package httpapi

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/tainalo2/devis_generator/backend/internal/microsoftmail"
	"github.com/tainalo2/devis_generator/backend/internal/paymentsecrets"
)

func (s *Server) microsoftRequest(r *http.Request) (paymentsecrets.Actor, bool) {
	user, ok := UserFromContext(r.Context())
	if !ok || s.microsoftMail == nil {
		return paymentsecrets.Actor{}, false
	}
	return paymentsecrets.Actor{UserID: user.ID, RequestID: middleware.GetReqID(r.Context())}, true
}

func (s *Server) handleMicrosoftAuthorize(w http.ResponseWriter, r *http.Request) {
	actor, ok := s.microsoftRequest(r)
	if !ok {
		writeAPIError(w, http.StatusServiceUnavailable, "Microsoft OAuth indisponible.")
		return
	}
	authorizationURL, err := s.microsoftMail.AuthorizationURL(actor.UserID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Impossible de démarrer la connexion Microsoft.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"authorization_url": authorizationURL})
}

func (s *Server) handleMicrosoftCallback(w http.ResponseWriter, r *http.Request) {
	actor, ok := s.microsoftRequest(r)
	if !ok {
		writeAPIError(w, http.StatusServiceUnavailable, "Microsoft OAuth indisponible.")
		return
	}
	if err := s.microsoftMail.HandleCallback(r.Context(), actor, actor.UserID, r.URL.Query().Get("state"), r.URL.Query().Get("code")); err != nil {
		if errors.Is(err, microsoftmail.ErrInvalidState) {
			writeAPIError(w, http.StatusBadRequest, "État OAuth invalide ou expiré.")
			return
		}
		writeAPIError(w, http.StatusBadGateway, "Échec de la connexion Microsoft.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "connected"})
}

func (s *Server) handleMicrosoftStatus(w http.ResponseWriter, r *http.Request) {
	actor, ok := s.microsoftRequest(r)
	if !ok {
		writeAPIError(w, http.StatusServiceUnavailable, "Microsoft OAuth indisponible.")
		return
	}
	connected, err := s.microsoftMail.Status(r.Context(), actor, actor.UserID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Impossible de vérifier la connexion Microsoft.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"connected": connected})
}

func (s *Server) handleMicrosoftSend(w http.ResponseWriter, r *http.Request) {
	actor, ok := s.microsoftRequest(r)
	if !ok {
		writeAPIError(w, http.StatusServiceUnavailable, "Microsoft OAuth indisponible.")
		return
	}
	var message microsoftmail.Message
	if err := decodeJSON(r, &message); err != nil {
		writeAPIError(w, http.StatusBadRequest, "Requête invalide.")
		return
	}
	if err := s.microsoftMail.Send(r.Context(), actor, actor.UserID, message); err != nil {
		if errors.Is(err, microsoftmail.ErrNotConnected) {
			writeAPIError(w, http.StatusConflict, "Aucune boîte Microsoft connectée.")
			return
		}
		writeAPIError(w, http.StatusBadGateway, "Échec de l'envoi Microsoft.")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "sent"})
}

func (s *Server) handleMicrosoftDisconnect(w http.ResponseWriter, r *http.Request) {
	actor, ok := s.microsoftRequest(r)
	if !ok {
		writeAPIError(w, http.StatusServiceUnavailable, "Microsoft OAuth indisponible.")
		return
	}
	if err := s.microsoftMail.Disconnect(r.Context(), actor, actor.UserID); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Échec de la déconnexion Microsoft.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "disconnected"})
}
