package httpapi

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/tainalo2/devis_generator/backend/internal/doclink"
)

func TestPublicDocumentPageIsServedBySPA(t *testing.T) {
	staticDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(staticDir, "index.html"), []byte("<!doctype html><div id=\"root\">app</div>"), 0o600); err != nil {
		t.Fatal(err)
	}
	opts := testOptions()
	opts.StaticDir = staticDir
	srv := New(opts)

	req := httptest.NewRequest(http.MethodGet, "/d/signed-token", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `id="root"`) {
		t.Fatalf("body = %q, want SPA index", rec.Body.String())
	}
}

func TestPublicDocumentPDFRejectsInvalidToken(t *testing.T) {
	srv := New(testOptions())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/documents/not-a-token/pdf", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "lien invalide") {
		t.Fatalf("body = %q, want invalid-link error", rec.Body.String())
	}
}

func TestPublicDocumentPDFRejectsExpiredQuoteToken(t *testing.T) {
	opts := testOptions()
	token, err := doclink.Generate(opts.DocLinkSecret, uuid.NewString(), doclink.KindQuote, 1, time.Now().Add(-doclink.QuoteTTL-time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	srv := New(opts)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/documents/"+token+"/pdf", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusGone {
		t.Fatalf("status = %d, want 410", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "lien expire") {
		t.Fatalf("body = %q, want expired-link error", rec.Body.String())
	}
}
