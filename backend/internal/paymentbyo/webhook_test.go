package paymentbyo

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProviderWebhooksFailClosed(t *testing.T) {
	if _, err := (&MollieAdapter{}).VerifyWebhook(Config{}, nil, []byte(`id=x`)); err == nil {
		t.Fatal("mollie accepted without retrieval")
	}
	if _, err := (&PayPalAdapter{}).VerifyWebhook(Config{}, map[string]string{"Paypal-Transmission-Id": "x"}, nil); err == nil {
		t.Fatal("paypal accepted incomplete headers")
	}
}

func TestMollieWebhookRetrievesCanonicalPayment(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/payments/tr_1" {
			t.Errorf("path=%s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"id": "tr_1", "status": "paid", "metadata": map[string]string{"tenant_id": "tenant", "document_id": "doc"}})
	}))
	defer srv.Close()
	a := &MollieAdapter{HTTP: NewHTTPClient(srv.URL)}
	e, err := a.VerifyWebhook(Config{MollieAPIKey: "test_x"}, nil, []byte("id=tr_1"))
	if err != nil || e.Status != "paid" || e.TenantID != "tenant" || e.DocumentID != "doc" {
		t.Fatalf("event=%#v err=%v", e, err)
	}
}
