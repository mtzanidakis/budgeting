package auth

import (
	"net/http"
	"testing"
	"time"

	"github.com/mtzanidakis/budgeting/internal/models"
)

// fakeSessionRepo is an in-memory SessionRepository for unit tests.
type fakeSessionRepo struct {
	nextID   int64
	sessions map[string]*models.Session // keyed by token hash
	renews   int
}

func newFakeSessionRepo() *fakeSessionRepo {
	return &fakeSessionRepo{sessions: make(map[string]*models.Session)}
}

func (f *fakeSessionRepo) CreateSession(userID int64, tokenHash string, createdAt, expiresAt time.Time) (int64, error) {
	f.nextID++
	f.sessions[tokenHash] = &models.Session{
		ID:         f.nextID,
		UserID:     userID,
		Username:   "user",
		TokenHash:  tokenHash,
		CreatedAt:  createdAt,
		LastUsedAt: createdAt,
		ExpiresAt:  expiresAt,
	}
	return f.nextID, nil
}

func (f *fakeSessionRepo) GetSessionByHash(tokenHash string) (*models.Session, error) {
	s, ok := f.sessions[tokenHash]
	if !ok {
		return nil, nil
	}
	c := *s
	return &c, nil
}

func (f *fakeSessionRepo) RenewSession(id int64, lastUsedAt, expiresAt time.Time) error {
	f.renews++
	for _, s := range f.sessions {
		if s.ID == id {
			s.LastUsedAt = lastUsedAt
			s.ExpiresAt = expiresAt
		}
	}
	return nil
}

func (f *fakeSessionRepo) DeleteSessionByHash(tokenHash string) error {
	delete(f.sessions, tokenHash)
	return nil
}

func (f *fakeSessionRepo) DeleteExpiredSessions(now time.Time) (int64, error) {
	var n int64
	for h, s := range f.sessions {
		if !s.ExpiresAt.After(now) {
			delete(f.sessions, h)
			n++
		}
	}
	return n, nil
}

func newTestStore() (*SessionStore, *fakeSessionRepo, *time.Time) {
	repo := newFakeSessionRepo()
	store := NewSessionStore(repo)
	clock := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return clock }
	return store, repo, &clock
}

func TestSessionStore(t *testing.T) {
	store, repo, clock := newTestStore()

	token, created, err := store.Create(1, "testuser")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if token == "" {
		t.Fatal("Token is empty")
	}
	if !created.ExpiresAt.Equal(clock.Add(SessionTTL)) {
		t.Fatalf("Expected expiry %v, got %v", clock.Add(SessionTTL), created.ExpiresAt)
	}

	// Raw token must never be stored, only its hash.
	if _, ok := repo.sessions[token]; ok {
		t.Fatal("Raw token stored in repository")
	}
	if _, ok := repo.sessions[hashToken(token)]; !ok {
		t.Fatal("Token hash not stored in repository")
	}

	session, ok := store.Get(token)
	if !ok {
		t.Fatal("Get failed to retrieve session")
	}
	if session.UserID != 1 || session.ID != created.ID {
		t.Fatalf("Unexpected session: %+v", session)
	}

	if _, ok := store.Get("invalid-token"); ok {
		t.Fatal("Get should fail for invalid token")
	}
	if _, ok := store.Get(""); ok {
		t.Fatal("Get should fail for empty token")
	}

	if err := store.Delete(token); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if _, ok := store.Get(token); ok {
		t.Fatal("Session should be deleted")
	}
}

func TestSessionStoreUniqueness(t *testing.T) {
	store, _, _ := newTestStore()

	token1, _, err := store.Create(1, "user1")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	token2, _, err := store.Create(2, "user2")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if token1 == token2 {
		t.Fatal("Tokens should be unique")
	}

	s1, ok := store.Get(token1)
	if !ok || s1.UserID != 1 {
		t.Fatal("Session 1 data is incorrect")
	}
	s2, ok := store.Get(token2)
	if !ok || s2.UserID != 2 {
		t.Fatal("Session 2 data is incorrect")
	}
}

func TestSessionStoreExpiry(t *testing.T) {
	store, repo, clock := newTestStore()

	token, _, err := store.Create(1, "user")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	*clock = clock.Add(SessionTTL - time.Second)
	if _, ok := store.Get(token); !ok {
		t.Fatal("Session should still be valid just before expiry")
	}

	*clock = clock.Add(time.Second)
	if _, ok := store.Get(token); ok {
		t.Fatal("Session should be expired at expires_at")
	}
	if len(repo.sessions) != 0 {
		t.Fatal("Expired session should be deleted on lookup")
	}
}

func TestSessionStoreRenew(t *testing.T) {
	store, repo, clock := newTestStore()

	token, _, err := store.Create(1, "user")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Within the renew interval: no DB write.
	*clock = clock.Add(SessionRenewInterval - time.Second)
	sess, _ := store.Get(token)
	renewed, err := store.Renew(sess)
	if err != nil || renewed {
		t.Fatalf("Expected no renewal, got renewed=%v err=%v", renewed, err)
	}
	if repo.renews != 0 {
		t.Fatal("Renew should not hit the repository within the interval")
	}

	// Past the renew interval: expiry slides forward.
	*clock = clock.Add(time.Second)
	sess, _ = store.Get(token)
	renewed, err = store.Renew(sess)
	if err != nil || !renewed {
		t.Fatalf("Expected renewal, got renewed=%v err=%v", renewed, err)
	}
	if !sess.ExpiresAt.Equal(clock.Add(SessionTTL)) {
		t.Fatalf("Expected expiry %v, got %v", clock.Add(SessionTTL), sess.ExpiresAt)
	}

	// The renewed expiry is persisted: the session survives past the original TTL.
	*clock = clock.Add(SessionTTL - time.Minute)
	if _, ok := store.Get(token); !ok {
		t.Fatal("Renewed session should outlive the original expiry")
	}
}

func TestSessionStoreDeleteExpired(t *testing.T) {
	store, repo, clock := newTestStore()

	if _, _, err := store.Create(1, "old"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	*clock = clock.Add(24 * time.Hour)
	fresh, _, err := store.Create(2, "fresh")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	*clock = clock.Add(SessionTTL - time.Hour)
	n, err := store.DeleteExpired()
	if err != nil {
		t.Fatalf("DeleteExpired failed: %v", err)
	}
	if n != 1 || len(repo.sessions) != 1 {
		t.Fatalf("Expected 1 expired session removed, got n=%d remaining=%d", n, len(repo.sessions))
	}
	if _, ok := store.Get(fresh); !ok {
		t.Fatal("Fresh session should remain")
	}
}

func TestSessionCookies(t *testing.T) {
	expires := time.Now().Add(SessionTTL)
	c := NewSessionCookie("tok", expires)
	if c.Name != SessionCookieName || c.Value != "tok" || !c.HttpOnly || c.SameSite != http.SameSiteStrictMode || c.Path != "/" {
		t.Fatalf("Unexpected session cookie: %+v", c)
	}
	if c.MaxAge < int(SessionTTL.Seconds())-5 || c.MaxAge > int(SessionTTL.Seconds()) {
		t.Fatalf("Unexpected MaxAge %d", c.MaxAge)
	}

	cleared := ClearSessionCookie()
	if cleared.Name != SessionCookieName || cleared.MaxAge >= 0 || cleared.Value != "" {
		t.Fatalf("Unexpected clear cookie: %+v", cleared)
	}
}
