package payments

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
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
	Provider            paymentbyo.Provider `json:"provider"`
	Env                 paymentbyo.Env      `json:"env"`
	StripeSecretKey     string              `json:"stripe_secret_key,omitempty"`
	StripeWebhookSecret string              `json:"stripe_webhook_secret,omitempty"`
	MollieAPIKey        string              `json:"mollie_api_key,omitempty"`
	PayPalClientID      string              `json:"paypal_client_id,omitempty"`
	PayPalClientSecret  string              `json:"paypal_client_secret,omitempty"`
	PayPalWebhookID     string              `json:"paypal_webhook_id,omitempty"`
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
	Seal([]byte) (paymentsecrets.SealedSecret, error)
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
		return []paymentsecrets.Kind{}
	}
	return nil
}
func (s *Service) PutConfig(ctx context.Context, actor paymentsecrets.Actor, in ConfigInput) (ConfigView, error) {
	ad := s.adapters[in.Provider]
	if ad == nil {
		return ConfigView{}, fmt.Errorf("unsupported provider")
	}
	cfg := paymentbyo.Config{Provider: in.Provider, Env: in.Env, StripeSecretKey: in.StripeSecretKey, StripeWebhookSecret: in.StripeWebhookSecret, MollieAPIKey: in.MollieAPIKey, PayPalClientID: in.PayPalClientID, PayPalClientSecret: in.PayPalClientSecret, PayPalWebhookID: in.PayPalWebhookID}
	if err := ad.ValidateConfig(cfg); err != nil {
		return ConfigView{}, err
	}
	sealed := map[paymentsecrets.Kind]paymentsecrets.SealedSecret{}
	for _, k := range kindsFor(in.Provider) {
		m, err := s.vault.Seal([]byte(secretFields(in)[k]))
		if err != nil {
			return ConfigView{}, err
		}
		sealed[k] = m
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ConfigView{}, err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `INSERT INTO payment_provider_configs(owner_id,provider,environment) VALUES($1,$2,$3) ON CONFLICT(owner_id,provider) DO UPDATE SET environment=EXCLUDED.environment,updated_at=NOW()`, actor.UserID, in.Provider, in.Env); err != nil {
		return ConfigView{}, err
	}
	masked := map[string]string{}
	for k, m := range sealed {
		var id uuid.UUID
		var inserted bool
		err = tx.QueryRow(ctx, `INSERT INTO payment_secrets(owner_id,kind,reference,key_id,ciphertext,masked_value) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(owner_id,kind,reference) DO UPDATE SET key_id=EXCLUDED.key_id,ciphertext=EXCLUDED.ciphertext,masked_value=EXCLUDED.masked_value,updated_at=NOW() RETURNING id,(xmax=0)`, actor.UserID, k, string(in.Provider), m.Ciphertext[0], m.Ciphertext, m.MaskedValue).Scan(&id, &inserted)
		if err != nil {
			return ConfigView{}, err
		}
		action := paymentsecrets.ActionUpdate
		if inserted {
			action = paymentsecrets.ActionCreate
		}
		if _, err = tx.Exec(ctx, `INSERT INTO payment_secret_audit(secret_id,owner_id,kind,reference,actor_id,actor_request_id,action) VALUES($1,$2,$3,$4,$5,$6,$7)`, id, actor.UserID, k, string(in.Provider), actor.UserID, actor.RequestID, action); err != nil {
			return ConfigView{}, err
		}
		masked[string(k)] = m.MaskedValue
	}
	if err = tx.Commit(ctx); err != nil {
		return ConfigView{}, err
	}
	return ConfigView{Provider: in.Provider, Env: in.Env, Masked: masked}, nil
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
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	for _, k := range kindsFor(p) {
		var id uuid.UUID
		e := tx.QueryRow(ctx, `SELECT id FROM payment_secrets WHERE owner_id=$1 AND kind=$2 AND reference=$3 FOR UPDATE`, actor.UserID, k, string(p)).Scan(&id)
		if errors.Is(e, pgx.ErrNoRows) {
			continue
		}
		if e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, `INSERT INTO payment_secret_audit(secret_id,owner_id,kind,reference,actor_id,actor_request_id,action) VALUES($1,$2,$3,$4,$5,$6,$7)`, id, actor.UserID, k, string(p), actor.UserID, actor.RequestID, paymentsecrets.ActionDelete); e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, `DELETE FROM payment_secrets WHERE id=$1`, id); e != nil {
			return e
		}
	}
	tag, err := tx.Exec(ctx, `DELETE FROM payment_provider_configs WHERE owner_id=$1 AND provider=$2`, actor.UserID, p)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return tx.Commit(ctx)
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
func isPayableDocument(doc biz.Document) bool {
	return doc.Kind == "invoice" && (doc.Status == "sent" || doc.Status == "accepted") && doc.TotalCents > 0
}
func (s *Service) Checkout(ctx context.Context, actor paymentsecrets.Actor, docID uuid.UUID, in CheckoutInput) (paymentbyo.PaymentSession, error) {
	doc, err := s.docs.GetDocument(ctx, actor.UserID, docID)
	if err != nil {
		return paymentbyo.PaymentSession{}, ErrNotFound
	}
	if !isPayableDocument(doc) {
		return paymentbyo.PaymentSession{}, fmt.Errorf("document is not a payable invoice")
	}
	cfg, err := s.load(ctx, actor, in.Provider)
	if err != nil {
		return paymentbyo.PaymentSession{}, err
	}
	req := paymentbyo.PaymentRequest{DocumentID: doc.ID.String(), TenantID: actor.UserID.String(), Amount: doc.TotalCents, Currency: strings.ToUpper(doc.Currency), ReturnURL: in.ReturnURL, CancelURL: in.CancelURL, IdempotencyKey: IdempotencyKey(actor.UserID, doc.ID, in.Provider)}
	if in.Provider == paymentbyo.ProviderStripeConnect {
		var charges, payouts, cards, transfers bool
		if err = s.pool.QueryRow(ctx, `SELECT account_id,charges_enabled,payouts_enabled,card_payments_active,transfers_active FROM stripe_connect_accounts WHERE owner_id=$1`, actor.UserID).Scan(&req.ConnectedAccountID, &charges, &payouts, &cards, &transfers); errors.Is(err, pgx.ErrNoRows) {
			return paymentbyo.PaymentSession{}, ErrNotFound
		} else if err != nil {
			return paymentbyo.PaymentSession{}, err
		}
		if !charges || !payouts || !cards || !transfers {
			return paymentbyo.PaymentSession{}, fmt.Errorf("connected account capabilities inactive")
		}
	}
	ad := s.adapters[in.Provider]
	if ad == nil {
		return paymentbyo.PaymentSession{}, fmt.Errorf("unsupported provider")
	}
	var existingID, existingURL *string
	var state string
	var reserved bool
	err = s.pool.QueryRow(ctx, `INSERT INTO payment_attempts(owner_id,document_id,provider,idempotency_key,status,reserved_at,lease_expires_at) VALUES($1,$2,$3,$4,'creating',NOW(),NOW()+interval '5 minutes') ON CONFLICT(provider,idempotency_key) DO NOTHING RETURNING true`, actor.UserID, doc.ID, in.Provider, req.IdempotencyKey).Scan(&reserved)
	if errors.Is(err, pgx.ErrNoRows) {
		err = s.pool.QueryRow(ctx, `UPDATE payment_attempts SET status='creating',last_error=NULL,reserved_at=NOW(),lease_expires_at=NOW()+interval '5 minutes',updated_at=NOW() WHERE provider=$1 AND idempotency_key=$2 AND owner_id=$3 AND document_id=$4 AND (status='failed' OR (status='creating' AND lease_expires_at<=NOW())) RETURNING true`, in.Provider, req.IdempotencyKey, actor.UserID, doc.ID).Scan(&reserved)
		if errors.Is(err, pgx.ErrNoRows) {
			err = s.pool.QueryRow(ctx, `SELECT external_id,checkout_url,status FROM payment_attempts WHERE provider=$1 AND idempotency_key=$2 AND owner_id=$3 AND document_id=$4`, in.Provider, req.IdempotencyKey, actor.UserID, doc.ID).Scan(&existingID, &existingURL, &state)
		}
	}
	if err != nil {
		return paymentbyo.PaymentSession{}, err
	}
	if !reserved {
		if existingID != nil && existingURL != nil {
			return paymentbyo.PaymentSession{Provider: in.Provider, Env: cfg.Env, ExternalID: *existingID, CheckoutURL: *existingURL}, nil
		}
		return paymentbyo.PaymentSession{}, fmt.Errorf("payment attempt in progress")
	}
	session, err := ad.CreatePayment(ctx, cfg, req)
	if err != nil {
		_, _ = s.pool.Exec(ctx, `UPDATE payment_attempts SET status='failed',last_error=$1,updated_at=NOW() WHERE provider=$2 AND idempotency_key=$3`, err.Error(), in.Provider, req.IdempotencyKey)
		return session, err
	}
	tag, err := s.pool.Exec(ctx, `UPDATE payment_attempts SET external_id=$1,checkout_url=$2,status='pending',updated_at=NOW() WHERE owner_id=$3 AND provider=$4 AND idempotency_key=$5 AND status='creating'`, session.ExternalID, session.CheckoutURL, actor.UserID, in.Provider, req.IdempotencyKey)
	if err != nil || tag.RowsAffected() != 1 {
		if err == nil {
			err = fmt.Errorf("payment attempt finalization lost")
		}
		// Leave creating recoverable: after lease expiry a retry reuses the
		// stable provider idempotency key and can finalize the accepted session.
		return paymentbyo.PaymentSession{}, err
	}
	return session, nil
}

func webhookRouteIdentifier(p paymentbyo.Provider, headers map[string]string, body []byte) (string, error) {
	switch p {
	case paymentbyo.ProviderMollie:
		v, err := url.ParseQuery(string(body))
		if err != nil || v.Get("id") == "" {
			return "", fmt.Errorf("missing route identifier")
		}
		return v.Get("id"), nil
	case paymentbyo.ProviderStripe, paymentbyo.ProviderStripeConnect:
		var x struct {
			Account string `json:"account"`
			Data    struct {
				Object struct {
					ID string `json:"id"`
				} `json:"object"`
			} `json:"data"`
		}
		if json.Unmarshal(body, &x) != nil {
			return "", fmt.Errorf("invalid webhook")
		}
		if p == paymentbyo.ProviderStripeConnect {
			if x.Account == "" {
				return "", fmt.Errorf("missing account")
			}
			return x.Account, nil
		}
		if x.Data.Object.ID == "" {
			return "", fmt.Errorf("missing route identifier")
		}
		return x.Data.Object.ID, nil
	case paymentbyo.ProviderPayPal:
		var x struct {
			EventType string `json:"event_type"`
			Resource  struct {
				ID                string `json:"id"`
				SupplementaryData struct {
					RelatedIDs struct {
						OrderID string `json:"order_id"`
					} `json:"related_ids"`
				} `json:"supplementary_data"`
			} `json:"resource"`
		}
		if json.Unmarshal(body, &x) != nil {
			return "", fmt.Errorf("invalid webhook")
		}
		id := x.Resource.ID
		if x.EventType == "PAYMENT.CAPTURE.COMPLETED" || x.EventType == "PAYMENT.CAPTURE.DENIED" {
			id = x.Resource.SupplementaryData.RelatedIDs.OrderID
		}
		if id == "" {
			return "", fmt.Errorf("missing route identifier")
		}
		return id, nil
	}
	return "", fmt.Errorf("unsupported provider")
}

// ProcessWebhook verifies before opening the transaction. The UNIQUE ledger key
// serializes retries; the document and attempt status update commit atomically.
func (s *Service) ProcessWebhook(ctx context.Context, p paymentbyo.Provider, headers map[string]string, body []byte) error {
	ad := s.adapters[p]
	if ad == nil {
		return fmt.Errorf("unsupported provider")
	}
	routeID, err := webhookRouteIdentifier(p, headers, body)
	if err != nil {
		return err
	}
	var owner, attempt, doc uuid.UUID
	if p == paymentbyo.ProviderStripeConnect {
		err = s.pool.QueryRow(ctx, `SELECT owner_id FROM stripe_connect_accounts WHERE account_id=$1`, routeID).Scan(&owner)
	} else {
		err = s.pool.QueryRow(ctx, `SELECT owner_id,id,document_id FROM payment_attempts WHERE provider=$1 AND external_id=$2`, p, routeID).Scan(&owner, &attempt, &doc)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	actor := paymentsecrets.Actor{UserID: owner, RequestID: "webhook-verify"}
	cfg, err := s.load(ctx, actor, p)
	if err != nil {
		return err
	}
	ev, err := ad.VerifyWebhook(cfg, headers, body)
	if err != nil {
		return fmt.Errorf("webhook verification failed: %w", err)
	}
	if ev.EventID == "" || ev.ExternalID == "" || ev.TenantID != owner.String() {
		return fmt.Errorf("verified identity missing or mismatched")
	}
	var docID uuid.UUID
	if ev.DocumentID != "" {
		docID, err = uuid.Parse(ev.DocumentID)
		if err != nil {
			return fmt.Errorf("invalid verified document")
		}
	}
	if ev.Status != "pending" && ev.Status != "paid" && ev.Status != "failed" {
		return fmt.Errorf("invalid payment status")
	}
	if p == paymentbyo.ProviderStripeConnect {
		err = s.pool.QueryRow(ctx, `SELECT id,document_id FROM payment_attempts WHERE provider=$1 AND external_id=$2 AND owner_id=$3`, p, ev.ExternalID, owner).Scan(&attempt, &doc)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	if docID != uuid.Nil && doc != docID {
		return fmt.Errorf("verified document mismatch")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `INSERT INTO payment_webhook_events(provider,event_id,owner_id,payment_attempt_id) VALUES($1,$2,$3,$4) ON CONFLICT(provider,event_id) DO NOTHING`, p, ev.EventID, owner, attempt)
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
		tag, err = tx.Exec(ctx, `UPDATE documents SET status='paid',paid_at=COALESCE(paid_at,NOW()),updated_at=NOW() WHERE id=$1 AND user_id=$2 AND status IN ('sent','accepted')`, doc, owner)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return fmt.Errorf("document is not payable")
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
