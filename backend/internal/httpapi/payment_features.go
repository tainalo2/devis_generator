package httpapi

import (
	"context"
	"net/http"

	"github.com/tainalo2/devis_generator/backend/internal/paymentflags"
)

type paymentFlagsStore interface {
	Get(context.Context) (paymentflags.Features, error)
	Set(context.Context, paymentflags.Features) (paymentflags.Features, error)
}

func (s *Server) handleGetPaymentFeatures(w http.ResponseWriter, r *http.Request) {
	if s.paymentFlags == nil {
		writeJSON(w, http.StatusOK, paymentflags.Features{})
		return
	}
	features, err := s.paymentFlags.Get(r.Context())
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Configuration des paiements indisponible.")
		return
	}
	writeJSON(w, http.StatusOK, features)
}

func (s *Server) handlePutPaymentFeatures(w http.ResponseWriter, r *http.Request) {
	if s.paymentFlags == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "Configuration des paiements indisponible.")
		return
	}
	var features paymentflags.Features
	if err := decodeJSON(r, &features); err != nil {
		writeAPIError(w, http.StatusBadRequest, "Requête invalide.")
		return
	}
	saved, err := s.paymentFlags.Set(r.Context(), features)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Mise à jour des paiements impossible.")
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

// requireAdmin conceals admin-only endpoints from non-admin accounts.
func (s *Server) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := UserFromContext(r.Context())
		if !ok || !user.IsAdmin {
			http.NotFound(w, r)
			return
		}
		next(w, r)
	}
}

// requirePaymentFeature enforces a feature flag server-side. Payment-mode
// routes must wrap their handlers with this guard when they are registered.
func (s *Server) requirePaymentFeature(mode paymentflags.Mode, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.paymentFlags == nil {
			http.NotFound(w, r)
			return
		}
		features, err := s.paymentFlags.Get(r.Context())
		if err != nil || !features.Enabled(mode) {
			http.NotFound(w, r)
			return
		}
		next(w, r)
	}
}
