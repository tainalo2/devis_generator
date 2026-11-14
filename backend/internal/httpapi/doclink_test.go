package httpapi

import (
	"crypto/tls"
	"net/http/httptest"
	"testing"
)

func TestRequestSchemeUsesForwardedProtoFromTLSProxy(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.test", nil)
	req.Header.Set("X-Forwarded-Proto", "https")

	if got := requestScheme(req); got != "https" {
		t.Fatalf("requestScheme() = %q, want https", got)
	}
}

func TestRequestSchemeFallsBackToDirectTLS(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.test", nil)
	req.TLS = &tls.ConnectionState{}

	if got := requestScheme(req); got != "https" {
		t.Fatalf("requestScheme() = %q, want https", got)
	}
}

func TestRequestSchemePrioritizesForwardedProtoOverTLS(t *testing.T) {
	req := httptest.NewRequest("GET", "https://example.test", nil)
	req.TLS = &tls.ConnectionState{}
	req.Header.Set("X-Forwarded-Proto", "http")

	if got := requestScheme(req); got != "http" {
		t.Fatalf("requestScheme() = %q, want http", got)
	}
}
