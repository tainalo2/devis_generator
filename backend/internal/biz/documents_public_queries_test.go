package biz

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tainalo2/devis_generator/backend/internal/crypto"
	"github.com/tainalo2/devis_generator/backend/internal/db"
)

// queryCounter is a pgx tracer that really counts the SQL statements sent to
// PostgreSQL through the pool.
type queryCounter struct{ n atomic.Int64 }

func (c *queryCounter) TraceQueryStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryStartData) context.Context {
	c.n.Add(1)
	return ctx
}

func (c *queryCounter) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

// TestGetDocumentPublicRunsSameSQLForEveryOutcome pins the repository-level
// part of the public-link oracle policy: GetDocumentPublic sends the very same
// number of SQL statements (documents + document_lines) whether the document is
// absent, an existing one of another kind (kind mismatch), an existing one that
// the caller will treat as expired, or valid. Kind/expiry are decided by the
// caller after the load, so at repository level they are all "existing".
func TestGetDocumentPublicRunsSameSQLForEveryOutcome(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := db.RunMigrations(databaseURL); err != nil {
		t.Fatal(err)
	}
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	counter := &queryCounter{}
	cfg.ConnConfig.Tracer = counter
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	box, err := crypto.NewBox(bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatal(err)
	}
	store := NewStore(pool, box)

	ownerID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO users (id, email, display_name) VALUES ($1, $2, 'doclink count')`, ownerID, fmt.Sprintf("count-%s@example.test", ownerID)); err != nil {
		t.Fatal(err)
	}
	defer pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, ownerID)

	issuerSnap, err := store.sealIssuerSnapshot(sampleIssuer())
	if err != nil {
		t.Fatal(err)
	}
	clientSnap, err := store.sealClientSnapshot(ClientSnapshot{Name: "Client Test", Address: Address{Line1: "2 rue X", City: "Lyon"}})
	if err != nil {
		t.Fatal(err)
	}
	insert := func(kind string) uuid.UUID {
		t.Helper()
		id := uuid.New()
		if _, err := pool.Exec(ctx, `
			INSERT INTO documents (id, user_id, kind, status, issuer_snapshot, client_snapshot)
			VALUES ($1, $2, $3::document_kind, 'draft', $4, $5)`, id, ownerID, kind, issuerSnap, clientSnap); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO document_lines (document_id, position, description) VALUES ($1, 0, 'ligne')`, id); err != nil {
			t.Fatal(err)
		}
		return id
	}

	cases := []struct {
		name     string
		id       uuid.UUID
		notFound bool
	}{
		{"absent", uuid.New(), true},
		{"existing quote (valid or expired)", insert("quote"), false},
		{"existing invoice (kind mismatch)", insert("invoice"), false},
	}
	want := int64(-1)
	for _, tc := range cases {
		before := counter.n.Load()
		_, _, err := store.GetDocumentPublic(ctx, tc.id)
		got := counter.n.Load() - before
		if tc.notFound != errors.Is(err, ErrNotFound) || (!tc.notFound && err != nil) {
			t.Fatalf("%s: unexpected error %v", tc.name, err)
		}
		if want < 0 {
			want = got
		}
		if got != want {
			t.Errorf("%s: %d SQL statements, want %d (same as the other outcomes)", tc.name, got, want)
		}
	}
	if want != 2 {
		t.Errorf("expected exactly 2 statements (documents + lines), got %d", want)
	}
}
