package smtpconfig

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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

func (s *PGStore) Upsert(ctx context.Context, config Config) (Config, error) {
	err := s.pool.QueryRow(ctx, `
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
