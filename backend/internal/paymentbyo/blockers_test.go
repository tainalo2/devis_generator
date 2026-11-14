package paymentbyo

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"
)

func TestStripeCheckoutUsesFormContract(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
			t.Errorf("content-type=%q", got)
		}
		if got := r.Header.Get("Idempotency-Key"); got != "stable" {
			t.Errorf("idempotency=%q", got)
		}
		b, _ := io.ReadAll(r.Body)
		v, _ := url.ParseQuery(string(b))
		for k, want := range map[string]string{"mode": "payment", "line_items[0][price_data][unit_amount]": "1234", "line_items[0][price_data][currency]": "eur", "line_items[0][quantity]": "1", "metadata[tenant_id]": "tenant", "metadata[document_id]": "doc"} {
			if v.Get(k) != want {
				t.Errorf("%s=%q want %q", k, v.Get(k), want)
			}
		}
		json.NewEncoder(w).Encode(map[string]string{"id": "cs_1", "url": "https://checkout"})
	}))
	defer srv.Close()
	a := &StripeAdapter{HTTP: NewHTTPClient(srv.URL)}
	_, err := a.CreatePayment(t.Context(), Config{Env: EnvSandbox, StripeSecretKey: "sk_test_x", StripeWebhookSecret: "whsec_x"}, PaymentRequest{Amount: 1234, Currency: "EUR", TenantID: "tenant", DocumentID: "doc", ReturnURL: "https://ok", CancelURL: "https://cancel", IdempotencyKey: "stable"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestStripeWebhookRejectsOldTimestampAndReturnsPayloadIdentity(t *testing.T) {
	body := []byte(`{"id":"evt_1","type":"checkout.session.completed","data":{"object":{"id":"cs_1","payment_status":"paid","metadata":{"tenant_id":"tenant","document_id":"doc"}}}}`)
	secret := "whsec_x"
	ts := time.Now().Add(-6 * time.Minute).Unix()
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(ts, 10) + "." + string(body)))
	sig := "t=" + strconv.FormatInt(ts, 10) + ",v1=" + hex.EncodeToString(mac.Sum(nil))
	if _, err := (&StripeAdapter{Now: time.Now}).VerifyWebhook(Config{StripeWebhookSecret: secret}, map[string]string{"Stripe-Signature": sig}, body); err == nil {
		t.Fatal("accepted stale signature")
	}
}

func TestPayPalProtocolAndEnvironment(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			if r.Header.Get("Authorization") != "Basic "+base64.StdEncoding.EncodeToString([]byte("id:secret")) {
				t.Error("bad basic auth")
			}
			json.NewEncoder(w).Encode(map[string]string{"access_token": "tok"})
			return
		}
		if r.Header.Get("Content-Type") != "application/json" || r.Header.Get("PayPal-Request-Id") != "stable" {
			t.Error("bad order headers")
		}
		json.NewEncoder(w).Encode(map[string]any{"id": "o1", "links": []map[string]string{{"rel": "approve", "href": "https://approve"}}})
	}))
	defer srv.Close()
	a := NewPayPalAdapter(NewHTTPClient(srv.URL), NewHTTPClient(srv.URL))
	_, err := a.CreatePayment(t.Context(), Config{Env: EnvSandbox, PayPalClientID: "id", PayPalClientSecret: "secret", PayPalWebhookID: "wh"}, PaymentRequest{Amount: 100, Currency: "EUR", IdempotencyKey: "stable"})
	if err != nil {
		t.Fatal(err)
	}
}
