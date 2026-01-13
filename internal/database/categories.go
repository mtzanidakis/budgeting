package database

import (
	"fmt"
	"time"

	"github.com/manolis/budgeting/internal/models"
)

func (db *DB) CreateCategory(description string, actionType models.ActionType) (*models.Category, error) {
	now := time.Now()
	result, err := db.Exec(
		"INSERT INTO categories (description, action_type, created_at, updated_at) VALUES (?, ?, ?, ?)",
		description, actionType, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create category: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get category id: %w", err)
	}

	return &models.Category{
		ID:          id,
		Description: description,
		ActionType:  actionType,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func (db *DB) ListCategories(actionType string) ([]*models.Category, error) {
	query := "SELECT id, description, action_type, created_at, updated_at FROM categories"
	args := []interface{}{}

	if actionType != "" {
		query += " WHERE action_type = ?"
		args = append(args, actionType)
	}

	query += " ORDER BY description ASC"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list categories: %w", err)
	}
	defer rows.Close()

	var categories []*models.Category
	for rows.Next() {
		var category models.Category
		if err := rows.Scan(
			&category.ID, &category.Description, &category.ActionType,
			&category.CreatedAt, &category.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}
		categories = append(categories, &category)
	}

	return categories, nil
}

func (db *DB) GetCategoryByID(categoryID int64) (*models.Category, error) {
	var category models.Category
	err := db.QueryRow(
		"SELECT id, description, action_type, created_at, updated_at FROM categories WHERE id = ?",
		categoryID,
	).Scan(
		&category.ID, &category.Description, &category.ActionType,
		&category.CreatedAt, &category.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get category: %w", err)
	}

	return &category, nil
}

func (db *DB) UpdateCategory(categoryID int64, description string, actionType models.ActionType) (*models.Category, error) {
	now := time.Now()
	result, err := db.Exec(
		"UPDATE categories SET description = ?, action_type = ?, updated_at = ? WHERE id = ?",
		description, actionType, now, categoryID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update category: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return nil, fmt.Errorf("category not found")
	}

	return &models.Category{
		ID:          categoryID,
		Description: description,
		ActionType:  actionType,
		UpdatedAt:   now,
	}, nil
}

func (db *DB) DeleteCategory(categoryID int64) error {
	result, err := db.Exec(
		"DELETE FROM categories WHERE id = ?",
		categoryID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete category: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("category not found")
	}

	return nil
}
