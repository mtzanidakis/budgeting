package config

import (
	"errors"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Port           string `env:"PORT" envDefault:"4666"`
	DatabasePath   string `env:"DATABASE_PATH" envDefault:"./data/budgeting.db"`
	SessionSecret  string `env:"SESSION_SECRET,required"`
	LogLevel       string `env:"LOG_LEVEL" envDefault:"info"`
	Currency       string `env:"CURRENCY" envDefault:"€"`

	// Optional AI chat sidebar, backed by any OpenAI-compatible
	// chat completions API.
	AIChatEnabled bool   `env:"AI_CHAT_ENABLED" envDefault:"false"`
	OpenAIBaseURL string `env:"OPENAI_BASE_URL" envDefault:"https://api.openai.com/v1"`
	OpenAIAPIKey  string `env:"OPENAI_API_KEY"`
	OpenAIModel   string `env:"OPENAI_MODEL"`
}

func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	if cfg.AIChatEnabled && (cfg.OpenAIBaseURL == "" || cfg.OpenAIAPIKey == "" || cfg.OpenAIModel == "") {
		return nil, errors.New("AI_CHAT_ENABLED requires OPENAI_BASE_URL, OPENAI_API_KEY and OPENAI_MODEL")
	}
	return cfg, nil
}
