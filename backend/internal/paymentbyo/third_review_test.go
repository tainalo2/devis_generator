package paymentbyo

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestPayPalCaptureCompletedUsesRelatedOrderID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/oauth2/token" {
			_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "tok"})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"verification_status": "SUCCESS"})
	}))
	defer srv.Close()
	a := NewPayPalAdapter(NewHTTPClient(srv.URL), NewHTTPClient(srv.URL))
	body := []byte(`{"id":"WH-1","event_type":"PAYMENT.CAPTURE.COMPLETED","resource":{"id":"CAPTURE-1","status":"COMPLETED","custom_id":"tenant-1","supplementary_data":{"related_ids":{"order_id":"ORDER-1"}}}}`)
	h := map[string]string{"Paypal-Transmission-Id": "x", "Paypal-Transmission-Time": "x", "Paypal-Cert-Url": "x", "Paypal-Auth-Algo": "x", "Paypal-Transmission-Sig": "x"}
	ev, err := a.VerifyWebhook(Config{Env: EnvSandbox, PayPalClientID: "id", PayPalClientSecret: "secret", PayPalWebhookID: "wh"}, h, body)
	if err != nil {
		t.Fatal(err)
	}
	if ev.ExternalID != "ORDER-1" || ev.Status != "paid" {
		t.Fatalf("event=%+v", ev)
	}
}

func TestStripeConnectCheckoutUsesPlatformAccountAndBoundedFee(t *testing.T) {
	var auth, account string
	var form url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth, account = r.Header.Get("Authorization"), r.Header.Get("Stripe-Account")
		b, _ := io.ReadAll(r.Body)
		form, _ = url.ParseQuery(string(b))
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "cs_connect", "url": "https://checkout"})
	}))
	defer srv.Close()
	a := &StripeConnectAdapter{Stripe: &StripeAdapter{HTTP: NewHTTPClient(srv.URL)}, ComplianceAck: true, FeeBPS: 500, FeeFixedCents: 25, FeeCapCents: 100}
	s, err := a.CreatePayment(t.Context(), Config{Env: EnvSandbox, StripeConnectSecret: "sk_test_platform", StripeConnectWebhookSecret: "whsec"}, PaymentRequest{Amount: 5000, Currency: "EUR", ConnectedAccountID: "acct_1"})
	if err != nil {
		t.Fatal(err)
	}
	if s.ExternalID != "cs_connect" || auth != "Bearer sk_test_platform" || account != "acct_1" || form.Get("payment_intent_data[application_fee_amount]") != "100" {
		t.Fatalf("auth=%q account=%q form=%v", auth, account, form)
	}
}

func TestStripeConnectCheckoutRejectsMissingAccountAndCompliance(t *testing.T) {
	a := &StripeConnectAdapter{Stripe: &StripeAdapter{HTTP: NewHTTPClient("http://invalid")}}
	_, err := a.CreatePayment(t.Context(), Config{Env: EnvSandbox, StripeConnectSecret: "sk_test_platform", StripeConnectWebhookSecret: "whsec"}, PaymentRequest{Amount: 100, Currency: "EUR"})
	if err == nil {
		t.Fatal("expected fail-closed rejection")
	}
}
