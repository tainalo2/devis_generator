package paymentbyo

import "context"

type Env string

const (
	EnvSandbox Env = "sandbox"
	EnvLive    Env = "live"
)

type Provider string

const (
	ProviderStripe Provider = "stripe"
	ProviderMollie Provider = "mollie"
	ProviderPayPal Provider = "paypal"
)

type Config struct {
	Provider Provider
	Env      Env
	// Stripe
	StripeSecretKey     string
	StripeWebhookSecret string
	// Mollie
	MollieAPIKey string
	// PayPal
	PayPalClientID     string
	PayPalClientSecret string
	PayPalWebhookID    string
}

type PaymentRequest struct {
	DocumentID string
	Amount     int64 // minor units
	Currency   string
	ReturnURL  string
	CancelURL  string
}

type PaymentSession struct {
	Provider    Provider
	Env         Env
	ExternalID  string
	CheckoutURL string
}

type WebhookEvent struct {
	Provider   Provider
	ExternalID string
	DocumentID string
	Status     string // pending/paid/failed
	Raw        []byte
}

type ProviderAdapter interface {
	ValidateConfig(cfg Config) error
	CreatePayment(ctx context.Context, cfg Config, req PaymentRequest) (PaymentSession, error)
	VerifyWebhook(cfg Config, headers map[string]string, body []byte) (WebhookEvent, error)
	Revoke(cfg Config) error
	Fingerprint(cfg Config) string
}

func AllProviders() []Provider { return []Provider{ProviderStripe, ProviderMollie, ProviderPayPal} }
