package stripeconnect

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds Stripe Connect (platform) runtime configuration.
type Config struct {
	PlatformSecretKey    string
	PlatformPublishableKey string
	WebhookSecret        string
	PublicURL            string // HTTPS public URL
	ComplianceAck        bool   // operator acknowledged facilitator role
	FeeBPS               int64  // basis points
	FeeFixedCents        int64
	FeeCapCents          int64  // 0 = no cap
	Currency             string
}

func ConfigFromEnv() Config {
	bps, _ := strconv.ParseInt(os.Getenv("STRIPE_CONNECT_FEE_BPS"), 10, 64)
	fixed, _ := strconv.ParseInt(os.Getenv("STRIPE_CONNECT_FEE_FIXED_CENTS"), 10, 64)
	capC, _ := strconv.ParseInt(os.Getenv("STRIPE_CONNECT_FEE_CAP_CENTS"), 10, 64)
	ack := os.Getenv("STRIPE_CONNECT_COMPLIANCE_ACK") == "true"
	return Config{
		PlatformSecretKey:     os.Getenv("STRIPE_PLATFORM_SECRET_KEY"),
		PlatformPublishableKey: os.Getenv("STRIPE_PLATFORM_PUBLISHABLE_KEY"),
		WebhookSecret:         os.Getenv("STRIPE_CONNECT_WEBHOOK_SECRET"),
		PublicURL:             os.Getenv("STRIPE_PUBLIC_URL"),
		ComplianceAck:         ack,
		FeeBPS:                bps,
		FeeFixedCents:         fixed,
		FeeCapCents:           capC,
		Currency:              os.Getenv("STRIPE_CONNECT_CURRENCY"),
	}
}

func (c Config) IsTestMode() bool {
	return len(c.PlatformSecretKey) >= 8 && c.PlatformSecretKey[:8] == "sk_test_"
}

func (c Config) Enabled() (bool, string) {
	if os.Getenv("PAYMENT_STRIPE_CONNECT_ENABLED") != "true" {
		return false, "disabled"
	}
	if c.PlatformSecretKey == "" {
		return false, "missing_platform_secret"
	}
	if c.WebhookSecret == "" {
		return false, "missing_webhook_secret"
	}
	if c.PublicURL == "" || !(len(c.PublicURL) > 5 && (c.PublicURL[:5] == "https" )) {
		return false, "missing_public_https_url"
	}
	if !c.ComplianceAck {
		return false, "missing_compliance_ack"
	}
	if c.Currency == "" {
		return false, "missing_currency"
	}
	return true, ""
}

// ComputeFee returns application_fee_amount in cents. Server-authoritative.
func (c Config) ComputeFee(amountCents int64, currency string) (int64, error) {
	if amountCents <= 0 {
		return 0, fmt.Errorf("amount must be positive")
	}
	if currency != c.Currency && c.Currency != "" {
		// allow any? require match
		if currency != c.Currency {
			return 0, fmt.Errorf("currency mismatch")
		}
	}
	fee := (amountCents*c.FeeBPS)/10000 + c.FeeFixedCents
	if c.FeeCapCents > 0 && fee > c.FeeCapCents {
		fee = c.FeeCapCents
	}
	if fee < 0 {
		fee = 0
	}
	if fee > amountCents {
		fee = amountCents
	}
	return fee, nil
}