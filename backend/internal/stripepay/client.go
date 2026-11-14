package stripepay

// Config holds Stripe runtime configuration. Keys are expected to be test-mode
// keys (sk_test_/pk_test_) during validation; production keys are never required.
type Config struct {
	SecretKey      string
	PublishableKey string
	WebhookSecret  string
	// ConnectAccountID is an optional connected (Express/Standard) account id
	// used in test mode to simulate Stripe Connect flows.
	ConnectAccountID string
}

func (c Config) IsTestMode() bool {
	return len(c.SecretKey) >= 8 && c.SecretKey[:8] == "sk_test_"
}
