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
	if _, err := store.Upsert(ctx, want); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(ctx, ownerID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Host != want.Host || got.Port != want.Port || got.PasswordConfigured != true {
		t.Fatalf("got %#v", got)
	}
	if err := store.Delete(ctx, ownerID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, ownerID); err != ErrNotFound {
		t.Fatalf("after delete err=%v", err)
	}
}
