package auth

import (
	"testing"
)

func TestHashPassword(t *testing.T) {
	password := "test-password-123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if hash == "" {
		t.Fatal("Hash is empty")
	}

	if hash == password {
		t.Fatal("Hash should not equal password")
	}
}

func TestComparePassword(t *testing.T) {
	password := "test-password-123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	// Test correct password
	if err := ComparePassword(hash, password); err != nil {
		t.Fatalf("ComparePassword failed for correct password: %v", err)
	}

	// Test incorrect password
	if err := ComparePassword(hash, "wrong-password"); err == nil {
		t.Fatal("ComparePassword should fail for incorrect password")
	}
}

func TestGenerateRandomPassword(t *testing.T) {
	// Test with length < 6
	password, err := GenerateRandomPassword(3)
	if err != nil {
		t.Fatalf("GenerateRandomPassword failed: %v", err)
	}

	if len(password) < 6 {
		t.Fatalf("Password length should be at least 6, got %d", len(password))
	}

	// Test with length >= 6
	password, err = GenerateRandomPassword(20)
	if err != nil {
		t.Fatalf("GenerateRandomPassword failed: %v", err)
	}

	if len(password) != 20 {
		t.Fatalf("Password length should be 20, got %d", len(password))
	}

	// Test uniqueness
	password2, err := GenerateRandomPassword(20)
	if err != nil {
		t.Fatalf("GenerateRandomPassword failed: %v", err)
	}

	if password == password2 {
		t.Fatal("Generated passwords should be unique")
	}
}
