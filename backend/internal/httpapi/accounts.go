package httpapi

import (
	"context"

	"github.com/google/uuid"
)

// isUserCollaborator checks if a user has access to an account.
// It returns true if:
// - The user IS the account owner (account_id == user_id), OR
// - The user is listed as a collaborator in the account_collaborators table
//
// NOTE: This implementation currently only supports single-user accounts (account owner = user).
// Multi-account support with collaborators requires the account_collaborators table from Multi-T2.
func (s *Server) isUserCollaborator(ctx context.Context, accountID uuid.UUID, userID uuid.UUID) (bool, error) {
	// User owns the account themselves (single-user account)
	if accountID == userID {
		return true, nil
	}

	// Multi-account support with collaborators would query:
	// SELECT EXISTS(SELECT 1 FROM account_collaborators WHERE account_id = $1 AND user_id = $2)
	// For now, single-user accounts only: non-owners are forbidden
	return false, nil
}
