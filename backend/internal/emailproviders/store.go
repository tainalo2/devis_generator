package emailproviders

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EmailConfig holds an email account configuration for a user.
// Currently supports Microsoft OAuth 365, Gmail OAuth, and manual SMTP.
type EmailConfig struct {
	UserID          uuid.UUID
	Provider        string // "microsoft_graph", "gmail_api", "smtp"
	SMTPHost        string
	SMTPPort        int
	SMTPUsername    string
	SMTPUseTLS      bool
	SMTPFromAddr    string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Store persists email configurations per user.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore creates a new email config store.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// GetByUserID retrieves the email configuration for a user.
// Returns nil if not configured.
func (s *Store) GetByUserID(ctx context.Context, userID uuid.UUID) (*EmailConfig, error) {
	var ec EmailConfig
	err := s.pool.QueryRow(ctx, `
		SELECT user_id, smtp_host, smtp_port, smtp_username, smtp_use_tls, smtp_from_addr, created_at, updated_at
		FROM email_config WHERE user_id = $1
	`, userID).Scan(
		&ec.UserID, &ec.SMTPHost, &ec.SMTPPort, &ec.SMTPUsername, &ec.SMTPUseTLS, &ec.SMTPFromAddr,
		&ec.CreatedAt, &ec.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get email config: %w", err)
	}
	return &ec, nil
}

// Upsert creates or updates an email configuration.
func (s *Store) Upsert(ctx context.Context, ec *EmailConfig) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO email_config (user_id, smtp_host, smtp_port, smtp_username, smtp_use_tls, smtp_from_addr, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		ON CONFLICT (user_id) DO UPDATE
		SET smtp_host=EXCLUDED.smtp_host, smtp_port=EXCLUDED.smtp_port,
		    smtp_username=EXCLUDED.smtp_username, smtp_use_tls=EXCLUDED.smtp_use_tls,
		    smtp_from_addr=EXCLUDED.smtp_from_addr, updated_at=NOW()
	`, ec.UserID, ec.SMTPHost, ec.SMTPPort, ec.SMTPUsername, ec.SMTPUseTLS, ec.SMTPFromAddr)
	if err != nil {
		return fmt.Errorf("upsert email config: %w", err)
	}
	return nil
}

// DeleteByUserID removes the email configuration for a user.
func (s *Store) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, "DELETE FROM email_config WHERE user_id = $1", userID)
	if err != nil {
		return fmt.Errorf("delete email config: %w", err)
	}
	return nil
}
