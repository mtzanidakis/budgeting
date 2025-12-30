package handlers

import (
	"net/http"

	"github.com/manolis/budgeting/internal/database"
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
