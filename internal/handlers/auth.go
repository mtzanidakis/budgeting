package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/mtzanidakis/budgeting/internal/auth"
	"github.com/mtzanidakis/budgeting/internal/database"
	"github.com/mtzanidakis/budgeting/internal/middleware"
)

type AuthHandler struct {
	db           *database.DB
	sessionStore *auth.SessionStore
}

func NewAuthHandler(db *database.DB, sessionStore *auth.SessionStore) *AuthHandler {
	return &AuthHandler{
		db:           db,
		sessionStore: sessionStore,
	}
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	User    *struct {
		ID       int64  `json:"id"`
		Username string `json:"username"`
		Name     string `json:"name"`
	} `json:"user,omitempty"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, LoginResponse{
			Success: false,
			Message: "Invalid request",
		})
		return
	}

	user, err := h.db.GetUserByUsername(req.Username)
	if err != nil {
		respondJSON(w, http.StatusUnauthorized, LoginResponse{
			Success: false,
			Message: "Invalid credentials",
		})
		return
	}

	if err := auth.ComparePassword(user.Password, req.Password); err != nil {
		respondJSON(w, http.StatusUnauthorized, LoginResponse{
			Success: false,
			Message: "Invalid credentials",
		})
		return
	}

	// Drop any session this browser was already carrying so re-logins don't
	// leave orphaned rows behind until they expire.
	if old, err := r.Cookie(auth.SessionCookieName); err == nil {
		if err := h.sessionStore.Delete(old.Value); err != nil {
			slog.Warn("failed to delete previous session on login", "error", err)
		}
	}

	token, session, err := h.sessionStore.Create(user.ID, user.Username)
	if err != nil {
		slog.Error("failed to create session", "user_id", user.ID, "error", err)
		respondJSON(w, http.StatusInternalServerError, LoginResponse{
			Success: false,
			Message: "Failed to create session",
		})
		return
	}

	http.SetCookie(w, auth.NewSessionCookie(token, session.ExpiresAt))

	respondJSON(w, http.StatusOK, LoginResponse{
		Success: true,
		Message: "Login successful",
		User: &struct {
			ID       int64  `json:"id"`
			Username string `json:"username"`
			Name     string `json:"name"`
		}{
			ID:       user.ID,
			Username: user.Username,
			Name:     user.Name,
		},
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Always clear the cookie, even if deleting the server-side row fails, so
	// the browser can't silently resume the session on the next page load.
	http.SetCookie(w, auth.ClearSessionCookie())

	cookie, err := r.Cookie(auth.SessionCookieName)
	if err == nil {
		if err := h.sessionStore.Delete(cookie.Value); err != nil {
			slog.Error("failed to delete session on logout", "error", err)
			respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"message": "Failed to log out",
			})
			return
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Logout successful",
	})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	session, ok := middleware.GetSession(r)
	if !ok {
		respondJSON(w, http.StatusUnauthorized, LoginResponse{
			Success: false,
			Message: "Unauthorized",
		})
		return
	}

	user, err := h.db.GetUserByUsername(session.Username)
	if err != nil {
		respondJSON(w, http.StatusUnauthorized, LoginResponse{
			Success: false,
			Message: "User not found",
		})
		return
	}

	respondJSON(w, http.StatusOK, LoginResponse{
		Success: true,
		User: &struct {
			ID       int64  `json:"id"`
			Username string `json:"username"`
			Name     string `json:"name"`
		}{
			ID:       user.ID,
			Username: user.Username,
			Name:     user.Name,
		},
	})
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
