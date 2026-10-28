package config

import (
	"encoding/base64"
	"testing"
	"time"
)

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("APP_ENV", "development")
	if _, err := Load(); err == nil {
		t.Fatal("expected error when DATABASE_URL is missing")
	}
}

func TestLoadDefaultsDev(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/devis?sslmode=disable")
	t.Setenv("ADDR", "")
	t.Setenv("STATIC_DIR", "")
	t.Setenv("APP_ENV", "development")
	t.Setenv("SHUTDOWN_TIMEOUT", "")
	t.Setenv("STARTUP_TIMEOUT", "")
	t.Setenv("SESSION_SECRET", "")
	t.Setenv("ENCRYPTION_KEY", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if cfg.Addr != "0.0.0.0:8080" {
		t.Fatalf("Addr = %q, want 0.0.0.0:8080", cfg.Addr)
	}
	if cfg.SessionSecret != devSessionSecret {
		t.Fatalf("expected dev session secret")
	}
	if len(cfg.EncryptionKey) != 32 {
		t.Fatalf("encryption key len = %d", len(cfg.EncryptionKey))
	}
	if cfg.CookieSecure {
		t.Fatal("CookieSecure should be false in development by default")
	}
	if cfg.ShutdownTimeout != 10*time.Second {
		t.Fatalf("ShutdownTimeout = %v", cfg.ShutdownTimeout)
	}
}

func TestLoadProductionRequiresSecrets(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://db")
	t.Setenv("APP_ENV", "production")
	t.Setenv("SESSION_SECRET", "")
	t.Setenv("ENCRYPTION_KEY", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected error without secrets in production")
	}
}

func TestLoadProductionOK(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	t.Setenv("DATABASE_URL", "postgres://db")
	t.Setenv("APP_ENV", "production")
	t.Setenv("SESSION_SECRET", "production-session-secret-value-32ch")
	t.Setenv("ENCRYPTION_KEY", base64.StdEncoding.EncodeToString(key))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.CookieSecure {
		t.Fatal("CookieSecure should be true in production")
	}
	if len(cfg.EncryptionKey) != 32 {
		t.Fatal("bad key length")
	}
}

func TestLoadRejectsDisabledAdvisoryLock(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost/db?x-lock-strategy=off")
	t.Setenv("APP_ENV", "development")
	if _, err := Load(); err == nil {
		t.Fatal("expected error when advisory lock is disabled")
	}
}

func TestLoadRejectsWeakProductionSessionSecret(t *testing.T) {
	key := make([]byte, 32)
	t.Setenv("DATABASE_URL", "postgres://db")
	t.Setenv("APP_ENV", "production")
	t.Setenv("SESSION_SECRET", "too-short")
	t.Setenv("ENCRYPTION_KEY", base64.StdEncoding.EncodeToString(key))
	if _, err := Load(); err == nil {
		t.Fatal("expected weak SESSION_SECRET error")
	}
}
