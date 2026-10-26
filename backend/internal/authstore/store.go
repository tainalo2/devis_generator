package authstore

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrEmailTaken   = errors.New("email taken")
	ErrInvalidCreds = errors.New("invalid credentials")
)

// User is a registered account.
type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	DisplayName  string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Session is a server-side session row.
type Session struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	CSRFToken string
	CreatedAt time.Time
	ExpiresAt time.Time
	RevokedAt *time.Time
	UserAgent string
	IP        *string
}

// Store persists users and sessions.
type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) CreateUser(ctx context.Context, email, passwordHash, displayName string) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, display_name)
		VALUES ($1, $2, $3)
		RETURNING id, email, password_hash, display_name, created_at, updated_at
	`, email, passwordHash, displayName).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return User{}, ErrEmailTaken
		}
		return User{}, err
	}
	return u, nil
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `
		SELECT id, email, COALESCE(password_hash, ''), display_name, created_at, updated_at
		FROM users WHERE lower(email) = lower($1)
	`, email).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

func (s *Store) GetUserByID(ctx context.Context, id uuid.UUID) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `
		SELECT id, email, COALESCE(password_hash, ''), display_name, created_at, updated_at
		FROM users WHERE id = $1
	`, id).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

func (s *Store) CreateSession(ctx context.Context, userID uuid.UUID, tokenHash, csrf string, expiresAt time.Time, ua string, ip *string) (Session, error) {
	var sess Session
	err := s.pool.QueryRow(ctx, `
		INSERT INTO sessions (user_id, token_hash, csrf_token, expires_at, user_agent, ip)
		VALUES ($1, $2, $3, $4, $5, $6::inet)
		RETURNING id, user_id, token_hash, csrf_token, created_at, expires_at, revoked_at, user_agent, host(ip)::text
	`, userID, tokenHash, csrf, expiresAt, ua, ip).Scan(
		&sess.ID, &sess.UserID, &sess.TokenHash, &sess.CSRFToken, &sess.CreatedAt, &sess.ExpiresAt, &sess.RevokedAt, &sess.UserAgent, &sess.IP,
	)
	return sess, err
}

func (s *Store) GetSessionByTokenHash(ctx context.Context, tokenHash string) (Session, User, error) {
	var sess Session
	var u User
	err := s.pool.QueryRow(ctx, `
		SELECT s.id, s.user_id, s.token_hash, s.csrf_token, s.created_at, s.expires_at, s.revoked_at, s.user_agent, host(s.ip)::text,
		       u.id, u.email, COALESCE(u.password_hash, ''), u.display_name, u.created_at, u.updated_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = $1
	`, tokenHash).Scan(
		&sess.ID, &sess.UserID, &sess.TokenHash, &sess.CSRFToken, &sess.CreatedAt, &sess.ExpiresAt, &sess.RevokedAt, &sess.UserAgent, &sess.IP,
		&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, User{}, ErrNotFound
	}
	if err != nil {
		return Session{}, User{}, err
	}
	if sess.RevokedAt != nil || time.Now().After(sess.ExpiresAt) {
		return Session{}, User{}, ErrNotFound
	}
	return sess, u, nil
}

func (s *Store) RevokeSession(ctx context.Context, tokenHash string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE sessions SET revoked_at = NOW() WHERE token_hash = $1 AND revoked_at IS NULL
	`, tokenHash)
	return err
}

func (s *Store) RevokeAllUserSessions(ctx context.Context, userID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL
	`, userID)
	return err
}

func (s *Store) TouchSession(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `UPDATE sessions SET last_seen_at = NOW() WHERE id = $1`, id)
	return err
}

// DeleteUserHard removes the user; FK cascades remove sessions, clients, documents.
func (s *Store) DeleteUserHard(ctx context.Context, userID uuid.UUID) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, _ = tx.Exec(ctx, `DELETE FROM document_number_sequences WHERE issuer_key = $1`, userID.String())
	tag, err := tx.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return tx.Commit(ctx)
}

// ExportUserData returns the account part of the GDPR export. Business data
// (issuer, clients, documents) is added by the biz layer, which decrypts it.
func (s *Store) ExportUserData(ctx context.Context, userID uuid.UUID) (map[string]any, error) {
	u, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"exported_at": time.Now().UTC(),
		"user": map[string]any{
			"id": u.ID, "email": u.Email, "display_name": u.DisplayName,
			"created_at": u.CreatedAt, "updated_at": u.UpdatedAt,
		},
	}, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
