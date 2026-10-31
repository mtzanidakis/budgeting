package middleware

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/mtzanidakis/budgeting/internal/auth"
	"github.com/mtzanidakis/budgeting/internal/database"
)

func setupAuthTest(t *testing.T) (*database.DB, *auth.SessionStore, http.Handler) {
	t.Helper()
	db, err := database.New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	store := auth.NewSessionStore(db)
	handler := Auth(store, db)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s, ok := GetSession(r)
		if !ok {
			t.Error("session missing from context")
		}
		w.Header().Set("X-User", s.Username)
		w.WriteHeader(http.StatusOK)
	}))
	return db, store, handler
}

func doRequest(h http.Handler, cookie string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	if cookie != "" {
		req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: cookie})
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func sessionCookie(rec *httptest.ResponseRecorder) *http.Cookie {
	for _, c := range rec.Result().Cookies() {
		if c.Name == auth.SessionCookieName {
			return c
		}
	}
	return nil
}

func TestAuthSessionCookie(t *testing.T) {
	db, store, h := setupAuthTest(t)
	user, err := db.CreateUser("alice", "hash", "Alice")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	if rec := doRequest(h, ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("no cookie: expected 401, got %d", rec.Code)
	}
	if rec := doRequest(h, "bogus"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("unknown cookie: expected 401, got %d", rec.Code)
	}

	token, sess, err := store.Create(user.ID, user.Username)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Fresh session: authenticated, no renewal, no cookie re-issued.
	rec := doRequest(h, token)
	if rec.Code != http.StatusOK || rec.Header().Get("X-User") != "alice" {
		t.Fatalf("valid cookie: expected 200 for alice, got %d %q", rec.Code, rec.Header().Get("X-User"))
	}
	if c := sessionCookie(rec); c != nil {
		t.Fatalf("fresh session should not re-issue cookie, got %+v", c)
	}

	// Last renewal over an hour ago: expiry slides forward and cookie is re-issued.
	past := time.Now().Add(-2 * time.Hour)
	if err := db.RenewSession(sess.ID, past, past.Add(auth.SessionTTL)); err != nil {
		t.Fatalf("RenewSession failed: %v", err)
	}
	rec = doRequest(h, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("stale session: expected 200, got %d", rec.Code)
	}
	c := sessionCookie(rec)
	if c == nil || c.Value != token {
		t.Fatalf("renewed session should re-issue the same token cookie, got %+v", c)
	}
	if c.MaxAge < int(auth.SessionTTL.Seconds())-60 {
		t.Fatalf("re-issued cookie MaxAge too small: %d", c.MaxAge)
	}
	renewed, err := store.Get(token)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if time.Until(renewed.ExpiresAt) < auth.SessionTTL-time.Minute {
		t.Fatalf("expiry not extended in DB: %v", renewed.ExpiresAt)
	}

	// Expired session: rejected.
	if err := db.RenewSession(sess.ID, past, time.Now().Add(-time.Second)); err != nil {
		t.Fatalf("RenewSession failed: %v", err)
	}
	if rec := doRequest(h, token); rec.Code != http.StatusUnauthorized {
		t.Fatalf("expired session: expected 401, got %d", rec.Code)
	}
}

func TestAuthStorageErrorIsNotUnauthorized(t *testing.T) {
	db, _, h := setupAuthTest(t)
	_ = db.Close()

	if rec := doRequest(h, "some-token"); rec.Code != http.StatusInternalServerError {
		t.Fatalf("DB failure: expected 500 (not a logout), got %d", rec.Code)
	}
}
