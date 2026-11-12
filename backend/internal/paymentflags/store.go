package paymentflags

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Mode identifies a server-gated payment integration.
type Mode string

const (
	RIB           Mode = "rib"
	BYOProvider   Mode = "byo_provider"
	StripeConnect Mode = "stripe_connect"
)

// Features is the complete payment feature configuration. Its zero value is
// deliberately secure: every integration is disabled until an admin opts in.
type Features struct {
	RIB           bool `json:"rib"`
	BYOProvider   bool `json:"byo_provider"`
	StripeConnect bool `json:"stripe_connect"`
}

func (f Features) Enabled(mode Mode) bool {
	switch mode {
	case RIB:
		return f.RIB
	case BYOProvider:
		return f.BYOProvider
	case StripeConnect:
		return f.StripeConnect
	default:
		return false
	}
}

// Store persists the singleton server-wide payment feature configuration.
type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

func (s *Store) Get(ctx context.Context) (Features, error) {
	var features Features
	err := s.pool.QueryRow(ctx, `
		SELECT rib_enabled, byo_provider_enabled, stripe_connect_enabled
		FROM payment_feature_flags WHERE singleton = TRUE
	`).Scan(&features.RIB, &features.BYOProvider, &features.StripeConnect)
	if err != nil {
		return Features{}, fmt.Errorf("read payment feature flags: %w", err)
	}
	return features, nil
}

func (s *Store) Set(ctx context.Context, features Features) (Features, error) {
	var saved Features
	err := s.pool.QueryRow(ctx, `
		INSERT INTO payment_feature_flags
			(singleton, rib_enabled, byo_provider_enabled, stripe_connect_enabled, updated_at)
		VALUES (TRUE, $1, $2, $3, NOW())
		ON CONFLICT (singleton) DO UPDATE SET
			rib_enabled = EXCLUDED.rib_enabled,
			byo_provider_enabled = EXCLUDED.byo_provider_enabled,
			stripe_connect_enabled = EXCLUDED.stripe_connect_enabled,
			updated_at = NOW()
		RETURNING rib_enabled, byo_provider_enabled, stripe_connect_enabled
	`, features.RIB, features.BYOProvider, features.StripeConnect).Scan(
		&saved.RIB, &saved.BYOProvider, &saved.StripeConnect,
	)
	if err != nil {
		return Features{}, fmt.Errorf("update payment feature flags: %w", err)
	}
	return saved, nil
}
