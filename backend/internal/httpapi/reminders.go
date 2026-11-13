package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/tainalo2/devis_generator/backend/internal/reminders"
)

func (s *Server) handleSendReminder(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "Non authentifié.")
		return
	}
	if s.reminders == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "Service indisponible.")
		return
	}
	invoice, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "Identifiant invalide.")
		return
	}
	var body struct {
		InvoiceID  uuid.UUID `json:"invoice_id"`
		TemplateID uuid.UUID `json:"template_id"`
	}
	if err = json.NewDecoder(r.Body).Decode(&body); err != nil || body.TemplateID == uuid.Nil || (body.InvoiceID != uuid.Nil && body.InvoiceID != invoice) {
		writeAPIError(w, http.StatusBadRequest, "Requête invalide.")
		return
	}
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" {
		writeAPIError(w, http.StatusBadRequest, "Idempotency-Key requis.")
		return
	}
	item, err := s.reminders.Send(r.Context(), u.ID, invoice, body.TemplateID, key, "manual")
	if errors.Is(err, reminders.ErrNotFound) {
		writeAPIError(w, http.StatusNotFound, "Facture ou modèle introuvable.")
		return
	}
	if errors.Is(err, reminders.ErrInvalid) {
		writeAPIError(w, http.StatusBadRequest, "Requête invalide.")
		return
	}
	if errors.Is(err, reminders.ErrDelivery) {
		writeJSON(w, http.StatusBadGateway, map[string]any{"reminder": item, "error": "Échec de livraison."})
		return
	}
	if errors.Is(err, reminders.ErrRateLimited) {
		writeAPIError(w, http.StatusTooManyRequests, "Limite horaire de relances atteinte.")
		return
	}
	if errors.Is(err, reminders.ErrDocumentChanged) {
		writeJSON(w, http.StatusConflict, map[string]any{"reminder": item, "error": "Le document a changé; relance annulée."})
		return
	}
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"reminder": item})
}
func (s *Server) handleListReminders(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "Non authentifié.")
		return
	}
	if s.reminders == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "Service indisponible.")
		return
	}
	invoice, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "Identifiant invalide.")
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	items, total, err := s.reminders.List(r.Context(), u.ID, invoice, page, size)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}
	page, size = reminders.BoundPagination(page, size)
	writeJSON(w, http.StatusOK, map[string]any{"reminders": items, "page": page, "page_size": size, "total": total})
}

func (s *Server) handleGetReminderDeliveryMode(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "Non authentifié.")
		return
	}
	mode, err := s.reminders.Mode(r.Context(), u.ID)
	if errors.Is(err, reminders.ErrDeliveryMode) {
		writeAPIError(w, http.StatusNotFound, "Mode d’envoi non configuré.")
		return
	}
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"mode": mode})
}

func (s *Server) handleSetReminderDeliveryMode(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "Non authentifié.")
		return
	}
	var body struct {
		Mode reminders.DeliveryMode `json:"mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "Requête invalide.")
		return
	}
	if err := s.reminders.SetMode(r.Context(), u.ID, body.Mode); errors.Is(err, reminders.ErrInvalid) {
		writeAPIError(w, http.StatusBadRequest, "Mode d’envoi invalide.")
		return
	} else if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"mode": body.Mode})
}
