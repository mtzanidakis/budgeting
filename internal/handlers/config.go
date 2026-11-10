package handlers

import (
	"net/http"
)

type ConfigHandler struct {
	currency      string
	aiChatEnabled bool
}

func NewConfigHandler(currency string, aiChatEnabled bool) *ConfigHandler {
	return &ConfigHandler{
		currency:      currency,
		aiChatEnabled: aiChatEnabled,
	}
}

type ConfigResponse struct {
	Currency      string `json:"currency"`
	AIChatEnabled bool   `json:"ai_chat_enabled"`
}

func (h *ConfigHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, ConfigResponse{
		Currency:      h.currency,
		AIChatEnabled: h.aiChatEnabled,
	})
}
