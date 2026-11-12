package paymentsecrets

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditAction string

const (
	ActionCreate AuditAction = "create"
	ActionRead   AuditAction = "read"
	ActionUpdate AuditAction = "update"
	ActionDelete AuditAction = "delete"
)

type AuditEvent struct {
	ID             int64
	SecretID       *uuid.UUID
	OwnerID        uuid.UUID
	Kind           Kind
	Reference      string
	ActorID        *uuid.UUID
	ActorRequestID string
	Action         AuditAction
	CreatedAt      time.Time
}

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

func (r *PGRepository) PutCiphertext(ctx context.Context, actor Actor, ownerID uuid.UUID, kind Kind, reference string, ciphertext []byte, masked string) (Metadata, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Metadata{}, err
	}
	defer tx.Rollback(ctx)

	var metadata Metadata
	err = tx.QueryRow(ctx, `
		INSERT INTO payment_secrets (owner_id, kind, reference, key_id, ciphertext, masked_value)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (owner_id, kind, reference) DO UPDATE
		SET key_id = EXCLUDED.key_id,
		    ciphertext = EXCLUDED.ciphertext,
		    masked_value = EXCLUDED.masked_value,
		    updated_at = NOW()
		RETURNING id, owner_id, kind, reference, masked_value, created_at, updated_at
	`, ownerID, kind, reference, ciphertext[0], ciphertext, masked).Scan(
		&metadata.ID, &metadata.OwnerID, &metadata.Kind, &metadata.Reference, &metadata.MaskedValue, &metadata.CreatedAt, &metadata.UpdatedAt,
	)
	if err != nil {
		return Metadata{}, err
	}
	if err := insertAudit(ctx, tx, &metadata.ID, ownerID, kind, reference, actor, ActionCreate); err != nil {
		return Metadata{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Metadata{}, err
	}
	return metadata, nil
}

func (r *PGRepository) ReadCiphertext(ctx context.Context, actor Actor, ownerID uuid.UUID, kind Kind, reference string) ([]byte, Metadata, error) {
	var metadata Metadata
	var ciphertext []byte
	err := r.pool.QueryRow(ctx, `
		SELECT id, owner_id, kind, reference, key_id, ciphertext, masked_value, created_at, updated_at
		FROM payment_secrets
		WHERE owner_id = $1 AND kind = $2 AND reference = $3
	`, ownerID, kind, reference).Scan(
		&metadata.ID, &metadata.OwnerID, &metadata.Kind, &metadata.Reference, new(int16), &ciphertext, &metadata.MaskedValue, &metadata.CreatedAt, &metadata.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, Metadata{}, ErrNotFound
	}
	if err != nil {
		return nil, Metadata{}, err
	}
	if err := insertAudit(ctx, r.pool, &metadata.ID, ownerID, kind, reference, actor, ActionRead); err != nil {
		return nil, Metadata{}, err
	}
	return ciphertext, metadata, nil
}

func (r *PGRepository) Delete(ctx context.Context, actor Actor, ownerID uuid.UUID, kind Kind, reference string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var secretID uuid.UUID
	err = tx.QueryRow(ctx, `
		DELETE FROM payment_secrets
		WHERE owner_id = $1 AND kind = $2 AND reference = $3
		RETURNING id
	`, ownerID, kind, reference).Scan(&secretID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if err := insertAudit(ctx, tx, &secretID, ownerID, kind, reference, actor, ActionDelete); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) ListAudit(ctx context.Context, ownerID uuid.UUID, limit int) ([]AuditEvent, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, secret_id, owner_id, kind, reference, actor_id, actor_request_id, action, created_at
		FROM payment_secret_audit
		WHERE owner_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT $2
	`, ownerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []AuditEvent
	for rows.Next() {
		var event AuditEvent
		var actorID *uuid.UUID
		var secretID *uuid.UUID
		if err := rows.Scan(&event.ID, &secretID, &event.OwnerID, &event.Kind, &event.Reference, &actorID, &event.ActorRequestID, &event.Action, &event.CreatedAt); err != nil {
			return nil, err
		}
		event.SecretID = secretID
		event.ActorID = actorID
		events = append(events, event)
	}
	return events, rows.Err()
}

type auditExecer interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func insertAudit(ctx context.Context, execer auditExecer, secretID *uuid.UUID, ownerID uuid.UUID, kind Kind, reference string, actor Actor, action AuditAction) error {
	_, err := execer.Exec(ctx, `
		INSERT INTO payment_secret_audit (secret_id, owner_id, kind, reference, actor_id, actor_request_id, action)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, secretID, ownerID, kind, reference, actor.UserID, actor.RequestID, action)
	return err
}
