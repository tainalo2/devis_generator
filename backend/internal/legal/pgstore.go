package legal

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PGStore implements Store with legal_values (append-only) + legal_variables (projection).
type PGStore struct {
	pool *pgxpool.Pool
	// FreshFor marks a value fresh if younger than this duration.
	FreshFor time.Duration
}

func NewPGStore(pool *pgxpool.Pool) *PGStore {
	return &PGStore{pool: pool, FreshFor: 7 * 24 * time.Hour}
}

func (s *PGStore) Get(ctx context.Context, key string) (Variable, error) {
	v, _, err := s.Latest(ctx, key)
	return v, err
}

func (s *PGStore) Latest(ctx context.Context, key string) (Variable, string, error) {
	var v Variable
	var status string
	err := s.pool.QueryRow(ctx, `
		SELECT key, value, source, fetched_at, status
		FROM legal_values
		WHERE key = $1 AND status IN ('verified','manual','stale')
		ORDER BY fetched_at DESC
		LIMIT 1
	`, key).Scan(&v.Key, &v.Value, &v.Source, &v.FetchedAt, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		// Fallback projection
		err = s.pool.QueryRow(ctx, `
			SELECT key, value, source, fetched_at FROM legal_variables WHERE key = $1
		`, key).Scan(&v.Key, &v.Value, &v.Source, &v.FetchedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return Variable{}, "", ErrNotConfigured
		}
		if err != nil {
			return Variable{}, "", err
		}
		return v, StatusStale, nil
	}
	if err != nil {
		return Variable{}, "", err
	}
	v.Status = status
	return v, status, nil
}

func (s *PGStore) Upsert(ctx context.Context, v Variable) error {
	status := v.Status
	if status == "" {
		status = StatusVerified
	}
	if err := s.Append(ctx, v, status, ""); err != nil {
		return err
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO legal_variables (key, value, source, fetched_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (key) DO UPDATE
		SET value = EXCLUDED.value, source = EXCLUDED.source, fetched_at = EXCLUDED.fetched_at, updated_at = NOW()
	`, v.Key, v.Value, v.Source, v.FetchedAt)
	return err
}

func (s *PGStore) Append(ctx context.Context, v Variable, status, errMsg string) error {
	if status == "" {
		status = StatusVerified
	}
	if v.FetchedAt.IsZero() {
		v.FetchedAt = time.Now().UTC()
	}
	if len(v.Value) == 0 {
		v.Value = json.RawMessage(`{}`)
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO legal_values (key, value, source, status, fetched_at, error_message)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, v.Key, v.Value, v.Source, status, v.FetchedAt, errMsg)
	return err
}

func (s *PGStore) ListStatus(ctx context.Context) ([]StatusEntry, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT ON (key) key, value, source, fetched_at, status
		FROM legal_values
		WHERE status IN ('verified','manual','stale','error')
		ORDER BY key, fetched_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	freshFor := s.FreshFor
	if freshFor <= 0 {
		freshFor = 7 * 24 * time.Hour
	}
	now := time.Now().UTC()
	out := []StatusEntry{}
	for rows.Next() {
		var e StatusEntry
		if err := rows.Scan(&e.Key, &e.Value, &e.Source, &e.FetchedAt, &e.Status); err != nil {
			return nil, err
		}
		age := now.Sub(e.FetchedAt)
		e.AgeSecs = int64(age.Seconds())
		e.Fresh = (e.Status == StatusVerified && age <= freshFor) || e.Status == StatusManual
		out = append(out, e)
	}
	return out, rows.Err()
}
