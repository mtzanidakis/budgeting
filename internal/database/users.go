package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/manolis/budgeting/internal/models"
)

func (db *DB) CreateUser(username, password, name string) (*models.User, error) {
	now := time.Now()
	result, err := db.Exec(
		"INSERT INTO users (username, password, name, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		username, password, name, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get user id: %w", err)
	}

	return &models.User{
		ID:        id,
		Username:  username,
		Password:  password,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (db *DB) GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	err := db.QueryRow(
		"SELECT id, username, password, name, created_at, updated_at FROM users WHERE username = ?",
		username,
	).Scan(&user.ID, &user.Username, &user.Password, &user.Name, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

func (db *DB) GetUserByID(id int64) (*models.User, error) {
	var user models.User
	err := db.QueryRow(
		"SELECT id, username, password, name, created_at, updated_at FROM users WHERE id = ?",
		id,
	).Scan(&user.ID, &user.Username, &user.Password, &user.Name, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

func (db *DB) ListUsers() ([]*models.User, error) {
	rows, err := db.Query(
		"SELECT id, username, password, name, created_at, updated_at FROM users ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.ID, &user.Username, &user.Password, &user.Name, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, &user)
	}

	return users, nil
}

func (db *DB) UpdateUser(username string, password, name *string) error {
	user, err := db.GetUserByUsername(username)
	if err != nil {
		return err
	}

	now := time.Now()

	if password != nil && name != nil {
		_, err = db.Exec(
			"UPDATE users SET password = ?, name = ?, updated_at = ? WHERE id = ?",
			*password, *name, now, user.ID,
		)
	} else if password != nil {
		_, err = db.Exec(
			"UPDATE users SET password = ?, updated_at = ? WHERE id = ?",
			*password, now, user.ID,
		)
	} else if name != nil {
		_, err = db.Exec(
			"UPDATE users SET name = ?, updated_at = ? WHERE id = ?",
			*name, now, user.ID,
		)
	}

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

func (db *DB) DeleteUser(username string) error {
	result, err := db.Exec("DELETE FROM users WHERE username = ?", username)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}
