package payments

import (
	"testing"

	"github.com/google/uuid"
	"github.com/tainalo2/devis_generator/backend/internal/biz"
	"github.com/tainalo2/devis_generator/backend/internal/paymentbyo"
)

func TestOnlyIssuedUnpaidInvoicesArePayable(t *testing.T) {
	for _, status := range []string{"draft", "cancelled", "paid"} {
		if isPayableDocument(biz.Document{Kind: "invoice", Status: status, TotalCents: 100}) {
			t.Fatalf("%s was payable", status)
		}
	}
	for _, status := range []string{"sent", "accepted"} {
		if !isPayableDocument(biz.Document{Kind: "invoice", Status: status, TotalCents: 100}) {
			t.Fatalf("%s was not payable", status)
		}
	}
}

func TestIdempotencyKeyDeterministicAndTenantScoped(t *testing.T) {
	o := uuid.New()
	d := uuid.New()
	a := IdempotencyKey(o, d, paymentbyo.ProviderStripe)
	if a != IdempotencyKey(o, d, paymentbyo.ProviderStripe) {
		t.Fatal("not deterministic")
	}
	if a == IdempotencyKey(uuid.New(), d, paymentbyo.ProviderStripe) {
		t.Fatal("not tenant scoped")
	}
	if a == IdempotencyKey(o, d, paymentbyo.ProviderMollie) {
		t.Fatal("not provider scoped")
	}
}
