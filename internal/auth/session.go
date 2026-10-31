package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/mtzanidakis/budgeting/internal/models"
)

const (
	// SessionCookieName is the name of the HTTP-only session cookie.
	SessionCookieName = "session"

	// SessionTTL is how long a session stays valid without activity. Active
	// sessions slide forward by this amount (see SessionRenewInterval).
	SessionTTL = 30 * 24 * time.Hour

	// SessionRenewInterval is the minimum time between expiry extensions for
	// the same session, so we don't write to the DB on every request.
	SessionRenewInterval = time.Hour
)

// ErrSessionNotFound is returned by SessionStore.Get when the token does not
// match a live session (unknown, revoked or expired).
var ErrSessionNotFound = errors.New("session not found")

// Session is the authenticated identity attached to a request context.
// For API-token requests it is synthesized and ID is 0.
type Session struct {
	ID         int64
	UserID     int64
	Username   string
	CreatedAt  time.Time
	LastUsedAt time.Time
	ExpiresAt  time.Time
}

// SessionRepository is the persistence layer used by SessionStore.
// It is implemented by *database.DB.
type SessionRepository interface {
	CreateSession(userID int64, tokenHash string, createdAt, expiresAt time.Time) (int64, error)
	GetSessionByHash(tokenHash string) (*models.Session, error)
	RenewSession(id int64, lastUsedAt, expiresAt time.Time) error
	DeleteSessionByHash(tokenHash string) error
	DeleteExpiredSessions(now time.Time) (int64, error)
}

// SessionStore manages persistent browser sessions. Raw tokens are only ever
// held in the client cookie; the repository stores their SHA-256 hash.
type SessionStore struct {
	repo SessionRepository
	now  func() time.Time
}

func NewSessionStore(repo SessionRepository) *SessionStore {
	return &SessionStore{repo: repo, now: time.Now}
}

// Create starts a new session for the user and returns the raw token to put
// in the cookie together with the stored session.
func (s *SessionStore) Create(userID int64, username string) (string, *Session, error) {
	token, err := generateSessionToken()
	if err != nil {
		return "", nil, err
	}

	now := s.now()
	expiresAt := now.Add(SessionTTL)
	id, err := s.repo.CreateSession(userID, hashToken(token), now, expiresAt)
	if err != nil {
		return "", nil, fmt.Errorf("failed to store session: %w", err)
	}

	return token, &Session{
		ID:         id,
		UserID:     userID,
		Username:   username,
		CreatedAt:  now,
		LastUsedAt: now,
		ExpiresAt:  expiresAt,
	}, nil
}

// Get looks up a session by raw token. It returns ErrSessionNotFound for
// unknown or expired tokens (expired sessions are deleted) and any other
// error for storage failures, so callers can tell "log in again" apart from
// "try again later".
func (s *SessionStore) Get(token string) (*Session, error) {
	if token == "" {
		return nil, ErrSessionNotFound
	}

	hash := hashToken(token)
	rec, err := s.repo.GetSessionByHash(hash)
	if err != nil {
		return nil, fmt.Errorf("failed to look up session: %w", err)
	}
	if rec == nil {
		return nil, ErrSessionNotFound
	}

	if !rec.ExpiresAt.After(s.now()) {
		if err := s.repo.DeleteSessionByHash(hash); err != nil {
			slog.Warn("failed to delete expired session", "session_id", rec.ID, "error", err)
		}
		return nil, ErrSessionNotFound
	}

	return &Session{
		ID:         rec.ID,
		UserID:     rec.UserID,
		Username:   rec.Username,
		CreatedAt:  rec.CreatedAt,
		LastUsedAt: rec.LastUsedAt,
		ExpiresAt:  rec.ExpiresAt,
	}, nil
}

// Renew slides the session's expiry forward if it was last renewed more than
// SessionRenewInterval ago. It updates sess in place and reports whether a
// renewal happened (in which case the cookie should be re-issued).
func (s *SessionStore) Renew(sess *Session) (bool, error) {
	now := s.now()
	if now.Sub(sess.LastUsedAt) < SessionRenewInterval {
		return false, nil
	}

	expiresAt := now.Add(SessionTTL)
	if err := s.repo.RenewSession(sess.ID, now, expiresAt); err != nil {
		return false, err
	}
	sess.LastUsedAt = now
	sess.ExpiresAt = expiresAt
	return true, nil
}

// Delete removes the session identified by the raw token (logout).
func (s *SessionStore) Delete(token string) error {
	if token == "" {
		return nil
	}
	return s.repo.DeleteSessionByHash(hashToken(token))
}

// DeleteExpired removes all expired sessions and returns how many were removed.
func (s *SessionStore) DeleteExpired() (int64, error) {
	return s.repo.DeleteExpiredSessions(s.now())
}

// RunCleanup deletes expired sessions immediately and then every interval
// until ctx is cancelled. Intended to run in its own goroutine.
func (s *SessionStore) RunCleanup(ctx context.Context, interval time.Duration) {
	cleanup := func() {
		n, err := s.DeleteExpired()
		if err != nil {
			slog.Warn("failed to clean up expired sessions", "error", err)
			return
		}
		if n > 0 {
			slog.Info("cleaned up expired sessions", "count", n)
		}
	}

	cleanup()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cleanup()
		}
	}
}

// NewSessionCookie builds the session cookie for token, expiring at expiresAt.
func NewSessionCookie(token string, expiresAt time.Time) *http.Cookie {
	return &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteStrictMode,
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
	}
}

// ClearSessionCookie builds a cookie that removes the session cookie.
func ClearSessionCookie() *http.Cookie {
	return &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	}
}

func generateSessionToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate session token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}
