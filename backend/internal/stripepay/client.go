package stripepay

import (
	"os"

	stripe "github.com/stripe/stripe-go/v81"
)

// Config holds Stripe runtime configuration. Keys are expected to be test-mode
// keys (sk_test_/pk_test_) during validation; production keys are never required.
type Config struct {
	SecretKey     string
	PublishableKey string
	WebhookSecret  string
	// ConnectAccountID is an optional connected (Express/Standard) account id
	// used in test mode to simulate Stripe Connect flows.
	ConnectAccountID string
}

func ConfigFromEnv() Config {
	return Config{
		SecretKey:        os.Getenv("STRIPE_SECRET_KEY"),
		PublishableKey:   os.Getenv("STRIPE_PUBLISHABLE_KEY"),
		WebhookSecret:    os.Getenv("STRIPE_WEBHOOK_SECRET"),
		ConnectAccountID: os.Getenv("STRIPE_CONNECT_ACCOUNT_ID"),
	}
}

func (c Config) IsTestMode() bool {
	return len(c.SecretKey) >= 8 && c.SecretKey[:8] == "sk_test_"
}

func Init(cfg Config) {
	stripe.Key = cfg.SecretKey
}
