package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/mtzanidakis/budgeting/internal/models"
)

// Session timestamps are always stored in UTC so that SQL comparisons on the
// serialized DATETIME strings stay consistent across DST/timezone changes.

// CreateSession inserts a new session record and returns its id.
func (db *DB) CreateSession(userID int64, tokenHash string, createdAt, expiresAt time.Time) (int64, error) {
	createdAt = createdAt.UTC()
	result, err := db.Exec(
		`INSERT INTO sessions (user_id, token_hash, created_at, last_used_at, expires_at) VALUES (?, ?, ?, ?, ?)`,
		userID, tokenHash, createdAt, createdAt, expiresAt.UTC(),
	)
	if err != nil {
		return 0, fmt.Errorf("failed to create session: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get session id: %w", err)
	}
	return id, nil
}

// GetSessionByHash returns the session matching tokenHash, joined with the
// owning user's username. It returns (nil, nil) when no session matches.
// Expiry is not checked here; callers decide what to do with expired sessions.
func (db *DB) GetSessionByHash(tokenHash string) (*models.Session, error) {
	var s models.Session
	err := db.QueryRow(
		`SELECT s.id, s.user_id, u.username, s.created_at, s.last_used_at, s.expires_at
		 FROM sessions s
		 JOIN users u ON u.id = s.user_id
		 WHERE s.token_hash = ?`,
		tokenHash,
	).Scan(&s.ID, &s.UserID, &s.Username, &s.CreatedAt, &s.LastUsedAt, &s.ExpiresAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	return &s, nil
}

// RenewSession updates last_used_at and slides expires_at forward.
func (db *DB) RenewSession(id int64, lastUsedAt, expiresAt time.Time) error {
	_, err := db.Exec(
		`UPDATE sessions SET last_used_at = ?, expires_at = ? WHERE id = ?`,
		lastUsedAt.UTC(), expiresAt.UTC(), id,
	)
	if err != nil {
		return fmt.Errorf("failed to renew session: %w", err)
	}
	return nil
}

// DeleteSessionByHash removes the session matching tokenHash (logout).
func (db *DB) DeleteSessionByHash(tokenHash string) error {
	if _, err := db.Exec(`DELETE FROM sessions WHERE token_hash = ?`, tokenHash); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

// DeleteExpiredSessions removes sessions whose expires_at is at or before now.
// Returns the number of sessions removed.
func (db *DB) DeleteExpiredSessions(now time.Time) (int64, error) {
	result, err := db.Exec(`DELETE FROM sessions WHERE expires_at <= ?`, now.UTC())
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired sessions: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}
	return n, nil
}
