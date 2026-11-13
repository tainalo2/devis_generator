package paymentsecrets

import (
	"bytes"
	"context"
	"testing"

	"github.com/google/uuid"
	appcrypto "github.com/tainalo2/devis_generator/backend/internal/crypto"
)

type fakeRepository struct {
	ciphertext  []byte
	metadata    Metadata
	putActor    Actor
	readActor   Actor
	deleteActor Actor
	deleted     bool
}

func (r *fakeRepository) PutCiphertext(_ context.Context, actor Actor, ownerID uuid.UUID, kind Kind, reference string, ciphertext []byte, masked string) (Metadata, error) {
	r.putActor = actor
	r.ciphertext = append([]byte(nil), ciphertext...)
	r.metadata = Metadata{ID: uuid.New(), OwnerID: ownerID, Kind: kind, Reference: reference, MaskedValue: masked}
	return r.metadata, nil
}

func (r *fakeRepository) ReadCiphertext(_ context.Context, actor Actor, _ uuid.UUID, _ Kind, _ string) ([]byte, Metadata, error) {
	r.readActor = actor
	return append([]byte(nil), r.ciphertext...), r.metadata, nil
}

func (r *fakeRepository) DeleteCiphertext(_ context.Context, actor Actor, _ uuid.UUID, _ Kind, _ string) error {
	r.deleteActor = actor
	r.ciphertext = nil
	r.deleted = true
	return nil
}

func TestDeleteValidatesAndForwardsAuditActor(t *testing.T) {
	box, err := appcrypto.NewBox(bytes.Repeat([]byte{6}, 32))
	if err != nil {
		t.Fatal(err)
	}
	repo := &fakeRepository{}
	vault := NewVault(repo, box)
	actor := Actor{UserID: uuid.New(), RequestID: "req-delete"}

	if err := vault.Delete(context.Background(), actor, uuid.New(), KindSMTPPassword, "default"); err != nil {
		t.Fatal(err)
	}
	if !repo.deleted || repo.deleteActor != actor {
		t.Fatalf("delete not forwarded with actor: %#v", repo.deleteActor)
	}
	if err := vault.Delete(context.Background(), Actor{}, uuid.New(), KindSMTPPassword, "default"); err != ErrActorRequired {
		t.Fatalf("missing actor err = %v", err)
	}
}

func TestPutEncryptsAndReturnsOnlyMaskedMetadata(t *testing.T) {
	box, err := appcrypto.NewBox(bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatal(err)
	}
	repo := &fakeRepository{}
	vault := NewVault(repo, box)
	actor := Actor{UserID: uuid.New(), RequestID: "req-123"}
	ownerID := uuid.New()
	plaintext := "FR7612345678901234567890185"

	metadata, err := vault.Put(context.Background(), actor, ownerID, KindIBAN, "default", []byte(plaintext))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(repo.ciphertext, []byte(plaintext)) {
		t.Fatal("ciphertext contains plaintext")
	}
	if metadata.MaskedValue != "•••• 0185" {
		t.Fatalf("masked value = %q", metadata.MaskedValue)
	}
	if repo.putActor != actor {
		t.Fatalf("actor not forwarded: %#v", repo.putActor)
	}
}

func TestRevealDecryptsAndForwardsAuditActor(t *testing.T) {
	box, err := appcrypto.NewBox(bytes.Repeat([]byte{8}, 32))
	if err != nil {
		t.Fatal(err)
	}
	repo := &fakeRepository{}
	vault := NewVault(repo, box)
	actor := Actor{UserID: uuid.New(), RequestID: "req-read"}
	ownerID := uuid.New()
	secret := []byte("sk_live_1234567890")
	if _, err := vault.Put(context.Background(), actor, ownerID, KindBYOProviderAPIKey, "stripe", secret); err != nil {
		t.Fatal(err)
	}

	got, metadata, err := vault.Reveal(context.Background(), actor, ownerID, KindBYOProviderAPIKey, "stripe")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, secret) {
		t.Fatalf("got %q", got)
	}
	if metadata.MaskedValue != "•••• 7890" {
		t.Fatalf("masked value = %q", metadata.MaskedValue)
	}
	if repo.readActor != actor {
		t.Fatalf("read actor not forwarded: %#v", repo.readActor)
	}
}

func TestPutRejectsUnknownKindAndMissingActor(t *testing.T) {
	box, err := appcrypto.NewBox(bytes.Repeat([]byte{9}, 32))
	if err != nil {
		t.Fatal(err)
	}
	vault := NewVault(&fakeRepository{}, box)
	ownerID := uuid.New()
	actor := Actor{UserID: uuid.New()}

	if _, err := vault.Put(context.Background(), actor, ownerID, Kind("password"), "default", []byte("secret")); err != ErrInvalidKind {
		t.Fatalf("unknown kind err = %v", err)
	}
	if _, err := vault.Put(context.Background(), Actor{}, ownerID, KindIBAN, "default", []byte("FR76")); err != ErrActorRequired {
		t.Fatalf("missing actor err = %v", err)
	}
}

func TestEmailSecretsKindsAccepted(t *testing.T) {
	box, err := appcrypto.NewBox(bytes.Repeat([]byte{10}, 32))
	if err != nil {
		t.Fatal(err)
	}
	repo := &fakeRepository{}
	vault := NewVault(repo, box)
	actor := Actor{UserID: uuid.New(), RequestID: "req-email"}
	ownerID := uuid.New()

	// Test all three email credential kinds
	emailKinds := []Kind{KindOAuthGmailToken, KindOAuthMicrosoftToken, KindSMTPPassword}
	for _, kind := range emailKinds {
		plaintext := []byte("secret-token-" + string(kind))
		metadata, err := vault.Put(context.Background(), actor, ownerID, kind, "test", plaintext)
		if err != nil {
			t.Fatalf("Put(%s) failed: %v", kind, err)
		}
		if metadata.Kind != kind {
			t.Fatalf("kind = %s, want %s", metadata.Kind, kind)
		}

		// Verify Reveal works for email kinds
		revealed, _, err := vault.Reveal(context.Background(), actor, ownerID, kind, "test")
		if err != nil {
			t.Fatalf("Reveal(%s) failed: %v", kind, err)
		}
		if !bytes.Equal(revealed, plaintext) {
			t.Fatalf("revealed = %q, want %q", revealed, plaintext)
		}
	}
}

func TestDeleteForwardsAuditActor(t *testing.T) {
	box, err := appcrypto.NewBox(bytes.Repeat([]byte{11}, 32))
	if err != nil {
		t.Fatal(err)
	}
	repo := &fakeRepository{}
	vault := NewVault(repo, box)
	actor := Actor{UserID: uuid.New(), RequestID: "req-delete"}
	if err := vault.Delete(context.Background(), actor, actor.UserID, KindOAuthMicrosoftToken, "default"); err != nil {
		t.Fatal(err)
	}
	if repo.deleteActor != actor {
		t.Fatalf("delete actor not forwarded: %#v", repo.deleteActor)
	}
}
