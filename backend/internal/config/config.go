package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds runtime configuration loaded from the environment.
type Config struct {
	Addr            string
	DatabaseURL     string
	StaticDir       string
	MigrationsPath  string
	ShutdownTimeout time.Duration
	Environment     string
}

// Load reads configuration from environment variables.
// Optional .env loading is handled by the caller (godotenv).
func Load() (Config, error) {
	cfg := Config{
		Addr:            getEnv("ADDR", ":8080"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		StaticDir:       getEnv("STATIC_DIR", "../frontend/dist"),
		MigrationsPath:  getEnv("MIGRATIONS_PATH", "internal/db/migrations"),

		ShutdownTimeout: getDurationEnv("SHUTDOWN_TIMEOUT", 10*time.Second),
		Environment:     getEnv("APP_ENV", "development"),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	if d, err := time.ParseDuration(raw); err == nil {
		return d
	}
	if secs, err := strconv.Atoi(raw); err == nil {
		return time.Duration(secs) * time.Second
	}
	return fallback
}
