package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/tainalo2/devis_generator/backend/internal/billing"
	"github.com/tainalo2/devis_generator/backend/internal/biz"
	"github.com/tainalo2/devis_generator/backend/internal/legal"
	"github.com/tainalo2/devis_generator/backend/internal/pdfgen"
)

func (s *Server) handleListDocuments(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "Non authentifié.")
		return
	}
	list, err := s.biz.ListDocuments(r.Context(), u.ID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"documents": list})
}

func (s *Server) handleGetDocument(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "Non authentifié.")
		return
	}
	id, err := parseID(r)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "Identifiant invalide.")
		return
	}
	d, err := s.biz.GetDocument(r.Context(), u.ID, id)
	if errors.Is(err, biz.ErrNotFound) {
		writeAPIError(w, http.StatusNotFound, "Document introuvable.")
		return
	}
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"document": d})
}

func (s *Server) handleCreateDocument(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "Non authentifié.")
		return
	}
	var in biz.DocumentInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeAPIError(w, http.StatusBadRequest, "Requête invalide.")
		return
	}
	d, err := s.biz.CreateDocument(r.Context(), u.ID, in)
	if errors.Is(err, biz.ErrInvalidInput) {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "Impossible d'enregistrer le document.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"document": d})
}

func (s *Server) handleUpdateDocument(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "Non authentifié.")
		return
	}
	id, err := parseID(r)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "Identifiant invalide.")
		return
	}
	var in biz.DocumentInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeAPIError(w, http.StatusBadRequest, "Requête invalide.")
		return
	}
	d, err := s.biz.UpdateDocument(r.Context(), u.ID, id, in)
	if errors.Is(err, biz.ErrNotFound) {
		writeAPIError(w, http.StatusNotFound, "Document introuvable.")
		return
	}
	if errors.Is(err, biz.ErrImmutable) {
		writeAPIError(w, http.StatusConflict, "Document émis : modification impossible.")
		return
	}
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "Impossible d'enregistrer le document.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"document": d})
}

func (s *Server) handleDeleteDocument(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "Non authentifié.")
		return
	}
	id, err := parseID(r)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "Identifiant invalide.")
		return
	}
	err = s.biz.DeleteDocument(r.Context(), u.ID, id)
	if errors.Is(err, biz.ErrNotFound) {
		writeAPIError(w, http.StatusNotFound, "Document introuvable.")
		return
	}
	if errors.Is(err, biz.ErrImmutable) {
		writeAPIError(w, http.StatusConflict, "Document émis : suppression impossible.")
		return
	}
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleIssueDocument(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "Non authentifié.")
		return
	}
	id, err := parseID(r)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "Identifiant invalide.")
		return
	}
	d, err := s.biz.IssueDocument(r.Context(), u.ID, id)
	if errors.Is(err, biz.ErrNotFound) {
		writeAPIError(w, http.StatusNotFound, "Document introuvable.")
		return
	}
	if errors.Is(err, biz.ErrImmutable) {
		writeAPIError(w, http.StatusConflict, "Document déjà émis.")
		return
	}
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"document": d})
}

func (s *Server) handleDocumentStatus(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "Non authentifié.")
		return
	}
	id, err := parseID(r)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "Identifiant invalide.")
		return
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "Corps JSON invalide.")
		return
	}
	d, err := s.biz.UpdateDocumentStatus(r.Context(), u.ID, id, body.Status)
	if errors.Is(err, biz.ErrNotFound) {
		writeAPIError(w, http.StatusNotFound, "Document introuvable.")
		return
	}
	if errors.Is(err, biz.ErrInvalidInput) {
		writeAPIError(w, http.StatusConflict, "Ce changement de statut n’est pas possible pour ce document.")
		return
	}
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Mise à jour du statut impossible.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"document": d})
}

func (s *Server) handleConvertQuote(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "Non authentifié.")
		return
	}
	id, err := parseID(r)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "Identifiant invalide.")
		return
	}
	d, err := s.biz.ConvertQuoteToInvoice(r.Context(), u.ID, id)
	if errors.Is(err, biz.ErrNotFound) {
		writeAPIError(w, http.StatusNotFound, "Document introuvable.")
		return
	}
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"document": d})
}

func (s *Server) handleCreateDeposit(w http.ResponseWriter, r *http.Request) {
	s.createLinked(w, r, "deposit_invoice")
}

func (s *Server) handleCreateCreditNote(w http.ResponseWriter, r *http.Request) {
	s.createLinked(w, r, "credit_note")
}

func (s *Server) createLinked(w http.ResponseWriter, r *http.Request, kind string) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "Non authentifié.")
		return
	}
	id, err := parseID(r)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "Identifiant invalide.")
		return
	}
	var body struct {
		Lines []biz.DocLine `json:"lines"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	d, err := s.biz.CreateLinkedDocument(r.Context(), u.ID, id, kind, body.Lines)
	if errors.Is(err, biz.ErrNotFound) {
		writeAPIError(w, http.StatusNotFound, "Document introuvable.")
		return
	}
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"document": d})
}

func (s *Server) handleDocumentPDF(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "Non authentifié.")
		return
	}
	id, err := parseID(r)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "Identifiant invalide.")
		return
	}
	d, err := s.biz.GetDocument(r.Context(), u.ID, id)
	if errors.Is(err, biz.ErrNotFound) {
		writeAPIError(w, http.StatusNotFound, "Document introuvable.")
		return
	}
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}
	issuer, err := s.biz.GetIssuer(r.Context(), u.ID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}
	// Un document émis garde l'émetteur figé à l'émission.
	if len(d.IssuerSnapshot) > 0 {
		var snap biz.IssuerProfile
		if err := json.Unmarshal(d.IssuerSnapshot, &snap); err == nil {
			issuer = snap
		}
	}
	client := biz.Client{Name: "Client"}
	if d.ClientSnapshot != nil {
		// Document émis : le client figé à l'émission fait foi.
		client = d.ClientSnapshot.AsClient()
	} else if d.ClientID != nil {
		if c, err := s.biz.GetClient(r.Context(), u.ID, *d.ClientID); err == nil {
			client = c
		}
	}
	mentions := legal.DefaultMentions()
	if len(d.LegalSnapshot) > 0 {
		_ = json.Unmarshal(d.LegalSnapshot, &mentions)
	} else if s.legal != nil {
		mentions = s.legal.MentionsForIssue(r.Context())
	}
	pdf, err := pdfgen.Render(d, issuer, client, mentions)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Génération PDF impossible.")
		return
	}
	filename := "document.pdf"
	if d.Number != nil {
		filename = *d.Number + ".pdf"
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdf)
}

func (s *Server) handleGuestPDF(w http.ResponseWriter, r *http.Request) {
	var req biz.GuestPDFRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "Requête invalide.")
		return
	}
	if req.Currency == "" {
		req.Currency = "EUR"
	}
	if req.VATRegime == "" {
		req.VATRegime = billing.RegimeFranchise293B
	}
	inputs := make([]billing.LineInput, 0, len(req.Lines))
	for _, l := range req.Lines {
		inputs = append(inputs, billing.LineInput{
			Description: l.Description, QuantityMilli: l.QuantityMilli,
			UnitPriceCents: l.UnitPriceCents, DiscountBps: l.DiscountBps, TaxRateBps: l.TaxRateBps,
		})
	}
	calc, err := billing.Compute(req.VATRegime, inputs)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	lines := make([]biz.DocLine, 0, len(calc.Lines))
	for i, l := range calc.Lines {
		lines = append(lines, biz.DocLine{
			Position: i, Description: l.Description, QuantityMilli: l.QuantityMilli,
			UnitPriceCents: l.UnitPriceCents, DiscountBps: l.DiscountBps, TaxRateBps: l.TaxRateBps,
			TotalCents: l.TotalHTCents,
		})
	}
	num := req.Number
	doc := biz.Document{
		Kind: req.Kind, Status: "draft", Title: req.Title, Currency: req.Currency,
		VATRegime: req.VATRegime, Notes: req.Notes, PaymentTerms: req.PaymentTerms,
		SubtotalCents: calc.SubtotalCents, TaxCents: calc.TaxCents, TotalCents: calc.TotalCents,
		Lines: lines,
	}
	if req.IssueDate != "" {
		doc.IssueDate = &req.IssueDate
	}
	if req.DueDate != "" {
		doc.DueDate = &req.DueDate
	}
	if num != "" {
		doc.Number = &num
	}
	mentions := legal.DefaultMentions()
	if req.Mentions != nil {
		mentions = *req.Mentions
	} else if s.legal != nil {
		mentions = s.legal.MentionsForIssue(r.Context())
	}
	guestClient := biz.Client{
		Name: req.Client.Name, Email: req.Client.Email, Address: req.Client.Address,
		SIREN: req.Client.SIREN, SIRET: req.Client.SIRET, VATNumber: req.Client.VATNumber,
		Notes: req.Client.Notes,
	}
	pdf, err := pdfgen.Render(doc, req.Issuer, guestClient, mentions)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Génération PDF impossible.")
		return
	}
	// Ephemeral: nothing persisted.
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="document-invite.pdf"`)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdf)
}

func (s *Server) handleGetIssuer(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "Non authentifié.")
		return
	}
	p, err := s.biz.GetIssuer(r.Context(), u.ID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"issuer": p})
}

func (s *Server) handlePutIssuer(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "Non authentifié.")
		return
	}
	var in biz.IssuerProfile
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeAPIError(w, http.StatusBadRequest, "Requête invalide.")
		return
	}
	p, err := s.biz.UpsertIssuer(r.Context(), u.ID, in)
	if errors.Is(err, biz.ErrInvalidInput) {
		writeAPIError(w, http.StatusBadRequest, "Profil invalide.")
		return
	}
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"issuer": p})
}

func (s *Server) handlePreviewTotals(w http.ResponseWriter, r *http.Request) {
	var body struct {
		VATRegime string        `json:"vat_regime"`
		Lines     []biz.DocLine `json:"lines"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "Requête invalide.")
		return
	}
	inputs := make([]billing.LineInput, 0, len(body.Lines))
	for _, l := range body.Lines {
		inputs = append(inputs, billing.LineInput{
			Description: l.Description, QuantityMilli: l.QuantityMilli,
			UnitPriceCents: l.UnitPriceCents, DiscountBps: l.DiscountBps, TaxRateBps: l.TaxRateBps,
		})
	}
	res, err := billing.Compute(body.VATRegime, inputs)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func parseID(r *http.Request) (uuid.UUID, error) {
	return uuid.Parse(chi.URLParam(r, "id"))
}
