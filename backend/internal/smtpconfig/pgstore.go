package smtpconfig

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tainalo2/devis_generator/backend/internal/paymentsecrets"
)

type PGStore struct {
	pool *pgxpool.Pool
}

func NewPGStore(pool *pgxpool.Pool) *PGStore {
	return &PGStore{pool: pool}
}

func (s *PGStore) Get(ctx context.Context, ownerID uuid.UUID) (Config, error) {
	var config Config
	err := s.pool.QueryRow(ctx, `
		SELECT owner_id, host, port, username, from_address, tls_mode, created_at, updated_at
		FROM smtp_configs
		WHERE owner_id = $1
	`, ownerID).Scan(&config.OwnerID, &config.Host, &config.Port, &config.Username, &config.FromAddress, &config.TLSMode, &config.CreatedAt, &config.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Config{}, ErrNotFound
	}
	if err != nil {
		return Config{}, err
	}
	config.PasswordConfigured = true
	return config, nil
}

func (s *PGStore) Save(ctx context.Context, actor paymentsecrets.Actor, config Config, secret paymentsecrets.SealedSecret) (Config, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Config{}, err
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx, `
		INSERT INTO smtp_configs (owner_id, host, port, username, from_address, tls_mode)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (owner_id) DO UPDATE SET
			host = EXCLUDED.host,
			port = EXCLUDED.port,
			username = EXCLUDED.username,
			from_address = EXCLUDED.from_address,
			tls_mode = EXCLUDED.tls_mode,
			updated_at = NOW()
		RETURNING created_at, updated_at
	`, config.OwnerID, config.Host, config.Port, config.Username, config.FromAddress, config.TLSMode).Scan(&config.CreatedAt, &config.UpdatedAt)
	if err != nil {
		return Config{}, err
	}

	var secretID uuid.UUID
	var inserted bool
	err = tx.QueryRow(ctx, `
		INSERT INTO payment_secrets (owner_id, kind, reference, key_id, ciphertext, masked_value)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (owner_id, kind, reference) DO UPDATE SET
			key_id = EXCLUDED.key_id,
			ciphertext = EXCLUDED.ciphertext,
			masked_value = EXCLUDED.masked_value,
			updated_at = NOW()
		RETURNING id, (xmax = 0)
	`, config.OwnerID, paymentsecrets.KindSMTPPassword, secretReference, secret.Ciphertext[0], secret.Ciphertext, secret.MaskedValue).Scan(&secretID, &inserted)
	if err != nil {
		return Config{}, err
	}
	action := paymentsecrets.ActionUpdate
	if inserted {
		action = paymentsecrets.ActionCreate
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO payment_secret_audit
			(secret_id, owner_id, kind, reference, actor_id, actor_request_id, action)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, secretID, config.OwnerID, paymentsecrets.KindSMTPPassword, secretReference, actor.UserID, actor.RequestID, action); err != nil {
		return Config{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Config{}, err
	}
	config.PasswordConfigured = true
	return config, nil
}

func (s *PGStore) Delete(ctx context.Context, ownerID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM smtp_configs WHERE owner_id = $1`, ownerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
