package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port          string
	DatabaseURL   string
	Origins       []string
	MatchDuration time.Duration
	TickRate      int
}

func Load() Config {
	return Config{
		Port:          env("PORT", "8080"),
		DatabaseURL:   env("DATABASE_URL", "postgres://game:game@localhost:5432/obsidian_arena?sslmode=disable"),
		Origins:       split(env("ALLOWED_ORIGINS", "http://localhost:5173,http://localhost:8081")),
		MatchDuration: time.Duration(envInt("MATCH_DURATION_SECONDS", 60)) * time.Second,
		TickRate:      envInt("TICK_RATE", 20),
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" { return value }
	return fallback
}

func envInt(key string, fallback int) int {
	value, err := strconv.Atoi(env(key, ""))
	if err != nil || value <= 0 { return fallback }
	return value
}

func split(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if item := strings.TrimSpace(part); item != "" { result = append(result, item) }
	}
	return result
}
