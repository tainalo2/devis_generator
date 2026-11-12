package smtpconfig

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tainalo2/devis_generator/backend/internal/db"
	"github.com/tainalo2/devis_generator/backend/internal/paymentsecrets"
)

func TestPGStoreRoundTripAndDelete(t *testing.T) {
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
	if _, err := pool.Exec(ctx, `INSERT INTO users (id, email, display_name) VALUES ($1, $2, 'smtp test')`, ownerID, fmt.Sprintf("smtp-%s@example.test", ownerID)); err != nil {
		t.Fatal(err)
	}
	defer pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, ownerID)

	store := NewPGStore(pool)
	want := Config{OwnerID: ownerID, Host: "smtp.example.com", Port: 465, Username: "mailer", FromAddress: "mailer@example.com", TLSMode: TLSModeTLS, PasswordConfigured: true}
	actor := paymentsecrets.Actor{UserID: ownerID, RequestID: "smtp-integration"}
	sealed := paymentsecrets.SealedSecret{Ciphertext: []byte{1, 2, 3}, MaskedValue: "••••"}
	if _, err := store.Save(ctx, actor, want, sealed); err != nil {
		t.Fatal(err)
	}
	rotated := want
	rotated.Host = "smtp-rotated.example.com"
	rotatedSecret := paymentsecrets.SealedSecret{Ciphertext: []byte{1, 9, 9}, MaskedValue: "•••• 9999"}
	if _, err := store.Save(ctx, actor, rotated, rotatedSecret); err != nil {
		t.Fatal(err)
	}
	var action paymentsecrets.AuditAction
	if err := pool.QueryRow(ctx, `SELECT action FROM payment_secret_audit WHERE owner_id=$1 ORDER BY id DESC LIMIT 1`, ownerID).Scan(&action); err != nil {
		t.Fatal(err)
	}
	if action != paymentsecrets.ActionUpdate {
		t.Fatalf("rotation audit action=%q", action)
	}

	failed := rotated
	failed.Host = "must-rollback.example.com"
	if _, err := store.Save(ctx, paymentsecrets.Actor{UserID: uuid.New()}, failed, paymentsecrets.SealedSecret{Ciphertext: []byte{1, 8, 8}, MaskedValue: "•••• 8888"}); err == nil {
		t.Fatal("expected audit foreign-key failure")
	}
	var ciphertext []byte
	if err := pool.QueryRow(ctx, `SELECT ciphertext FROM payment_secrets WHERE owner_id=$1 AND kind='smtp_password'`, ownerID).Scan(&ciphertext); err != nil {
		t.Fatal(err)
	}
	if string(ciphertext) != string(rotatedSecret.Ciphertext) {
		t.Fatalf("secret changed after rolled-back save: %v", ciphertext)
	}
	got, err := store.Get(ctx, ownerID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Host != rotated.Host || got.Port != rotated.Port || got.PasswordConfigured != true {
		t.Fatalf("got %#v", got)
	}
	if err := store.Delete(ctx, ownerID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, ownerID); err != ErrNotFound {
		t.Fatalf("after delete err=%v", err)
	}
}
