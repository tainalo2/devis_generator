package payments

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tainalo2/devis_generator/backend/internal/db"
)

func paymentTestDB(t *testing.T) (*pgxpool.Pool, context.Context, uuid.UUID, uuid.UUID) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	if err := db.RunMigrations(dsn); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	owner, doc := uuid.New(), uuid.New()
	email := fmt.Sprintf("payments-%s@example.test", owner)
	if _, err = pool.Exec(ctx, `INSERT INTO users(id,email,display_name) VALUES($1,$2,'payments test')`, owner, email); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO documents(id,user_id,kind,status,currency,total_cents) VALUES($1,$2,'invoice','sent','EUR',1000)`, doc, owner); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, owner) })
	return pool, ctx, owner, doc
}

func TestPostgresConfigAndSecretsRollbackTogether(t *testing.T) {
	pool, ctx, owner, _ := paymentTestDB(t)
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO payment_provider_configs(owner_id,provider,environment) VALUES($1,'stripe','sandbox')`, owner)
	if err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO payment_secrets(owner_id,kind,reference,key_id,ciphertext,masked_value) VALUES($1,'stripe_api_key','stripe',1,$2,'masked')`, owner, []byte{1, 2})
	if err != nil {
		t.Fatal(err)
	}
	tx.Rollback(ctx)
	var n int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM payment_provider_configs WHERE owner_id=$1`, owner).Scan(&n); err != nil || n != 0 {
		t.Fatalf("config survived rollback: n=%d err=%v", n, err)
	}
}

func TestPostgresCheckoutReservationSerializesConcurrency(t *testing.T) {
	pool, ctx, owner, doc := paymentTestDB(t)
	key := uuid.NewString()
	var wg sync.WaitGroup
	results := make(chan int64, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tag, err := pool.Exec(ctx, `INSERT INTO payment_attempts(owner_id,document_id,provider,idempotency_key,status) VALUES($1,$2,'stripe',$3,'creating') ON CONFLICT(provider,idempotency_key) DO NOTHING`, owner, doc, key)
			if err != nil {
				t.Error(err)
				return
			}
			results <- tag.RowsAffected()
		}()
	}
	wg.Wait()
	close(results)
	var total int64
	for n := range results {
		total += n
	}
	if total != 1 {
		t.Fatalf("reservations=%d want 1", total)
	}
}

func TestPostgresWebhookDedupeAndTenantIsolation(t *testing.T) {
	pool, ctx, owner, doc := paymentTestDB(t)
	var attempt uuid.UUID
	if err := pool.QueryRow(ctx, `INSERT INTO payment_attempts(owner_id,document_id,provider,external_id,idempotency_key,status) VALUES($1,$2,'stripe',$3,$4,'pending') RETURNING id`, owner, doc, uuid.NewString(), uuid.NewString()).Scan(&attempt); err != nil {
		t.Fatal(err)
	}
	event := uuid.NewString()
	for i := 0; i < 2; i++ {
		_, err := pool.Exec(ctx, `INSERT INTO payment_webhook_events(provider,event_id,owner_id,payment_attempt_id) VALUES('stripe',$1,$2,$3) ON CONFLICT(provider,event_id) DO NOTHING`, event, owner, attempt)
		if err != nil {
			t.Fatal(err)
		}
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM payment_webhook_events WHERE provider='stripe' AND event_id=$1 AND owner_id=$2`, event, owner).Scan(&n); err != nil || n != 1 {
		t.Fatalf("dedupe n=%d err=%v", n, err)
	}
	other := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO users(id,email,display_name) VALUES($1,$2,'other')`, other, fmt.Sprintf("other-%s@example.test", other)); err != nil {
		t.Fatal(err)
	}
	defer pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, other)
	if _, err := pool.Exec(ctx, `INSERT INTO payment_webhook_events(provider,event_id,owner_id,payment_attempt_id) VALUES('stripe',$1,$2,$3)`, uuid.NewString(), other, attempt); err == nil {
		t.Fatal("cross-tenant webhook accepted")
	}
}
