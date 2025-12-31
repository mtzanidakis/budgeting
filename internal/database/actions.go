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
	Offset    int
}

type MonthlySummary struct {
	Year    int     `json:"year"`
	Month   int     `json:"month"`
	Income  float64 `json:"income"`
	Expense float64 `json:"expense"`
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

		if filters.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, filters.Offset)
		}
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

func (db *DB) CountActions(filters ActionFilters) (int, error) {
	query := "SELECT COUNT(*) FROM actions"
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

	var count int
	err := db.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count actions: %w", err)
	}

	return count, nil
}

func (db *DB) GetMonthlySummary(year int) ([]MonthlySummary, error) {
	query := `
		SELECT
			CAST(strftime('%Y', date) AS INTEGER) as year,
			CAST(strftime('%m', date) AS INTEGER) as month,
			SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END) as income,
			SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END) as expense
		FROM actions
		WHERE CAST(strftime('%Y', date) AS INTEGER) = ?
		GROUP BY year, month
		ORDER BY month ASC
	`

	rows, err := db.Query(query, year)
	if err != nil {
		return nil, fmt.Errorf("failed to get monthly summary: %w", err)
	}
	defer rows.Close()

	var summaries []MonthlySummary
	for rows.Next() {
		var summary MonthlySummary
		if err := rows.Scan(&summary.Year, &summary.Month, &summary.Income, &summary.Expense); err != nil {
			return nil, fmt.Errorf("failed to scan monthly summary: %w", err)
		}
		summaries = append(summaries, summary)
	}

	return summaries, nil
}

func (db *DB) GetActionByID(actionID int64) (*models.Action, error) {
	var action models.Action
	err := db.QueryRow(
		"SELECT id, user_id, type, date, description, amount, created_at, updated_at FROM actions WHERE id = ?",
		actionID,
	).Scan(
		&action.ID, &action.UserID, &action.Type, &action.Date,
		&action.Description, &action.Amount, &action.CreatedAt, &action.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get action: %w", err)
	}

	return &action, nil
}

func (db *DB) UpdateAction(actionID, userID int64, actionType models.ActionType, date, description string, amount float64) (*models.Action, error) {
	now := time.Now()
	result, err := db.Exec(
		"UPDATE actions SET type = ?, date = ?, description = ?, amount = ?, updated_at = ? WHERE id = ? AND user_id = ?",
		actionType, date, description, amount, now, actionID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update action: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return nil, fmt.Errorf("action not found or not owned by user")
	}

	return &models.Action{
		ID:          actionID,
		UserID:      userID,
		Type:        actionType,
		Date:        date,
		Description: description,
		Amount:      amount,
		UpdatedAt:   now,
	}, nil
}

func (db *DB) DeleteAction(actionID, userID int64) error {
	result, err := db.Exec(
		"DELETE FROM actions WHERE id = ? AND user_id = ?",
		actionID, userID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete action: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("action not found or not owned by user")
	}

	return nil
}
