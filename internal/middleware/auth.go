package middleware

import (
	"context"
	"net/http"

	"github.com/manolis/budgeting/internal/auth"
)

type contextKey string

const (
	SessionContextKey contextKey = "session"
)

func Auth(sessionStore *auth.SessionStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetSession(r *http.Request) (*auth.Session, bool) {
	session, ok := r.Context().Value(SessionContextKey).(*auth.Session)
	return session, ok
}
