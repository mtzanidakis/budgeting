package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/manolis/budgeting/internal/auth"
	"github.com/manolis/budgeting/internal/database"
	"github.com/manolis/budgeting/internal/middleware"
	"github.com/manolis/budgeting/internal/models"
)

type TokensHandler struct {
	db *database.DB
}

func NewTokensHandler(db *database.DB) *TokensHandler {
	return &TokensHandler{db: db}
}

type CreateTokenRequest struct {
	Name      string  `json:"name"`
	ExpiresAt *string `json:"expires_at,omitempty"` // YYYY-MM-DD or RFC3339; nil = never
}

func (h *TokensHandler) List(w http.ResponseWriter, r *http.Request) {
	session, ok := middleware.GetSession(r)
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
		return
	}

	tokens, err := h.db.ListAPITokensByUser(session.UserID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to list tokens"})
		return
	}
	if tokens == nil {
		tokens = []*models.APIToken{}
	}
	respondJSON(w, http.StatusOK, tokens)
}

func (h *TokensHandler) Create(w http.ResponseWriter, r *http.Request) {
	session, ok := middleware.GetSession(r)
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
		return
	}

	var req CreateTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Name is required"})
		return
	}
	if len(name) > 100 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Name too long (max 100)"})
		return
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil && strings.TrimSpace(*req.ExpiresAt) != "" {
		t, err := parseExpiry(*req.ExpiresAt)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if !t.After(time.Now()) {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Expiry must be in the future"})
			return
		}
		expiresAt = &t
	}

	raw, hash, err := auth.GenerateAPIToken()
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to generate token"})
		return
	}

	token, err := h.db.CreateAPIToken(session.UserID, name, hash, expiresAt)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create token"})
		return
	}
	// Return the raw token exactly once.
	token.Token = raw
	respondJSON(w, http.StatusCreated, token)
}

func (h *TokensHandler) Delete(w http.ResponseWriter, r *http.Request) {
	session, ok := middleware.GetSession(r)
	if !ok {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid id"})
		return
	}

	if err := h.db.SoftDeleteAPIToken(id, session.UserID); err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Token not found"})
		return
	}
	respondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// parseExpiry accepts YYYY-MM-DD or RFC3339. Date-only is interpreted as end-of-day UTC.
func parseExpiry(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t.Add(24*time.Hour - time.Second), nil
	}
	return time.Time{}, errInvalidExpiry
}

var errInvalidExpiry = &expiryError{"Invalid expiry format (use YYYY-MM-DD or RFC3339)"}

type expiryError struct{ msg string }

func (e *expiryError) Error() string { return e.msg }
