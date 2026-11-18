package signatures

import (
	"testing"
	"time"
)

func TestComputeSignatureHash(t *testing.T) {
	content1 := []byte("test pdf content")
	content2 := []byte("different content")

	hash1 := ComputeSignatureHash(content1)
	hash2 := ComputeSignatureHash(content1)
	hash3 := ComputeSignatureHash(content2)

	// Same content should produce same hash
	if hash1 != hash2 {
		t.Errorf("Same content produced different hashes: %s vs %s", hash1, hash2)
	}

	// Different content should produce different hash
	if hash1 == hash3 {
		t.Errorf("Different content produced same hash: %s", hash1)
	}

	// Hash should be 64 characters (SHA256 hex)
	if len(hash1) != 64 {
		t.Errorf("Hash length is %d, expected 64", len(hash1))
	}
}

func TestValidateSignatureHash(t *testing.T) {
	content := []byte("test pdf content")
	hash := ComputeSignatureHash(content)

	// Valid hash should return true
	if !ValidateSignatureHash(content, hash) {
		t.Error("ValidateSignatureHash failed for valid content and hash")
	}

	// Invalid hash should return false
	if ValidateSignatureHash(content, "invalid_hash") {
		t.Error("ValidateSignatureHash succeeded for invalid hash")
	}

	// Different content should not validate
	if ValidateSignatureHash([]byte("different"), hash) {
		t.Error("ValidateSignatureHash succeeded for different content")
	}
}

func TestDocumentSignatureStateIsSignable(t *testing.T) {
	tests := []struct {
		name       string
		state      DocumentSignatureState
		isSignable bool
	}{
		{
			name:       "Unsigned and not locked",
			state:      DocumentSignatureState{Status: StatusUnsigned, IsLocked: false},
			isSignable: true,
		},
		{
			name:       "Signed and locked",
			state:      DocumentSignatureState{Status: StatusSigned, IsLocked: true},
			isSignable: false,
		},
		{
			name:       "Unsigned but locked",
			state:      DocumentSignatureState{Status: StatusUnsigned, IsLocked: true},
			isSignable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.state.IsSignable() != tt.isSignable {
				t.Errorf("IsSignable() = %v, want %v", tt.state.IsSignable(), tt.isSignable)
			}
		})
	}
}

func TestDocumentSignatureStateIsSigned(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		state    DocumentSignatureState
		isSigned bool
	}{
		{
			name:     "Unsigned",
			state:    DocumentSignatureState{Status: StatusUnsigned, SignedAt: nil},
			isSigned: false,
		},
		{
			name:     "Signed with timestamp",
			state:    DocumentSignatureState{Status: StatusSigned, SignedAt: &now},
			isSigned: true,
		},
		{
			name:     "Signed but no timestamp",
			state:    DocumentSignatureState{Status: StatusSigned, SignedAt: nil},
			isSigned: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.state.IsSigned() != tt.isSigned {
				t.Errorf("IsSigned() = %v, want %v", tt.state.IsSigned(), tt.isSigned)
			}
		})
	}
}
