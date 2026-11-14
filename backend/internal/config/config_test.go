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
	t.Setenv("DATABASE_URL", "postgres://user:***@localhost:5432/devis?sslmode=disable")
	t.Setenv("ADDR", "")
	t.Setenv("STATIC_DIR", "")
	t.Setenv("APP_ENV", "development")
	t.Setenv("SHUTDOWN_TIMEOUT", "")
	t.Setenv("STARTUP_TIMEOUT", "")
	t.Setenv("SESSION_SECRET", "")
	t.Setenv("DOC_LINK_SECRET", "")
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
	if cfg.DocLinkSecret != devSessionSecret {
		t.Fatalf("expected DocLinkSecret to fall back to SessionSecret")
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
	t.Setenv("DOC_LINK_SECRET", "production-doclink-secret-value-32ch-")
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
	if cfg.DocLinkSecret != "production-doclink-secret-value-32ch-" {
		t.Fatalf("DocLinkSecret not set in production")
	}
}

func TestLoadProductionRequiresDocLinkSecret(t *testing.T) {
	key := make([]byte, 32)
	t.Setenv("DATABASE_URL", "postgres://db")
	t.Setenv("APP_ENV", "production")
	t.Setenv("SESSION_SECRET", "production-session-secret-value-32ch")
	t.Setenv("DOC_LINK_SECRET", "")
	t.Setenv("ENCRYPTION_KEY", base64.StdEncoding.EncodeToString(key))

	if _, err := Load(); err == nil {
		t.Fatal("expected error when DOC_LINK_SECRET is missing in production")
	}
}

func TestLoadMicrosoftOAuthConfiguration(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://db")
	t.Setenv("APP_ENV", "development")
	t.Setenv("MICROSOFT_OAUTH_CLIENT_ID", "client-id")
	t.Setenv("MICROSOFT_OAUTH_CLIENT_SECRET", "client-secret")
	t.Setenv("MICROSOFT_OAUTH_REDIRECT_URL", "https://app.example.test/api/v1/email/microsoft/callback")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.MicrosoftOAuthClientID != "client-id" || cfg.MicrosoftOAuthClientSecret != "client-secret" {
		t.Fatalf("unexpected Microsoft credentials: %#v", cfg)
	}
	if cfg.MicrosoftOAuthRedirectURL != "https://app.example.test/api/v1/email/microsoft/callback" {
		t.Fatalf("redirect URL = %q", cfg.MicrosoftOAuthRedirectURL)
	}
}

func TestLoadRejectsPartialMicrosoftOAuthConfiguration(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://db")
	t.Setenv("APP_ENV", "development")
	t.Setenv("MICROSOFT_OAUTH_CLIENT_ID", "client-id")
	t.Setenv("MICROSOFT_OAUTH_CLIENT_SECRET", "")
	t.Setenv("MICROSOFT_OAUTH_REDIRECT_URL", "")

	if _, err := Load(); err == nil {
		t.Fatal("expected partial Microsoft OAuth configuration error")
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

func TestLoadGoogleOAuthConfiguration(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://db")
	t.Setenv("APP_ENV", "development")
	t.Setenv("GOOGLE_OAUTH_CLIENT_ID", "google-client-id")
	t.Setenv("GOOGLE_OAUTH_CLIENT_SECRET", "google-client-secret")
	t.Setenv("GOOGLE_OAUTH_REDIRECT_URL", "https://app.example.test/api/v1/email/gmail/callback")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.GoogleOAuthClientID != "google-client-id" || cfg.GoogleOAuthClientSecret != "google-client-secret" {
		t.Fatalf("unexpected Google credentials: %#v", cfg)
	}
	if cfg.GoogleOAuthRedirectURL != "https://app.example.test/api/v1/email/gmail/callback" {
		t.Fatalf("redirect URL = %q", cfg.GoogleOAuthRedirectURL)
	}
}

func TestLoadRejectsPartialGoogleOAuthConfiguration(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://db")
	t.Setenv("APP_ENV", "development")
	t.Setenv("GOOGLE_OAUTH_CLIENT_ID", "google-client-id")
	t.Setenv("GOOGLE_OAUTH_CLIENT_SECRET", "")
	t.Setenv("GOOGLE_OAUTH_REDIRECT_URL", "")

	if _, err := Load(); err == nil {
		t.Fatal("expected partial Google OAuth configuration error")
	}
}

func TestLoadDocLinkSecretFallback(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:***@localhost:5432/devis?sslmode=disable")
	t.Setenv("APP_ENV", "development")
	t.Setenv("SESSION_SECRET", "my-custom-session-secret-32ch-x")
	t.Setenv("DOC_LINK_SECRET", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DocLinkSecret != cfg.SessionSecret {
		t.Fatalf("DocLinkSecret = %q, want %q (fallback to SessionSecret)", cfg.DocLinkSecret, cfg.SessionSecret)
	}
}

func TestLoadDocLinkSecretExplicit(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:***@localhost:5432/devis?sslmode=disable")
	t.Setenv("APP_ENV", "development")
	t.Setenv("SESSION_SECRET", "my-custom-session-secret-32ch-x")
	t.Setenv("DOC_LINK_SECRET", "different-doc-link-secret-32ch-y")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DocLinkSecret != "different-doc-link-secret-32ch-y" {
		t.Fatalf("DocLinkSecret = %q, want explicit value", cfg.DocLinkSecret)
	}
	if cfg.DocLinkSecret == cfg.SessionSecret {
		t.Fatal("DocLinkSecret should be different from SessionSecret when explicitly set")
	}
}
