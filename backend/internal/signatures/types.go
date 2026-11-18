package signatures

import (
	"crypto/sha256"
	"fmt"
	"time"
)

// Status represents the signature status of a document
type Status string

const (
	StatusUnsigned Status = "unsigned"
	StatusSigned   Status = "signed"
	StatusLocked   Status = "locked"
)

// Signature represents a document signature proof
type Signature struct {
	ID            string        `json:"id"`
	DocumentID    string        `json:"document_id"`
	UserID        *string       `json:"user_id,omitempty"`
	SignerEmail   string        `json:"signer_email"`
	SignedAt      time.Time     `json:"signed_at"`
	SignatureHash string        `json:"signature_hash"`
	IPAddress     *string       `json:"ip_address,omitempty"`
	UserAgent     *string       `json:"user_agent,omitempty"`
	ProofToken    string        `json:"proof_token"`
	CreatedAt     time.Time     `json:"created_at"`
}

// SignatureProof contains all information about a completed signature
type SignatureProof struct {
	DocumentID    string
	SignerEmail   string
	SignedAt      time.Time
	SignatureHash string
	ProofToken    string
}

// ComputeSignatureHash computes SHA256 hash of PDF content
// Returns the hex-encoded hash string
func ComputeSignatureHash(pdfContent []byte) string {
	hash := sha256.Sum256(pdfContent)
	return fmt.Sprintf("%x", hash)
}

// ValidateSignatureHash verifies that content matches the stored hash
func ValidateSignatureHash(pdfContent []byte, expectedHash string) bool {
	computed := ComputeSignatureHash(pdfContent)
	return computed == expectedHash
}

// DocumentSignatureState tracks the signature state of a document
type DocumentSignatureState struct {
	DocumentID    string
	Status        Status
	SignedAt      *time.Time
	SignedByEmail *string
	IsLocked      bool
	Signature     *Signature
}

// IsSignable returns true if document can be signed
func (s *DocumentSignatureState) IsSignable() bool {
	return s.Status == StatusUnsigned && !s.IsLocked
}

// IsSigned returns true if document is signed
func (s *DocumentSignatureState) IsSigned() bool {
	return s.Status == StatusSigned && s.SignedAt != nil
}
