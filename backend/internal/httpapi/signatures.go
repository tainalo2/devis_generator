package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/tainalo2/devis_generator/backend/internal/biz"
	"github.com/tainalo2/devis_generator/backend/internal/doclink"
	"github.com/tainalo2/devis_generator/backend/internal/legal"
	"github.com/tainalo2/devis_generator/backend/internal/pdfgen"
	"github.com/tainalo2/devis_generator/backend/internal/signatures"
)

// SignatureRequest represents a request to sign a document via public link
type SignatureRequest struct {
	SignerEmail string `json:"signer_email"`
	SignerName  string `json:"signer_name,omitempty"`
}

// SignatureResponse represents the signature created
type SignatureResponse struct {
	DocumentID  string    `json:"document_id"`
	SignedAt    time.Time `json:"signed_at"`
	SignerEmail string    `json:"signer_email"`
	ProofToken  string    `json:"proof_token"`
}

// DocumentSignatureStateResponse represents the signature state of a document
type DocumentSignatureStateResponse struct {
	DocumentID    string     `json:"document_id"`
	Status        string     `json:"status"`
	SignedAt      *time.Time `json:"signed_at,omitempty"`
	SignedByEmail *string    `json:"signed_by_email,omitempty"`
	IsLocked      bool       `json:"is_locked"`
	ProofToken    *string    `json:"proof_token,omitempty"`
}

// handleGetDocumentSignatureStatePublic returns the signature state of a document via public link
// No authentication required; endpoint is rate-limited
func (s *Server) handleGetDocumentSignatureStatePublic(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	if s.publicLinkLimiter != nil && !s.publicLinkLimiter.Allow("sig-state:"+s.clientIP(r)) {
		writeAPIError(w, http.StatusTooManyRequests, "Trop de requêtes. Réessayez plus tard.")
		return
	}

	token := chi.URLParam(r, "token")
	p, err := doclink.Verify(s.docLinkSecret, token)
	if err != nil {
		publicLinkUnavailable(w)
		return
	}

	// Parse document ID from token
	docID, perr := uuid.Parse(p.DocID)
	if perr != nil {
		docID = uuid.Nil
	}

	// Get document
	document, _, err := s.publicDocs.GetDocumentPublic(r.Context(), docID)
	notFound := errors.Is(err, biz.ErrNotFound)
	if err != nil && !notFound {
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}

	if notFound || perr != nil || document.Kind != p.Kind || doclink.Expired(p, time.Now()) {
		publicLinkUnavailable(w)
		return
	}

	// Get signature state
	state, err := s.signatures.GetDocumentSignatureState(r.Context(), document.ID.String())
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get signature state: %v", err))
		return
	}

	resp := DocumentSignatureStateResponse{
		DocumentID:    state.DocumentID,
		Status:        string(state.Status),
		SignedAt:      state.SignedAt,
		SignedByEmail: state.SignedByEmail,
		IsLocked:      state.IsLocked,
	}

	if state.Signature != nil {
		resp.ProofToken = &state.Signature.ProofToken
	}

	writeJSON(w, http.StatusOK, resp)
}

// handleSignDocumentPublic accepts and records a signature for a document via public link
// Requires signer email; PDF content is generated from stored document
func (s *Server) handleSignDocumentPublic(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	if s.publicLinkLimiter != nil && !s.publicLinkLimiter.Allow("sig:"+s.clientIP(r)) {
		writeAPIError(w, http.StatusTooManyRequests, "Trop de requêtes. Réessayez plus tard.")
		return
	}

	if r.Method != http.MethodPost {
		writeAPIError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}

	token := chi.URLParam(r, "token")
	p, err := doclink.Verify(s.docLinkSecret, token)
	if err != nil {
		publicLinkUnavailable(w)
		return
	}

	// Parse document ID from token
	docID, perr := uuid.Parse(p.DocID)
	if perr != nil {
		docID = uuid.Nil
	}

	// Get document
	document, ownerID, err := s.publicDocs.GetDocumentPublic(r.Context(), docID)
	notFound := errors.Is(err, biz.ErrNotFound)
	if err != nil && !notFound {
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}

	if notFound || perr != nil || document.Kind != p.Kind || doclink.Expired(p, time.Now()) {
		publicLinkUnavailable(w)
		return
	}

	// Parse signature request
	var sigReq SignatureRequest
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	defer r.Body.Close()

	if err := json.Unmarshal(body, &sigReq); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if sigReq.SignerEmail == "" {
		writeAPIError(w, http.StatusBadRequest, "signer_email required")
		return
	}

	// Check if document is already signed
	state, err := s.signatures.GetDocumentSignatureState(r.Context(), document.ID.String())
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get signature state: %v", err))
		return
	}

	if state.IsLocked {
		writeAPIError(w, http.StatusConflict, "document is already signed and locked")
		return
	}

	// Regenerate PDF to get current content
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

	// Compute signature hash
	sigHash := signatures.ComputeSignatureHash(pdf)

	// Create signature record
	sig := &signatures.Signature{
		ID:            uuid.New().String(),
		DocumentID:    document.ID.String(),
		SignerEmail:   sigReq.SignerEmail,
		SignedAt:      time.Now().UTC(),
		SignatureHash: sigHash,
		IPAddress:     getClientIP(r),
		UserAgent:     getClientUserAgent(r),
		ProofToken:    uuid.New().String(),
	}

	// Store signature
	if err := s.signatures.CreateSignature(r.Context(), sig); err != nil {
		writeAPIError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create signature: %v", err))
		return
	}

	// Lock document
	if err := s.signatures.LockDocument(r.Context(), document.ID.String(), sig.SignedAt, sigReq.SignerEmail); err != nil {
		writeAPIError(w, http.StatusInternalServerError, fmt.Sprintf("failed to lock document: %v", err))
		return
	}

	// Return signature response
	resp := SignatureResponse{
		DocumentID:  document.ID.String(),
		SignedAt:    sig.SignedAt,
		SignerEmail: sig.SignerEmail,
		ProofToken:  sig.ProofToken,
	}

	writeJSON(w, http.StatusCreated, resp)
}

// handleGetSignatureProof returns a signature proof by proof token
// Public endpoint: no authentication required, but rate-limited
func (s *Server) handleGetSignatureProof(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	if s.publicLinkLimiter != nil && !s.publicLinkLimiter.Allow("proof:"+s.clientIP(r)) {
		writeAPIError(w, http.StatusTooManyRequests, "Trop de requêtes. Réessayez plus tard.")
		return
	}

	proofToken := chi.URLParam(r, "proof_token")
	if proofToken == "" {
		writeAPIError(w, http.StatusBadRequest, "proof_token required")
		return
	}

	sig, err := s.signatures.GetSignatureByProofToken(r.Context(), proofToken)
	if err != nil {
		writeAPIError(w, http.StatusNotFound, "signature not found")
		return
	}

	resp := map[string]interface{}{
		"document_id":     sig.DocumentID,
		"signer_email":    sig.SignerEmail,
		"signed_at":       sig.SignedAt,
		"signature_hash":  sig.SignatureHash,
		"proof_token":     sig.ProofToken,
		"created_at":      sig.CreatedAt,
	}

	writeJSON(w, http.StatusOK, resp)
}

// handleGetDocumentSignatureStateForUser returns signature state for authenticated user
// Only visible to document owner
func (s *Server) handleGetDocumentSignatureStateForUser(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "Non authentifié.")
		return
	}

	docID, err := parseID(r)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "Identifiant invalide.")
		return
	}

	// Verify user owns the document
	doc, err := s.biz.GetDocument(r.Context(), user.ID, docID)
	if errors.Is(err, biz.ErrNotFound) {
		writeAPIError(w, http.StatusNotFound, "Document introuvable.")
		return
	}
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}

	// Get signature state
	state, err := s.signatures.GetDocumentSignatureState(r.Context(), doc.ID.String())
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get signature state: %v", err))
		return
	}

	resp := DocumentSignatureStateResponse{
		DocumentID:    state.DocumentID,
		Status:        string(state.Status),
		SignedAt:      state.SignedAt,
		SignedByEmail: state.SignedByEmail,
		IsLocked:      state.IsLocked,
	}

	if state.Signature != nil {
		resp.ProofToken = &state.Signature.ProofToken
	}

	writeJSON(w, http.StatusOK, resp)
}

// Helper functions
func getClientIP(r *http.Request) *string {
	// Check X-Forwarded-For first (proxy headers)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return &xff
	}
	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	return &ip
}

func getClientUserAgent(r *http.Request) *string {
	ua := r.Header.Get("User-Agent")
	if ua == "" {
		return nil
	}
	return &ua
}
