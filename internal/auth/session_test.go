package auth

import (
	"testing"
)

func TestSessionStore(t *testing.T) {
	store := NewSessionStore()

	// Test Create
	token, err := store.Create(1, "testuser")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if token == "" {
		t.Fatal("Token is empty")
	}

	// Test Get
	session, ok := store.Get(token)
	if !ok {
		t.Fatal("Get failed to retrieve session")
	}

	if session.UserID != 1 {
		t.Fatalf("Expected UserID 1, got %d", session.UserID)
	}

	if session.Username != "testuser" {
		t.Fatalf("Expected Username 'testuser', got '%s'", session.Username)
	}

	// Test Get with invalid token
	_, ok = store.Get("invalid-token")
	if ok {
		t.Fatal("Get should fail for invalid token")
	}

	// Test Delete
	store.Delete(token)

	_, ok = store.Get(token)
	if ok {
		t.Fatal("Session should be deleted")
	}
}

func TestSessionStoreUniqueness(t *testing.T) {
	store := NewSessionStore()

	token1, err := store.Create(1, "user1")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	token2, err := store.Create(2, "user2")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if token1 == token2 {
		t.Fatal("Tokens should be unique")
	}

	// Verify each session has correct data
	session1, ok := store.Get(token1)
	if !ok || session1.UserID != 1 {
		t.Fatal("Session 1 data is incorrect")
	}

	session2, ok := store.Get(token2)
	if !ok || session2.UserID != 2 {
		t.Fatal("Session 2 data is incorrect")
	}
}
