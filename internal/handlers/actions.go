package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/manolis/budgeting/internal/database"
	"github.com/manolis/budgeting/internal/middleware"
	"github.com/manolis/budgeting/internal/models"
)

type ActionsHandler struct {
	db *database.DB
}

func NewActionsHandler(db *database.DB) *ActionsHandler {
	return &ActionsHandler{db: db}
}

type CreateActionRequest struct {
	Type        string  `json:"type"`
	Date        string  `json:"date"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
}

type ActionResponse struct {
	ID          int64   `json:"id"`
	UserID      int64   `json:"user_id"`
	Username    string  `json:"username"`
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	Date        string  `json:"date"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	CreatedAt   string  `json:"created_at"`
}

func (h *ActionsHandler) List(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	filters := database.ActionFilters{
		Username: query.Get("username"),
		Type:     query.Get("type"),
		DateFrom: query.Get("date_from"),
		DateTo:   query.Get("date_to"),
		Limit:    20,
	}

	if limitStr := query.Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			filters.Limit = limit
		}
	}

	actions, err := h.db.ListActions(filters)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch actions",
		})
		return
	}

	// Enrich actions with user information
	var response []ActionResponse
	for _, action := range actions {
		user, err := h.db.GetUserByID(action.UserID)
		if err != nil {
			continue
		}

		response = append(response, ActionResponse{
			ID:          action.ID,
			UserID:      action.UserID,
			Username:    user.Username,
			Name:        user.Name,
			Type:        string(action.Type),
			Date:        action.Date,
			Description: action.Description,
			Amount:      action.Amount,
			CreatedAt:   action.CreatedAt.Format(time.RFC3339),
		})
	}

	if response == nil {
		response = []ActionResponse{}
	}

	respondJSON(w, http.StatusOK, response)
}

func (h *ActionsHandler) Create(w http.ResponseWriter, r *http.Request) {
	session, ok := middleware.GetSession(r)
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
		return
	}

	var req CreateActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Invalid request",
		})
		return
	}

	// Validate request
	if req.Type != "income" && req.Type != "expense" {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Type must be 'income' or 'expense'",
		})
		return
	}

	if req.Description == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Description is required",
		})
		return
	}

	if req.Amount <= 0 {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Amount must be positive",
		})
		return
	}

	// Default to today if date is empty
	if req.Date == "" {
		req.Date = time.Now().Format("2006-01-02")
	}

	actionType := models.ActionTypeExpense
	if req.Type == "income" {
		actionType = models.ActionTypeIncome
	}

	action, err := h.db.CreateAction(session.UserID, actionType, req.Date, req.Description, req.Amount)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to create action",
		})
		return
	}

	user, _ := h.db.GetUserByID(action.UserID)

	respondJSON(w, http.StatusCreated, ActionResponse{
		ID:          action.ID,
		UserID:      action.UserID,
		Username:    user.Username,
		Name:        user.Name,
		Type:        string(action.Type),
		Date:        action.Date,
		Description: action.Description,
		Amount:      action.Amount,
		CreatedAt:   action.CreatedAt.Format(time.RFC3339),
	})
}
