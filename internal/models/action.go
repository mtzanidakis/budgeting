package models

import (
	"time"
)

type ActionType string

const (
	ActionTypeIncome  ActionType = "income"
	ActionTypeExpense ActionType = "expense"
)

type Action struct {
	ID          int64      `json:"id"`
	UserID      int64      `json:"user_id"`
	Type        ActionType `json:"type"`
	Date        string     `json:"date"` // YYYY-MM-DD format
	Description string     `json:"description"`
	Amount      float64    `json:"amount"`
	CategoryID  *int64     `json:"category_id,omitempty"` // Optional foreign key to categories
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
