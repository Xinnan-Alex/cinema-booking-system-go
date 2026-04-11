package config

import (
	"os"
	"time"
)

type Config struct {
	DatabaseURL string
	ServerPort  string
	HoldTTL     time.Duration
}

func Load() Config {
	cfg := Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgres://cinema:cinema@localhost:5432/cinema?sslmode=disable"),
		ServerPort:  getEnv("SERVER_PORT", "8080"),
		HoldTTL:     parseDuration(getEnv("HOLD_TTL", "2m")),
	}
	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 2 * time.Minute
	}
	return d
}
