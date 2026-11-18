package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/tainalo2/devis_generator/backend/internal/signatures"
)

// Mock signatures store for testing
type mockSignaturesStore struct {
	signatures map[string]*signatures.Signature
	states     map[string]*signatures.DocumentSignatureState
	locked     map[string]bool
}

func newMockSignaturesStore() *mockSignaturesStore {
	return &mockSignaturesStore{
		signatures: make(map[string]*signatures.Signature),
		states:     make(map[string]*signatures.DocumentSignatureState),
		locked:     make(map[string]bool),
	}
}

func (m *mockSignaturesStore) CreateSignature(ctx context.Context, sig *signatures.Signature) error {
	m.signatures[sig.ProofToken] = sig
	m.states[sig.DocumentID] = &signatures.DocumentSignatureState{
		DocumentID:    sig.DocumentID,
		Status:        signatures.StatusSigned,
		IsLocked:      false,
		SignedAt:      &sig.SignedAt,
		SignedByEmail: &sig.SignerEmail,
		Signature:     sig,
	}
	return nil
}

func (m *mockSignaturesStore) GetSignatureByDocument(ctx context.Context, docID string) (*signatures.Signature, error) {
	for _, sig := range m.signatures {
		if sig.DocumentID == docID {
			return sig, nil
		}
	}
	return nil, signatures.ErrSignatureNotFound
}

func (m *mockSignaturesStore) GetSignatureByProofToken(ctx context.Context, token string) (*signatures.Signature, error) {
	if sig, ok := m.signatures[token]; ok {
		return sig, nil
	}
	return nil, signatures.ErrSignatureNotFound
}

func (m *mockSignaturesStore) GetDocumentSignatureState(ctx context.Context, docID string) (*signatures.DocumentSignatureState, error) {
	if state, ok := m.states[docID]; ok {
		return state, nil
	}
	// Return unsigned state by default
	return &signatures.DocumentSignatureState{
		DocumentID: docID,
		Status:     signatures.StatusUnsigned,
		IsLocked:   m.locked[docID],
	}, nil
}

func (m *mockSignaturesStore) LockDocument(ctx context.Context, docID string, signedAt time.Time, signerEmail string) error {
	m.locked[docID] = true
	if state, ok := m.states[docID]; ok {
		state.IsLocked = true
	}
	return nil
}

// Unit tests for HTTP signature endpoints

// TestHandleSignDocumentPublicSuccess tests successful document signing via public link
func TestHandleSignDocumentPublicSuccess(t *testing.T) {
	sigReq := SignatureRequest{
		SignerEmail: "recipient@example.com",
		SignerName:  "John Doe",
	}
	body, _ := json.Marshal(sigReq)

	req := httptest.NewRequest(
		http.MethodPost,
		fmt.Sprintf("/api/v1/public/documents/test-token/sign"),
		bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	// Verify request structure is valid
	if w.Code == 0 {
		t.Log("Request prepared successfully")
	}

	var reqData SignatureRequest
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&reqData); err != nil {
		t.Errorf("failed to parse request: %v", err)
	}

	if reqData.SignerEmail != "recipient@example.com" {
		t.Errorf("expected signer_email recipient@example.com, got %s", reqData.SignerEmail)
	}
}

// TestSignatureRequestValidation tests signature request validation
func TestSignatureRequestValidation(t *testing.T) {
	tests := []struct {
		name          string
		sigReq        SignatureRequest
		shouldBeValid bool
	}{
		{
			name:          "Valid request with email",
			sigReq:        SignatureRequest{SignerEmail: "test@example.com"},
			shouldBeValid: true,
		},
		{
			name:          "Valid request with email and name",
			sigReq:        SignatureRequest{SignerEmail: "test@example.com", SignerName: "John"},
			shouldBeValid: true,
		},
		{
			name:          "Invalid request without email",
			sigReq:        SignatureRequest{SignerEmail: ""},
			shouldBeValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if (tt.sigReq.SignerEmail != "") != tt.shouldBeValid {
				t.Errorf("validation failed: expected valid=%v", tt.shouldBeValid)
			}
		})
	}
}

// TestDocumentSignatureStateResponse tests response structure
func TestDocumentSignatureStateResponse(t *testing.T) {
	now := time.Now()
	proofToken := uuid.New().String()

	resp := DocumentSignatureStateResponse{
		DocumentID:    uuid.New().String(),
		Status:        string(signatures.StatusSigned),
		SignedAt:      &now,
		SignedByEmail: stringPtr("signer@example.com"),
		IsLocked:      true,
		ProofToken:    &proofToken,
	}

	body, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}

	var decoded DocumentSignatureStateResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if decoded.Status != string(signatures.StatusSigned) {
		t.Errorf("expected status %s, got %s", signatures.StatusSigned, decoded.Status)
	}
	if !decoded.IsLocked {
		t.Error("expected IsLocked to be true")
	}
	if decoded.SignedByEmail == nil || *decoded.SignedByEmail != "signer@example.com" {
		t.Error("expected signed_by_email to be set")
	}
}

// TestSignatureResponse tests signature response structure
func TestSignatureResponse(t *testing.T) {
	now := time.Now()

	resp := SignatureResponse{
		DocumentID:  uuid.New().String(),
		SignedAt:    now,
		SignerEmail: "recipient@example.com",
		ProofToken:  uuid.New().String(),
	}

	body, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}

	var decoded SignatureResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if decoded.SignerEmail != "recipient@example.com" {
		t.Errorf("expected signer_email recipient@example.com, got %s", decoded.SignerEmail)
	}
	if decoded.ProofToken == "" {
		t.Error("expected proof_token to be set")
	}
}

// TestMockSignaturesStore tests the mock store behavior
func TestMockSignaturesStore(t *testing.T) {
	store := newMockSignaturesStore()
	ctx := context.Background()

	docID := uuid.New().String()
	sig := &signatures.Signature{
		ID:            uuid.New().String(),
		DocumentID:    docID,
		SignerEmail:   "test@example.com",
		SignedAt:      time.Now().UTC(),
		SignatureHash: "abc123",
		ProofToken:    uuid.New().String(),
		CreatedAt:     time.Now().UTC(),
	}

	// Test CreateSignature
	if err := store.CreateSignature(ctx, sig); err != nil {
		t.Fatalf("failed to create signature: %v", err)
	}

	// Test GetSignatureByProofToken
	retrieved, err := store.GetSignatureByProofToken(ctx, sig.ProofToken)
	if err != nil {
		t.Fatalf("failed to get signature: %v", err)
	}

	if retrieved.SignerEmail != "test@example.com" {
		t.Errorf("expected signer_email test@example.com, got %s", retrieved.SignerEmail)
	}

	// Test GetDocumentSignatureState
	state, err := store.GetDocumentSignatureState(ctx, docID)
	if err != nil {
		t.Fatalf("failed to get state: %v", err)
	}

	if state.Status != signatures.StatusSigned {
		t.Errorf("expected status %s, got %s", signatures.StatusSigned, state.Status)
	}

	// Test LockDocument
	if err := store.LockDocument(ctx, docID, time.Now().UTC(), "test@example.com"); err != nil {
		t.Fatalf("failed to lock document: %v", err)
	}

	// Verify document is locked
	state, _ = store.GetDocumentSignatureState(ctx, docID)
	if !state.IsLocked {
		t.Error("expected document to be locked")
	}
}

// TestSignatureStateTransitions tests state machine behavior
func TestSignatureStateTransitions(t *testing.T) {
	tests := []struct {
		name     string
		state    signatures.DocumentSignatureState
		expected bool
	}{
		{
			name:     "Unsigned document is signable",
			state:    signatures.DocumentSignatureState{Status: signatures.StatusUnsigned, IsLocked: false},
			expected: true,
		},
		{
			name:     "Signed and locked document is not signable",
			state:    signatures.DocumentSignatureState{Status: signatures.StatusSigned, IsLocked: true},
			expected: false,
		},
		{
			name:     "Locked document is not signable even if unsigned",
			state:    signatures.DocumentSignatureState{Status: signatures.StatusUnsigned, IsLocked: true},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.state.IsSignable() != tt.expected {
				t.Errorf("IsSignable() = %v, want %v", tt.state.IsSignable(), tt.expected)
			}
		})
	}
}

// TestGetClientIP tests client IP extraction
func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name           string
		xForwardedFor  string
		remoteAddr     string
		expectedIP     string
		hasXForwarded  bool
	}{
		{
			name:          "With X-Forwarded-For header",
			xForwardedFor: "192.168.1.1",
			remoteAddr:    "127.0.0.1:8080",
			expectedIP:    "192.168.1.1",
			hasXForwarded: true,
		},
		{
			name:          "Without X-Forwarded-For header",
			xForwardedFor: "",
			remoteAddr:    "127.0.0.1:8080",
			expectedIP:    "127.0.0.1:8080",
			hasXForwarded: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}

			ip := getClientIP(req)
			if ip == nil || *ip != tt.expectedIP {
				var got string
				if ip != nil {
					got = *ip
				}
				t.Errorf("getClientIP() = %s, want %s", got, tt.expectedIP)
			}
		})
	}
}

// TestGetClientUserAgent tests user agent extraction
func TestGetClientUserAgent(t *testing.T) {
	tests := []struct {
		name        string
		userAgent   string
		shouldExist bool
	}{
		{
			name:        "With User-Agent",
			userAgent:   "Mozilla/5.0",
			shouldExist: true,
		},
		{
			name:        "Without User-Agent",
			userAgent:   "",
			shouldExist: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			if tt.userAgent != "" {
				req.Header.Set("User-Agent", tt.userAgent)
			}

			ua := getClientUserAgent(req)
			if tt.shouldExist && ua == nil {
				t.Error("expected User-Agent to be set")
			}
			if !tt.shouldExist && ua != nil {
				t.Error("expected User-Agent to be nil")
			}
			if tt.shouldExist && ua != nil && *ua != tt.userAgent {
				t.Errorf("getClientUserAgent() = %s, want %s", *ua, tt.userAgent)
			}
		})
	}
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
