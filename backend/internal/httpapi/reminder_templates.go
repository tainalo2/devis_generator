package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/tainalo2/devis_generator/backend/internal/reminders"
)

func (s *Server) handleListReminderTemplates(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "Non authentifié.")
		return
	}
	items, err := s.reminders.ListTemplates(r.Context(), u.ID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"templates": items})
}
func (s *Server) handleCreateReminderTemplate(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "Non authentifié.")
		return
	}
	var v reminders.Template
	if json.NewDecoder(r.Body).Decode(&v) != nil {
		writeAPIError(w, http.StatusBadRequest, "Requête invalide.")
		return
	}
	v.ID = uuid.Nil
	item, err := s.reminders.CreateTemplate(r.Context(), u.ID, v)
	if errors.Is(err, reminders.ErrInvalid) {
		writeAPIError(w, http.StatusBadRequest, "Template invalide.")
		return
	}
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}
	writeJSON(w, http.StatusCreated, item)
}
func (s *Server) handleUpdateReminderTemplate(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "Non authentifié.")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "Identifiant invalide.")
		return
	}
	var v reminders.Template
	if json.NewDecoder(r.Body).Decode(&v) != nil {
		writeAPIError(w, http.StatusBadRequest, "Requête invalide.")
		return
	}
	v.ID = id
	item, err := s.reminders.UpdateTemplate(r.Context(), u.ID, v)
	if errors.Is(err, reminders.ErrInvalid) {
		writeAPIError(w, http.StatusBadRequest, "Template invalide.")
		return
	}
	if errors.Is(err, reminders.ErrNotFound) {
		writeAPIError(w, http.StatusNotFound, "Template introuvable.")
		return
	}
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}
	writeJSON(w, http.StatusOK, item)
}
func (s *Server) handleDeleteReminderTemplate(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "Non authentifié.")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "Identifiant invalide.")
		return
	}
	err = s.reminders.DeleteTemplate(r.Context(), u.ID, id)
	if errors.Is(err, reminders.ErrNotFound) {
		writeAPIError(w, http.StatusNotFound, "Template introuvable.")
		return
	}
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
