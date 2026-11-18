package signatures

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PGStore implements Store interface using PostgreSQL
type PGStore struct {
	pool *pgxpool.Pool
}

// NewPGStore creates a new PostgreSQL signature store
func NewPGStore(pool *pgxpool.Pool) *PGStore {
	return &PGStore{pool: pool}
}

// CreateSignature inserts a new signature record
func (s *PGStore) CreateSignature(ctx context.Context, sig *Signature) error {
	const query = `
		INSERT INTO document_signatures (
			id, document_id, user_id, signer_email, signed_at,
			signature_hash, ip_address, user_agent, proof_token, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		)
	`

	now := time.Now().UTC()
	_, err := s.pool.Exec(ctx, query,
		sig.ID, sig.DocumentID, sig.UserID, sig.SignerEmail, sig.SignedAt,
		sig.SignatureHash, sig.IPAddress, sig.UserAgent, sig.ProofToken, now,
	)

	if err != nil {
		if err.Error() == "violates check constraint \"one_signature_per_document\"" {
			return ErrAlreadySigned
		}
		return fmt.Errorf("failed to create signature: %w", err)
	}

	return nil
}

// GetSignatureByDocument retrieves a signature by document ID
func (s *PGStore) GetSignatureByDocument(ctx context.Context, documentID string) (*Signature, error) {
	const query = `
		SELECT
			id, document_id, user_id, signer_email, signed_at,
			signature_hash, ip_address, user_agent, proof_token, created_at
		FROM document_signatures
		WHERE document_id = $1
		LIMIT 1
	`

	var sig Signature
	err := s.pool.QueryRow(ctx, query, documentID).Scan(
		&sig.ID, &sig.DocumentID, &sig.UserID, &sig.SignerEmail, &sig.SignedAt,
		&sig.SignatureHash, &sig.IPAddress, &sig.UserAgent, &sig.ProofToken, &sig.CreatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, ErrSignatureNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query signature: %w", err)
	}

	return &sig, nil
}

// GetSignatureByProofToken retrieves a signature by its proof token
func (s *PGStore) GetSignatureByProofToken(ctx context.Context, proofToken string) (*Signature, error) {
	const query = `
		SELECT
			id, document_id, user_id, signer_email, signed_at,
			signature_hash, ip_address, user_agent, proof_token, created_at
		FROM document_signatures
		WHERE proof_token = $1
		LIMIT 1
	`

	var sig Signature
	err := s.pool.QueryRow(ctx, query, proofToken).Scan(
		&sig.ID, &sig.DocumentID, &sig.UserID, &sig.SignerEmail, &sig.SignedAt,
		&sig.SignatureHash, &sig.IPAddress, &sig.UserAgent, &sig.ProofToken, &sig.CreatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, ErrSignatureNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query signature by proof token: %w", err)
	}

	return &sig, nil
}

// LockDocument marks a document as signed and locked
func (s *PGStore) LockDocument(ctx context.Context, documentID string, signedAt time.Time, signedByEmail string) error {
	const query = `
		UPDATE documents
		SET signed_at = $2, is_locked = TRUE, signed_by_email = $3, status = 'signed', updated_at = NOW()
		WHERE id = $1 AND is_locked = FALSE
	`

	result, err := s.pool.Exec(ctx, query, documentID, signedAt, signedByEmail)
	if err != nil {
		return fmt.Errorf("failed to lock document: %w", err)
	}

	if result.RowsAffected() == 0 {
		// Either document doesn't exist or is already locked
		return ErrDocumentLocked
	}

	return nil
}

// GetDocumentSignatureState returns the signature state of a document
func (s *PGStore) GetDocumentSignatureState(ctx context.Context, documentID string) (*DocumentSignatureState, error) {
	const query = `
		SELECT
			id, signed_at, is_locked, signed_by_email, status
		FROM documents
		WHERE id = $1
	`

	var docID string
	var signedAt *time.Time
	var isLocked bool
	var signedByEmail *string
	var status string

	err := s.pool.QueryRow(ctx, query, documentID).Scan(
		&docID, &signedAt, &isLocked, &signedByEmail, &status,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("document not found: %w", err)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query document state: %w", err)
	}

	// Determine signature status
	var sigStatus Status
	if isLocked && signedAt != nil {
		sigStatus = StatusSigned
	} else {
		sigStatus = StatusUnsigned
	}

	state := &DocumentSignatureState{
		DocumentID:    docID,
		Status:        sigStatus,
		SignedAt:      signedAt,
		SignedByEmail: signedByEmail,
		IsLocked:      isLocked,
	}

	// If signed, fetch the signature proof
	if sigStatus == StatusSigned {
		sig, err := s.GetSignatureByDocument(ctx, documentID)
		if err == nil {
			state.Signature = sig
		}
		// If signature lookup fails, that's okay — we still report the state
	}

	return state, nil
}
