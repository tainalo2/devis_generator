package payments

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tainalo2/devis_generator/backend/internal/biz"
	"github.com/tainalo2/devis_generator/backend/internal/paymentbyo"
	"github.com/tainalo2/devis_generator/backend/internal/paymentsecrets"
)

var ErrNotFound = errors.New("payment configuration not found")

type ConfigInput struct {
	Provider                   paymentbyo.Provider `json:"provider"`
	Env                        paymentbyo.Env      `json:"env"`
	StripeSecretKey            string              `json:"stripe_secret_key,omitempty"`
	StripeWebhookSecret        string              `json:"stripe_webhook_secret,omitempty"`
	MollieAPIKey               string              `json:"mollie_api_key,omitempty"`
	PayPalClientID             string              `json:"paypal_client_id,omitempty"`
	PayPalClientSecret         string              `json:"paypal_client_secret,omitempty"`
	PayPalWebhookID            string              `json:"paypal_webhook_id,omitempty"`
	StripeConnectSecret        string              `json:"stripe_connect_secret,omitempty"`
	StripeConnectWebhookSecret string              `json:"stripe_connect_webhook_secret,omitempty"`
}

type ConfigView struct {
	Provider paymentbyo.Provider `json:"provider"`
	Env      paymentbyo.Env      `json:"env"`
	Masked   map[string]string   `json:"masked"`
}
type CheckoutInput struct {
	Provider  paymentbyo.Provider `json:"provider"`
	ReturnURL string              `json:"return_url"`
	CancelURL string              `json:"cancel_url"`
}
type DocumentStore interface {
	GetDocument(context.Context, uuid.UUID, uuid.UUID) (biz.Document, error)
}
type Vault interface {
	Put(context.Context, paymentsecrets.Actor, uuid.UUID, paymentsecrets.Kind, string, []byte) (paymentsecrets.Metadata, error)
	Reveal(context.Context, paymentsecrets.Actor, uuid.UUID, paymentsecrets.Kind, string) ([]byte, paymentsecrets.Metadata, error)
	Delete(context.Context, paymentsecrets.Actor, uuid.UUID, paymentsecrets.Kind, string) error
}

type Service struct {
	pool     *pgxpool.Pool
	vault    Vault
	docs     DocumentStore
	adapters map[paymentbyo.Provider]paymentbyo.ProviderAdapter
}

func New(pool *pgxpool.Pool, vault Vault, docs DocumentStore, adapters map[paymentbyo.Provider]paymentbyo.ProviderAdapter) *Service {
	return &Service{pool: pool, vault: vault, docs: docs, adapters: adapters}
}

func secretFields(in ConfigInput) map[paymentsecrets.Kind]string {
	return map[paymentsecrets.Kind]string{
		paymentsecrets.KindStripeAPIKey: in.StripeSecretKey, paymentsecrets.KindStripeWebhookSecret: in.StripeWebhookSecret,
		paymentsecrets.KindMollieAPIKey: in.MollieAPIKey, paymentsecrets.KindPayPalClientID: in.PayPalClientID,
		paymentsecrets.KindPayPalClientSecret: in.PayPalClientSecret, paymentsecrets.KindPayPalWebhookID: in.PayPalWebhookID,
		paymentsecrets.KindStripeConnectSecret: in.StripeConnectSecret, paymentsecrets.KindStripeConnectWebhookSecret: in.StripeConnectWebhookSecret,
	}
}
func kindsFor(p paymentbyo.Provider) []paymentsecrets.Kind {
	switch p {
	case paymentbyo.ProviderStripe:
		return []paymentsecrets.Kind{paymentsecrets.KindStripeAPIKey, paymentsecrets.KindStripeWebhookSecret}
	case paymentbyo.ProviderMollie:
		return []paymentsecrets.Kind{paymentsecrets.KindMollieAPIKey}
	case paymentbyo.ProviderPayPal:
		return []paymentsecrets.Kind{paymentsecrets.KindPayPalClientID, paymentsecrets.KindPayPalClientSecret, paymentsecrets.KindPayPalWebhookID}
	case paymentbyo.ProviderStripeConnect:
		return []paymentsecrets.Kind{paymentsecrets.KindStripeConnectSecret, paymentsecrets.KindStripeConnectWebhookSecret}
	}
	return nil
}
func (s *Service) PutConfig(ctx context.Context, actor paymentsecrets.Actor, in ConfigInput) (ConfigView, error) {
	ad := s.adapters[in.Provider]
	if ad == nil && in.Provider != paymentbyo.ProviderStripeConnect {
		return ConfigView{}, fmt.Errorf("unsupported provider")
	}
	cfg := paymentbyo.Config{Provider: in.Provider, Env: in.Env, StripeSecretKey: in.StripeSecretKey, StripeWebhookSecret: in.StripeWebhookSecret, MollieAPIKey: in.MollieAPIKey, PayPalClientID: in.PayPalClientID, PayPalClientSecret: in.PayPalClientSecret, PayPalWebhookID: in.PayPalWebhookID}
	if in.Provider == paymentbyo.ProviderStripeConnect {
		if in.StripeConnectSecret == "" || in.StripeConnectWebhookSecret == "" {
			return ConfigView{}, fmt.Errorf("missing connect credentials")
		}
	} else if err := ad.ValidateConfig(cfg); err != nil {
		return ConfigView{}, err
	}
	masked := map[string]string{}
	for _, k := range kindsFor(in.Provider) {
		v := secretFields(in)[k]
		m, err := s.vault.Put(ctx, actor, actor.UserID, k, string(in.Provider), []byte(v))
		if err != nil {
			return ConfigView{}, err
		}
		masked[string(k)] = m.MaskedValue
	}
	_, err := s.pool.Exec(ctx, `INSERT INTO payment_provider_configs(owner_id,provider,environment) VALUES($1,$2,$3) ON CONFLICT(owner_id,provider) DO UPDATE SET environment=EXCLUDED.environment,updated_at=NOW()`, actor.UserID, in.Provider, in.Env)
	return ConfigView{Provider: in.Provider, Env: in.Env, Masked: masked}, err
}
func (s *Service) GetConfig(ctx context.Context, actor paymentsecrets.Actor, p paymentbyo.Provider) (ConfigView, error) {
	var env paymentbyo.Env
	if err := s.pool.QueryRow(ctx, `SELECT environment FROM payment_provider_configs WHERE owner_id=$1 AND provider=$2`, actor.UserID, p).Scan(&env); errors.Is(err, pgx.ErrNoRows) {
		return ConfigView{}, ErrNotFound
	} else if err != nil {
		return ConfigView{}, err
	}
	out := ConfigView{Provider: p, Env: env, Masked: map[string]string{}}
	for _, k := range kindsFor(p) {
		_, m, e := s.vault.Reveal(ctx, actor, actor.UserID, k, string(p))
		if e != nil {
			return ConfigView{}, e
		}
		out.Masked[string(k)] = m.MaskedValue
	}
	return out, nil
}
func (s *Service) DeleteConfig(ctx context.Context, actor paymentsecrets.Actor, p paymentbyo.Provider) error {
	for _, k := range kindsFor(p) {
		if err := s.vault.Delete(ctx, actor, actor.UserID, k, string(p)); err != nil && !errors.Is(err, paymentsecrets.ErrNotFound) {
			return err
		}
	}
	tag, err := s.pool.Exec(ctx, `DELETE FROM payment_provider_configs WHERE owner_id=$1 AND provider=$2`, actor.UserID, p)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}
func (s *Service) load(ctx context.Context, actor paymentsecrets.Actor, p paymentbyo.Provider) (paymentbyo.Config, error) {
	v, e := s.GetConfig(ctx, actor, p)
	if e != nil {
		return paymentbyo.Config{}, e
	}
	c := paymentbyo.Config{Provider: p, Env: v.Env}
	for _, k := range kindsFor(p) {
		b, _, e := s.vault.Reveal(ctx, actor, actor.UserID, k, string(p))
		if e != nil {
			return c, e
		}
		switch k {
		case paymentsecrets.KindStripeAPIKey:
			c.StripeSecretKey = string(b)
		case paymentsecrets.KindStripeWebhookSecret:
			c.StripeWebhookSecret = string(b)
		case paymentsecrets.KindMollieAPIKey:
			c.MollieAPIKey = string(b)
		case paymentsecrets.KindPayPalClientID:
			c.PayPalClientID = string(b)
		case paymentsecrets.KindPayPalClientSecret:
			c.PayPalClientSecret = string(b)
		case paymentsecrets.KindPayPalWebhookID:
			c.PayPalWebhookID = string(b)
		}
	}
	return c, nil
}
func IdempotencyKey(owner, doc uuid.UUID, p paymentbyo.Provider) string {
	h := sha256.Sum256([]byte(owner.String() + ":" + doc.String() + ":" + string(p)))
	return hex.EncodeToString(h[:])
}
func (s *Service) Checkout(ctx context.Context, actor paymentsecrets.Actor, docID uuid.UUID, in CheckoutInput) (paymentbyo.PaymentSession, error) {
	doc, err := s.docs.GetDocument(ctx, actor.UserID, docID)
	if err != nil {
		return paymentbyo.PaymentSession{}, ErrNotFound
	}
	if doc.Kind != "invoice" || doc.TotalCents <= 0 {
		return paymentbyo.PaymentSession{}, fmt.Errorf("document is not a payable invoice")
	}
	cfg, err := s.load(ctx, actor, in.Provider)
	if err != nil {
		return paymentbyo.PaymentSession{}, err
	}
	req := paymentbyo.PaymentRequest{DocumentID: doc.ID.String(), TenantID: actor.UserID.String(), Amount: doc.TotalCents, Currency: strings.ToUpper(doc.Currency), ReturnURL: in.ReturnURL, CancelURL: in.CancelURL, IdempotencyKey: IdempotencyKey(actor.UserID, doc.ID, in.Provider)}
	session, err := s.adapters[in.Provider].CreatePayment(ctx, cfg, req)
	if err != nil {
		return session, err
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO payment_attempts(owner_id,document_id,provider,external_id,idempotency_key) VALUES($1,$2,$3,$4,$5) ON CONFLICT(provider,idempotency_key) DO UPDATE SET external_id=COALESCE(payment_attempts.external_id,EXCLUDED.external_id)`, actor.UserID, doc.ID, in.Provider, session.ExternalID, req.IdempotencyKey)
	return session, err
}

// ProcessWebhook verifies before opening the transaction. The UNIQUE ledger key
// serializes retries; the document and attempt status update commit atomically.
func (s *Service) ProcessWebhook(ctx context.Context, p paymentbyo.Provider, eventID, externalID string, headers map[string]string, body []byte) error {
	if eventID == "" || externalID == "" {
		return fmt.Errorf("missing event identity")
	}
	var owner, attempt, doc uuid.UUID
	if err := s.pool.QueryRow(ctx, `SELECT owner_id,id,document_id FROM payment_attempts WHERE provider=$1 AND external_id=$2`, p, externalID).Scan(&owner, &attempt, &doc); errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	actor := paymentsecrets.Actor{UserID: owner, RequestID: "webhook:" + eventID}
	cfg, err := s.load(ctx, actor, p)
	if err != nil {
		return err
	}
	ev, err := s.adapters[p].VerifyWebhook(cfg, headers, body)
	if err != nil {
		return err
	}
	if ev.ExternalID != "" && ev.ExternalID != externalID {
		return fmt.Errorf("external id mismatch")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `INSERT INTO payment_webhook_events(provider,event_id,owner_id,payment_attempt_id) VALUES($1,$2,$3,$4) ON CONFLICT(provider,event_id) DO NOTHING`, p, eventID, owner, attempt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return tx.Commit(ctx)
	}
	if _, err = tx.Exec(ctx, `UPDATE payment_attempts SET status=$1,updated_at=NOW() WHERE id=$2 AND owner_id=$3`, ev.Status, attempt, owner); err != nil {
		return err
	}
	if ev.Status == "paid" {
		if _, err = tx.Exec(ctx, `UPDATE documents SET status='paid',paid_at=COALESCE(paid_at,NOW()),updated_at=NOW() WHERE id=$1 AND user_id=$2`, doc, owner); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// ResolveConnectOwner trusts only the persisted unique account mapping.
func (s *Service) ResolveConnectOwner(ctx context.Context, accountID string) (uuid.UUID, error) {
	var id uuid.UUID
	err := s.pool.QueryRow(ctx, `SELECT owner_id FROM stripe_connect_accounts WHERE account_id=$1`, accountID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrNotFound
	}
	return id, err
}
