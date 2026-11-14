package paymentbyo

import (
	"context"
	"fmt"
)

// StripeConnectAdapter gives stripe_connect its own registered, fail-closed
// provider path while reusing Stripe's protocol implementation.
type StripeConnectAdapter struct{ Stripe *StripeAdapter }

func (a *StripeConnectAdapter) stripeConfig(c Config) Config {
	return Config{Provider: ProviderStripe, Env: c.Env, StripeSecretKey: c.StripeConnectSecret, StripeWebhookSecret: c.StripeConnectWebhookSecret}
}
func (a *StripeConnectAdapter) ValidateConfig(c Config) error {
	if a == nil || a.Stripe == nil {
		return fmt.Errorf("stripe connect adapter unavailable")
	}
	return a.Stripe.ValidateConfig(a.stripeConfig(c))
}
func (a *StripeConnectAdapter) CreatePayment(ctx context.Context, c Config, r PaymentRequest) (PaymentSession, error) {
	if err := a.ValidateConfig(c); err != nil {
		return PaymentSession{}, err
	}
	s, err := a.Stripe.CreatePayment(ctx, a.stripeConfig(c), r)
	s.Provider = ProviderStripeConnect
	return s, err
}
func (a *StripeConnectAdapter) VerifyWebhook(c Config, h map[string]string, b []byte) (WebhookEvent, error) {
	if err := a.ValidateConfig(c); err != nil {
		return WebhookEvent{}, err
	}
	e, err := a.Stripe.VerifyWebhook(a.stripeConfig(c), h, b)
	e.Provider = ProviderStripeConnect
	return e, err
}
func (a *StripeConnectAdapter) Revoke(Config) error { return nil }
func (a *StripeConnectAdapter) Fingerprint(c Config) string {
	return "stripe_connect:" + last4(c.StripeConnectSecret)
}
