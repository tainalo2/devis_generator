package paymentsecrets

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	appcrypto "github.com/tainalo2/devis_generator/backend/internal/crypto"
	"github.com/tainalo2/devis_generator/backend/internal/db"
)

func TestPGRepositoryEncryptsAtRestAndAuditsEveryAccess(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := db.RunMigrations(databaseURL); err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	ownerID := uuid.New()
	actorID := uuid.New()
	for _, id := range []uuid.UUID{ownerID, actorID} {
		if _, err := pool.Exec(ctx, `INSERT INTO users (id, email, display_name) VALUES ($1, $2, 'test')`, id, fmt.Sprintf("payment-secret-%s@example.test", id)); err != nil {
			t.Fatal(err)
		}
		defer pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, id)
	}

	box, err := appcrypto.NewBox(bytes.Repeat([]byte{3}, 32))
	if err != nil {
		t.Fatal(err)
	}
	repo := NewPGRepository(pool)
	vault := NewVault(repo, box)
	actor := Actor{UserID: actorID, RequestID: "integration-request"}
	plain := []byte("sk_test_super_secret_1234")

	metadata, err := vault.Put(ctx, actor, ownerID, KindBYOProviderAPIKey, "stripe", plain)
	if err != nil {
		t.Fatal(err)
	}
	var ciphertext []byte
	if err := pool.QueryRow(ctx, `SELECT ciphertext FROM payment_secrets WHERE id = $1`, metadata.ID).Scan(&ciphertext); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(ciphertext, plain) {
		t.Fatal("database ciphertext contains plaintext")
	}

	got, _, err := vault.Reveal(ctx, actor, ownerID, KindBYOProviderAPIKey, "stripe")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("got %q", got)
	}

	if err := vault.Delete(ctx, actor, ownerID, KindBYOProviderAPIKey, "stripe"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := vault.Reveal(ctx, actor, ownerID, KindBYOProviderAPIKey, "stripe"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("reveal after delete err = %v", err)
	}

	events, err := repo.ListAudit(ctx, ownerID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 || events[0].Action != ActionDelete || events[1].Action != ActionRead || events[2].Action != ActionCreate {
		t.Fatalf("unexpected audit events: %#v", events)
	}
	for _, event := range events {
		if event.ActorID == nil || *event.ActorID != actorID || event.ActorRequestID != actor.RequestID {
			t.Fatalf("audit actor/correlation missing: %#v", event)
		}
	}
}
