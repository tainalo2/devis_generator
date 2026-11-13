package httpapi

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/tainalo2/devis_generator/backend/internal/gmailmail"
	"github.com/tainalo2/devis_generator/backend/internal/paymentsecrets"
)

func (s *Server) gmailRequest(r *http.Request) (paymentsecrets.Actor, bool) {
	user, ok := UserFromContext(r.Context())
	if !ok || s.gmailMail == nil {
		return paymentsecrets.Actor{}, false
	}
	return paymentsecrets.Actor{UserID: user.ID, RequestID: middleware.GetReqID(r.Context())}, true
}

func (s *Server) handleGmailAuthorize(w http.ResponseWriter, r *http.Request) {
	actor, ok := s.gmailRequest(r)
	if !ok {
		writeAPIError(w, http.StatusServiceUnavailable, "OAuth Gmail indisponible.")
		return
	}
	authorizationURL, err := s.gmailMail.AuthorizationURL(actor.UserID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Impossible de démarrer la connexion Gmail.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"authorization_url": authorizationURL})
}

func (s *Server) handleGmailCallback(w http.ResponseWriter, r *http.Request) {
	actor, ok := s.gmailRequest(r)
	if !ok {
		writeAPIError(w, http.StatusServiceUnavailable, "OAuth Gmail indisponible.")
		return
	}
	if err := s.gmailMail.HandleCallback(r.Context(), actor, actor.UserID, r.URL.Query().Get("state"), r.URL.Query().Get("code")); err != nil {
		if errors.Is(err, gmailmail.ErrInvalidState) {
			writeAPIError(w, http.StatusBadRequest, "État OAuth invalide ou expiré.")
			return
		}
		writeAPIError(w, http.StatusBadGateway, "Échec de la connexion Gmail.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "connected"})
}

func (s *Server) handleGmailStatus(w http.ResponseWriter, r *http.Request) {
	actor, ok := s.gmailRequest(r)
	if !ok {
		writeAPIError(w, http.StatusServiceUnavailable, "OAuth Gmail indisponible.")
		return
	}
	connected, err := s.gmailMail.Status(r.Context(), actor, actor.UserID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Impossible de vérifier la connexion Gmail.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"connected": connected})
}

func (s *Server) handleGmailSend(w http.ResponseWriter, r *http.Request) {
	actor, ok := s.gmailRequest(r)
	if !ok {
		writeAPIError(w, http.StatusServiceUnavailable, "OAuth Gmail indisponible.")
		return
	}
	var message gmailmail.Message
	if err := decodeJSON(r, &message); err != nil {
		writeAPIError(w, http.StatusBadRequest, "Requête invalide.")
		return
	}
	if err := s.gmailMail.Send(r.Context(), actor, actor.UserID, message); err != nil {
		if errors.Is(err, gmailmail.ErrNotConnected) {
			writeAPIError(w, http.StatusConflict, "Aucun compte Gmail connecté.")
			return
		}
		writeAPIError(w, http.StatusBadGateway, "Échec de l'envoi Gmail.")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "sent"})
}

func (s *Server) handleGmailDisconnect(w http.ResponseWriter, r *http.Request) {
	actor, ok := s.gmailRequest(r)
	if !ok {
		writeAPIError(w, http.StatusServiceUnavailable, "OAuth Gmail indisponible.")
		return
	}
	if err := s.gmailMail.Disconnect(r.Context(), actor, actor.UserID); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Échec de la déconnexion Gmail.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "disconnected"})
}
