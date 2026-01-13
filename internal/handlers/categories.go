package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/manolis/budgeting/internal/database"
	"github.com/manolis/budgeting/internal/models"
)

type CategoriesHandler struct {
	db *database.DB
}

func NewCategoriesHandler(db *database.DB) *CategoriesHandler {
	return &CategoriesHandler{db: db}
}

type CreateCategoryRequest struct {
	Description string `json:"description"`
	ActionType  string `json:"action_type"`
}

type UpdateCategoryRequest struct {
	Description string `json:"description"`
	ActionType  string `json:"action_type"`
}

type CategoryResponse struct {
	ID          int64  `json:"id"`
	Description string `json:"description"`
	ActionType  string `json:"action_type"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func (h *CategoriesHandler) List(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	actionType := query.Get("action_type")

	categories, err := h.db.ListCategories(actionType)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch categories",
		})
		return
	}

	var response []CategoryResponse
	for _, category := range categories {
		response = append(response, CategoryResponse{
			ID:          category.ID,
			Description: category.Description,
			ActionType:  string(category.ActionType),
			CreatedAt:   category.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:   category.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	if response == nil {
		response = []CategoryResponse{}
	}

	respondJSON(w, http.StatusOK, response)
}

func (h *CategoriesHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Invalid request",
		})
		return
	}

	// Validate request
	if req.ActionType != "income" && req.ActionType != "expense" {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Action type must be 'income' or 'expense'",
		})
		return
	}

	if req.Description == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Description is required",
		})
		return
	}

	actionType := models.ActionTypeExpense
	if req.ActionType == "income" {
		actionType = models.ActionTypeIncome
	}

	category, err := h.db.CreateCategory(req.Description, actionType)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to create category",
		})
		return
	}

	respondJSON(w, http.StatusCreated, CategoryResponse{
		ID:          category.ID,
		Description: category.Description,
		ActionType:  string(category.ActionType),
		CreatedAt:   category.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   category.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

func (h *CategoriesHandler) Update(w http.ResponseWriter, r *http.Request) {
	categoryIDStr := chi.URLParam(r, "id")
	categoryID, err := strconv.ParseInt(categoryIDStr, 10, 64)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Invalid category ID",
		})
		return
	}

	var req UpdateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Invalid request",
		})
		return
	}

	// Validate request
	if req.ActionType != "income" && req.ActionType != "expense" {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Action type must be 'income' or 'expense'",
		})
		return
	}

	if req.Description == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Description is required",
		})
		return
	}

	actionType := models.ActionTypeExpense
	if req.ActionType == "income" {
		actionType = models.ActionTypeIncome
	}

	category, err := h.db.UpdateCategory(categoryID, req.Description, actionType)
	if err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{
			"error": "Category not found",
		})
		return
	}

	respondJSON(w, http.StatusOK, CategoryResponse{
		ID:          category.ID,
		Description: category.Description,
		ActionType:  string(category.ActionType),
		UpdatedAt:   category.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

func (h *CategoriesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	categoryIDStr := chi.URLParam(r, "id")
	categoryID, err := strconv.ParseInt(categoryIDStr, 10, 64)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Invalid category ID",
		})
		return
	}

	err = h.db.DeleteCategory(categoryID)
	if err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{
			"error": "Category not found",
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Category deleted successfully",
	})
}
