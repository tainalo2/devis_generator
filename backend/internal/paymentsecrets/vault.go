// Package paymentsecrets stores payment credentials encrypted at the
// application boundary and records every plaintext read or write.
package paymentsecrets

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	appcrypto "github.com/tainalo2/devis_generator/backend/internal/crypto"
)

type Kind string

const (
	KindIBAN                    Kind = "iban"
	KindBIC                     Kind = "bic"
	KindBYOProviderAPIKey       Kind = "byo_provider_api_key"
	KindStripeConnectCredential Kind = "stripe_connect_credential"
	KindOAuthGmailToken         Kind = "oauth_gmail_token"
	KindOAuthMicrosoftToken     Kind = "oauth_microsoft_token"
	KindSMTPPassword            Kind = "smtp_password"
)

var (
	ErrInvalidKind      = errors.New("invalid payment secret kind")
	ErrActorRequired    = errors.New("payment secret access requires an actor")
	ErrOwnerRequired    = errors.New("payment secret requires an owner")
	ErrEmptySecret      = errors.New("payment secret must not be empty")
	ErrInvalidReference = errors.New("payment secret reference must be between 1 and 128 characters")
	ErrNotFound         = errors.New("payment secret not found")
)

// Actor is persisted with every plaintext access. RequestID is correlation
// metadata only and must never contain credentials.
type Actor struct {
	UserID    uuid.UUID
	RequestID string
}

// Metadata is safe for JSON/API responses: it intentionally has no ciphertext
// or plaintext field.
type Metadata struct {
	ID          uuid.UUID `json:"id"`
	OwnerID     uuid.UUID `json:"owner_id"`
	Kind        Kind      `json:"kind"`
	Reference   string    `json:"reference"`
	MaskedValue string    `json:"masked_value"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SealedSecret is encrypted data safe to pass to a transactional repository.
type SealedSecret struct {
	Ciphertext  []byte
	MaskedValue string
}

type ciphertextRepository interface {
	PutCiphertext(context.Context, Actor, uuid.UUID, Kind, string, []byte, string) (Metadata, error)
	ReadCiphertext(context.Context, Actor, uuid.UUID, Kind, string) ([]byte, Metadata, error)
	DeleteCiphertext(context.Context, Actor, uuid.UUID, Kind, string) error
}

// Vault is the only payment-secret plaintext boundary. Repositories receive
// ciphertext only; reads pass through the repository first so they are audited.
type Vault struct {
	repo ciphertextRepository
	box  *appcrypto.Box
}

func NewVault(repo ciphertextRepository, box *appcrypto.Box) *Vault {
	return &Vault{repo: repo, box: box}
}

func (v *Vault) Put(ctx context.Context, actor Actor, ownerID uuid.UUID, kind Kind, reference string, plaintext []byte) (Metadata, error) {
	reference, err := validateAccess(actor, ownerID, kind, reference)
	if err != nil {
		return Metadata{}, err
	}
	if len(plaintext) == 0 {
		return Metadata{}, ErrEmptySecret
	}
	sealed, err := v.Seal(plaintext)
	if err != nil {
		return Metadata{}, err
	}
	return v.repo.PutCiphertext(ctx, actor, ownerID, kind, reference, sealed.Ciphertext, sealed.MaskedValue)
}

// Seal encrypts a secret without persisting it, allowing a caller to commit
// the ciphertext atomically with related database state.
func (v *Vault) Seal(plaintext []byte) (SealedSecret, error) {
	if len(plaintext) == 0 {
		return SealedSecret{}, ErrEmptySecret
	}
	ciphertext, err := v.box.Encrypt(plaintext)
	if err != nil {
		return SealedSecret{}, err
	}
	return SealedSecret{Ciphertext: ciphertext, MaskedValue: Mask(plaintext)}, nil
}

func (v *Vault) Reveal(ctx context.Context, actor Actor, ownerID uuid.UUID, kind Kind, reference string) ([]byte, Metadata, error) {
	reference, err := validateAccess(actor, ownerID, kind, reference)
	if err != nil {
		return nil, Metadata{}, err
	}
	ciphertext, metadata, err := v.repo.ReadCiphertext(ctx, actor, ownerID, kind, reference)
	if err != nil {
		return nil, Metadata{}, err
	}
	plaintext, err := v.box.Decrypt(ciphertext)
	if err != nil {
		return nil, Metadata{}, err
	}
	return plaintext, metadata, nil
}

// Delete removes a secret through the audited repository boundary.
func (v *Vault) Delete(ctx context.Context, actor Actor, ownerID uuid.UUID, kind Kind, reference string) error {
	reference, err := validateAccess(actor, ownerID, kind, reference)
	if err != nil {
		return err
	}
	return v.repo.DeleteCiphertext(ctx, actor, ownerID, kind, reference)
}

func validateAccess(actor Actor, ownerID uuid.UUID, kind Kind, reference string) (string, error) {
	if actor.UserID == uuid.Nil {
		return "", ErrActorRequired
	}
	if ownerID == uuid.Nil {
		return "", ErrOwnerRequired
	}
	if !kind.Valid() {
		return "", ErrInvalidKind
	}
	reference = strings.TrimSpace(reference)
	if n := utf8.RuneCountInString(reference); n == 0 || n > 128 {
		return "", ErrInvalidReference
	}
	return reference, nil
}

func (k Kind) Valid() bool {
	switch k {
	case KindIBAN, KindBIC, KindBYOProviderAPIKey, KindStripeConnectCredential,
		KindOAuthGmailToken, KindOAuthMicrosoftToken, KindSMTPPassword:
		return true
	default:
		return false
	}
}

// Mask returns only the last four runes, never the full secret. Short values
// are completely hidden.
func Mask(secret []byte) string {
	runes := []rune(strings.TrimSpace(string(secret)))
	if len(runes) <= 4 {
		return "••••"
	}
	return "•••• " + string(runes[len(runes)-4:])
}
