package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tainalo2/devis_generator/backend/internal/legal"
)

func TestLegalStatusWithoutService(t *testing.T) {
	s := New(Options{Version: "test"})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/legal/status", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestLegalStatusAndSearchGuest(t *testing.T) {
	store := legal.NewMemStore()
	_ = store.Upsert(context.Background(), legal.Variable{
		Key: legal.KeyFlatIndemnity, Value: json.RawMessage(`{"cents":4000}`),
		Source: "manual-surveillance", FetchedAt: time.Now().UTC(), Status: legal.StatusManual,
	})

	searchSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"results":[{"siren":"123456789","nom_complet":"ACME","siege":{"siret":"12345678900011","code_postal":"75001","libelle_commune":"PARIS"}}]}`))
	}))
	defer searchSrv.Close()

	viesSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"valid":true,"name":"ACME SAS","address":"PARIS"}`))
	}))
	defer viesSrv.Close()

	recherche := legal.NewRechercheClient(searchSrv.Client())
	recherche.BaseURL = searchSrv.URL
	recherche.Limiter = legal.NewRateLimiter(time.Millisecond)

	vies := legal.NewVIESClient(viesSrv.Client())
	vies.URL = viesSrv.URL
	vies.Limiter = legal.NewRateLimiter(time.Millisecond)
	vies.Retry = legal.RetryPolicy{MaxAttempts: 1}

	svc := &legal.Service{Store: store, Recherche: recherche, VIES: vies}
	s := New(Options{Version: "test", Legal: svc})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/legal/status", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/legal/companies/search?q=acme", nil)
	rec = httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("search=%d %s", rec.Code, rec.Body.String())
	}
	var searchBody map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &searchBody)
	results, _ := searchBody["results"].([]any)
	if len(results) != 1 {
		t.Fatalf("results=%v", searchBody)
	}

	body := bytes.NewBufferString(`{"vat_number":"FR12345678901"}`)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/legal/vat/validate", body)
	req.Header.Set("Content-Type", "application/json")
	// CSRF: set matching cookies
	req.AddCookie(&http.Cookie{Name: "vibed_csrf", Value: "test-csrf"})
	req.Header.Set("X-CSRF-Token", "test-csrf")
	rec = httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("vat=%d %s", rec.Code, rec.Body.String())
	}
}

func TestCompanySearchRateLimit(t *testing.T) {
	store := legal.NewMemStore()
	svc := &legal.Service{
		Store: store,
		Recherche: &legal.RechercheClient{
			BaseURL: "http://127.0.0.1:1",
			Client:  http.DefaultClient,
			Limiter: legal.NewRateLimiter(time.Millisecond),
			Retry:   legal.RetryPolicy{MaxAttempts: 1},
		},
	}
	s := New(Options{Version: "test", Legal: svc})
	s.legalGuestLimiter = NewRateLimiter(2, time.Hour)

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/legal/companies/search?q=ab", nil)
		req.RemoteAddr = "9.9.9.9:1234"
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, req)
		// May be 502 (upstream) or 200 — not 429 yet
		if rec.Code == http.StatusTooManyRequests {
			t.Fatalf("unexpected 429 on attempt %d", i)
		}
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/legal/companies/search?q=ab", nil)
	req.RemoteAddr = "9.9.9.9:1234"
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("want 429, got %d", rec.Code)
	}
}
