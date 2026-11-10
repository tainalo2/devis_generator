package httpapi

import (
	"context"
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/tainalo2/devis_generator/backend/internal/biz"
	"github.com/tainalo2/devis_generator/backend/internal/doclink"
)

type documentLinkTarget = biz.DocumentLinkTarget

type documentLinkResolver func(context.Context, uuid.UUID) (documentLinkTarget, error)

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
	token, err := doclink.Generate(s.docLinkSecret, document.ID.String(), document.Kind, document.QuoteVersion, document.CreatedAt)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Lien indisponible.")
		return
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	link := (&url.URL{Scheme: scheme, Host: r.Host, Path: "/d/" + token}).String()
	response := map[string]any{"link": link, "permanent": document.Kind == doclink.KindInvoice || document.Kind == doclink.KindDeposit}
	if document.Kind == doclink.KindQuote {
		response["expires_at"] = document.CreatedAt.Add(doclink.QuoteTTL).UTC().Format(time.RFC3339)
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleDocumentLink(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	payload, err := doclink.Verify(s.docLinkSecret, token)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "lien invalide"})
		return
	}
	id, err := uuid.Parse(payload.DocID)
	if err != nil || s.resolveDocumentLink == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "lien invalide"})
		return
	}
	target, err := s.resolveDocumentLink(r.Context(), id)
	if errors.Is(err, biz.ErrNotFound) || (err == nil && target.Kind != payload.Kind) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "document introuvable"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "lien indisponible"})
		return
	}
	if target.Replaced {
		currentToken, err := doclink.Generate(s.docLinkSecret, target.ID.String(), target.Kind, target.Version, target.CreatedAt)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "lien indisponible"})
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		http.Redirect(w, r, "/d/"+currentToken+"?replaced=1", http.StatusTemporaryRedirect)
		return
	}
	currentPayload := doclink.Payload{
		DocID: target.ID.String(), Kind: target.Kind, Version: target.Version, Issued: target.CreatedAt.Unix(),
	}
	if doclink.Expired(currentPayload, time.Now()) {
		writeJSON(w, http.StatusGone, map[string]string{"error": "lien expire"})
		return
	}
	message := ""
	if r.URL.Query().Get("replaced") == "1" {
		message = doclink.ReplacedMessage
	}
	if target.Kind == doclink.KindQuote {
		w.Header().Set("Cache-Control", "no-store")
	}
	if strings.Contains(r.Header.Get("Accept"), "text/html") {
		writeDocumentLinkPage(w, target, message)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"document_id": target.ID.String(),
		"kind":        target.Kind,
		"version":     target.Version,
		"message":     message,
	})
}

func writeDocumentLinkPage(w http.ResponseWriter, target documentLinkTarget, message string) {
	notice := ""
	if message != "" {
		notice = `<div class="notice"><strong>` + html.EscapeString(message) + `.</strong> Vous consultez automatiquement la version actuelle.</div>`
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`<!doctype html><html lang="fr"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Document partagé</title><style>body{font-family:system-ui,sans-serif;background:#f6f4ef;color:#1d1d1b;margin:0;padding:2rem}.card{max-width:42rem;margin:8vh auto;background:white;border-radius:1rem;padding:2rem;box-shadow:0 12px 36px #0001}.notice{border-left:4px solid #b56b2d;background:#fff6e9;padding:1rem;margin:1rem 0}code{word-break:break-all}</style></head><body><main class="card"><h1>Document partagé</h1>` + notice + `<p>Devis version ` + fmt.Sprint(target.Version) + `</p><p><small>Identifiant : <code>` + html.EscapeString(target.ID.String()) + `</code></small></p></main></body></html>`))
}
