package models

import (
	"time"
)

type Category struct {
	ID          int64      `json:"id"`
	Description string     `json:"description"`
	ActionType  ActionType `json:"action_type"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
