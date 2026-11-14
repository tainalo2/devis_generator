package payments

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/tainalo2/devis_generator/backend/internal/biz"
	"github.com/tainalo2/devis_generator/backend/internal/paymentbyo"
	"github.com/tainalo2/devis_generator/backend/internal/paymentsecrets"
)

type reviewVault struct{ reveals int }

func (*reviewVault) Seal([]byte) (paymentsecrets.SealedSecret, error) {
	return paymentsecrets.SealedSecret{}, nil
}
func (*reviewVault) Put(context.Context, paymentsecrets.Actor, uuid.UUID, paymentsecrets.Kind, string, []byte) (paymentsecrets.Metadata, error) {
	return paymentsecrets.Metadata{}, nil
}
func (v *reviewVault) Reveal(context.Context, paymentsecrets.Actor, uuid.UUID, paymentsecrets.Kind, string) ([]byte, paymentsecrets.Metadata, error) {
	v.reveals++
	return []byte("secret"), paymentsecrets.Metadata{}, nil
}
func (*reviewVault) Delete(context.Context, paymentsecrets.Actor, uuid.UUID, paymentsecrets.Kind, string) error {
	return nil
}

type reviewAdapter struct {
	verifies int
	event    paymentbyo.WebhookEvent
	create   func()
}

func (*reviewAdapter) ValidateConfig(paymentbyo.Config) error { return nil }
func (a *reviewAdapter) CreatePayment(context.Context, paymentbyo.Config, paymentbyo.PaymentRequest) (paymentbyo.PaymentSession, error) {
	if a.create != nil {
		a.create()
	}
	return paymentbyo.PaymentSession{ExternalID: "ext-created", CheckoutURL: "https://checkout"}, nil
}
func (a *reviewAdapter) VerifyWebhook(paymentbyo.Config, map[string]string, []byte) (paymentbyo.WebhookEvent, error) {
	a.verifies++
	return a.event, nil
}
func (*reviewAdapter) Revoke(paymentbyo.Config) error       { return nil }
func (*reviewAdapter) Fingerprint(paymentbyo.Config) string { return "fake" }

type reviewDocs struct{ doc biz.Document }

func (d reviewDocs) GetDocument(context.Context, uuid.UUID, uuid.UUID) (biz.Document, error) {
	return d.doc, nil
}

func TestPostgresWebhookRoutesOneTenantAndUnknownReadsNoSecrets(t *testing.T) {
	pool, ctx, owner, doc := paymentTestDB(t)
	ext := "ext-route"
	if _, err := pool.Exec(ctx, `INSERT INTO payment_provider_configs(owner_id,provider,environment) VALUES($1,'stripe','sandbox')`, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO payment_attempts(owner_id,document_id,provider,external_id,idempotency_key,status) VALUES($1,$2,'stripe',$3,$4,'pending')`, owner, doc, ext, uuid.NewString()); err != nil {
		t.Fatal(err)
	}
	v := &reviewVault{}
	a := &reviewAdapter{event: paymentbyo.WebhookEvent{EventID: "evt-route", ExternalID: ext, TenantID: owner.String(), DocumentID: doc.String(), Status: "pending"}}
	s := New(pool, v, reviewDocs{}, map[paymentbyo.Provider]paymentbyo.ProviderAdapter{paymentbyo.ProviderStripe: a})
	if err := s.ProcessWebhook(ctx, paymentbyo.ProviderStripe, nil, []byte(`{"data":{"object":{"id":"ext-route"}}}`)); err != nil {
		t.Fatal(err)
	}
	if a.verifies != 1 {
		t.Fatalf("verifiers=%d", a.verifies)
	}
	before := v.reveals
	if err := s.ProcessWebhook(ctx, paymentbyo.ProviderStripe, nil, []byte(`{"data":{"object":{"id":"unknown"}}}`)); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err=%v", err)
	}
	if a.verifies != 1 || v.reveals != before {
		t.Fatalf("unknown verifier=%d reveals=%d before=%d", a.verifies, v.reveals, before)
	}
}

func TestPostgresCheckoutFinalizationLostReturnsNoSession(t *testing.T) {
	pool, ctx, owner, docID := paymentTestDB(t)
	if _, err := pool.Exec(ctx, `INSERT INTO payment_provider_configs(owner_id,provider,environment) VALUES($1,'stripe','sandbox')`, owner); err != nil {
		t.Fatal(err)
	}
	doc := biz.Document{ID: docID, Kind: "invoice", Status: "sent", TotalCents: 1000, Currency: "EUR"}
	a := &reviewAdapter{}
	a.create = func() { _, _ = pool.Exec(ctx, `DELETE FROM payment_attempts WHERE owner_id=$1`, owner) }
	s := New(pool, &reviewVault{}, reviewDocs{doc}, map[paymentbyo.Provider]paymentbyo.ProviderAdapter{paymentbyo.ProviderStripe: a})
	sess, err := s.Checkout(ctx, paymentsecrets.Actor{UserID: owner}, docID, CheckoutInput{Provider: paymentbyo.ProviderStripe})
	if err == nil || sess.ExternalID != "" {
		t.Fatalf("session=%+v err=%v", sess, err)
	}
}

func TestPostgresPayPalCaptureCorrelatesOrderAndMarksPaid(t *testing.T) {
	pool, ctx, owner, doc := paymentTestDB(t)
	if _, err := pool.Exec(ctx, `INSERT INTO payment_provider_configs(owner_id,provider,environment) VALUES($1,'paypal','sandbox')`, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO payment_attempts(owner_id,document_id,provider,external_id,idempotency_key,status) VALUES($1,$2,'paypal','ORDER-1',$3,'pending')`, owner, doc, uuid.NewString()); err != nil {
		t.Fatal(err)
	}
	a := &reviewAdapter{event: paymentbyo.WebhookEvent{EventID: "WH-1", ExternalID: "ORDER-1", TenantID: owner.String(), Status: "paid"}}
	s := New(pool, &reviewVault{}, reviewDocs{}, map[paymentbyo.Provider]paymentbyo.ProviderAdapter{paymentbyo.ProviderPayPal: a})
	body := []byte(`{"event_type":"PAYMENT.CAPTURE.COMPLETED","resource":{"id":"CAPTURE-1","supplementary_data":{"related_ids":{"order_id":"ORDER-1"}}}}`)
	if err := s.ProcessWebhook(ctx, paymentbyo.ProviderPayPal, nil, body); err != nil {
		t.Fatal(err)
	}
	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM documents WHERE id=$1`, doc).Scan(&status); err != nil || status != "paid" {
		t.Fatalf("status=%q err=%v", status, err)
	}
}
