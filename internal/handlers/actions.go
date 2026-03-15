package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
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
	CategoryID  *int64  `json:"category_id,omitempty"`
}

type UpdateActionRequest struct {
	Type        string  `json:"type"`
	Date        string  `json:"date"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	CategoryID  *int64  `json:"category_id,omitempty"`
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
	CategoryID  *int64  `json:"category_id,omitempty"`
	CreatedAt   string  `json:"created_at"`
}

type PaginatedActionsResponse struct {
	Actions []ActionResponse `json:"actions"`
	Total   int              `json:"total"`
}

type MonthlyChartData struct {
	Month   string  `json:"month"`
	Income  float64 `json:"income"`
	Expense float64 `json:"expense"`
}

type ChartDataResponse struct {
	Year int                `json:"year"`
	Data []MonthlyChartData `json:"data"`
}

type CategoryChartDataResponse struct {
	Year     int                       `json:"year"`
	Month    int                       `json:"month"`
	Expenses []database.CategorySummary `json:"expenses"`
	Income   []database.CategorySummary `json:"income"`
}

func (h *ActionsHandler) List(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	filters := database.ActionFilters{
		Username: query.Get("username"),
		Type:     query.Get("type"),
		DateFrom: query.Get("date_from"),
		DateTo:   query.Get("date_to"),
		Search:   query.Get("search"),
		Limit:    20,
	}

	if limitStr := query.Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			filters.Limit = limit
		}
	}

	// Check if offset parameter is provided for pagination
	isPaginated := false
	if offsetStr := query.Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			filters.Offset = offset
			isPaginated = true
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
			CategoryID:  action.CategoryID,
			CreatedAt:   action.CreatedAt.Format(time.RFC3339),
		})
	}

	if response == nil {
		response = []ActionResponse{}
	}

	// If paginated, return paginated response with total count
	if isPaginated {
		total, err := h.db.CountActions(filters)
		if err != nil {
			respondJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "Failed to count actions",
			})
			return
		}

		respondJSON(w, http.StatusOK, PaginatedActionsResponse{
			Actions: response,
			Total:   total,
		})
		return
	}

	// Otherwise, return old format (backward compatible)
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

	if req.CategoryID == nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Category is required",
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

	action, err := h.db.CreateAction(session.UserID, actionType, req.Date, req.Description, req.Amount, req.CategoryID)
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
		CategoryID:  action.CategoryID,
		CreatedAt:   action.CreatedAt.Format(time.RFC3339),
	})
}

func (h *ActionsHandler) GetChartData(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	yearStr := query.Get("year")
	username := query.Get("username")

	year := time.Now().Year()
	if yearStr != "" {
		if y, err := strconv.Atoi(yearStr); err == nil {
			year = y
		}
	}

	summaries, err := h.db.GetMonthlySummary(year, username)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch chart data",
		})
		return
	}

	// Create map for quick lookup
	summaryMap := make(map[int]database.MonthlySummary)
	for _, s := range summaries {
		summaryMap[s.Month] = s
	}

	// Build response with all 12 months
	monthNames := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun",
		"Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}

	var data []MonthlyChartData
	for i := 1; i <= 12; i++ {
		summary, exists := summaryMap[i]
		chartData := MonthlyChartData{
			Month: monthNames[i-1],
		}
		if exists {
			chartData.Income = summary.Income
			chartData.Expense = summary.Expense
		}
		data = append(data, chartData)
	}

	respondJSON(w, http.StatusOK, ChartDataResponse{
		Year: year,
		Data: data,
	})
}

func (h *ActionsHandler) GetCategoryChartData(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	year := time.Now().Year()
	if yearStr := query.Get("year"); yearStr != "" {
		if y, err := strconv.Atoi(yearStr); err == nil {
			year = y
		}
	}

	month := int(time.Now().Month())
	if monthStr := query.Get("month"); monthStr != "" {
		if m, err := strconv.Atoi(monthStr); err == nil {
			month = m
		}
	}

	username := query.Get("username")

	expenseSummaries, err := h.db.GetCategorySummary(year, month, "expense", username)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch expense category data",
		})
		return
	}

	incomeSummaries, err := h.db.GetCategorySummary(year, month, "income", username)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch income category data",
		})
		return
	}

	if expenseSummaries == nil {
		expenseSummaries = []database.CategorySummary{}
	}
	if incomeSummaries == nil {
		incomeSummaries = []database.CategorySummary{}
	}

	respondJSON(w, http.StatusOK, CategoryChartDataResponse{
		Year:     year,
		Month:    month,
		Expenses: expenseSummaries,
		Income:   incomeSummaries,
	})
}

func (h *ActionsHandler) Update(w http.ResponseWriter, r *http.Request) {
	session, ok := middleware.GetSession(r)
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
		return
	}

	actionIDStr := chi.URLParam(r, "id")
	actionID, err := strconv.ParseInt(actionIDStr, 10, 64)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Invalid action ID",
		})
		return
	}

	var req UpdateActionRequest
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

	if req.CategoryID == nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Category is required",
		})
		return
	}

	actionType := models.ActionTypeExpense
	if req.Type == "income" {
		actionType = models.ActionTypeIncome
	}

	action, err := h.db.UpdateAction(actionID, session.UserID, actionType, req.Date, req.Description, req.Amount, req.CategoryID)
	if err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{
			"error": "Action not found or not owned by user",
		})
		return
	}

	user, _ := h.db.GetUserByID(action.UserID)

	respondJSON(w, http.StatusOK, ActionResponse{
		ID:          action.ID,
		UserID:      action.UserID,
		Username:    user.Username,
		Name:        user.Name,
		Type:        string(action.Type),
		Date:        action.Date,
		Description: action.Description,
		Amount:      action.Amount,
		CategoryID:  action.CategoryID,
		CreatedAt:   action.CreatedAt.Format(time.RFC3339),
	})
}

func (h *ActionsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	session, ok := middleware.GetSession(r)
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
		return
	}

	actionIDStr := chi.URLParam(r, "id")
	actionID, err := strconv.ParseInt(actionIDStr, 10, 64)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Invalid action ID",
		})
		return
	}

	err = h.db.DeleteAction(actionID, session.UserID)
	if err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{
			"error": "Action not found or not owned by user",
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Action deleted successfully",
	})
}
