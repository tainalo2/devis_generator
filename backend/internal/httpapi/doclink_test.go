package httpapi

import (
	"net/http/httptest"
	"testing"
)

func TestRequestSchemeUsesForwardedHTTPSWithoutTLS(t *testing.T) {
	req := httptest.NewRequest("GET", "http://app.example.test/api/v1/documents/id/link", nil)
	req.Header.Set("X-Forwarded-Proto", "https")

	if req.TLS != nil {
		t.Fatal("test setup: TLS must be nil behind a TLS-terminating proxy")
	}
	if got := requestScheme(req); got != "https" {
		t.Fatalf("requestScheme() = %q, want https", got)
	}
}
