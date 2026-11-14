package payments

import (
	"testing"

	"github.com/tainalo2/devis_generator/backend/internal/paymentbyo"
)

func TestWebhookRouteIdentifier(t *testing.T) {
	if got, _ := webhookRouteIdentifier(paymentbyo.ProviderStripe, nil, []byte(`{"data":{"object":{"id":"cs_123"}}}`)); got != "cs_123" {
		t.Fatalf("stripe=%q", got)
	}
	if got, _ := webhookRouteIdentifier(paymentbyo.ProviderPayPal, nil, []byte(`{"event_type":"PAYMENT.CAPTURE.COMPLETED","resource":{"id":"CAP","supplementary_data":{"related_ids":{"order_id":"ORDER"}}}}`)); got != "ORDER" {
		t.Fatalf("paypal=%q", got)
	}
	if got, _ := webhookRouteIdentifier(paymentbyo.ProviderMollie, nil, []byte("id=tr_1")); got != "tr_1" {
		t.Fatalf("mollie=%q", got)
	}
}
