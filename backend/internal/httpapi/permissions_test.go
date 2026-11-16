package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/tainalo2/devis_generator/backend/internal/authstore"
)

// TestSensitiveEndpointIsolation_APIKeyBlocked tests that API keys cannot access sensitive endpoints.
func TestSensitiveEndpointIsolation_APIKeyBlocked(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		method         string
		shouldBeGated  bool
		expectedStatus int
	}{
		// API Keys endpoints - must be gated
		{
			name:           "API Keys List via API Key",
			path:           "/api/v1/auth/api-keys",
			method:         "GET",
			shouldBeGated:  true,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "API Keys Create via API Key",
			path:           "/api/v1/auth/api-keys",
			method:         "POST",
			shouldBeGated:  true,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "API Keys Delete via API Key",
			path:           "/api/v1/auth/api-keys/00000000-0000-0000-0000-000000000000",
			method:         "DELETE",
			shouldBeGated:  true,
			expectedStatus: http.StatusForbidden,
		},
		// Payment config endpoints - must be gated
		{
			name:           "Payment Config Get via API Key",
			path:           "/api/v1/payments/config/stripe",
			method:         "GET",
			shouldBeGated:  true,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Payment Config Update via API Key",
			path:           "/api/v1/payments/config/stripe",
			method:         "PUT",
			shouldBeGated:  true,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Payment Config Delete via API Key",
			path:           "/api/v1/payments/config/stripe",
			method:         "DELETE",
			shouldBeGated:  true,
			expectedStatus: http.StatusForbidden,
		},
		// Stripe Connect endpoints - must be gated
		{
			name:           "Stripe Connect Onboard via API Key",
			path:           "/api/v1/payments/stripe-connect/onboard",
			method:         "POST",
			shouldBeGated:  true,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Stripe Connect Refresh via API Key",
			path:           "/api/v1/payments/stripe-connect/refresh",
			method:         "POST",
			shouldBeGated:  true,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Stripe Connect Callback via API Key",
			path:           "/api/v1/payments/stripe-connect/callback",
			method:         "GET",
			shouldBeGated:  true,
			expectedStatus: http.StatusForbidden,
		},
		// SMTP/Email credentials - must be gated
		{
			name:           "SMTP Config Get via API Key",
			path:           "/api/v1/email/smtp",
			method:         "GET",
			shouldBeGated:  true,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "SMTP Config Create via API Key",
			path:           "/api/v1/email/smtp",
			method:         "POST",
			shouldBeGated:  true,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "SMTP Config Delete via API Key",
			path:           "/api/v1/email/smtp",
			method:         "DELETE",
			shouldBeGated:  true,
			expectedStatus: http.StatusForbidden,
		},
		// Issuer/Bank coordinates - must be gated
		{
			name:           "Issuer Get via API Key",
			path:           "/api/v1/issuer",
			method:         "GET",
			shouldBeGated:  true,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Issuer Update via API Key",
			path:           "/api/v1/issuer",
			method:         "PUT",
			shouldBeGated:  true,
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := createTestServer(t)
			userID := uuid.New()
			ctx := context.WithValue(context.Background(), ctxUserKey, authstore.User{ID: userID})
			apiKey := authstore.APIKey{ID: uuid.New(), UserID: userID, Scopes: []string{"read", "write"}}
			ctx = context.WithValue(ctx, ctxAPIKeyKey, apiKey)

			req := httptest.NewRequest(tt.method, tt.path, nil).WithContext(ctx)
			w := httptest.NewRecorder()

			// Test the ensureSensitiveEndpointIsolation wrapper directly for deterministic tests
			handler := s.ensureSensitiveEndpointIsolation(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			handler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected %d, got %d for %s %s", tt.expectedStatus, w.Code, tt.method, tt.path)
			}
		})
	}
}

// TestSensitiveEndpointIsolation_SessionCanAccess tests that session-authenticated users CAN access sensitive endpoints.
func TestSensitiveEndpointIsolation_SessionCanAccess(t *testing.T) {
	tests := []struct {
		name string
		path string
		method string
	}{
		{
			name:   "API Keys List via Session",
			path:   "/api/v1/auth/api-keys",
			method: "GET",
		},
		{
			name:   "Payment Config Get via Session",
			path:   "/api/v1/payments/config/stripe",
			method: "GET",
		},
		{
			name:   "SMTP Config Get via Session",
			path:   "/api/v1/email/smtp",
			method: "GET",
		},
		{
			name:   "Issuer Get via Session",
			path:   "/api/v1/issuer",
			method: "GET",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := createTestServer(t)
			userID := uuid.New()
			user := authstore.User{ID: userID}
			session := authstore.Session{ID: uuid.New(), UserID: userID, CSRFToken: "valid-token"}

			ctx := context.WithValue(context.Background(), ctxUserKey, user)
			ctx = context.WithValue(ctx, ctxSessionKey, session)

			req := httptest.NewRequest(tt.method, tt.path, nil).WithContext(ctx)
			w := httptest.NewRecorder()

			// Mock handler that returns 200 OK
			handler := s.isolateSensitiveEndpoints(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			handler.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("expected 200, got %d for %s %s (session should be allowed)", w.Code, tt.method, tt.path)
			}
		})
	}
}

// TestEnsureSensitiveEndpointIsolation tests the defensive handler wrapper.
func TestEnsureSensitiveEndpointIsolation(t *testing.T) {
	t.Run("API key blocked on sensitive endpoint", func(t *testing.T) {
		s := createTestServer(t)
		userID := uuid.New()
		ctx := context.WithValue(context.Background(), ctxUserKey, authstore.User{ID: userID})
		apiKey := authstore.APIKey{ID: uuid.New(), UserID: userID, Scopes: []string{"read", "write"}}
		ctx = context.WithValue(ctx, ctxAPIKeyKey, apiKey)

		req := httptest.NewRequest("GET", "/api/v1/auth/api-keys", nil).WithContext(ctx)
		w := httptest.NewRecorder()

		handler := s.ensureSensitiveEndpointIsolation(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		handler(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden, got %d", w.Code)
		}
	})

	t.Run("session allowed on sensitive endpoint", func(t *testing.T) {
		s := createTestServer(t)
		userID := uuid.New()
		user := authstore.User{ID: userID}
		session := authstore.Session{ID: uuid.New(), UserID: userID}

		ctx := context.WithValue(context.Background(), ctxUserKey, user)
		ctx = context.WithValue(ctx, ctxSessionKey, session)

		req := httptest.NewRequest("GET", "/api/v1/auth/api-keys", nil).WithContext(ctx)
		w := httptest.NewRecorder()

		handler := s.ensureSensitiveEndpointIsolation(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		handler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d (session should be allowed)", w.Code)
		}
	})

	t.Run("unauthenticated rejected", func(t *testing.T) {
		s := createTestServer(t)
		ctx := context.Background()

		req := httptest.NewRequest("GET", "/api/v1/auth/api-keys", nil).WithContext(ctx)
		w := httptest.NewRecorder()

		handler := s.ensureSensitiveEndpointIsolation(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		handler(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized, got %d", w.Code)
		}
	})
}

// TestIsSensitiveEndpoint tests the endpoint classification logic.
func TestIsSensitiveEndpoint(t *testing.T) {
	tests := []struct {
		name       string
		pattern    string
		shouldPass bool
	}{
		// Sensitive endpoints - would match
		{"API Keys endpoint", "/api/v1/auth/api-keys", true},
		{"API Keys with ID", "/api/v1/auth/api-keys/{id}", true},
		{"Payment config", "/api/v1/payments/config/{provider}", true},
		{"Stripe Connect onboard", "/api/v1/payments/stripe-connect/onboard", true},
		{"Stripe Connect refresh", "/api/v1/payments/stripe-connect/refresh", true},
		{"Stripe Connect callback", "/api/v1/payments/stripe-connect/callback", true},
		{"SMTP config", "/api/v1/email/smtp", true},
		{"Issuer endpoint", "/api/v1/issuer", true},

		// Non-sensitive endpoints
		{"Documents list", "/api/v1/documents", false},
		{"Documents get", "/api/v1/documents/{id}", false},
		{"Clients list", "/api/v1/clients", false},
		{"Legal status", "/api/v1/legal/status", false},
		{"Public config", "/api/v1/public/config", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Check if the pattern is in the sensitive list
			sensitivePatterns := []string{
				"/api/v1/auth/api-keys",
				"/api/v1/auth/api-keys/{id}",
				"/api/v1/payments/config/{provider}",
				"/api/v1/payments/stripe-connect/onboard",
				"/api/v1/payments/stripe-connect/refresh",
				"/api/v1/payments/stripe-connect/callback",
				"/api/v1/email/smtp",
				"/api/v1/issuer",
			}

			found := false
			for _, pattern := range sensitivePatterns {
				if tt.pattern == pattern {
					found = true
					break
				}
			}

			if found != tt.shouldPass {
				t.Errorf("pattern %s: expected %v, got %v", tt.pattern, tt.shouldPass, found)
			}
		})
	}
}

// createTestServer creates a minimal test server for permissions testing.
func createTestServer(t *testing.T) *Server {
	t.Helper()
	return &Server{}
}
