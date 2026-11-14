package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tainalo2/devis_generator/backend/internal/payments"
)

func TestStripeConnectTenantSecretsAreForbiddenJSONFields(t *testing.T) {
	for _, body := range []string{`{"env":"live","stripe_connect_secret":"tenant"}`, `{"env":"live","stripe_connect_webhook_secret":"tenant"}`} {
		r := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(body))
		var in payments.ConfigInput
		if err := decodeJSON(r, &in); err == nil {
			t.Fatalf("tenant operator secret accepted: %s", body)
		}
	}
}

func TestWebhookAdmissionRejectsBeforeProviderOrService(t *testing.T) {
	ipLimiter := NewRateLimiter(1, time.Hour)
	if !ipLimiter.Allow("203.0.113.9") {
		t.Fatal("failed to prime limiter")
	}
	s := &Server{webhookIPLimiter: ipLimiter, webhookGlobalLimiter: NewRateLimiter(1, time.Hour)}
	r := httptest.NewRequest(http.MethodPost, "/api/v1/payments/webhooks/existing-or-unknown", strings.NewReader(`{"id":"candidate"}`))
	r.RemoteAddr = "203.0.113.9:1234"
	w := httptest.NewRecorder()
	s.handlePaymentWebhook(w, r)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("status=%d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Webhook invalide") {
		t.Fatalf("non-uniform body: %s", w.Body.String())
	}
}

func TestWebhookUnknownProviderUsesUniformInvalidResponse(t *testing.T) {
	s := &Server{webhookIPLimiter: NewRateLimiter(10, time.Hour), webhookGlobalLimiter: NewRateLimiter(10, time.Hour)}
	r := httptest.NewRequest(http.MethodPost, "/api/v1/payments/webhooks/unknown", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	s.handlePaymentWebhook(w, r)
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "Webhook invalide") {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}
