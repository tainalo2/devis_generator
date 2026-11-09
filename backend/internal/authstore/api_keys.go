package authstore

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

type APIKey struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Name      string
	KeyPrefix string
	Scopes    []string
	CreatedAt time.Time
	RevokedAt *time.Time
	LastUsed  *time.Time
}

func (s *Store) CreateAPIKey(ctx context.Context, userID uuid.UUID, name string, keyHash, prefix string, scopes []string) (APIKey, error) {
	var k APIKey
	err := s.pool.QueryRow(ctx, `
		INSERT INTO api_keys (user_id, name, key_hash, key_prefix, scopes)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, name, key_prefix, scopes, created_at, revoked_at, last_used_at
	`, userID, name, keyHash, prefix, scopes).Scan(
		&k.ID, &k.UserID, &k.Name, &k.KeyPrefix, &k.Scopes, &k.CreatedAt, &k.RevokedAt, &k.LastUsed,
	)
	return k, err
}

func (s *Store) ListAPIKeys(ctx context.Context, userID uuid.UUID) ([]APIKey, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, user_id, name, key_prefix, scopes, created_at, revoked_at, last_used_at
		FROM api_keys WHERE user_id = $1 ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []APIKey
	for rows.Next() {
		var k APIKey
		if err := rows.Scan(&k.ID, &k.UserID, &k.Name, &k.KeyPrefix, &k.Scopes, &k.CreatedAt, &k.RevokedAt, &k.LastUsed); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

func (s *Store) RevokeAPIKey(ctx context.Context, userID, keyID uuid.UUID) error {
	ct, err := s.pool.Exec(ctx, `
		UPDATE api_keys SET revoked_at = NOW()
		WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL
	`, keyID, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) GetAPIKeyByHash(ctx context.Context, keyHash string) (APIKey, User, error) {
	var k APIKey
	var u User
	err := s.pool.QueryRow(ctx, `
		SELECT k.id, k.user_id, k.name, k.key_prefix, k.scopes, k.created_at, k.revoked_at, k.last_used_at,
		       u.id, u.email, COALESCE(u.password_hash, ''), u.display_name, u.created_at, u.updated_at
		FROM api_keys k JOIN users u ON u.id = k.user_id
		WHERE k.key_hash = $1 AND k.revoked_at IS NULL
	`, keyHash).Scan(
		&k.ID, &k.UserID, &k.Name, &k.KeyPrefix, &k.Scopes, &k.CreatedAt, &k.RevokedAt, &k.LastUsed,
		&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, ErrNotFound) || err != nil {
		// normalize
	}
	if err != nil {
		return APIKey{}, User{}, ErrNotFound
	}
	return k, u, nil
}

func (s *Store) TouchAPIKey(ctx context.Context, id uuid.UUID) {
	_, _ = s.pool.Exec(ctx, `UPDATE api_keys SET last_used_at = NOW() WHERE id = $1`, id)
}
