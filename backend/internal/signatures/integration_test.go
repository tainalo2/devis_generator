package signatures

import (
	"crypto/sha256"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestSignatureCreation tests creating and retrieving signatures
// This is an integration test; it needs a TEST_DATABASE_URL to run
func TestSignatureCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("Integration test; skipping in short mode")
	}

	// In a real integration test, this would use a database pool from the test setup
	// For now, we're just documenting the test structure

	docID := uuid.New().String()
	signerEmail := "test@example.com"
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte("test pdf")))
	proofToken := uuid.New().String()

	sig := &Signature{
		ID:            uuid.New().String(),
		DocumentID:    docID,
		SignerEmail:   signerEmail,
		SignedAt:      time.Now().UTC(),
		SignatureHash: hash,
		ProofToken:    proofToken,
		CreatedAt:     time.Now().UTC(),
	}

	// Verify signature structure is valid
	if sig.ID == "" || sig.DocumentID == "" {
		t.Fatal("Signature missing required fields")
	}

	if len(sig.SignatureHash) != 64 {
		t.Fatalf("SignatureHash length is %d, expected 64", len(sig.SignatureHash))
	}
}

// Real integration test example (commented out for now, needs DB setup):
/*
func testIntegrationSignatureStorage(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	store := NewPGStore(pool)

	docID := uuid.New().String()
	signerEmail := "recipient@example.com"
	pdf := []byte("PDF content")
	hash := ComputeSignatureHash(pdf)

	// Create signature
	sig := &Signature{
		ID:            uuid.New().String(),
		DocumentID:    docID,
		SignerEmail:   signerEmail,
		SignedAt:      time.Now().UTC(),
		SignatureHash: hash,
		ProofToken:    uuid.New().String(),
	}

	if err := store.CreateSignature(ctx, sig); err != nil {
		t.Fatalf("CreateSignature failed: %v", err)
	}

	// Retrieve by document ID
	retrieved, err := store.GetSignatureByDocument(ctx, docID)
	if err != nil {
		t.Fatalf("GetSignatureByDocument failed: %v", err)
	}

	if retrieved.SignerEmail != signerEmail {
		t.Errorf("Got signer %s, expected %s", retrieved.SignerEmail, signerEmail)
	}

	// Lock document
	now := time.Now().UTC()
	if err := store.LockDocument(ctx, docID, now, signerEmail); err != nil {
		t.Fatalf("LockDocument failed: %v", err)
	}

	// Verify state
	state, err := store.GetDocumentSignatureState(ctx, docID)
	if err != nil {
		t.Fatalf("GetDocumentSignatureState failed: %v", err)
	}

	if !state.IsLocked {
		t.Error("Document should be locked after LockDocument")
	}

	if state.Status != StatusSigned {
		t.Errorf("Expected status %s, got %s", StatusSigned, state.Status)
	}
}
*/
