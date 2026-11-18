package biz

import (
	"bytes"
	"context"
	"errors"
	"net/url"
	"os"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tainalo2/devis_generator/backend/internal/crypto"
	"github.com/tainalo2/devis_generator/backend/internal/db"
)

// newSignatureDatabase migrates a dedicated database per test. A schema is not enough here:
// migration 000003 looks for the 'issued' enum value in every schema, so a sibling schema
// (or the public one, migrated by other tests) makes it skip the value and issuing fails.
func newSignatureDatabase(t *testing.T) (*pgxpool.Pool, context.Context) {
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
	name := "biz_sig_" + uuid.NewString()[:8]
	if _, err := admin.Exec(ctx, `CREATE DATABASE `+name); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec(context.Background(), `DROP DATABASE IF EXISTS `+name+` WITH (FORCE)`)
		admin.Close()
	})
	u, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	u.Path = "/" + name
	if err := db.RunMigrations(u.String()); err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.New(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool, ctx
}

// signingFixture returns a store able to issue documents (encryption box) and a
// quote already issued by a fresh owner.
func signingFixture(t *testing.T, kind string) (*Store, *pgxpool.Pool, context.Context, uuid.UUID, Document) {
	t.Helper()
	pool, ctx := newSignatureDatabase(t)
	box, err := crypto.NewBox(bytes.Repeat([]byte{9}, 32))
	if err != nil {
		t.Fatal(err)
	}
	store := NewStore(pool, box)
	owner := seedIdempotencyUser(t, ctx, pool)
	if _, err := store.UpsertIssuer(ctx, owner, IssuerProfile{LegalName: "Atelier", SIRET: "12345678900012", Address: Address{Line1: "1 rue A", PostalCode: "75001", City: "Paris", Country: "FR"}}); err != nil {
		t.Fatal(err)
	}
	in := quoteInput(10000)
	in.Kind = kind
	draft, err := store.CreateDocument(ctx, owner, in)
	if err != nil {
		t.Fatal(err)
	}
	issued, err := store.IssueDocument(ctx, owner, draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	return store, pool, ctx, owner, issued
}

var hex64 = regexp.MustCompile(`^[0-9a-f]{64}$`)

func TestSignQuoteRecordsProofAndAcceptsQuote(t *testing.T) {
	store, _, ctx, owner, doc := signingFixture(t, "quote")
	sig, err := store.SignQuote(ctx, doc.ID, SignInput{Name: "Jeanne Client", IP: "203.0.113.7", UserAgent: "UA/1"})
	if err != nil {
		t.Fatal(err)
	}
	if !hex64.MatchString(sig.DocumentHash) || sig.SignerName != "Jeanne Client" || sig.SignerIP != "203.0.113.7" || sig.SignerUserAgent != "UA/1" || sig.SignedAt.IsZero() {
		t.Fatalf("preuve incomplète: %+v", sig)
	}
	got, err := store.GetDocument(ctx, owner, doc.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "accepted" || got.AcceptedAt == nil || got.SignedAt == nil {
		t.Fatalf("statut=%s accepted_at=%v signed_at=%v", got.Status, got.AcceptedAt, got.SignedAt)
	}
	list, err := store.ListDocuments(ctx, owner)
	if err != nil || len(list) != 1 || list[0].SignedAt == nil {
		t.Fatalf("liste sans signed_at: %v %+v", err, list)
	}
}

func TestSignQuoteTwiceIsRejectedAndKeepsFirstProof(t *testing.T) {
	store, pool, ctx, _, doc := signingFixture(t, "quote")
	first, err := store.SignQuote(ctx, doc.ID, SignInput{Name: "Premier Signataire"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.SignQuote(ctx, doc.ID, SignInput{Name: "Second Signataire"}); !errors.Is(err, ErrAlreadySigned) {
		t.Fatalf("err=%v, want ErrAlreadySigned", err)
	}
	var n int
	var name string
	if err := pool.QueryRow(ctx, `SELECT count(*), min(signer_name) FROM document_signatures WHERE document_id=$1`, doc.ID).Scan(&n, &name); err != nil {
		t.Fatal(err)
	}
	if n != 1 || name != first.SignerName {
		t.Fatalf("n=%d name=%q", n, name)
	}
}

func TestSignQuoteConcurrentOnlyOneWins(t *testing.T) {
	store, pool, ctx, _, doc := signingFixture(t, "quote")
	const workers = 8
	var wg sync.WaitGroup
	results := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := store.SignQuote(ctx, doc.ID, SignInput{Name: "Signataire Concurrent"})
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	ok, already := 0, 0
	for err := range results {
		switch {
		case err == nil:
			ok++
		case errors.Is(err, ErrAlreadySigned):
			already++
		default:
			t.Fatalf("erreur inattendue: %v", err)
		}
	}
	var n int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM document_signatures WHERE document_id=$1`, doc.ID).Scan(&n)
	if ok != 1 || already != workers-1 || n != 1 {
		t.Fatalf("ok=%d already=%d rows=%d", ok, already, n)
	}
}

func TestSignQuoteRejectsNonSignableDocuments(t *testing.T) {
	t.Run("facture", func(t *testing.T) {
		store, _, ctx, _, doc := signingFixture(t, "invoice")
		if _, err := store.SignQuote(ctx, doc.ID, SignInput{Name: "Jeanne Client"}); !errors.Is(err, ErrNotSignable) {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("brouillon", func(t *testing.T) {
		store, _, ctx, owner, _ := signingFixture(t, "quote")
		draft, err := store.CreateDocument(ctx, owner, quoteInput(500))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := store.SignQuote(ctx, draft.ID, SignInput{Name: "Jeanne Client"}); !errors.Is(err, ErrNotSignable) {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("devis refusé", func(t *testing.T) {
		store, _, ctx, owner, doc := signingFixture(t, "quote")
		if _, err := store.UpdateDocumentStatus(ctx, owner, doc.ID, "rejected"); err != nil {
			t.Fatal(err)
		}
		if _, err := store.SignQuote(ctx, doc.ID, SignInput{Name: "Jeanne Client"}); !errors.Is(err, ErrNotSignable) {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("devis remplacé", func(t *testing.T) {
		store, pool, ctx, owner, doc := signingFixture(t, "quote")
		repl, err := store.CreateDocument(ctx, owner, quoteInput(777))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `UPDATE documents SET replaces_document_id=$1 WHERE id=$2`, doc.ID, repl.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := store.SignQuote(ctx, doc.ID, SignInput{Name: "Jeanne Client"}); !errors.Is(err, ErrNotSignable) {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("document inconnu", func(t *testing.T) {
		store, _, ctx, _, _ := signingFixture(t, "quote")
		if _, err := store.SignQuote(ctx, uuid.New(), SignInput{Name: "Jeanne Client"}); !errors.Is(err, ErrNotFound) {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("nom invalide", func(t *testing.T) {
		store, _, ctx, _, doc := signingFixture(t, "quote")
		for _, name := range []string{"", " ", "A"} {
			if _, err := store.SignQuote(ctx, doc.ID, SignInput{Name: name}); !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("name=%q err=%v", name, err)
			}
		}
	})
}

func TestSignedQuoteCannotBeRejectedDeletedOrEdited(t *testing.T) {
	store, pool, ctx, owner, doc := signingFixture(t, "quote")
	if _, err := store.SignQuote(ctx, doc.ID, SignInput{Name: "Jeanne Client"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateDocumentStatus(ctx, owner, doc.ID, "rejected"); err == nil {
		t.Fatal("un devis signé ne peut pas être refusé")
	}
	if err := store.DeleteDocument(ctx, owner, doc.ID); !errors.Is(err, ErrImmutable) {
		t.Fatalf("delete err=%v", err)
	}
	if _, err := store.UpdateDocument(ctx, owner, doc.ID, quoteInput(1)); !errors.Is(err, ErrImmutable) {
		t.Fatalf("update err=%v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE document_signatures SET signer_name='Autre' WHERE document_id=$1`, doc.ID); err == nil {
		t.Fatal("la preuve doit être en ajout seul")
	}
}

func TestSignatureIntegrityDetectsTampering(t *testing.T) {
	store, pool, ctx, owner, doc := signingFixture(t, "quote")
	if _, err := store.SignQuote(ctx, doc.ID, SignInput{Name: "Jeanne Client"}); err != nil {
		t.Fatal(err)
	}
	proof, err := store.GetSignatureProof(ctx, owner, doc.ID)
	if err != nil || !proof.IntegrityOK {
		t.Fatalf("err=%v proof=%+v", err, proof)
	}
	if _, err := pool.Exec(ctx, `UPDATE document_lines SET unit_price_cents = unit_price_cents + 1 WHERE document_id=$1`, doc.ID); err != nil {
		t.Fatal(err)
	}
	proof, err = store.GetSignatureProof(ctx, owner, doc.ID)
	if err != nil || proof.IntegrityOK {
		t.Fatalf("falsification non détectée: err=%v proof=%+v", err, proof)
	}
}

func TestSignatureProofIsOwnerScoped(t *testing.T) {
	store, pool, ctx, _, doc := signingFixture(t, "quote")
	if _, err := store.SignQuote(ctx, doc.ID, SignInput{Name: "Jeanne Client"}); err != nil {
		t.Fatal(err)
	}
	other := seedIdempotencyUser(t, ctx, pool)
	if _, err := store.GetSignatureProof(ctx, other, doc.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("un autre compte ne doit pas lire la preuve: %v", err)
	}
}

func TestGetSignatureProofUnsignedIsNotFound(t *testing.T) {
	store, _, ctx, owner, doc := signingFixture(t, "quote")
	if _, err := store.GetSignatureProof(ctx, owner, doc.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err=%v", err)
	}
}
