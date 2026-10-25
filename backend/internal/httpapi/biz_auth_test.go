package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBusinessRoutesRequireAuth(t *testing.T) {
	s := New(Options{Version: "test"})
	paths := []string{
		"/api/v1/clients",
		"/api/v1/documents",
		"/api/v1/issuer",
	}
	for _, p := range paths {
		req := httptest.NewRequest(http.MethodGet, p, nil)
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s status=%d want 401", p, rec.Code)
		}
	}
}

func TestGuestPDFRejectsBadPayload(t *testing.T) {
	s := New(Options{Version: "test"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/guest/pdf", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rec.Code)
	}
}
