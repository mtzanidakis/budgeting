package database

import (
	"os"
	"testing"

	"github.com/mtzanidakis/budgeting/internal/auth"
	"github.com/mtzanidakis/budgeting/internal/models"
)

func setupTestDB(t *testing.T) *DB {
	dbPath := "./test.db"

	// Remove existing test database
	_ = os.Remove(dbPath)

	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
		_ = os.Remove(dbPath)
	})

	return db
}

func TestUserCRUD(t *testing.T) {
	db := setupTestDB(t)

	// Create user
	password, _ := auth.HashPassword("test-password")
	user, err := db.CreateUser("testuser", password, "Test User")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	if user.ID == 0 {
		t.Fatal("User ID should not be 0")
	}

	// Get user by username
	retrieved, err := db.GetUserByUsername("testuser")
	if err != nil {
		t.Fatalf("GetUserByUsername failed: %v", err)
	}

	if retrieved.Name != "Test User" {
		t.Fatalf("Expected name 'Test User', got '%s'", retrieved.Name)
	}

	// Get user by ID
	retrieved, err = db.GetUserByID(user.ID)
	if err != nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}

	if retrieved.Username != "testuser" {
		t.Fatalf("Expected username 'testuser', got '%s'", retrieved.Username)
	}

	// Update user
	newName := "Updated Name"
	err = db.UpdateUser("testuser", nil, &newName)
	if err != nil {
		t.Fatalf("UpdateUser failed: %v", err)
	}

	retrieved, _ = db.GetUserByUsername("testuser")
	if retrieved.Name != "Updated Name" {
		t.Fatalf("Expected updated name 'Updated Name', got '%s'", retrieved.Name)
	}

	// List users
	users, err := db.ListUsers()
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}

	if len(users) != 1 {
		t.Fatalf("Expected 1 user, got %d", len(users))
	}

	// Delete user
	err = db.DeleteUser("testuser")
	if err != nil {
		t.Fatalf("DeleteUser failed: %v", err)
	}

	_, err = db.GetUserByUsername("testuser")
	if err == nil {
		t.Fatal("User should be deleted")
	}
}

func TestActionsCRUD(t *testing.T) {
	db := setupTestDB(t)

	// Create user first
	password, _ := auth.HashPassword("test-password")
	user, _ := db.CreateUser("testuser", password, "Test User")

	// Create action
	action, err := db.CreateAction(user.ID, models.ActionTypeExpense, "2024-01-01", "Test expense", 100.50, nil)
	if err != nil {
		t.Fatalf("CreateAction failed: %v", err)
	}

	if action.ID == 0 {
		t.Fatal("Action ID should not be 0")
	}

	// List actions
	actions, err := db.ListActions(ActionFilters{Limit: 20})
	if err != nil {
		t.Fatalf("ListActions failed: %v", err)
	}

	if len(actions) != 1 {
		t.Fatalf("Expected 1 action, got %d", len(actions))
	}

	// Filter by username
	actions, err = db.ListActions(ActionFilters{Username: "testuser", Limit: 20})
	if err != nil {
		t.Fatalf("ListActions with username filter failed: %v", err)
	}

	if len(actions) != 1 {
		t.Fatalf("Expected 1 action, got %d", len(actions))
	}

	// Filter by type
	actions, err = db.ListActions(ActionFilters{Type: "expense", Limit: 20})
	if err != nil {
		t.Fatalf("ListActions with type filter failed: %v", err)
	}

	if len(actions) != 1 {
		t.Fatalf("Expected 1 action, got %d", len(actions))
	}

	// Filter by wrong type
	actions, err = db.ListActions(ActionFilters{Type: "income", Limit: 20})
	if err != nil {
		t.Fatalf("ListActions with type filter failed: %v", err)
	}

	if len(actions) != 0 {
		t.Fatalf("Expected 0 actions, got %d", len(actions))
	}
}

func TestUpdatePreservesCreatedAt(t *testing.T) {
	db := setupTestDB(t)

	password, _ := auth.HashPassword("test-password")
	user, _ := db.CreateUser("testuser", password, "Test User")

	category, err := db.CreateCategory("Groceries", models.ActionTypeExpense)
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}

	updatedCategory, err := db.UpdateCategory(category.ID, "Food", models.ActionTypeExpense)
	if err != nil {
		t.Fatalf("UpdateCategory failed: %v", err)
	}

	if updatedCategory.CreatedAt.IsZero() {
		t.Fatal("UpdateCategory returned zero CreatedAt")
	}

	if updatedCategory.Description != "Food" {
		t.Fatalf("Expected description 'Food', got '%s'", updatedCategory.Description)
	}

	action, err := db.CreateAction(user.ID, models.ActionTypeExpense, "2024-01-01", "Test expense", 100.50, &category.ID)
	if err != nil {
		t.Fatalf("CreateAction failed: %v", err)
	}

	updatedAction, err := db.UpdateAction(action.ID, user.ID, models.ActionTypeExpense, "2024-01-02", "Updated expense", 42, &category.ID)
	if err != nil {
		t.Fatalf("UpdateAction failed: %v", err)
	}

	if updatedAction.CreatedAt.IsZero() {
		t.Fatal("UpdateAction returned zero CreatedAt")
	}

	if !updatedAction.CreatedAt.Equal(action.CreatedAt) {
		t.Fatalf("Expected CreatedAt %v, got %v", action.CreatedAt, updatedAction.CreatedAt)
	}

	if updatedAction.Description != "Updated expense" || updatedAction.Amount != 42 || updatedAction.Date != "2024-01-02" {
		t.Fatalf("UpdateAction returned stale fields: %+v", updatedAction)
	}

	if _, err := db.UpdateAction(action.ID, user.ID+1, models.ActionTypeExpense, "2024-01-02", "x", 1, &category.ID); err == nil {
		t.Fatal("UpdateAction should fail for a non-owner")
	}
}
