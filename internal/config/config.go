package config

import (
	"github.com/caarlos0/env/v11"
)

type Config struct {
	Port           string `env:"PORT" envDefault:"8080"`
	DatabasePath   string `env:"DATABASE_PATH" envDefault:"./data/budgeting.db"`
	SessionSecret  string `env:"SESSION_SECRET,required"`
	LogLevel       string `env:"LOG_LEVEL" envDefault:"info"`
	Currency       string `env:"CURRENCY" envDefault:"€"`
}

func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
