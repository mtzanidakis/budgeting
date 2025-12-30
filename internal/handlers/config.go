package handlers

import (
	"net/http"
)

type ConfigHandler struct {
	currency string
}

func NewConfigHandler(currency string) *ConfigHandler {
	return &ConfigHandler{
		currency: currency,
	}
}

type ConfigResponse struct {
	Currency string `json:"currency"`
}

func (h *ConfigHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, ConfigResponse{
		Currency: h.currency,
	})
}
