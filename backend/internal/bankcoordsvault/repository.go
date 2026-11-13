// Package bankcoordsvault manages encrypted bank coordinates (IBAN/BIC/holder name)
// stored via the paymentsecrets vault.
package bankcoordsvault

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tainalo2/devis_generator/backend/internal/bankcoord"
	"github.com/tainalo2/devis_generator/backend/internal/paymentsecrets"
)

var (
	ErrNotFound    = errors.New("bank coordinates not found")
	ErrConflict    = errors.New("bank coordinates already exist for this account")
	ErrUnauthorized = errors.New("unauthorized access to bank coordinates")
)

// BankCoordinates represents the stored metadata for encrypted bank credentials.
// The actual IBAN and BIC are stored in the paymentsecrets vault.
type BankCoordinates struct {
	ID                 uuid.UUID
	UserID             uuid.UUID
	AccountHolderName  string
	SecretID           uuid.UUID // payment_secrets.id for IBAN
	BICSecretID        *uuid.UUID // payment_secrets.id for BIC (optional)
	BankName           string
	CreatedAt          string
	UpdatedAt          string
}

// Repository manages bank coordinates in PostgreSQL.
type Repository struct {
	pool  *pgxpool.Pool
	vault *paymentsecrets.Vault
}

// NewRepository creates a new bank coordinates repository.
func NewRepository(pool *pgxpool.Pool, vault *paymentsecrets.Vault) *Repository {
	return &Repository{pool: pool, vault: vault}
}

// CreateOrUpdate creates or updates bank coordinates for a user.
// - Validates IBAN and optional BIC via bankcoord package
// - Stores IBAN and BIC encrypted via paymentsecrets vault
// - Creates or updates a single bank_coordinates row per user (UNIQUE constraint)
func (r *Repository) CreateOrUpdate(ctx context.Context, actor paymentsecrets.Actor, userID uuid.UUID, 
	accountHolderName, rawIBAN, rawBIC, bankName string) (*BankCoordinates, error) {
	
	// Validate IBAN
	iban, err := bankcoord.ValidateIBAN(rawIBAN)
	if err != nil {
		return nil, err
	}
	
	// Validate optional BIC
	bic, err := bankcoord.ValidateBIC(rawBIC)
	if err != nil {
		return nil, err
	}
	
	// Validate account holder name
	if len(accountHolderName) == 0 || len(accountHolderName) > 255 {
		return nil, bankcoord.ErrMissingName
	}
	
	// Store IBAN in vault
	ibanMetadata, err := r.vault.Put(ctx, actor, userID, paymentsecrets.KindIBAN, "primary", []byte(iban))
	if err != nil {
		return nil, err
	}
	
	// Store BIC in vault if provided
	var bicSecretID *uuid.UUID
	if bic != "" {
		bicMetadata, err := r.vault.Put(ctx, actor, userID, paymentsecrets.KindBIC, "bic", []byte(bic))
		if err != nil {
			return nil, err
		}
		bicSecretID = &bicMetadata.ID
	}
	
	// Upsert bank_coordinates row
	var result BankCoordinates
	err = r.pool.QueryRow(ctx, `
		INSERT INTO bank_coordinates (user_id, account_holder_name, secret_id, bic_secret_id, bank_name)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE
		SET account_holder_name = EXCLUDED.account_holder_name,
		    secret_id = EXCLUDED.secret_id,
		    bic_secret_id = EXCLUDED.bic_secret_id,
		    bank_name = EXCLUDED.bank_name,
		    updated_at = NOW()
		RETURNING id, user_id, account_holder_name, secret_id, bic_secret_id, bank_name, created_at, updated_at
	`, userID, accountHolderName, ibanMetadata.ID, bicSecretID, bankName).Scan(
		&result.ID, &result.UserID, &result.AccountHolderName, &result.SecretID, 
		&result.BICSecretID, &result.BankName, &result.CreatedAt, &result.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &result, nil
}

// Read retrieves bank coordinates for a user (plaintext IBAN/BIC via vault reveal).
// Returns: (accountHolderName, IBAN, BIC, bankName, error)
func (r *Repository) Read(ctx context.Context, actor paymentsecrets.Actor, userID uuid.UUID) (
	accountHolderName, iban, bic, bankName string, err error) {
	
	var coords BankCoordinates
	err = r.pool.QueryRow(ctx, `
		SELECT id, user_id, account_holder_name, secret_id, bic_secret_id, bank_name, created_at, updated_at
		FROM bank_coordinates
		WHERE user_id = $1
	`, userID).Scan(
		&coords.ID, &coords.UserID, &coords.AccountHolderName, &coords.SecretID, 
		&coords.BICSecretID, &coords.BankName, &coords.CreatedAt, &coords.UpdatedAt,
	)
	
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", "", "", ErrNotFound
	}
	if err != nil {
		return "", "", "", "", err
	}
	
	// Reveal IBAN from vault
	ibanBytes, _, err := r.vault.Reveal(ctx, actor, userID, paymentsecrets.KindIBAN, "primary")
	if err != nil {
		return "", "", "", "", err
	}
	
	// Reveal BIC if available
	if coords.BICSecretID != nil {
		bicBytes, _, err := r.vault.Reveal(ctx, actor, userID, paymentsecrets.KindBIC, "bic")
		if err != nil {
			return "", "", "", "", err
		}
		bic = string(bicBytes)
	}
	
	return coords.AccountHolderName, string(ibanBytes), bic, coords.BankName, nil
}

// GetMetadata retrieves bank coordinates metadata without revealing secrets.
// Useful for listing or displaying masked values.
func (r *Repository) GetMetadata(ctx context.Context, userID uuid.UUID) (*BankCoordinates, error) {
	var coords BankCoordinates
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, account_holder_name, secret_id, bic_secret_id, bank_name, created_at, updated_at
		FROM bank_coordinates
		WHERE user_id = $1
	`, userID).Scan(
		&coords.ID, &coords.UserID, &coords.AccountHolderName, &coords.SecretID, 
		&coords.BICSecretID, &coords.BankName, &coords.CreatedAt, &coords.UpdatedAt,
	)
	
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	
	return &coords, err
}

// Delete removes bank coordinates for a user (cascade deletes secrets via foreign key).
func (r *Repository) Delete(ctx context.Context, actor paymentsecrets.Actor, userID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM bank_coordinates
		WHERE user_id = $1
	`, userID)
	
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	
	return err
}

// Exists checks if bank coordinates exist for a user without revealing secrets.
func (r *Repository) Exists(ctx context.Context, userID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM bank_coordinates WHERE user_id = $1)
	`, userID).Scan(&exists)
	
	return exists, err
}
