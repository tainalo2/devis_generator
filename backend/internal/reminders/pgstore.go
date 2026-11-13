package reminders

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PGStore struct{ pool *pgxpool.Pool }

func NewPGStore(pool *pgxpool.Pool) *PGStore { return &PGStore{pool: pool} }
func (s *PGStore) Invoice(ctx context.Context, owner, id uuid.UUID) (Invoice, error) {
	var v Invoice
	v.ID = id
	v.OwnerID = owner
	err := s.pool.QueryRow(ctx, `SELECT d.kind::text, COALESCE(NULLIF((d.client_snapshot->>'email'),''),c.email,'') FROM documents d LEFT JOIN clients c ON c.id=d.client_id WHERE d.id=$1 AND d.user_id=$2`, id, owner).Scan(&v.Kind, &v.Recipient)
	if errors.Is(err, pgx.ErrNoRows) {
		err = ErrNotFound
	}
	return v, err
}
func (s *PGStore) Template(ctx context.Context, owner, id uuid.UUID) (Template, error) {
	v := Template{ID: id, OwnerID: owner}
	err := s.pool.QueryRow(ctx, `SELECT subject,html FROM reminder_templates WHERE id=$1 AND user_id=$2`, id, owner).Scan(&v.Subject, &v.HTML)
	if errors.Is(err, pgx.ErrNoRows) {
		err = ErrNotFound
	}
	return v, err
}
func scanReminder(row pgx.Row) (Reminder, error) {
	var r Reminder
	err := row.Scan(&r.ID, &r.OwnerID, &r.InvoiceID, &r.TemplateID, &r.Type, &r.Recipient, &r.Status, &r.Error, &r.CreatedAt, &r.SentAt)
	return r, err
}
func (s *PGStore) Prepare(ctx context.Context, owner, invoice, template uuid.UUID, key, kind string, now time.Time) (Reminder, bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Reminder{}, false, err
	}
	defer tx.Rollback(ctx)
	var r Reminder
	err = tx.QueryRow(ctx, `INSERT INTO reminders(user_id,invoice_id,template_id,reminder_type,recipient,status,idempotency_key,created_at) SELECT $1,d.id,$3,$4,COALESCE(NULLIF(d.client_snapshot->>'email',''),c.email,''),'queued',$5,$6 FROM documents d LEFT JOIN clients c ON c.id=d.client_id WHERE d.id=$2 AND d.user_id=$1 AND d.kind='invoice' ON CONFLICT(user_id,idempotency_key) DO NOTHING RETURNING id,user_id,invoice_id,template_id,reminder_type,recipient,status,error_message,created_at,sent_at`, owner, invoice, template, kind, key, now).Scan(&r.ID, &r.OwnerID, &r.InvoiceID, &r.TemplateID, &r.Type, &r.Recipient, &r.Status, &r.Error, &r.CreatedAt, &r.SentAt)
	created := err == nil
	if errors.Is(err, pgx.ErrNoRows) {
		r, err = scanReminder(tx.QueryRow(ctx, `SELECT id,user_id,invoice_id,template_id,reminder_type,recipient,status,error_message,created_at,sent_at FROM reminders WHERE user_id=$1 AND idempotency_key=$2`, owner, key))
	}
	if err != nil {
		return Reminder{}, false, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Reminder{}, false, err
	}
	return r, created, nil
}
func (s *PGStore) MarkSent(ctx context.Context, owner, id uuid.UUID, now time.Time) (Reminder, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Reminder{}, err
	}
	defer tx.Rollback(ctx)
	r, err := scanReminder(tx.QueryRow(ctx, `UPDATE reminders SET status='sent',sent_at=$3,error_message=NULL WHERE id=$1 AND user_id=$2 AND status='queued' RETURNING id,user_id,invoice_id,template_id,reminder_type,recipient,status,error_message,created_at,sent_at`, id, owner, now))
	if errors.Is(err, pgx.ErrNoRows) {
		return Reminder{}, ErrNotFound
	}
	if err != nil {
		return Reminder{}, err
	}
	tag, err := tx.Exec(ctx, `UPDATE documents SET reminder_count=reminder_count+1,last_reminder_at=$3,updated_at=$3 WHERE id=$1 AND user_id=$2`, r.InvoiceID, owner, now)
	if err != nil || tag.RowsAffected() != 1 {
		if err == nil {
			err = ErrNotFound
		}
		return Reminder{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Reminder{}, err
	}
	return r, nil
}
func (s *PGStore) MarkFailed(ctx context.Context, owner, id uuid.UUID, msg string, now time.Time) (Reminder, error) {
	r, err := scanReminder(s.pool.QueryRow(ctx, `UPDATE reminders SET status='failed',error_message=$3 WHERE id=$1 AND user_id=$2 AND status='queued' RETURNING id,user_id,invoice_id,template_id,reminder_type,recipient,status,error_message,created_at,sent_at`, id, owner, msg))
	if errors.Is(err, pgx.ErrNoRows) {
		err = ErrNotFound
	}
	return r, err
}
func (s *PGStore) List(ctx context.Context, owner, invoice uuid.UUID, page, size int) ([]Reminder, int, error) {
	offset := (page - 1) * size
	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM reminders WHERE user_id=$1 AND invoice_id=$2`, owner, invoice).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.pool.Query(ctx, `SELECT id,user_id,invoice_id,template_id,reminder_type,recipient,status,error_message,created_at,sent_at FROM reminders WHERE user_id=$1 AND invoice_id=$2 ORDER BY created_at DESC,id DESC LIMIT $3 OFFSET $4`, owner, invoice, size, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := make([]Reminder, 0)
	for rows.Next() {
		r, err := scanReminder(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, r)
	}
	return out, total, rows.Err()
}
