package paymentbyo

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestContract(t *testing.T) {
	// Stripe
	stripeSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": "cs_1", "url": "https://stripe/checkout"})
	}))
	defer stripeSrv.Close()
	sa := &StripeAdapter{HTTP: NewHTTPClient(stripeSrv.URL)}
	cfg := Config{Provider: ProviderStripe, Env: EnvSandbox, StripeSecretKey: "sk_test_abc1234", StripeWebhookSecret: "whsec_test"}
	if err := sa.ValidateConfig(cfg); err != nil { t.Fatal(err) }
	sess, err := sa.CreatePayment(t.Context(), cfg, PaymentRequest{Amount: 1000, Currency: "EUR", ReturnURL: "https://r"})
	if err != nil { t.Fatal(err) }
	if sess.CheckoutURL == "" { t.Fatal("no url") }
	// invalid env mix
	if err := sa.ValidateConfig(Config{Provider: ProviderStripe, Env: EnvLive, StripeSecretKey: "sk_test_x", StripeWebhookSecret: "w"}); err == nil { t.Fatal("expected error") }

	// Mollie
	mollieSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": "tr_1", "_links": map[string]any{"checkout": map[string]string{"href": "https://mollie/checkout"}}})
	}))
	defer mollieSrv.Close()
	ma := &MollieAdapter{HTTP: NewHTTPClient(mollieSrv.URL)}
	mcfg := Config{Provider: ProviderMollie, Env: EnvSandbox, MollieAPIKey: "test_key1234"}
	if err := ma.ValidateConfig(mcfg); err != nil { t.Fatal(err) }
	msess, err := ma.CreatePayment(t.Context(), mcfg, PaymentRequest{Amount: 1000, Currency: "EUR"})
	if err != nil { t.Fatal(err) }
	if msess.ExternalID != "tr_1" { t.Fatal("id") }

	// PayPal
	paypalSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/v1/oauth2/token" {
			json.NewEncoder(w).Encode(map[string]string{"access_token": "tok"})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"id": "order_1", "links": []map[string]string{{"rel": "approve", "href": "https://paypal/approve"}}})
	}))
	defer paypalSrv.Close()
	pa := &PayPalAdapter{HTTP: NewHTTPClient(paypalSrv.URL)}
	pcfg := Config{Provider: ProviderPayPal, Env: EnvSandbox, PayPalClientID: "cid", PayPalClientSecret: "csec", PayPalWebhookID: "wh"}
	if err := pa.ValidateConfig(pcfg); err != nil { t.Fatal(err) }
	psess, err := pa.CreatePayment(t.Context(), pcfg, PaymentRequest{Amount: 1000, Currency: "USD"})
	if err != nil { t.Fatal(err) }
	if psess.CheckoutURL != "https://paypal/approve" { t.Fatal("url") }
}
