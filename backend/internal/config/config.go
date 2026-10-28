package config

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tainalo2/devis_generator/backend/internal/auth"
)

// Dev defaults — intentionally weak; never use in production.
const (
	devSessionSecret    = "dev-only-session-secret-change-me-32b"
	devEncryptionKeyB64 = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=" // 32 zero bytes
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

	SessionSecret string
	EncryptionKey []byte // 32 raw bytes
	CookieSecure  bool
	SessionTTL    time.Duration
	// TrustedProxies : réseaux des reverse proxies autorisés à fournir
	// X-Forwarded-For (TRUSTED_PROXIES, CIDR séparés par des virgules).
	TrustedProxies []*net.IPNet
	Argon2         auth.Argon2Params

	// Legal sources (optional; defaults are public endpoints, no API keys).
	LegalRefreshEnabled  bool
	LegalRefreshInterval time.Duration
	LegalHTTPTimeout     time.Duration
	LegalRechercheURL    string
	LegalVIESURL         string
	LegalTEDBURL         string
	LegalBCEURL          string

	// Site public (landing / pages légales) — aucune identité inventée.
	DonateURL           string
	LegalPublisherName  string
	LegalPublisherAddr  string
	LegalPublisherEmail string
	LegalPublisherSIRET string
	LegalHostName       string
	LegalHostAddr       string
	LegalHostURL        string
}

func (c Config) IsProduction() bool {
	return strings.EqualFold(c.Environment, "production")
}

// Load reads configuration from environment variables.
func Load() (Config, error) {
	cfg := Config{
		Addr:            getEnv("ADDR", "0.0.0.0:8080"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		StaticDir:       getEnv("STATIC_DIR", "../frontend/dist"),
		MigrationsPath:  getEnv("MIGRATIONS_PATH", "internal/db/migrations"),
		ShutdownTimeout: getDurationEnv("SHUTDOWN_TIMEOUT", 10*time.Second),
		StartupTimeout:  getDurationEnv("STARTUP_TIMEOUT", 2*time.Minute),
		Environment:     getEnv("APP_ENV", "development"),
		SessionTTL:      getDurationEnv("SESSION_TTL", 30*24*time.Hour),
		Argon2: auth.Argon2Params{
			Time:    uint32(getIntEnv("ARGON2_TIME", 3)),
			Memory:  uint32(getIntEnv("ARGON2_MEMORY", 64*1024)),
			Threads: uint8(getIntEnv("ARGON2_THREADS", 2)),
			KeyLen:  uint32(getIntEnv("ARGON2_KEYLEN", 32)),
		},
		LegalRefreshEnabled:  getBoolEnv("LEGAL_REFRESH_ENABLED", true),
		LegalRefreshInterval: getDurationEnv("LEGAL_REFRESH_INTERVAL", 24*time.Hour),
		LegalHTTPTimeout:     getDurationEnv("LEGAL_HTTP_TIMEOUT", 8*time.Second),
		LegalRechercheURL:    getEnv("LEGAL_RECHERCHE_BASE_URL", ""),
		LegalVIESURL:         getEnv("LEGAL_VIES_URL", ""),
		LegalTEDBURL:         getEnv("LEGAL_TEDB_URL", ""),
		LegalBCEURL:          getEnv("LEGAL_BCE_URL", ""),
		DonateURL:            getEnv("DONATE_URL", ""),
		LegalPublisherName:   getEnv("LEGAL_PUBLISHER_NAME", ""),
		LegalPublisherAddr:   getEnv("LEGAL_PUBLISHER_ADDRESS", ""),
		LegalPublisherEmail:  getEnv("LEGAL_PUBLISHER_EMAIL", ""),
		LegalPublisherSIRET:  getEnv("LEGAL_PUBLISHER_SIRET", ""),
		LegalHostName:        getEnv("LEGAL_HOST_NAME", ""),
		LegalHostAddr:        getEnv("LEGAL_HOST_ADDRESS", ""),
		LegalHostURL:         getEnv("LEGAL_HOST_URL", ""),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if !MigrateURLLooksSafe(cfg.DatabaseURL) {
		return Config{}, fmt.Errorf("DATABASE_URL must not disable golang-migrate advisory locking (unsafe for multi-replica)")
	}

	proxies, err := parseCIDRs(os.Getenv("TRUSTED_PROXIES"))
	if err != nil {
		return Config{}, fmt.Errorf("TRUSTED_PROXIES: %w", err)
	}
	cfg.TrustedProxies = proxies

	sessionSecret := strings.TrimSpace(os.Getenv("SESSION_SECRET"))
	encKeyB64 := strings.TrimSpace(os.Getenv("ENCRYPTION_KEY"))

	if cfg.IsProduction() {
		if err := validateProductionSecrets(sessionSecret, encKeyB64); err != nil {
			return Config{}, err
		}
		cfg.SessionSecret = sessionSecret
		key, err := decodeEncryptionKey(encKeyB64)
		if err != nil {
			return Config{}, err
		}
		cfg.EncryptionKey = key
		cfg.CookieSecure = true
	} else {
		if sessionSecret == "" {
			slog.Warn("SESSION_SECRET unset; using insecure development default")
			sessionSecret = devSessionSecret
		} else if len(sessionSecret) < 32 {
			slog.Warn("SESSION_SECRET is shorter than 32 characters (ok in development only)")
		}
		cfg.SessionSecret = sessionSecret

		if encKeyB64 == "" {
			slog.Warn("ENCRYPTION_KEY unset; using insecure development default (32 zero bytes)")
			encKeyB64 = devEncryptionKeyB64
		}
		key, err := decodeEncryptionKey(encKeyB64)
		if err != nil {
			return Config{}, fmt.Errorf("ENCRYPTION_KEY: %w", err)
		}
		cfg.EncryptionKey = key
		cfg.CookieSecure = getBoolEnv("COOKIE_SECURE", false)
	}

	return cfg, nil
}

// parseCIDRs lit une liste « 10.0.0.0/8, 172.16.0.1 » ; une IP seule vaut /32 (ou /128).
func parseCIDRs(raw string) ([]*net.IPNet, error) {
	var out []*net.IPNet
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if !strings.Contains(part, "/") {
			ip := net.ParseIP(part)
			if ip == nil {
				return nil, fmt.Errorf("adresse invalide %q", part)
			}
			bits := 128
			if ip.To4() != nil {
				bits = 32
			}
			part = fmt.Sprintf("%s/%d", ip.String(), bits)
		}
		_, n, err := net.ParseCIDR(part)
		if err != nil {
			return nil, fmt.Errorf("réseau invalide %q", part)
		}
		out = append(out, n)
	}
	return out, nil
}

func validateProductionSecrets(sessionSecret, encKeyB64 string) error {
	if sessionSecret == "" {
		return fmt.Errorf("SESSION_SECRET is required in production")
	}
	if len(sessionSecret) < 32 {
		return fmt.Errorf("SESSION_SECRET must be at least 32 characters in production")
	}
	if sessionSecret == devSessionSecret {
		return fmt.Errorf("SESSION_SECRET must not use the development default in production")
	}
	if encKeyB64 == "" {
		return fmt.Errorf("ENCRYPTION_KEY is required in production")
	}
	if encKeyB64 == devEncryptionKeyB64 {
		return fmt.Errorf("ENCRYPTION_KEY must not use the development default in production")
	}
	if _, err := decodeEncryptionKey(encKeyB64); err != nil {
		return fmt.Errorf("ENCRYPTION_KEY: %w", err)
	}
	return nil
}

func decodeEncryptionKey(b64 string) ([]byte, error) {
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("must be standard base64: %w", err)
	}
	if len(raw) != 32 {
		return nil, fmt.Errorf("must decode to exactly 32 bytes, got %d", len(raw))
	}
	return raw, nil
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

func getIntEnv(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return n
}

func getBoolEnv(key string, fallback bool) bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if raw == "" {
		return fallback
	}
	switch raw {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}
