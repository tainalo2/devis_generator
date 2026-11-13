package httpapi

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/tainalo2/devis_generator/backend/internal/paymentsecrets"
	"github.com/tainalo2/devis_generator/backend/internal/smtpconfig"
)

func (s *Server) handleGetSMTPConfig(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "Non authentifié.")
		return
	}
	if s.smtp == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "Service SMTP indisponible.")
		return
	}
	config, err := s.smtp.Get(r.Context(), user.ID)
	if errors.Is(err, smtpconfig.ErrNotFound) {
		writeAPIError(w, http.StatusNotFound, "Configuration SMTP introuvable.")
		return
	}
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"config": config})
}

func (s *Server) handleSaveSMTPConfig(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "Non authentifié.")
		return
	}
	if s.smtp == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "Service SMTP indisponible.")
		return
	}
	var input smtpconfig.Input
	if err := decodeJSON(r, &input); err != nil {
		writeAPIError(w, http.StatusBadRequest, "Requête invalide.")
		return
	}
	actor := paymentsecrets.Actor{UserID: user.ID, RequestID: middleware.GetReqID(r.Context())}
	config, err := s.smtp.Save(r.Context(), actor, user.ID, input)
	if errors.Is(err, smtpconfig.ErrInvalidConfig) {
		writeAPIError(w, http.StatusBadRequest, "Configuration SMTP invalide.")
		return
	}
	if errors.Is(err, smtpconfig.ErrSMTPTestFailed) {
		writeAPIError(w, http.StatusBadGateway, "Le test SMTP a échoué.")
		return
	}
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"config": config})
}

func (s *Server) handleDeleteSMTPConfig(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "Non authentifié.")
		return
	}
	if s.smtp == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "Service SMTP indisponible.")
		return
	}
	actor := paymentsecrets.Actor{UserID: user.ID, RequestID: middleware.GetReqID(r.Context())}
	if err := s.smtp.Delete(r.Context(), actor, user.ID); errors.Is(err, smtpconfig.ErrNotFound) {
		writeAPIError(w, http.StatusNotFound, "Configuration SMTP introuvable.")
		return
	} else if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
