package paymentbyo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// StripeConnectAdapter gives stripe_connect its own registered, fail-closed
// provider path while reusing Stripe's protocol implementation.
type StripeConnectAdapter struct {
	Stripe                             *StripeAdapter
	ComplianceAck                      bool
	FeeBPS, FeeFixedCents, FeeCapCents int64
}

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
	if !a.ComplianceAck {
		return PaymentSession{}, fmt.Errorf("stripe connect compliance acknowledgement required")
	}
	if r.ConnectedAccountID == "" {
		return PaymentSession{}, fmt.Errorf("connected account required")
	}
	if a.FeeBPS < 0 || a.FeeBPS > 2500 || a.FeeFixedCents < 0 || a.FeeCapCents < 0 {
		return PaymentSession{}, fmt.Errorf("unsafe platform fee")
	}
	fee := r.Amount*a.FeeBPS/10000 + a.FeeFixedCents
	if a.FeeCapCents > 0 && fee > a.FeeCapCents {
		fee = a.FeeCapCents
	}
	if fee > r.Amount {
		fee = r.Amount
	}
	v := url.Values{"mode": {"payment"}, "success_url": {r.ReturnURL}, "cancel_url": {r.CancelURL}, "line_items[0][quantity]": {"1"}, "line_items[0][price_data][unit_amount]": {strconv.FormatInt(r.Amount, 10)}, "line_items[0][price_data][currency]": {strings.ToLower(r.Currency)}, "line_items[0][price_data][product_data][name]": {"Invoice " + r.DocumentID}, "metadata[tenant_id]": {r.TenantID}, "metadata[document_id]": {r.DocumentID}, "payment_intent_data[application_fee_amount]": {strconv.FormatInt(fee, 10)}}
	b, code, err := a.Stripe.HTTP.do(ctx, "POST", "/v1/checkout/sessions", []byte(v.Encode()), map[string]string{"Authorization": "Bearer " + c.StripeConnectSecret, "Stripe-Account": r.ConnectedAccountID, "Content-Type": "application/x-www-form-urlencoded", "Idempotency-Key": r.IdempotencyKey})
	if err != nil || code >= 400 {
		if err == nil {
			err = fmt.Errorf("stripe connect %d: %s", code, b)
		}
		return PaymentSession{}, err
	}
	var out struct{ ID, URL string }
	if err = json.Unmarshal(b, &out); err != nil {
		return PaymentSession{}, err
	}
	return PaymentSession{Provider: ProviderStripeConnect, Env: c.Env, ExternalID: out.ID, CheckoutURL: out.URL}, nil
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
