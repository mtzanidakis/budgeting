package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/manolis/budgeting/internal/models"
)

// CreateAPIToken inserts a new token record. The raw token is not stored; only tokenHash.
func (db *DB) CreateAPIToken(userID int64, name, tokenHash string, expiresAt *time.Time) (*models.APIToken, error) {
	now := time.Now()
	result, err := db.Exec(
		`INSERT INTO api_tokens (user_id, token_hash, name, expires_at, created_at) VALUES (?, ?, ?, ?, ?)`,
		userID, tokenHash, name, expiresAt, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create api token: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get api token id: %w", err)
	}

	return &models.APIToken{
		ID:        id,
		UserID:    userID,
		TokenHash: tokenHash,
		Name:      name,
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}, nil
}

// ListAPITokensByUser returns active (non-deleted) tokens for a user, newest first.
func (db *DB) ListAPITokensByUser(userID int64) ([]*models.APIToken, error) {
	rows, err := db.Query(
		`SELECT id, user_id, token_hash, name, last_used_at, expires_at, created_at
		 FROM api_tokens
		 WHERE user_id = ? AND deleted_at IS NULL
		 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list api tokens: %w", err)
	}
	defer rows.Close()

	var tokens []*models.APIToken
	for rows.Next() {
		var t models.APIToken
		if err := rows.Scan(&t.ID, &t.UserID, &t.TokenHash, &t.Name, &t.LastUsedAt, &t.ExpiresAt, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan api token: %w", err)
		}
		tokens = append(tokens, &t)
	}
	return tokens, nil
}

// GetAPITokenByHash returns the active, non-expired token matching the hash.
func (db *DB) GetAPITokenByHash(tokenHash string) (*models.APIToken, error) {
	var t models.APIToken
	err := db.QueryRow(
		`SELECT id, user_id, token_hash, name, last_used_at, expires_at, created_at
		 FROM api_tokens
		 WHERE token_hash = ? AND deleted_at IS NULL`,
		tokenHash,
	).Scan(&t.ID, &t.UserID, &t.TokenHash, &t.Name, &t.LastUsedAt, &t.ExpiresAt, &t.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("api token not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get api token: %w", err)
	}

	if t.ExpiresAt != nil && !t.ExpiresAt.After(time.Now()) {
		return nil, fmt.Errorf("api token expired")
	}

	return &t, nil
}

// SoftDeleteAPIToken marks the token deleted, scoped to the owning user.
func (db *DB) SoftDeleteAPIToken(id, userID int64) error {
	result, err := db.Exec(
		`UPDATE api_tokens SET deleted_at = ? WHERE id = ? AND user_id = ? AND deleted_at IS NULL`,
		time.Now(), id, userID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete api token: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("api token not found")
	}
	return nil
}

// UpdateAPITokenLastUsed sets last_used_at = now() for the given id.
func (db *DB) UpdateAPITokenLastUsed(id int64) error {
	_, err := db.Exec(`UPDATE api_tokens SET last_used_at = ? WHERE id = ?`, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update api token last_used_at: %w", err)
	}
	return nil
}
