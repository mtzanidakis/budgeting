package models

import "time"

// Session is a persisted browser login session. Only the SHA-256 hash of the
// session token is stored; the raw token lives in the user's cookie.
type Session struct {
	ID         int64
	UserID     int64
	Username   string
	TokenHash  string
	CreatedAt  time.Time
	LastUsedAt time.Time
	ExpiresAt  time.Time
}
