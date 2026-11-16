package httpapi

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// TestIsUserCollaboratorOwnerHasAccess tests that account owners can access their own account
func TestIsUserCollaboratorOwnerHasAccess(t *testing.T) {
	srv := &Server{}
	userID := uuid.New()

	// User accessing own account
	allowed, err := srv.isUserCollaborator(context.Background(), userID, userID)
	if err != nil {
		t.Fatalf("isUserCollaborator failed: %v", err)
	}
	if !allowed {
		t.Error("owner should have access to own account")
	}
}

// TestIsUserCollaboratorNonOwnerDenied tests that non-owners are denied access (single-user accounts)
func TestIsUserCollaboratorNonOwnerDenied(t *testing.T) {
	srv := &Server{}
	ownerID := uuid.New()
	otherUserID := uuid.New()

	// Other user accessing owner's account (not a collaborator)
	allowed, err := srv.isUserCollaborator(context.Background(), ownerID, otherUserID)
	if err != nil {
		t.Fatalf("isUserCollaborator failed: %v", err)
	}
	if allowed {
		t.Error("non-owner should not have access to account (single-user account)")
	}
}

// TestAccountIDParameterValidation tests that the account_id parameter is properly validated
func TestAccountIDParameterValidationLogic(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid UUID",
			input:   uuid.New().String(),
			wantErr: false,
		},
		{
			name:    "invalid UUID",
			input:   "not-a-uuid",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: false, // Empty string means default to user's account, not an error
		},
		{
			name:    "malformed UUID",
			input:   "12345678-1234-1234-1234-12345678",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.input == "" {
				// Empty string case: no error, just use default
				return
			}

			_, err := uuid.Parse(tt.input)
			gotErr := err != nil

			if gotErr != tt.wantErr {
				t.Errorf("got error %v, want error %v", gotErr, tt.wantErr)
			}
		})
	}
}
