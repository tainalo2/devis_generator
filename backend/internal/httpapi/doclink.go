package httpapi

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/tainalo2/devis_generator/backend/internal/biz"
	"github.com/tainalo2/devis_generator/backend/internal/doclink"
)

func (s *Server) handleCreateDocumentLink(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "Non authentifié.")
		return
	}
	id, err := parseID(r)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "Identifiant invalide.")
		return
	}
	document, err := s.biz.GetDocument(r.Context(), user.ID, id)
	if errors.Is(err, biz.ErrNotFound) {
		writeAPIError(w, http.StatusNotFound, "Document introuvable.")
		return
	}
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}
	token, err := doclink.Generate(s.docLinkSecret, document.ID.String(), document.Kind, 1, document.CreatedAt)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Lien indisponible.")
		return
	}
	scheme := requestScheme(r)
	link := (&url.URL{Scheme: scheme, Host: r.Host, Path: "/d/" + token}).String()
	response := map[string]any{"link": link, "permanent": document.Kind == doclink.KindInvoice || document.Kind == doclink.KindDeposit}
	if document.Kind == doclink.KindQuote {
		response["expires_at"] = document.CreatedAt.Add(doclink.QuoteTTL).UTC().Format(time.RFC3339)
	}
	writeJSON(w, http.StatusOK, response)
}

func requestScheme(r *http.Request) string {
	forwardedProto := strings.ToLower(strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Proto"), ",")[0]))
	if forwardedProto == "http" || forwardedProto == "https" {
		return forwardedProto
	}
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

func (s *Server) handleDocumentLink(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	p, err := doclink.Verify(s.docLinkSecret, token)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "lien invalide"})
		return
	}
	if doclink.Expired(p, time.Now()) {
		writeJSON(w, http.StatusGone, map[string]string{"error": "lien expire"})
		return
	}
	msg := ""
	if r.URL.Query().Get("replaced") == "1" {
		msg = doclink.ReplacedMessage
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"document_id": p.DocID,
		"kind":        p.Kind,
		"version":     p.Version,
		"message":     msg,
	})
}
