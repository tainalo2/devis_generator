package stripeconnect

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AccountClient is injected so onboarding and refresh can be tested without Stripe.
type AccountClient interface {
	CreateAccount(context.Context, string, string) (*ConnectedAccount, error)
	CreateAccountLink(context.Context, string, string, string) (string, error)
	RetrieveAccount(context.Context, string) (*ConnectedAccount, error)
}

type Service struct {
	pool      *pgxpool.Pool
	client    AccountClient
	country   string
	currency  string
	publicURL string
}

func NewService(pool *pgxpool.Pool, client AccountClient, cfg Config) *Service {
	return &Service{pool: pool, client: client, country: cfg.Country, currency: cfg.Currency, publicURL: cfg.PublicURL}
}

func (s *Service) Start(ctx context.Context, owner uuid.UUID, email string) (string, error) {
	acc, err := s.client.CreateAccount(ctx, email, s.country)
	if err != nil {
		return "", err
	}
	if err = s.save(ctx, owner, acc); err != nil {
		return "", err
	}
	return s.client.CreateAccountLink(ctx, acc.ID, s.publicURL+"/api/v1/payments/stripe-connect/refresh", s.publicURL+"/api/v1/payments/stripe-connect/callback")
}

func (s *Service) Refresh(ctx context.Context, owner uuid.UUID) (*ConnectedAccount, error) {
	var accountID string
	if err := s.pool.QueryRow(ctx, `SELECT account_id FROM stripe_connect_accounts WHERE owner_id=$1`, owner).Scan(&accountID); err != nil {
		return nil, err
	}
	acc, err := s.client.RetrieveAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}
	return acc, s.save(ctx, owner, acc)
}

func (s *Service) save(ctx context.Context, owner uuid.UUID, acc *ConnectedAccount) error {
	if acc == nil || acc.ID == "" {
		return fmt.Errorf("invalid connected account")
	}
	caps, _ := json.Marshal(acc.Capabilities)
	compliance, _ := json.Marshal(map[string]bool{"details_submitted": acc.DetailsSubmitted})
	_, err := s.pool.Exec(ctx, `INSERT INTO stripe_connect_accounts(owner_id,account_id,country,default_currency,capabilities,compliance,charges_enabled,payouts_enabled,card_payments_active,transfers_active) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT(owner_id) DO UPDATE SET account_id=EXCLUDED.account_id,country=EXCLUDED.country,default_currency=EXCLUDED.default_currency,capabilities=EXCLUDED.capabilities,compliance=EXCLUDED.compliance,charges_enabled=EXCLUDED.charges_enabled,payouts_enabled=EXCLUDED.payouts_enabled,card_payments_active=EXCLUDED.card_payments_active,transfers_active=EXCLUDED.transfers_active,updated_at=NOW()`, owner, acc.ID, acc.Country, acc.DefaultCurrency, caps, compliance, acc.ChargesEnabled, acc.PayoutsEnabled, acc.Capabilities["card_payments"] == "active", acc.Capabilities["transfers"] == "active")
	return err
}
