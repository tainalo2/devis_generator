package biz

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Collaborator represents a user with access to an account
type Collaborator struct {
	ID        uuid.UUID
	Email     string
	Role      string
	CreatedAt time.Time
}

// Invitation represents a pending invitation
type Invitation struct {
	ID        uuid.UUID
	Email     string
	Role      string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// ErrDuplicateInvitation is returned when an active invitation already exists
var ErrDuplicateInvitation = errors.New("active invitation already exists")

// ErrCannotInviteSelf is returned when trying to invite yourself
var ErrCannotInviteSelf = errors.New("cannot invite yourself")

// ErrCannotRevokeLastOwner is returned when trying to revoke the last owner
var ErrCannotRevokeLastOwner = errors.New("cannot revoke the last owner")

// ListCollaborators returns all collaborators for an account (owner sees all, collaborators see all in account)
func (s *Store) ListCollaborators(ctx context.Context, accountID uuid.UUID) ([]Collaborator, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
			ac.id,
			u.email,
			ac.role,
			ac.created_at
		FROM account_collaborators ac
		JOIN users u ON ac.user_id = u.id
		WHERE ac.account_id = $1
		ORDER BY ac.created_at ASC
	`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var collaborators []Collaborator
	for rows.Next() {
		var c Collaborator
		if err := rows.Scan(&c.ID, &c.Email, &c.Role, &c.CreatedAt); err != nil {
			return nil, err
		}
		collaborators = append(collaborators, c)
	}
	return collaborators, rows.Err()
}

// ListPendingInvitations returns all non-expired invitations for an account
func (s *Store) ListPendingInvitations(ctx context.Context, accountID uuid.UUID) ([]Invitation, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
			id,
			email,
			role,
			expires_at,
			created_at
		FROM invitations
		WHERE account_id = $1 AND expires_at > NOW()
		ORDER BY created_at DESC
	`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invitations []Invitation
	for rows.Next() {
		var inv Invitation
		if err := rows.Scan(&inv.ID, &inv.Email, &inv.Role, &inv.ExpiresAt, &inv.CreatedAt); err != nil {
			return nil, err
		}
		invitations = append(invitations, inv)
	}
	return invitations, rows.Err()
}

// CreateInvitation creates a new invitation for a collaborator
func (s *Store) CreateInvitation(
	ctx context.Context,
	invitationID uuid.UUID,
	accountID uuid.UUID,
	email string,
	role string,
	token string,
	expiresAt time.Time,
	createdBy uuid.UUID,
) (*Invitation, error) {
	// Check if the user is trying to invite themselves
	var existingUser uuid.UUID
	err := s.pool.QueryRow(ctx, "SELECT id FROM users WHERE id = $1 AND email = $2", createdBy, email).Scan(&existingUser)
	if err == nil {
		return nil, ErrCannotInviteSelf
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	// Insert invitation with ON CONFLICT to handle duplicate active invitations
	var createdAt time.Time
	err = s.pool.QueryRow(ctx, `
		INSERT INTO invitations (id, account_id, email, role, token, expires_at, created_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (account_id, email) WHERE expires_at > NOW()
		DO UPDATE SET
			token = EXCLUDED.token,
			expires_at = EXCLUDED.expires_at,
			created_at = EXCLUDED.created_at,
			created_by = EXCLUDED.created_by
		RETURNING id, email, role, expires_at, created_at
	`, invitationID, accountID, email, role, token, expiresAt, time.Now(), createdBy).Scan(
		&invitationID, &email, &role, &expiresAt, &createdAt,
	)

	if err != nil {
		return nil, err
	}

	return &Invitation{
		ID:        invitationID,
		Email:     email,
		Role:      role,
		ExpiresAt: expiresAt,
		CreatedAt: createdAt,
	}, nil
}

// RevokeCollaborator removes a collaborator from an account
func (s *Store) RevokeCollaborator(ctx context.Context, accountID uuid.UUID, collaboratorID uuid.UUID) error {
	// Verify the collaborator belongs to this account and get the role
	var collabRole string
	err := s.pool.QueryRow(ctx, `
		SELECT role
		FROM account_collaborators
		WHERE id = $1 AND account_id = $2
	`, collaboratorID, accountID).Scan(&collabRole)

	if err != nil {
		return err
	}

	// Prevent revoking the last owner
	if collabRole == "owner" {
		var ownerCount int64
		err = s.pool.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM account_collaborators
			WHERE account_id = $1 AND role = 'owner'
		`, accountID).Scan(&ownerCount)

		if err != nil {
			return err
		}

		if ownerCount == 1 {
			return ErrCannotRevokeLastOwner
		}
	}

	// Delete the collaborator
	_, err = s.pool.Exec(ctx, `
		DELETE FROM account_collaborators
		WHERE id = $1
	`, collaboratorID)

	return err
}

// CancelInvitation cancels a pending invitation
func (s *Store) CancelInvitation(ctx context.Context, accountID uuid.UUID, invitationID uuid.UUID) error {
	// Verify the invitation belongs to this account and hasn't expired
	var invExists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM invitations
			WHERE id = $1 AND account_id = $2 AND expires_at > NOW()
		)
	`, invitationID, accountID).Scan(&invExists)

	if err != nil {
		return err
	}

	if !invExists {
		return pgx.ErrNoRows
	}

	// Delete the invitation
	_, err = s.pool.Exec(ctx, `
		DELETE FROM invitations
		WHERE id = $1
	`, invitationID)

	return err
}

// LogAction logs a collaborator-related action for audit trail
func (s *Store) LogAction(ctx context.Context, actor interface{}, action string, details string) error {
	// This is a simplified version - in production you'd want to use proper audit logging
	// For now, we'll skip logging to keep it simple
	return nil
}

// IsUserOwner checks if a user is an owner of their own account
func (s *Store) IsUserOwner(ctx context.Context, userID uuid.UUID) (bool, error) {
	var role string
	err := s.pool.QueryRow(ctx, `
		SELECT role FROM account_collaborators
		WHERE account_id = $1 AND user_id = $2 AND role = 'owner'
	`, userID, userID).Scan(&role)

	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
