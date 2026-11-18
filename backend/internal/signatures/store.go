package signatures

import (
	"context"
	"errors"
	"time"
)

// Store defines the interface for signature persistence
type Store interface {
	// CreateSignature stores a new signature proof for a document
	// Returns error if document is already signed or locked
	CreateSignature(ctx context.Context, sig *Signature) error

	// GetSignatureByDocument returns the signature for a document (if any)
	GetSignatureByDocument(ctx context.Context, documentID string) (*Signature, error)

	// GetSignatureByProofToken returns a signature by its proof token
	// Used for public verification endpoints
	GetSignatureByProofToken(ctx context.Context, proofToken string) (*Signature, error)

	// LockDocument marks a document as locked/signed
	// Sets signed_at, is_locked=true, and updates status to 'signed'
	LockDocument(ctx context.Context, documentID string, signedAt time.Time, signedByEmail string) error

	// GetDocumentSignatureState returns the signature state of a document
	GetDocumentSignatureState(ctx context.Context, documentID string) (*DocumentSignatureState, error)
}

// Error types for signature operations
var (
	ErrAlreadySigned     = errors.New("document is already signed")
	ErrDocumentLocked    = errors.New("document is locked")
	ErrSignatureNotFound = errors.New("signature not found")
	ErrInvalidHash       = errors.New("signature hash validation failed")
)
