package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/tainalo2/devis_generator/backend/internal/biz"
	"github.com/tainalo2/devis_generator/backend/internal/doclink"
	"github.com/tainalo2/devis_generator/backend/internal/legal"
	"github.com/tainalo2/devis_generator/backend/internal/pdfgen"
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

// publicDocStore is the read-only data access used by the public document link.
type publicDocStore interface {
	GetDocumentPublic(ctx context.Context, docID uuid.UUID) (biz.Document, uuid.UUID, error)
	GetIssuer(ctx context.Context, userID uuid.UUID) (biz.IssuerProfile, error)
	GetClient(ctx context.Context, userID, id uuid.UUID) (biz.Client, error)
}

// publicLinkUnavailable is the single response for every unusable link
// (invalid, tampered, expired, unknown document, kind mismatch) so that the
// status, body and headers reveal nothing about why the link failed.
func publicLinkUnavailable(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, http.StatusNotFound, map[string]string{"error": publicLinkUnavailableMessage})
}

const publicLinkUnavailableMessage = "Lien indisponible."

// redactIssuerForPublic removes every bank coordinate from the issuer view used
// for public rendering: a public link must never expose the RIB/IBAN.
func redactIssuerForPublic(issuer biz.IssuerProfile) biz.IssuerProfile {
	issuer.IBAN = ""
	return issuer
}

func (s *Server) handleDocumentLink(w http.ResponseWriter, r *http.Request) {
	if s.publicDocs == nil {
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}
	token := chi.URLParam(r, "token")
	p, err := doclink.Verify(s.docLinkSecret, token)
	if err != nil || doclink.Expired(p, time.Now()) {
		publicLinkUnavailable(w)
		return
	}
	docID, err := uuid.Parse(p.DocID)
	if err != nil {
		publicLinkUnavailable(w)
		return
	}
	document, ownerID, err := s.publicDocs.GetDocumentPublic(r.Context(), docID)
	if errors.Is(err, biz.ErrNotFound) || (err == nil && document.Kind != p.Kind) {
		publicLinkUnavailable(w)
		return
	}
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}

	issuer, err := s.publicDocs.GetIssuer(r.Context(), ownerID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}
	if len(document.IssuerSnapshot) > 0 {
		_ = json.Unmarshal(document.IssuerSnapshot, &issuer)
	}
	issuer = redactIssuerForPublic(issuer)
	client := biz.Client{Name: "Client"}
	if document.ClientSnapshot != nil {
		client = document.ClientSnapshot.AsClient()
	} else if document.ClientID != nil {
		if current, err := s.publicDocs.GetClient(r.Context(), ownerID, *document.ClientID); err == nil {
			client = current
		}
	}
	mentions := legal.DefaultMentions()
	if len(document.LegalSnapshot) > 0 {
		_ = json.Unmarshal(document.LegalSnapshot, &mentions)
	} else if s.legal != nil {
		mentions = s.legal.MentionsForIssue(r.Context())
	}
	pdf, err := pdfgen.Render(document, issuer, client, mentions)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Génération PDF impossible.")
		return
	}
	filename := "document.pdf"
	if document.Number != nil {
		filename = *document.Number + ".pdf"
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `inline; filename="`+filename+`"`)
	w.Header().Set("Cache-Control", "private, no-store")
	if r.URL.Query().Get("replaced") == "1" {
		w.Header().Set("X-Document-Link-Message", doclink.ReplacedMessage)
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdf)
}
