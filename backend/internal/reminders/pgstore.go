package reminders

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PGStore struct{ pool *pgxpool.Pool }

func NewPGStore(pool *pgxpool.Pool) *PGStore { return &PGStore{pool: pool} }
func (s *PGStore) Mode(ctx context.Context, owner uuid.UUID) (DeliveryMode, error) {
	var mode DeliveryMode
	err := s.pool.QueryRow(ctx, `SELECT mode FROM email_delivery_settings WHERE user_id=$1`, owner).Scan(&mode)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrDeliveryMode
	}
	return mode, err
}
func (s *PGStore) SetMode(ctx context.Context, owner uuid.UUID, mode DeliveryMode) error {
	if owner == uuid.Nil || (mode != DeliveryModeGmail && mode != DeliveryModeMicrosoft && mode != DeliveryModeSMTP) {
		return ErrInvalid
	}
	_, err := s.pool.Exec(ctx, `INSERT INTO email_delivery_settings(user_id,mode) VALUES($1,$2) ON CONFLICT(user_id) DO UPDATE SET mode=EXCLUDED.mode,updated_at=NOW()`, owner, mode)
	return err
}
func (s *PGStore) Invoice(ctx context.Context, owner, id uuid.UUID) (Invoice, error) {
	var v Invoice
	v.ID = id
	v.OwnerID = owner
	err := s.pool.QueryRow(ctx, `SELECT d.kind::text, d.status::text, COALESCE(d.client_snapshot->>'email',''), d.updated_at FROM documents d WHERE d.id=$1 AND d.user_id=$2`, id, owner).Scan(&v.Kind, &v.Status, &v.Recipient, &v.UpdatedAt)
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
func (s *PGStore) CreateTemplate(ctx context.Context, owner uuid.UUID, v Template) (Template, error) {
	v.OwnerID = owner
	err := s.pool.QueryRow(ctx, `INSERT INTO reminder_templates(user_id,name,subject,html) VALUES($1,$2,$3,$4) RETURNING id,created_at,updated_at`, owner, strings.TrimSpace(v.Name), strings.TrimSpace(v.Subject), v.HTML).Scan(&v.ID, &v.CreatedAt, &v.UpdatedAt)
	return v, err
}
func (s *PGStore) ListTemplates(ctx context.Context, owner uuid.UUID) ([]Template, error) {
	rows, err := s.pool.Query(ctx, `SELECT id,user_id,name,subject,html,created_at,updated_at FROM reminder_templates WHERE user_id=$1 ORDER BY created_at,id`, owner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Template
	for rows.Next() {
		var v Template
		if err := rows.Scan(&v.ID, &v.OwnerID, &v.Name, &v.Subject, &v.HTML, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *PGStore) UpdateTemplate(ctx context.Context, owner uuid.UUID, v Template) (Template, error) {
	v.OwnerID = owner
	err := s.pool.QueryRow(ctx, `UPDATE reminder_templates SET name=$3,subject=$4,html=$5,updated_at=NOW() WHERE id=$1 AND user_id=$2 RETURNING created_at,updated_at`, v.ID, owner, strings.TrimSpace(v.Name), strings.TrimSpace(v.Subject), v.HTML).Scan(&v.CreatedAt, &v.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		err = ErrNotFound
	}
	return v, err
}
func (s *PGStore) DeleteTemplate(ctx context.Context, owner, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM reminder_templates WHERE id=$1 AND user_id=$2`, id, owner)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

func scanReminder(row pgx.Row) (Reminder, error) {
	var r Reminder
	err := row.Scan(&r.ID, &r.OwnerID, &r.InvoiceID, &r.TemplateID, &r.Type, &r.Recipient, &r.Status, &r.Error, &r.CreatedAt, &r.SentAt, &r.LeaseUntil, &r.Attempts)
	return r, err
}
func (s *PGStore) Prepare(ctx context.Context, owner, invoice, template uuid.UUID, key, kind string, now time.Time) (Reminder, bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Reminder{}, false, err
	}
	defer tx.Rollback(ctx)
	// Serialize reservations per account so concurrent workers cannot bypass the hourly cap.
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1::text, 0))`, owner); err != nil {
		return Reminder{}, false, err
	}
	existing, err := scanReminder(tx.QueryRow(ctx, `SELECT id,user_id,invoice_id,template_id,reminder_type,recipient,status,error_message,created_at,sent_at,lease_until,attempt_count FROM reminders WHERE user_id=$1 AND invoice_id=$2 AND template_id=$3 AND reminder_type=$4 AND idempotency_key=$5 FOR UPDATE`, owner, invoice, template, kind, key))
	if err == nil {
		claimable := existing.Status == StatusQueued || existing.Status == StatusFailed || (existing.Status == StatusSending && existing.LeaseUntil != nil && !existing.LeaseUntil.After(now))
		if claimable {
			var hourly int
			if err = tx.QueryRow(ctx, `SELECT count(*) FROM reminder_delivery_attempts WHERE user_id=$1 AND attempted_at >= $2 - interval '1 hour'`, owner, now).Scan(&hourly); err != nil {
				return Reminder{}, false, err
			}
			if hourly >= 20 {
				return Reminder{}, false, ErrRateLimited
			}
			existing, err = scanReminder(tx.QueryRow(ctx, `UPDATE reminders SET status='sending',lease_until=$3 + interval '5 minutes',attempt_count=attempt_count+1,error_message=NULL WHERE id=$1 AND user_id=$2 RETURNING id,user_id,invoice_id,template_id,reminder_type,recipient,status,error_message,created_at,sent_at,lease_until,attempt_count`, existing.ID, owner, now))
			if err != nil {
				return Reminder{}, false, err
			}
			if _, err = tx.Exec(ctx, `INSERT INTO reminder_delivery_attempts(reminder_id,user_id,attempted_at) VALUES($1,$2,$3)`, existing.ID, owner, now); err != nil {
				return Reminder{}, false, err
			}
		}
		if err = tx.Commit(ctx); err != nil {
			return Reminder{}, false, err
		}
		return existing, claimable, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Reminder{}, false, err
	}
	var hourly int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM reminder_delivery_attempts WHERE user_id=$1 AND attempted_at >= $2 - interval '1 hour'`, owner, now).Scan(&hourly); err != nil {
		return Reminder{}, false, err
	}
	if hourly >= 20 {
		return Reminder{}, false, ErrRateLimited
	}
	var r Reminder
	err = tx.QueryRow(ctx, `INSERT INTO reminders(user_id,invoice_id,template_id,reminder_type,recipient,status,idempotency_key,created_at,lease_until,attempt_count) SELECT $1,d.id,t.id,$4,d.client_snapshot->>'email','sending',$5,$6,$6 + interval '5 minutes',1 FROM documents d JOIN reminder_templates t ON t.id=$3 AND t.user_id=$1 WHERE d.id=$2 AND d.user_id=$1 AND d.kind IN ('invoice','quote') AND d.status='sent' AND COALESCE(d.client_snapshot->>'email','') <> '' ON CONFLICT(user_id,invoice_id,template_id,reminder_type,idempotency_key) DO NOTHING RETURNING id,user_id,invoice_id,template_id,reminder_type,recipient,status,error_message,created_at,sent_at,lease_until,attempt_count`, owner, invoice, template, kind, key, now).Scan(&r.ID, &r.OwnerID, &r.InvoiceID, &r.TemplateID, &r.Type, &r.Recipient, &r.Status, &r.Error, &r.CreatedAt, &r.SentAt, &r.LeaseUntil, &r.Attempts)
	if errors.Is(err, pgx.ErrNoRows) {
		return Reminder{}, false, ErrDocumentChanged
	}
	if err != nil {
		return Reminder{}, false, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO reminder_delivery_attempts(reminder_id,user_id,attempted_at) VALUES($1,$2,$3)`, r.ID, owner, now); err != nil {
		return Reminder{}, false, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Reminder{}, false, err
	}
	return r, true, nil
}
func (s *PGStore) MarkSent(ctx context.Context, owner, id uuid.UUID, now time.Time) (Reminder, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Reminder{}, err
	}
	defer tx.Rollback(ctx)
	r, err := scanReminder(tx.QueryRow(ctx, `UPDATE reminders SET status='sent',sent_at=$3,lease_until=NULL,error_message=NULL WHERE id=$1 AND user_id=$2 AND status='sending' RETURNING id,user_id,invoice_id,template_id,reminder_type,recipient,status,error_message,created_at,sent_at,lease_until,attempt_count`, id, owner, now))
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
	r, err := scanReminder(s.pool.QueryRow(ctx, `UPDATE reminders SET status='failed',lease_until=NULL,error_message=$3 WHERE id=$1 AND user_id=$2 AND status='sending' RETURNING id,user_id,invoice_id,template_id,reminder_type,recipient,status,error_message,created_at,sent_at,lease_until,attempt_count`, id, owner, msg))
	if errors.Is(err, pgx.ErrNoRows) {
		err = ErrNotFound
	}
	return r, err
}
func (s *PGStore) MarkSkipped(ctx context.Context, owner, id uuid.UUID, msg string, now time.Time) (Reminder, error) {
	r, err := scanReminder(s.pool.QueryRow(ctx, `UPDATE reminders SET status='skipped',lease_until=NULL,error_message=$3 WHERE id=$1 AND user_id=$2 AND status='sending' RETURNING id,user_id,invoice_id,template_id,reminder_type,recipient,status,error_message,created_at,sent_at,lease_until,attempt_count`, id, owner, msg))
	if errors.Is(err, pgx.ErrNoRows) {
		err = ErrNotFound
	}
	return r, err
}
func (s *PGStore) LockForDelivery(ctx context.Context, owner, id uuid.UUID, expectedUpdatedAt time.Time, expectedRecipient string) (func(), error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	var kind, status, recipient string
	var updatedAt time.Time
	err = tx.QueryRow(ctx, `SELECT kind::text,status::text,COALESCE(client_snapshot->>'email',''),updated_at FROM documents WHERE id=$1 AND user_id=$2 FOR UPDATE`, id, owner).Scan(&kind, &status, &recipient, &updatedAt)
	if err != nil || (kind != "invoice" && kind != "quote") || status != "sent" || !updatedAt.Equal(expectedUpdatedAt) || recipient != expectedRecipient {
		_ = tx.Rollback(context.Background())
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		return nil, ErrDocumentChanged
	}
	var once sync.Once
	release := func() { once.Do(func() { _ = tx.Rollback(context.Background()) }) }
	return release, nil
}
func (s *PGStore) List(ctx context.Context, owner, invoice uuid.UUID, page, size int) ([]Reminder, int, error) {
	offset := (page - 1) * size
	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM reminders WHERE user_id=$1 AND invoice_id=$2`, owner, invoice).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.pool.Query(ctx, `SELECT id,user_id,invoice_id,template_id,reminder_type,recipient,status,error_message,created_at,sent_at,lease_until,attempt_count FROM reminders WHERE user_id=$1 AND invoice_id=$2 ORDER BY created_at DESC,id DESC LIMIT $3 OFFSET $4`, owner, invoice, size, offset)
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
