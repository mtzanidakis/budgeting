package database

import (
	"fmt"
	"strings"
	"time"

	"github.com/manolis/budgeting/internal/models"
)

type ActionFilters struct {
	Username  string
	Type      string
	DateFrom  string
	DateTo    string
	Limit     int
}

func (db *DB) CreateAction(userID int64, actionType models.ActionType, date, description string, amount float64) (*models.Action, error) {
	now := time.Now()
	result, err := db.Exec(
		"INSERT INTO actions (user_id, type, date, description, amount, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		userID, actionType, date, description, amount, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create action: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get action id: %w", err)
	}

	return &models.Action{
		ID:          id,
		UserID:      userID,
		Type:        actionType,
		Date:        date,
		Description: description,
		Amount:      amount,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func (db *DB) ListActions(filters ActionFilters) ([]*models.Action, error) {
	query := "SELECT id, user_id, type, date, description, amount, created_at, updated_at FROM actions"
	var conditions []string
	var args []interface{}

	if filters.Username != "" {
		conditions = append(conditions, "user_id = (SELECT id FROM users WHERE username = ?)")
		args = append(args, filters.Username)
	}

	if filters.Type != "" {
		conditions = append(conditions, "type = ?")
		args = append(args, filters.Type)
	}

	if filters.DateFrom != "" {
		conditions = append(conditions, "date >= ?")
		args = append(args, filters.DateFrom)
	}

	if filters.DateTo != "" {
		conditions = append(conditions, "date <= ?")
		args = append(args, filters.DateTo)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY date DESC, created_at DESC"

	if filters.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filters.Limit)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list actions: %w", err)
	}
	defer rows.Close()

	var actions []*models.Action
	for rows.Next() {
		var action models.Action
		if err := rows.Scan(
			&action.ID, &action.UserID, &action.Type, &action.Date,
			&action.Description, &action.Amount, &action.CreatedAt, &action.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan action: %w", err)
		}
		actions = append(actions, &action)
	}

	return actions, nil
}
