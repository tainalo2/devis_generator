package biz

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tainalo2/devis_generator/backend/internal/db"
)

func newIdempotencyStore(t *testing.T) (*Store, *pgxpool.Pool, context.Context) {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set; PostgreSQL integration test skipped")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	admin, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	schema := "biz_idem_test_" + fmt.Sprintf("%x", uuid.New())
	if _, err := admin.Exec(ctx, `CREATE SCHEMA `+schema); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec(context.Background(), `DROP SCHEMA IF EXISTS `+schema+` CASCADE`)
		admin.Close()
	})
	u, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	if err := db.RunMigrations(u.String()); err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.New(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return NewStore(pool, nil), pool, ctx
}

func seedIdempotencyUser(t *testing.T, ctx context.Context, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO users(id,email,display_name) VALUES($1,$2,'idem')`, id, id.String()+"@example.test"); err != nil {
		t.Fatal(err)
	}
	return id
}

func quoteInput(price int64) DocumentInput {
	return DocumentInput{
		Kind: "quote", Title: "Devis", VATRegime: "standard", Currency: "EUR",
		Lines: []DocLine{{Description: "Conseil", QuantityMilli: 1000, UnitPriceCents: price, TaxRateBps: 2000}},
	}
}

func idem(t *testing.T, op, key string, in DocumentInput) Idempotency {
	t.Helper()
	i, err := NewIdempotency(op, key, in)
	if err != nil {
		t.Fatal(err)
	}
	return i
}

func countDocuments(t *testing.T, ctx context.Context, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM documents`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestIdempotentCreateReplaysIdenticalRetry(t *testing.T) {
	store, pool, ctx := newIdempotencyStore(t)
	owner := seedIdempotencyUser(t, ctx, pool)
	in := quoteInput(10000)
	first, replayed, err := store.CreateDocumentIdempotent(ctx, owner, in, idem(t, OpCreateDocument, "k1", in))
	if err != nil || replayed {
		t.Fatalf("first: replayed=%v err=%v", replayed, err)
	}
	second, replayed, err := store.CreateDocumentIdempotent(ctx, owner, in, idem(t, OpCreateDocument, "k1", in))
	if err != nil || !replayed {
		t.Fatalf("retry: replayed=%v err=%v", replayed, err)
	}
	if first.ID != second.ID || first.TotalCents != second.TotalCents {
		t.Fatalf("retry returned a different document: %v vs %v", first.ID, second.ID)
	}
	if n := countDocuments(t, ctx, pool); n != 1 {
		t.Fatalf("documents=%d, want 1", n)
	}
}

func TestIdempotentCreateRejectsDifferentPayload(t *testing.T) {
	store, pool, ctx := newIdempotencyStore(t)
	owner := seedIdempotencyUser(t, ctx, pool)
	in := quoteInput(10000)
	if _, _, err := store.CreateDocumentIdempotent(ctx, owner, in, idem(t, OpCreateDocument, "k1", in)); err != nil {
		t.Fatal(err)
	}
	other := quoteInput(20000)
	_, _, err := store.CreateDocumentIdempotent(ctx, owner, other, idem(t, OpCreateDocument, "k1", other))
	if !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("err=%v, want ErrIdempotencyConflict", err)
	}
	if n := countDocuments(t, ctx, pool); n != 1 {
		t.Fatalf("documents=%d, want 1", n)
	}
}

func TestIdempotencyIsScopedByTenantAndOperation(t *testing.T) {
	store, pool, ctx := newIdempotencyStore(t)
	a, b := seedIdempotencyUser(t, ctx, pool), seedIdempotencyUser(t, ctx, pool)
	in := quoteInput(10000)
	da, _, err := store.CreateDocumentIdempotent(ctx, a, in, idem(t, OpCreateDocument, "shared", in))
	if err != nil {
		t.Fatal(err)
	}
	db, replayed, err := store.CreateDocumentIdempotent(ctx, b, in, idem(t, OpCreateDocument, "shared", in))
	if err != nil || replayed || db.ID == da.ID {
		t.Fatalf("tenant B must not replay tenant A: replayed=%v err=%v same=%v", replayed, err, db.ID == da.ID)
	}
	// Tenant B with a different payload under the same key is NOT a conflict with A.
	other := quoteInput(1)
	if _, _, err := store.CreateDocumentIdempotent(ctx, b, other, idem(t, OpCreateDocument, "other", other)); err != nil {
		t.Fatal(err)
	}
	// Same key, other operation: independent namespace.
	dop, replayed, err := store.CreateDocumentIdempotent(ctx, a, in, idem(t, "other_operation", "shared", in))
	if err != nil || replayed || dop.ID == da.ID {
		t.Fatalf("operation scope: replayed=%v err=%v", replayed, err)
	}
	if n := countDocuments(t, ctx, pool); n != 4 {
		t.Fatalf("documents=%d, want 4", n)
	}
}

func TestIdempotentCreateSerializesConcurrentSameKey(t *testing.T) {
	store, pool, ctx := newIdempotencyStore(t)
	owner := seedIdempotencyUser(t, ctx, pool)
	in := quoteInput(10000)
	const workers = 12
	type result struct {
		id       uuid.UUID
		replayed bool
		err      error
	}
	results := make([]result, workers)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			d, replayed, err := store.CreateDocumentIdempotent(ctx, owner, in, idem(t, OpCreateDocument, "race", in))
			results[i] = result{d.ID, replayed, err}
		}(i)
	}
	close(start)
	wg.Wait()
	created := 0
	for i, r := range results {
		if r.err != nil {
			t.Fatalf("worker %d: %v", i, r.err)
		}
		if !r.replayed {
			created++
		}
		if r.id != results[0].id {
			t.Fatalf("worker %d got a different document", i)
		}
	}
	if created != 1 {
		t.Fatalf("created=%d, want exactly 1", created)
	}
	if n := countDocuments(t, ctx, pool); n != 1 {
		t.Fatalf("documents=%d, want 1", n)
	}
}

func TestIdempotentCreateConcurrentDifferentPayloadsConflict(t *testing.T) {
	store, pool, ctx := newIdempotencyStore(t)
	owner := seedIdempotencyUser(t, ctx, pool)
	const workers = 8
	errs := make([]error, workers)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			in := quoteInput(int64(1000 + i))
			<-start
			_, _, errs[i] = store.CreateDocumentIdempotent(ctx, owner, in, idem(t, OpCreateDocument, "race", in))
		}(i)
	}
	close(start)
	wg.Wait()
	ok, conflicts := 0, 0
	for _, err := range errs {
		switch {
		case err == nil:
			ok++
		case errors.Is(err, ErrIdempotencyConflict):
			conflicts++
		default:
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if ok != 1 || conflicts != workers-1 {
		t.Fatalf("ok=%d conflicts=%d", ok, conflicts)
	}
	if n := countDocuments(t, ctx, pool); n != 1 {
		t.Fatalf("documents=%d, want 1", n)
	}
}

func TestIdempotencyKeyIsNotBurnedByFailedCreation(t *testing.T) {
	store, pool, ctx := newIdempotencyStore(t)
	owner := seedIdempotencyUser(t, ctx, pool)
	bad := quoteInput(10000)
	bad.Lines[0].TaxRateBps = 1234 // taux non autorisé : échec dans la transaction de calcul
	if _, _, err := store.CreateDocumentIdempotent(ctx, owner, bad, idem(t, OpCreateDocument, "k", bad)); err == nil {
		t.Fatal("expected invalid input error")
	}
	var keys int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM document_idempotency_keys`).Scan(&keys); err != nil || keys != 0 {
		t.Fatalf("keys=%d err=%v", keys, err)
	}
	good := quoteInput(10000)
	if _, replayed, err := store.CreateDocumentIdempotent(ctx, owner, good, idem(t, OpCreateDocument, "k", good)); err != nil || replayed {
		t.Fatalf("replayed=%v err=%v", replayed, err)
	}
}

func TestIdempotencyKeysExpireAndArePurged(t *testing.T) {
	store, pool, ctx := newIdempotencyStore(t)
	owner := seedIdempotencyUser(t, ctx, pool)
	in := quoteInput(10000)
	first, _, err := store.CreateDocumentIdempotent(ctx, owner, in, idem(t, OpCreateDocument, "old", in))
	if err != nil {
		t.Fatal(err)
	}
	var ttl time.Duration
	if err := pool.QueryRow(ctx, `SELECT expires_at - created_at FROM document_idempotency_keys`).Scan(&ttl); err != nil {
		t.Fatal(err)
	}
	if ttl < IdempotencyTTL-time.Second || ttl > IdempotencyTTL+time.Second {
		t.Fatalf("ttl=%v, want %v", ttl, IdempotencyTTL)
	}
	if _, err := pool.Exec(ctx, `UPDATE document_idempotency_keys SET expires_at = NOW() - INTERVAL '1 second'`); err != nil {
		t.Fatal(err)
	}
	// Une clé expirée n'est plus rejouée : nouvelle création, même avec un payload différent.
	other := quoteInput(5)
	second, replayed, err := store.CreateDocumentIdempotent(ctx, owner, other, idem(t, OpCreateDocument, "old", other))
	if err != nil || replayed || second.ID == first.ID {
		t.Fatalf("expired key must allow a new creation: replayed=%v err=%v", replayed, err)
	}
	// Purge explicite.
	if _, err := pool.Exec(ctx, `UPDATE document_idempotency_keys SET expires_at = NOW() - INTERVAL '1 hour'`); err != nil {
		t.Fatal(err)
	}
	n, err := store.PurgeExpiredIdempotency(ctx)
	if err != nil || n != 1 {
		t.Fatalf("purged=%d err=%v", n, err)
	}
	var left int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM document_idempotency_keys`).Scan(&left); err != nil || left != 0 {
		t.Fatalf("left=%d err=%v", left, err)
	}
}

func TestIdempotentConvertAndLinkedCreationsAreReplayed(t *testing.T) {
	store, pool, ctx := newIdempotencyStore(t)
	owner := seedIdempotencyUser(t, ctx, pool)
	quote, err := store.CreateDocument(ctx, owner, quoteInput(10000))
	if err != nil {
		t.Fatal(err)
	}
	conv := func(key string) (Document, bool, error) {
		i, err := NewIdempotency(OpConvertQuote, key, map[string]any{"quote_id": quote.ID})
		if err != nil {
			t.Fatal(err)
		}
		return store.ConvertQuoteToInvoiceIdempotent(ctx, owner, quote.ID, i)
	}
	inv1, r1, err := conv("c1")
	if err != nil || r1 {
		t.Fatalf("convert: replayed=%v err=%v", r1, err)
	}
	inv2, r2, err := conv("c1")
	if err != nil || !r2 || inv1.ID != inv2.ID {
		t.Fatalf("convert retry: replayed=%v err=%v", r2, err)
	}
	lines := []DocLine{{Description: "Acompte", QuantityMilli: 1000, UnitPriceCents: 3000, TaxRateBps: 2000}}
	dep := func(key string, l []DocLine) (Document, bool, error) {
		i, err := NewIdempotency(OpCreateDeposit, key, map[string]any{"origin": quote.ID, "lines": l})
		if err != nil {
			t.Fatal(err)
		}
		return store.CreateLinkedDocumentIdempotent(ctx, owner, quote.ID, "deposit_invoice", l, i)
	}
	d1, _, err := dep("d1", lines)
	if err != nil {
		t.Fatal(err)
	}
	d2, replayed, err := dep("d1", lines)
	if err != nil || !replayed || d1.ID != d2.ID {
		t.Fatalf("deposit retry: replayed=%v err=%v", replayed, err)
	}
	changed := []DocLine{{Description: "Acompte", QuantityMilli: 1000, UnitPriceCents: 9999, TaxRateBps: 2000}}
	if _, _, err := dep("d1", changed); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("err=%v, want conflict", err)
	}
	if n := countDocuments(t, ctx, pool); n != 3 { // devis + facture + acompte
		t.Fatalf("documents=%d, want 3", n)
	}
}

func TestIdempotencyMigrationDownThenUp(t *testing.T) {
	_, pool, ctx := newIdempotencyStore(t)
	down, err := os.ReadFile("../db/migrations/000022_document_creation_idempotency.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	up, err := os.ReadFile("../db/migrations/000022_document_creation_idempotency.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range []struct {
		name string
		sql  []byte
	}{{"down", down}, {"up", up}} {
		if _, err := pool.Exec(ctx, string(step.sql)); err != nil {
			t.Fatalf("%s: %v", step.name, err)
		}
	}
	var exists bool
	if err := pool.QueryRow(ctx, `SELECT to_regclass('document_idempotency_keys') IS NOT NULL`).Scan(&exists); err != nil || !exists {
		t.Fatalf("table missing after up: %v", err)
	}
}
