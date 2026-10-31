package database

import (
	"os"
	"testing"
	"time"

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

func TestSessions(t *testing.T) {
	db := setupTestDB(t)

	password, _ := auth.HashPassword("test-password")
	user, err := db.CreateUser("sessuser", password, "Session User")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	other, err := db.CreateUser("otheruser", password, "Other User")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	now := time.Now()
	id1, err := db.CreateSession(user.ID, "hash1", now, now.Add(time.Hour))
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	id2, err := db.CreateSession(user.ID, "hash2", now, now.Add(time.Hour))
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	if _, err := db.CreateSession(other.ID, "hash3", now, now.Add(time.Hour)); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Duplicate token hash must be rejected.
	if _, err := db.CreateSession(user.ID, "hash1", now, now.Add(time.Hour)); err == nil {
		t.Fatal("Expected duplicate token hash to fail")
	}

	s, err := db.GetSessionByHash("hash1")
	if err != nil || s == nil {
		t.Fatalf("GetSessionByHash failed: %v", err)
	}
	if s.ID != id1 || s.UserID != user.ID || s.Username != "sessuser" {
		t.Fatalf("Unexpected session: %+v", s)
	}
	if !s.ExpiresAt.Equal(now.Add(time.Hour).UTC()) {
		t.Fatalf("ExpiresAt round-trip mismatch: want %v got %v", now.Add(time.Hour).UTC(), s.ExpiresAt)
	}

	if s, err := db.GetSessionByHash("missing"); err != nil || s != nil {
		t.Fatalf("Expected (nil, nil) for missing session, got %v, %v", s, err)
	}

	// Renew
	later := now.Add(10 * time.Minute)
	if err := db.RenewSession(id1, later, later.Add(48*time.Hour)); err != nil {
		t.Fatalf("RenewSession failed: %v", err)
	}
	s, _ = db.GetSessionByHash("hash1")
	if !s.LastUsedAt.Equal(later.UTC()) || !s.ExpiresAt.Equal(later.Add(48*time.Hour).UTC()) {
		t.Fatalf("Renew not persisted: %+v", s)
	}

	// Delete all of a user's sessions except the current one.
	n, err := db.DeleteUserSessions(user.ID, id1)
	if err != nil || n != 1 {
		t.Fatalf("DeleteUserSessions: n=%d err=%v", n, err)
	}
	if s, _ := db.GetSessionByHash("hash2"); s != nil {
		t.Fatalf("Session %d should have been revoked", id2)
	}
	if s, _ := db.GetSessionByHash("hash1"); s == nil {
		t.Fatal("Current session should be kept")
	}
	if s, _ := db.GetSessionByHash("hash3"); s == nil {
		t.Fatal("Other user's session should be untouched")
	}

	// Delete expired: hash3 expires in 1h, hash1 in ~48h.
	n, err = db.DeleteExpiredSessions(now.Add(2 * time.Hour))
	if err != nil || n != 1 {
		t.Fatalf("DeleteExpiredSessions: n=%d err=%v", n, err)
	}
	if s, _ := db.GetSessionByHash("hash3"); s != nil {
		t.Fatal("Expired session should be removed")
	}

	// Logout
	if err := db.DeleteSessionByHash("hash1"); err != nil {
		t.Fatalf("DeleteSessionByHash failed: %v", err)
	}
	if s, _ := db.GetSessionByHash("hash1"); s != nil {
		t.Fatal("Session should be deleted")
	}

	// Cascade on user delete
	if _, err := db.CreateSession(other.ID, "hash4", now, now.Add(time.Hour)); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	if err := db.DeleteUser("otheruser"); err != nil {
		t.Fatalf("DeleteUser failed: %v", err)
	}
	if s, _ := db.GetSessionByHash("hash4"); s != nil {
		t.Fatal("Sessions should be removed when the user is deleted")
	}
}
