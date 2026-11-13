package reminders

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tainalo2/devis_generator/backend/internal/db"
)

func TestPGStoreConcurrentDeliveryWorkflow(t *testing.T) {
	pool, ctx := newReminderIntegrationPool(t)
	store := NewPGStore(pool)
	owner, invoice, template := seedReminderFixture(t, ctx, pool, "primary")
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	t.Run("same scope and key has one delivery claim", func(t *testing.T) {
		const key = "concurrent-same-scope"
		start := make(chan struct{})
		type result struct {
			r       Reminder
			claimed bool
			err     error
		}
		results := make(chan result, 2)
		var ready sync.WaitGroup
		ready.Add(2)
		for range 2 {
			go func() {
				ready.Done()
				<-start
				r, claimed, err := store.Prepare(ctx, owner, invoice, template, key, "manual", now)
				results <- result{r, claimed, err}
			}()
		}
		ready.Wait()
		close(start)

		claims := 0
		var id uuid.UUID
		for range 2 {
			got := <-results
			if got.err != nil {
				t.Fatal(got.err)
			}
			if got.claimed {
				claims++
			}
			if id != uuid.Nil && got.r.ID != id {
				t.Fatalf("different reservations: %s and %s", id, got.r.ID)
			}
			id = got.r.ID
		}
		if claims != 1 {
			t.Fatalf("claims=%d, want 1", claims)
		}
		var count, attempts int
		var status string
		if err := pool.QueryRow(ctx, `SELECT count(*), min(status), min(attempt_count) FROM reminders WHERE user_id=$1 AND invoice_id=$2 AND template_id=$3 AND reminder_type='manual' AND idempotency_key=$4`, owner, invoice, template, key).Scan(&count, &status, &attempts); err != nil {
			t.Fatal(err)
		}
		if count != 1 || status != "sending" || attempts != 1 {
			t.Fatalf("count=%d status=%q attempts=%d", count, status, attempts)
		}
	})

	t.Run("same key is allowed across delivery scope", func(t *testing.T) {
		_, invoice2, template2 := seedReminderFixtureForOwner(t, ctx, pool, owner, "secondary")
		cases := []struct {
			invoice  uuid.UUID
			template uuid.UUID
			kind     string
		}{
			{invoice, template, "scheduled"},
			{invoice2, template, "manual"},
			{invoice, template2, "manual"},
		}
		for i, tc := range cases {
			r, claimed, err := store.Prepare(ctx, owner, tc.invoice, tc.template, "shared-across-scope", tc.kind, now.Add(time.Duration(i)*time.Second))
			if err != nil || !claimed || r.Status != StatusSending {
				t.Fatalf("case %d: reminder=%#v claimed=%v err=%v", i, r, claimed, err)
			}
		}
	})

	t.Run("cross tenant document and template are rejected", func(t *testing.T) {
		other, otherInvoice, otherTemplate := seedReminderFixture(t, ctx, pool, "other-tenant")
		_ = other
		for name, ids := range map[string]struct {
			inv  uuid.UUID
			tmpl uuid.UUID
		}{
			"document": {otherInvoice, template},
			"template": {invoice, otherTemplate},
		} {
			t.Run(name, func(t *testing.T) {
				_, claimed, err := store.Prepare(ctx, owner, ids.inv, ids.tmpl, "cross-tenant-"+name, "manual", now)
				if !errors.Is(err, ErrDocumentChanged) || claimed {
					t.Fatalf("claimed=%v err=%v, want ErrDocumentChanged", claimed, err)
				}
				_, err = pool.Exec(ctx, `INSERT INTO reminders(user_id,invoice_id,template_id,reminder_type,recipient,status,idempotency_key) VALUES($1,$2,$3,'manual','recipient@example.test','queued',$4)`, owner, ids.inv, ids.tmpl, "constraint-cross-tenant-"+name)
				if err == nil {
					t.Fatal("cross-tenant reminder insert unexpectedly satisfied tenant foreign keys")
				}
			})
		}
	})

	t.Run("expired lease is reclaimed and increments attempts", func(t *testing.T) {
		r, claimed, err := store.Prepare(ctx, owner, invoice, template, "expired-lease", "job", now)
		if err != nil || !claimed {
			t.Fatalf("initial claim=%v err=%v", claimed, err)
		}
		later := now.Add(6 * time.Minute)
		r, claimed, err = store.Prepare(ctx, owner, invoice, template, "expired-lease", "job", later)
		if err != nil || !claimed || r.Attempts != 2 || r.LeaseUntil == nil || !r.LeaseUntil.Equal(later.Add(5*time.Minute)) {
			t.Fatalf("reclaim=%#v claimed=%v err=%v", r, claimed, err)
		}
	})

	t.Run("unexpired lease is not reclaimed", func(t *testing.T) {
		r, claimed, err := store.Prepare(ctx, owner, invoice, template, "live-lease", "job", now)
		if err != nil || !claimed {
			t.Fatalf("initial claim=%v err=%v", claimed, err)
		}
		got, claimed, err := store.Prepare(ctx, owner, invoice, template, "live-lease", "job", now.Add(time.Minute))
		if err != nil || claimed || got.ID != r.ID || got.Attempts != 1 {
			t.Fatalf("second=%#v claimed=%v err=%v", got, claimed, err)
		}
	})

	t.Run("MarkSent increments document exactly once", func(t *testing.T) {
		r, claimed, err := store.Prepare(ctx, owner, invoice, template, "mark-sent-once", "manual", now)
		if err != nil || !claimed {
			t.Fatalf("prepare claim=%v err=%v", claimed, err)
		}
		sentAt := now.Add(time.Minute)
		got, err := store.MarkSent(ctx, owner, r.ID, sentAt)
		if err != nil || got.Status != StatusSent {
			t.Fatalf("MarkSent=%#v err=%v", got, err)
		}
		if _, err := store.MarkSent(ctx, owner, r.ID, sentAt.Add(time.Second)); !errors.Is(err, ErrNotFound) {
			t.Fatalf("second MarkSent err=%v, want ErrNotFound", err)
		}
		var count int
		var last time.Time
		if err := pool.QueryRow(ctx, `SELECT reminder_count,last_reminder_at FROM documents WHERE id=$1`, invoice).Scan(&count, &last); err != nil {
			t.Fatal(err)
		}
		if count != 1 || !last.Equal(sentAt) {
			t.Fatalf("reminder_count=%d last_reminder_at=%s", count, last)
		}
	})
}

func TestMigration000017UpDown(t *testing.T) {
	pool, ctx := newReminderIntegrationPool(t)
	for _, direction := range []string{"down", "up"} {
		path := filepath.Join("..", "db", "migrations", "000017_reminder_delivery_safeguards."+direction+".sql")
		sql, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("migration 000017 %s: %v", direction, err)
		}
	}
	var columns int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM information_schema.columns WHERE table_schema=current_schema() AND table_name='reminders' AND column_name IN ('lease_until','attempt_count')`).Scan(&columns); err != nil {
		t.Fatal(err)
	}
	if columns != 2 {
		t.Fatalf("safeguard columns=%d, want 2", columns)
	}
}

func newReminderIntegrationPool(t *testing.T) (*pgxpool.Pool, context.Context) {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set; PostgreSQL integration test skipped")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	t.Cleanup(cancel)

	admin, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	schema := "reminders_test_" + fmt.Sprintf("%x", uuid.New())
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
	isolatedURL := u.String()
	if err := db.RunMigrations(isolatedURL); err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.New(ctx, isolatedURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool, ctx
}

func seedReminderFixture(t *testing.T, ctx context.Context, pool *pgxpool.Pool, suffix string) (uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	owner := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO users(id,email,display_name) VALUES($1,$2,'integration')`, owner, suffix+"-"+owner.String()+"@example.test"); err != nil {
		t.Fatal(err)
	}
	return seedReminderFixtureForOwner(t, ctx, pool, owner, suffix)
}

func seedReminderFixtureForOwner(t *testing.T, ctx context.Context, pool *pgxpool.Pool, owner uuid.UUID, suffix string) (uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	invoice, template := uuid.New(), uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO documents(id,user_id,kind,status,client_snapshot) VALUES($1,$2,'invoice','sent',jsonb_build_object('email',$3))`, invoice, owner, suffix+"@example.test"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO reminder_templates(id,user_id,name,subject,html) VALUES($1,$2,$3,'subject','<p>body</p>')`, template, owner, suffix); err != nil {
		t.Fatal(err)
	}
	return owner, invoice, template
}
