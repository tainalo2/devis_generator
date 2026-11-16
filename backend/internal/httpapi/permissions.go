package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// isSensitiveEndpoint returns true if this endpoint exposes sensitive account data (API keys, payment config, bank coords).
// A non-owner collaborator must never be able to access these endpoints, even if they have read access to documents.
func (s *Server) isSensitiveEndpoint(r *http.Request) bool {
	// Extract the route pattern for matching
	routeCtx := chi.RouteContext(r.Context())
	routePattern := routeCtx.RoutePattern()

	// Routes that expose sensitive account configuration
	sensitivePatterns := []string{
		// API keys management
		"/api/v1/auth/api-keys",
		"/api/v1/auth/api-keys/{id}",

		// Payment configuration - these contain API keys and secrets
		"/api/v1/payments/config/{provider}",

		// Stripe Connect - these contain connected account secrets
		"/api/v1/payments/stripe-connect/onboard",
		"/api/v1/payments/stripe-connect/refresh",
		"/api/v1/payments/stripe-connect/callback",

		// SMTP configuration - email credentials
		"/api/v1/email/smtp",

		// Bank coordinates - stored encrypted, but access should be owner-only
		// (These are accessed through the issuer endpoint which contains bank data)
		"/api/v1/issuer",
	}

	for _, pattern := range sensitivePatterns {
		if routePattern == pattern {
			return true
		}
	}
	return false
}

// requireOwner checks that the authenticated user is the account owner.
// For now, since there is no collaborator feature yet, every authenticated user is the owner.
// This middleware prepares the system for collaborator support: in the future, we'll check
// if the user has ownership rights on the account.
func (s *Server) requireOwner(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := UserFromContext(r.Context())
		if !ok {
			writeAPIError(w, http.StatusUnauthorized, "Non authentifié.")
			return
		}

		// TODO: When collaborators are implemented, check if user.ID == owner_id.
		// For now, any authenticated user is the owner of their own account.
		// This check will gate access to:
		// - API keys
		// - Payment configuration (Stripe, PayPal, etc.)
		// - Bank coordinates (RIB/IBAN)
		// - Stripe Connect credentials

		next.ServeHTTP(w, r)
	})
}

// requireSessionOwner ensures the user is accessing their own account via a session (not API key).
// API keys are limited to read/write scopes but cannot access owner-only endpoints.
func (s *Server) requireSessionOwner(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := UserFromContext(r.Context())
		if !ok {
			writeAPIError(w, http.StatusUnauthorized, "Non authentifié.")
			return
		}

		// If authenticated via API key, reject access to owner-only endpoints
		if _, ok := APIKeyFromContext(r.Context()); ok {
			writeAPIError(w, http.StatusForbidden, "Accès refusé : clés API ne peuvent pas gérer les configurations sensibles.")
			return
		}

		// If not via API key, we have a session - proceed
		if _, ok := SessionFromContext(r.Context()); ok {
			next.ServeHTTP(w, r)
			return
		}

		writeAPIError(w, http.StatusForbidden, "Session requise pour cette action.")
	})
}

// isolateSensitiveEndpoints is a middleware that enforces permission isolation on sensitive endpoints.
// It is applied to the entire /api/v1 route group and checks each request against the sensitive endpoints list.
func (s *Server) isolateSensitiveEndpoints(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only check authenticated requests
		_, ok := UserFromContext(r.Context())
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		// Check if this is a sensitive endpoint
		if s.isSensitiveEndpoint(r) {
			// Check if accessing via API key - API keys cannot access sensitive endpoints
			if _, ok := APIKeyFromContext(r.Context()); ok {
				writeAPIError(w, http.StatusForbidden, "Accès refusé : clés API ne peuvent pas accéder aux configurations sensibles.")
				return
			}

			// TODO: When collaborators are implemented, check if user is an owner.
			// For now, any authenticated session user can access owner endpoints on their own account.
			// This will be checked in the future:
			// if !userIsOwnerOfAccount(user.ID) {
			//     writeAPIError(w, http.StatusForbidden, "Accès refusé : seul le propriétaire du compte peut accéder à cette ressource.")
			//     return
			// }
		}

		next.ServeHTTP(w, r)
	})
}

// ensureSensitiveEndpointIsolation wraps a sensitive endpoint handler to ensure proper access control.
// This is a defensive measure - in addition to the middleware, we explicitly gate access in handlers.
func (s *Server) ensureSensitiveEndpointIsolation(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, ok := UserFromContext(r.Context())
		if !ok {
			writeAPIError(w, http.StatusUnauthorized, "Non authentifié.")
			return
		}

		// API keys cannot access sensitive endpoints
		if _, ok := APIKeyFromContext(r.Context()); ok {
			writeAPIError(w, http.StatusForbidden, "Accès refusé : clés API ne peuvent pas accéder aux configurations sensibles.")
			return
		}

		// TODO: Check if user is owner when collaborators are implemented
		// For now, any session-authenticated user can access their own account

		next(w, r)
	}
}
