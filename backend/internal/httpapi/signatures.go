package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/tainalo2/devis_generator/backend/internal/biz"
	"github.com/tainalo2/devis_generator/backend/internal/doclink"
)

// publicSigner is the signature data access used by the public link.
type publicSigner interface {
	SignQuote(ctx context.Context, docID uuid.UUID, in biz.SignInput) (biz.Signature, error)
	GetSignaturePublic(ctx context.Context, docID uuid.UUID) (biz.Signature, error)
}

// resolvePublicDocument verifies the link token and loads the document. Every unusable link
// (tampered, expired, unknown, kind mismatch) yields the same 404 as the PDF route, so the
// signature routes are not an oracle either. ok=false means the response was already written.
func (s *Server) resolvePublicDocument(w http.ResponseWriter, r *http.Request) (biz.Document, bool) {
	if s.publicDocs == nil || s.publicSigner == nil {
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return biz.Document{}, false
	}
	p, err := doclink.Verify(s.docLinkSecret, chi.URLParam(r, "token"))
	if err != nil {
		publicLinkUnavailable(w)
		return biz.Document{}, false
	}
	docID, perr := uuid.Parse(p.DocID)
	if perr != nil {
		docID = uuid.Nil
	}
	document, _, err := s.publicDocs.GetDocumentPublic(r.Context(), docID)
	notFound := errors.Is(err, biz.ErrNotFound)
	if err != nil && !notFound {
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return biz.Document{}, false
	}
	if notFound || perr != nil || document.Kind != p.Kind || doclink.Expired(p, time.Now()) {
		publicLinkUnavailable(w)
		return biz.Document{}, false
	}
	return document, true
}

func isSignableState(d biz.Document) bool {
	return d.Kind == doclink.KindQuote && (d.Status == "issued" || d.Status == "sent")
}

// handlePublicSignatureState : GET /public/documents/{token}/signature.
// Public view only: no IP, user-agent or hash.
func (s *Server) handlePublicSignatureState(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	if s.publicLinkLimiter != nil && !s.publicLinkLimiter.Allow("sigstate:"+s.clientIP(r)) {
		writeAPIError(w, http.StatusTooManyRequests, "Trop de requêtes. Réessayez plus tard.")
		return
	}
	document, ok := s.resolvePublicDocument(w, r)
	if !ok {
		return
	}
	out := map[string]any{"signed": false, "signable": isSignableState(document)}
	sig, err := s.publicSigner.GetSignaturePublic(r.Context(), document.ID)
	switch {
	case err == nil:
		out["signed"], out["signable"] = true, false
		out["signed_at"] = sig.SignedAt.UTC().Format(time.RFC3339)
		out["signer_name"] = sig.SignerName
	case !errors.Is(err, biz.ErrNotFound):
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

type signRequest struct {
	Name string `json:"name"`
}

// handlePublicSign : POST /public/documents/{token}/sign. The signer declares only a name;
// time, IP, user-agent and content hash are captured server-side.
func (s *Server) handlePublicSign(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	if s.signLimiter != nil && !s.signLimiter.Allow("sign:"+s.clientIP(r)) {
		writeAPIError(w, http.StatusTooManyRequests, "Trop de requêtes. Réessayez plus tard.")
		return
	}
	document, ok := s.resolvePublicDocument(w, r)
	if !ok {
		return
	}
	var req signRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "Requête invalide.")
		return
	}
	sig, err := s.publicSigner.SignQuote(r.Context(), document.ID, biz.SignInput{
		Name: req.Name, IP: s.clientIP(r), UserAgent: r.UserAgent(),
	})
	switch {
	case err == nil:
		writeJSON(w, http.StatusCreated, map[string]any{
			"signed_at":   sig.SignedAt.UTC().Format(time.RFC3339),
			"signer_name": sig.SignerName,
		})
	case errors.Is(err, biz.ErrAlreadySigned):
		writeAPIError(w, http.StatusConflict, "Ce devis est déjà signé.")
	case errors.Is(err, biz.ErrNotSignable):
		writeAPIError(w, http.StatusConflict, "Ce document ne peut pas être signé.")
	case errors.Is(err, biz.ErrInvalidInput):
		writeAPIError(w, http.StatusBadRequest, "Indiquez votre nom (2 à 200 caractères).")
	case errors.Is(err, biz.ErrNotFound):
		publicLinkUnavailable(w)
	default:
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
	}
}

// handleGetSignatureProof : GET /documents/{id}/signature, owner only, with integrity check.
func (s *Server) handleGetSignatureProof(w http.ResponseWriter, r *http.Request) {
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
	proof, err := s.biz.GetSignatureProof(r.Context(), user.ID, id)
	if errors.Is(err, biz.ErrNotFound) {
		writeAPIError(w, http.StatusNotFound, "Aucune signature pour ce document.")
		return
	}
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}
	writeJSON(w, http.StatusOK, proof)
}
