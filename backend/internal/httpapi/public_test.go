package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPublicConfig(t *testing.T) {
	opts := testOptions()
	opts.Version = "test"
	opts.PublicConfig = &PublicSiteConfig{
		DonateURL:          "https://example.org/dons",
		LegalPublisherName: "",
	}
	s := New(opts)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/config", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	var body PublicSiteConfig
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.DonateURL != "https://example.org/dons" {
		t.Fatalf("donate=%q", body.DonateURL)
	}
	if body.LegalPublisherName != "" {
		t.Fatalf("publisher should stay empty when unset")
	}
}
