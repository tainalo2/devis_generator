package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/tainalo2/devis_generator/backend/internal/authstore"
	"github.com/tainalo2/devis_generator/backend/internal/paymentflags"
)

type memoryPaymentFlags struct {
	features paymentflags.Features
	err      error
}

func (m *memoryPaymentFlags) Get(context.Context) (paymentflags.Features, error) {
	return m.features, m.err
}

func (m *memoryPaymentFlags) Set(_ context.Context, features paymentflags.Features) (paymentflags.Features, error) {
	if m.err != nil {
		return paymentflags.Features{}, m.err
	}
	m.features = features
	return features, nil
}

func requestAsUser(method, target string, body *bytes.Reader, user authstore.User) *http.Request {
	req := httptest.NewRequest(method, target, body)
	return req.WithContext(context.WithValue(req.Context(), ctxUserKey, user))
}

func TestPaymentFeaturesDefaultToDisabled(t *testing.T) {
	store := &memoryPaymentFlags{}
	s := New(Options{PaymentFlags: store})

	req := requestAsUser(http.MethodGet, "/api/v1/payment/features", bytes.NewReader(nil), authstore.User{ID: uuid.New()})
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var got paymentflags.Features
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.RIB || got.BYOProvider || got.StripeConnect {
		t.Fatalf("features must be opt-in, got %+v", got)
	}
}

func TestAdminCanTogglePaymentFeaturesIndependently(t *testing.T) {
	store := &memoryPaymentFlags{}
	s := New(Options{PaymentFlags: store})
	admin := authstore.User{ID: uuid.New(), IsAdmin: true}
	body := bytes.NewReader([]byte(`{"rib":true,"byo_provider":false,"stripe_connect":true}`))
	req := requestAsUser(http.MethodPut, "/api/v1/admin/payment-features", body, admin)
	rec := httptest.NewRecorder()
	s.handlePutPaymentFeatures(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !store.features.RIB || store.features.BYOProvider || !store.features.StripeConnect {
		t.Fatalf("features not updated independently: %+v", store.features)
	}
}

func TestNonAdminCannotReadOrChangeAdminPaymentFeatures(t *testing.T) {
	s := New(Options{PaymentFlags: &memoryPaymentFlags{}})
	user := authstore.User{ID: uuid.New()}

	for _, method := range []string{http.MethodGet, http.MethodPut} {
		req := requestAsUser(method, "/api/v1/admin/payment-features", bytes.NewReader([]byte(`{}`)), user)
		rec := httptest.NewRecorder()
		s.requireAdmin(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s status=%d, want 404", method, rec.Code)
		}
	}
}

func TestDisabledPaymentModeEndpointIsInaccessible(t *testing.T) {
	store := &memoryPaymentFlags{features: paymentflags.Features{RIB: true}}
	s := New(Options{PaymentFlags: store})
	hit := false
	handler := s.requirePaymentFeature(paymentflags.BYOProvider, func(w http.ResponseWriter, _ *http.Request) {
		hit = true
		w.WriteHeader(http.StatusNoContent)
	})

	rec := httptest.NewRecorder()
	handler(rec, httptest.NewRequest(http.MethodPost, "/api/v1/payments/byo/checkout", nil))
	if rec.Code != http.StatusNotFound || hit {
		t.Fatalf("disabled endpoint status=%d hit=%v", rec.Code, hit)
	}

	store.features.BYOProvider = true
	rec = httptest.NewRecorder()
	handler(rec, httptest.NewRequest(http.MethodPost, "/api/v1/payments/byo/checkout", nil))
	if rec.Code != http.StatusNoContent || !hit {
		t.Fatalf("enabled endpoint status=%d hit=%v", rec.Code, hit)
	}
}
