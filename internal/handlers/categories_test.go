package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mtzanidakis/budgeting/internal/database"
	"github.com/mtzanidakis/budgeting/internal/models"
)

func TestCategoryUpdateResponseIncludesCreatedAt(t *testing.T) {
	db, err := database.New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	category, err := db.CreateCategory("Groceries", models.ActionTypeExpense)
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}

	r := chi.NewRouter()
	r.Put("/categories/{id}", NewCategoriesHandler(db).Update)

	body := strings.NewReader(`{"description":"Food","action_type":"expense"}`)
	req := httptest.NewRequest(http.MethodPut, "/categories/"+strconv.FormatInt(category.ID, 10), body)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp CategoryResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	createdAt, err := time.Parse(time.RFC3339, resp.CreatedAt)
	if err != nil {
		t.Fatalf("created_at %q is not RFC3339: %v", resp.CreatedAt, err)
	}

	if createdAt.Year() < 2000 {
		t.Fatalf("created_at looks like a zero value: %q", resp.CreatedAt)
	}

	if resp.Description != "Food" {
		t.Fatalf("Expected description 'Food', got '%s'", resp.Description)
	}
}
