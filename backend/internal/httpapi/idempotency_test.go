package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tainalo2/devis_generator/backend/internal/authstore"
	"github.com/tainalo2/devis_generator/backend/internal/biz"
	"github.com/tainalo2/devis_generator/backend/internal/db"
)

func TestIdempotencyKeyHeaderValidation(t *testing.T) {
	cases := map[string]bool{
		"":                           true, // absent : comportement historique
		"abc-123_DEF.x:y":            true,
		strings.Repeat("a", 255):     true,
		strings.Repeat("a", 256):     false,
		"has space":                  false,
		"tab\t":                      false,
		"accentué":                   false,
		"new\nline":                  false,
		"ctrl\x01":                   false,
		"slash/ok":                   true,
		"  padded  ":                 false,
		"quote\"":                    false,
		"semi;colon":                 false,
		"unicode\u2028sep":           false,
		strings.Repeat("é", 10):      false,
		"0":                          true,
		"~!@#$%^&*()_+-={}[]|:,.<>?": false,
	}
	for key, valid := range cases {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		if key != "" {
			req.Header.Set("Idempotency-Key", key)
		}
		got, present, err := idempotencyKeyFromRequest(req)
		if key == "" {
			if err != nil || present {
				t.Fatalf("absent key: present=%v err=%v", present, err)
			}
			continue
		}
		if valid && (err != nil || !present || got != key) {
			t.Fatalf("key %q should be valid: got=%q present=%v err=%v", key, got, present, err)
		}
		if !valid && err == nil {
			t.Fatalf("key %q should be rejected", key)
		}
	}
}

func TestCreateDocumentRejectsInvalidIdempotencyKeyBeforeStore(t *testing.T) {
	s := &Server{} // biz nil : le rejet doit précéder tout accès au store
	req := httptest.NewRequest(http.MethodPost, "/documents", strings.NewReader(`{"kind":"quote","vat_regime":"standard"}`))
	req.Header.Set("Idempotency-Key", strings.Repeat("a", 300))
	req = req.WithContext(context.WithValue(req.Context(), ctxUserKey, authstore.User{ID: uuid.New()}))
	rec := httptest.NewRecorder()
	s.handleCreateDocument(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body)
	}
}

func newIdempotencyHTTPServer(t *testing.T) (*Server, *pgxpool.Pool) {
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
	schema := "httpapi_idem_test_" + fmt.Sprintf("%x", uuid.New())
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
	return &Server{biz: biz.NewStore(pool, nil)}, pool
}

func idempotencyUser(t *testing.T, pool *pgxpool.Pool) authstore.User {
	t.Helper()
	id := uuid.New()
	if _, err := pool.Exec(context.Background(), `INSERT INTO users(id,email,display_name) VALUES($1,$2,'idem')`, id, id.String()+"@example.test"); err != nil {
		t.Fatal(err)
	}
	return authstore.User{ID: id}
}

func postJSON(t *testing.T, h http.HandlerFunc, user authstore.User, path, key, body string, params map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	ctx := context.WithValue(req.Context(), ctxUserKey, user)
	if len(params) > 0 {
		rc := chi.NewRouteContext()
		for k, v := range params {
			rc.URLParams.Add(k, v)
		}
		ctx = context.WithValue(ctx, chi.RouteCtxKey, rc)
	}
	rec := httptest.NewRecorder()
	h(rec, req.WithContext(ctx))
	return rec
}

func docID(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var out struct {
		Document struct {
			ID string `json:"id"`
		} `json:"document"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil || out.Document.ID == "" {
		t.Fatalf("body=%s err=%v", rec.Body, err)
	}
	return out.Document.ID
}

const quoteBody = `{"kind":"quote","title":"Devis","vat_regime":"standard","currency":"EUR","lines":[{"description":"Conseil","quantity_milli":1000,"unit_price_cents":%d,"tax_rate_bps":2000}]}`

func TestCreateDocumentHTTPIdempotency(t *testing.T) {
	s, pool := newIdempotencyHTTPServer(t)
	a, b := idempotencyUser(t, pool), idempotencyUser(t, pool)
	body := fmt.Sprintf(quoteBody, 10000)

	first := postJSON(t, s.handleCreateDocument, a, "/documents", "k1", body, nil)
	if first.Code != http.StatusCreated || first.Header().Get("Idempotent-Replayed") != "" {
		t.Fatalf("first: %d %s", first.Code, first.Body)
	}
	retry := postJSON(t, s.handleCreateDocument, a, "/documents", "k1", body, nil)
	if retry.Code != http.StatusOK || retry.Header().Get("Idempotent-Replayed") != "true" || docID(t, retry) != docID(t, first) {
		t.Fatalf("retry: %d replayed=%q %s", retry.Code, retry.Header().Get("Idempotent-Replayed"), retry.Body)
	}
	conflict := postJSON(t, s.handleCreateDocument, a, "/documents", "k1", fmt.Sprintf(quoteBody, 20000), nil)
	if conflict.Code != http.StatusConflict {
		t.Fatalf("conflict: %d %s", conflict.Code, conflict.Body)
	}
	other := postJSON(t, s.handleCreateDocument, b, "/documents", "k1", body, nil)
	if other.Code != http.StatusCreated || docID(t, other) == docID(t, first) {
		t.Fatalf("tenant isolation: %d %s", other.Code, other.Body)
	}
	// Sans en-tête : création classique, non dédupliquée (comportement historique).
	n1 := postJSON(t, s.handleCreateDocument, a, "/documents", "", body, nil)
	n2 := postJSON(t, s.handleCreateDocument, a, "/documents", "", body, nil)
	if n1.Code != http.StatusCreated || n2.Code != http.StatusCreated || docID(t, n1) == docID(t, n2) {
		t.Fatalf("no key: %d/%d", n1.Code, n2.Code)
	}
	var docs int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM documents`).Scan(&docs); err != nil || docs != 4 {
		t.Fatalf("documents=%d err=%v", docs, err)
	}
}

func TestCreateDocumentHTTPConcurrentRetries(t *testing.T) {
	s, pool := newIdempotencyHTTPServer(t)
	user := idempotencyUser(t, pool)
	body := fmt.Sprintf(quoteBody, 10000)
	const workers = 10
	ids := make([]string, workers)
	codes := make([]int, workers)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			rec := postJSON(t, s.handleCreateDocument, user, "/documents", "race", body, nil)
			codes[i] = rec.Code
			if rec.Code < 300 {
				var out struct {
					Document struct{ ID string } `json:"document"`
				}
				_ = json.Unmarshal(rec.Body.Bytes(), &out)
				ids[i] = out.Document.ID
			}
		}(i)
	}
	close(start)
	wg.Wait()
	created := 0
	for i := range ids {
		if codes[i] == http.StatusCreated {
			created++
		} else if codes[i] != http.StatusOK {
			t.Fatalf("worker %d status=%d", i, codes[i])
		}
		if ids[i] == "" || ids[i] != ids[0] {
			t.Fatalf("worker %d id=%q want %q", i, ids[i], ids[0])
		}
	}
	if created != 1 {
		t.Fatalf("created=%d, want 1", created)
	}
}

func TestConvertAndDepositHTTPIdempotency(t *testing.T) {
	s, pool := newIdempotencyHTTPServer(t)
	user := idempotencyUser(t, pool)
	quote := postJSON(t, s.handleCreateDocument, user, "/documents", "", fmt.Sprintf(quoteBody, 10000), nil)
	id := docID(t, quote)
	params := map[string]string{"id": id}

	c1 := postJSON(t, s.handleConvertQuote, user, "/convert", "conv", "", params)
	c2 := postJSON(t, s.handleConvertQuote, user, "/convert", "conv", "", params)
	if c1.Code != http.StatusCreated || c2.Code != http.StatusOK || docID(t, c1) != docID(t, c2) {
		t.Fatalf("convert: %d %s / %d %s", c1.Code, c1.Body, c2.Code, c2.Body)
	}
	dep := `{"lines":[{"description":"Acompte","quantity_milli":1000,"unit_price_cents":3000,"tax_rate_bps":2000}]}`
	handler := func(w http.ResponseWriter, r *http.Request) { s.handleCreateDeposit(w, r) }
	d1 := postJSON(t, handler, user, "/deposit", "dep", dep, params)
	d2 := postJSON(t, handler, user, "/deposit", "dep", dep, params)
	if d1.Code != http.StatusCreated || d2.Code != http.StatusOK || docID(t, d1) != docID(t, d2) {
		t.Fatalf("deposit: %d %s / %d %s", d1.Code, d1.Body, d2.Code, d2.Body)
	}
	changed := strings.Replace(dep, "3000", "4000", 1)
	if d3 := postJSON(t, handler, user, "/deposit", "dep", changed, params); d3.Code != http.StatusConflict {
		t.Fatalf("deposit conflict: %d %s", d3.Code, d3.Body)
	}
	var docs int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM documents`).Scan(&docs); err != nil || docs != 3 {
		t.Fatalf("documents=%d err=%v", docs, err)
	}
}

func TestCreateDocumentHTTPCanonicalReplayAndDeletedTombstone(t *testing.T) {
	s, pool := newIdempotencyHTTPServer(t)
	user := idempotencyUser(t, pool)
	withEUR := fmt.Sprintf(quoteBody, 10000)
	// Même document métier : devise omise, ordre de clés différent, champs dérivés présents.
	omitted := `{"lines":[{"total_cents":1,"position":9,"tax_rate_bps":2000,"unit_price_cents":10000,"quantity_milli":1000,"description":"Conseil"}],"vat_regime":"standard","title":"Devis","kind":"quote"}`

	first := postJSON(t, s.handleCreateDocument, user, "/documents", "canon", withEUR, nil)
	if first.Code != http.StatusCreated {
		t.Fatalf("first: %d %s", first.Code, first.Body)
	}
	retry := postJSON(t, s.handleCreateDocument, user, "/documents", "canon", omitted, nil)
	if retry.Code != http.StatusOK || retry.Header().Get("Idempotent-Replayed") != "true" || docID(t, retry) != docID(t, first) {
		t.Fatalf("devise omise == EUR doit rejouer: %d %s", retry.Code, retry.Body)
	}

	id := docID(t, first)
	del := postJSON(t, s.handleDeleteDocument, user, "/documents/"+id, "", "", map[string]string{"id": id})
	if del.Code >= 300 {
		t.Fatalf("delete: %d %s", del.Code, del.Body)
	}
	replay := postJSON(t, s.handleCreateDocument, user, "/documents", "canon", withEUR, nil)
	if replay.Code != http.StatusConflict || !strings.Contains(replay.Body.String(), "nouvelle Idempotency-Key") {
		t.Fatalf("rejeu après suppression: %d %s", replay.Code, replay.Body)
	}
	var docs int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM documents`).Scan(&docs); err != nil || docs != 0 {
		t.Fatalf("aucun document recréé: documents=%d err=%v", docs, err)
	}
}
