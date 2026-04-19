package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/mtzanidakis/budgeting/internal/auth"
	"github.com/mtzanidakis/budgeting/internal/database"
)

type contextKey string

const (
	SessionContextKey    contextKey = "session"
	AuthMethodContextKey contextKey = "auth_method"

	AuthMethodSession = "session"
	AuthMethodToken   = "token"

	// Minimum interval between last_used_at updates for the same token.
	lastUsedThrottle = time.Minute
)

// Auth authenticates a request via session cookie or Bearer API token.
// Successful auth injects the session and auth-method into the request context.
func Auth(sessionStore *auth.SessionStore, db *database.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if session, ok := authenticateToken(r, db); ok {
				ctx := context.WithValue(r.Context(), SessionContextKey, session)
				ctx = context.WithValue(ctx, AuthMethodContextKey, AuthMethodToken)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			cookie, err := r.Cookie("session")
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			session, ok := sessionStore.Get(cookie.Value)
			if !ok {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), SessionContextKey, session)
			ctx = context.WithValue(ctx, AuthMethodContextKey, AuthMethodSession)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireSessionAuth rejects requests that authenticated via API token.
// Use this for endpoints that should only be reachable from an interactive session,
// e.g. token management endpoints.
func RequireSessionAuth() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			method, _ := r.Context().Value(AuthMethodContextKey).(string)
			if method != AuthMethodSession {
				http.Error(w, "Forbidden: session authentication required", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func GetSession(r *http.Request) (*auth.Session, bool) {
	session, ok := r.Context().Value(SessionContextKey).(*auth.Session)
	return session, ok
}

// authenticateToken validates a Bearer API token. On success it constructs an
// in-memory Session (not registered in the SessionStore) and schedules a
// throttled last_used_at update.
func authenticateToken(r *http.Request, db *database.DB) (*auth.Session, bool) {
	header := r.Header.Get("Authorization")
	if header == "" {
		return nil, false
	}
	const bearer = "Bearer "
	if !strings.HasPrefix(header, bearer) {
		return nil, false
	}
	raw := strings.TrimSpace(header[len(bearer):])
	if !auth.IsAPITokenFormat(raw) {
		return nil, false
	}

	tokenRec, err := db.GetAPITokenByHash(auth.HashAPIToken(raw))
	if err != nil {
		return nil, false
	}

	user, err := db.GetUserByID(tokenRec.UserID)
	if err != nil {
		return nil, false
	}

	if tokenRec.LastUsedAt == nil || time.Since(*tokenRec.LastUsedAt) > lastUsedThrottle {
		tokenID := tokenRec.ID
		go func() {
			if err := db.UpdateAPITokenLastUsed(tokenID); err != nil {
				slog.Warn("failed to update api token last_used_at", "token_id", tokenID, "error", err)
			}
		}()
	}

	return &auth.Session{
		UserID:    user.ID,
		Username:  user.Username,
		CreatedAt: time.Now(),
	}, true
}
