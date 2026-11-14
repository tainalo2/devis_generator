package paymentbyo

import "testing"

func TestProviderWebhooksFailClosedWithoutVerifier(t *testing.T) {
	cfg := Config{}
	if _, err := (&MollieAdapter{}).VerifyWebhook(cfg, nil, []byte(`{"id":"x"}`)); err == nil {
		t.Fatal("mollie accepted unverified webhook")
	}
	if _, err := (&PayPalAdapter{}).VerifyWebhook(cfg, map[string]string{"Paypal-Transmission-Id": "x"}, nil); err == nil {
		t.Fatal("paypal accepted unverified webhook")
	}
}
func TestInjectedWebhookVerifierIsUsed(t *testing.T) {
	called := false
	m := &MollieAdapter{WebhookVerifier: func(Config, map[string]string, []byte) (WebhookEvent, error) {
		called = true
		return WebhookEvent{ExternalID: "p1"}, nil
	}}
	e, err := m.VerifyWebhook(Config{}, nil, nil)
	if err != nil || !called || e.ExternalID != "p1" {
		t.Fatalf("injected verifier not used: %#v %v", e, err)
	}
}
