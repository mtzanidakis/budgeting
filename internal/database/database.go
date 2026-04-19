package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattn/go-sqlite3"
)

type DB struct {
	*sql.DB
}

func init() {
	// Register custom SQLite driver with Unicode-aware LOWER function
	sql.Register("sqlite3_unicode",
		&sqlite3.SQLiteDriver{
			ConnectHook: func(conn *sqlite3.SQLiteConn) error {
				// Register LOWER_UNICODE function that properly handles Greek/Unicode
				return conn.RegisterFunc("lower_unicode", func(s string) string {
					return strings.ToLower(s)
				}, true)
			},
		})
}

func New(dbPath string) (*DB, error) {
	// Create the directory if it doesn't exist
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite3_unicode", dbPath+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{db}, nil
}

func (db *DB) Migrate() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password TEXT NOT NULL,
			name TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS categories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			description TEXT NOT NULL,
			action_type TEXT NOT NULL CHECK(action_type IN ('income', 'expense')),
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS actions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			type TEXT NOT NULL CHECK(type IN ('income', 'expense')),
			date DATE NOT NULL,
			description TEXT NOT NULL,
			amount REAL NOT NULL,
			category_id INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE SET NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_actions_user_id ON actions(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_actions_date ON actions(date)`,
		`CREATE INDEX IF NOT EXISTS idx_actions_type ON actions(type)`,
		`CREATE INDEX IF NOT EXISTS idx_categories_action_type ON categories(action_type)`,
		`CREATE TABLE IF NOT EXISTS api_tokens (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			token_hash TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			last_used_at DATETIME,
			expires_at DATETIME,
			deleted_at DATETIME,
			created_at DATETIME NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_api_tokens_user ON api_tokens(user_id) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_api_tokens_hash ON api_tokens(token_hash) WHERE deleted_at IS NULL`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	// Add category_id column to existing actions table if it doesn't exist
	var columnExists bool
	err := db.QueryRow(`
		SELECT COUNT(*) > 0
		FROM pragma_table_info('actions')
		WHERE name='category_id'
	`).Scan(&columnExists)

	if err != nil {
		return fmt.Errorf("failed to check for category_id column: %w", err)
	}

	if !columnExists {
		if _, err := db.Exec(`ALTER TABLE actions ADD COLUMN category_id INTEGER REFERENCES categories(id) ON DELETE SET NULL`); err != nil {
			return fmt.Errorf("failed to add category_id column: %w", err)
		}

		// Create index for the new column
		if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_actions_category_id ON actions(category_id)`); err != nil {
			return fmt.Errorf("failed to create category_id index: %w", err)
		}
	}

	return nil
}
