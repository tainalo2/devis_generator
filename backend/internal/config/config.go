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
	StartupTimeout  time.Duration
	Environment     string
}

// Load reads configuration from environment variables.
// Optional .env loading is handled by the caller (godotenv).
func Load() (Config, error) {
	cfg := Config{
		// ":8080" and "0.0.0.0:8080" both listen on all interfaces in Go.
		Addr:            getEnv("ADDR", "0.0.0.0:8080"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		StaticDir:       getEnv("STATIC_DIR", "../frontend/dist"),
		MigrationsPath:  getEnv("MIGRATIONS_PATH", "internal/db/migrations"),
		ShutdownTimeout: getDurationEnv("SHUTDOWN_TIMEOUT", 10*time.Second),
		StartupTimeout:  getDurationEnv("STARTUP_TIMEOUT", 2*time.Minute),
		Environment:     getEnv("APP_ENV", "development"),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if !MigrateURLLooksSafe(cfg.DatabaseURL) {
		return Config{}, fmt.Errorf("DATABASE_URL must not disable golang-migrate advisory locking (unsafe for multi-replica)")
	}

	return cfg, nil
}

// MigrateURLLooksSafe rejects URLs that turn off migrate advisory locks.
func MigrateURLLooksSafe(databaseURL string) bool {
	lower := strings.ToLower(databaseURL)
	for _, bad := range []string{
		"x-lock-strategy=off",
		"x-lock-strategy=false",
		"x-lock-strategy=none",
		"x-advisory-locking=false",
	} {
		if strings.Contains(lower, bad) {
			return false
		}
	}
	return true
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
