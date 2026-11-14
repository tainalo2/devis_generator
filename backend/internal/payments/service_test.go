package payments

import (
	"github.com/google/uuid"
	"github.com/tainalo2/devis_generator/backend/internal/paymentbyo"
	"testing"
)

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
