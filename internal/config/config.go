// Package config reads service configuration from environment variables.
package config

import "errors"

type Config struct {
	Addr        string
	DatabaseURL string
}

func Load(getenv func(string) string) (Config, error) {
	cfg := Config{Addr: getenv("ADDR"), DatabaseURL: getenv("DATABASE_URL")}
	if cfg.Addr == "" {
		cfg.Addr = ":8080"
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	return cfg, nil
}
