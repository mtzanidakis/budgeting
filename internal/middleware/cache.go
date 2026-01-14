package middleware

import (
	"net/http"
	"strings"
)

// CacheControl middleware sets appropriate cache headers based on the request
func CacheControl(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if this is a versioned asset (has ?v= query parameter)
		if strings.Contains(r.URL.RawQuery, "v=") {
			// Versioned assets get long-term caching (1 year) with immutable flag
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			// Non-versioned assets get short-term caching (1 hour)
			w.Header().Set("Cache-Control", "public, max-age=3600")
		}

		next.ServeHTTP(w, r)
	})
}
