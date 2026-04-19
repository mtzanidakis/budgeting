package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/mtzanidakis/budgeting/internal/auth"
	"github.com/mtzanidakis/budgeting/internal/database"
	"github.com/mtzanidakis/budgeting/internal/middleware"
)

type UsersHandler struct {
	db *database.DB
}

func NewUsersHandler(db *database.DB) *UsersHandler {
	return &UsersHandler{db: db}
}

type UserListResponse struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
}

func (h *UsersHandler) List(w http.ResponseWriter, r *http.Request) {
	users, err := h.db.ListUsers()
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch users",
		})
		return
	}

	var response []UserListResponse
	for _, user := range users {
		response = append(response, UserListResponse{
			ID:       user.ID,
			Username: user.Username,
			Name:     user.Name,
		})
	}

	if response == nil {
		response = []UserListResponse{}
	}

	respondJSON(w, http.StatusOK, response)
}

type UpdateProfileRequest struct {
	Name            string `json:"name"`
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type UpdateProfileResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	User    *struct {
		ID       int64  `json:"id"`
		Username string `json:"username"`
		Name     string `json:"name"`
	} `json:"user,omitempty"`
}

func (h *UsersHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	// Get session from middleware
	session, ok := middleware.GetSession(r)
	if !ok {
		respondJSON(w, http.StatusUnauthorized, UpdateProfileResponse{
			Success: false,
			Message: "Unauthorized",
		})
		return
	}

	// Decode request
	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, UpdateProfileResponse{
			Success: false,
			Message: "Invalid request format",
		})
		return
	}

	// Validate name (required)
	if strings.TrimSpace(req.Name) == "" {
		respondJSON(w, http.StatusBadRequest, UpdateProfileResponse{
			Success: false,
			Message: "Name is required",
		})
		return
	}

	// Get current user from database
	user, err := h.db.GetUserByUsername(session.Username)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, UpdateProfileResponse{
			Success: false,
			Message: "Failed to retrieve user",
		})
		return
	}

	// Prepare update parameters
	var passwordPtr *string
	var namePtr *string

	// Name is always updated
	trimmedName := strings.TrimSpace(req.Name)
	namePtr = &trimmedName

	// Password update logic
	if req.NewPassword != "" {
		// Current password is REQUIRED when changing password
		if req.CurrentPassword == "" {
			respondJSON(w, http.StatusBadRequest, UpdateProfileResponse{
				Success: false,
				Message: "Current password is required to change password",
			})
			return
		}

		// Verify current password
		if err := auth.ComparePassword(user.Password, req.CurrentPassword); err != nil {
			respondJSON(w, http.StatusUnauthorized, UpdateProfileResponse{
				Success: false,
				Message: "Current password is incorrect",
			})
			return
		}

		// Validate new password strength
		if len(req.NewPassword) < 6 {
			respondJSON(w, http.StatusBadRequest, UpdateProfileResponse{
				Success: false,
				Message: "New password must be at least 6 characters",
			})
			return
		}

		// Hash new password
		hashedPassword, err := auth.HashPassword(req.NewPassword)
		if err != nil {
			respondJSON(w, http.StatusInternalServerError, UpdateProfileResponse{
				Success: false,
				Message: "Failed to hash password",
			})
			return
		}
		passwordPtr = &hashedPassword
	}

	// Update user in database
	if err := h.db.UpdateUser(session.Username, passwordPtr, namePtr); err != nil {
		respondJSON(w, http.StatusInternalServerError, UpdateProfileResponse{
			Success: false,
			Message: "Failed to update profile",
		})
		return
	}

	// Return success with updated user info
	respondJSON(w, http.StatusOK, UpdateProfileResponse{
		Success: true,
		Message: "Profile updated successfully",
		User: &struct {
			ID       int64  `json:"id"`
			Username string `json:"username"`
			Name     string `json:"name"`
		}{
			ID:       user.ID,
			Username: user.Username,
			Name:     trimmedName,
		},
	})
}
