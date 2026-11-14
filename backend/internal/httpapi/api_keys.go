package httpapi

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/tainalo2/devis_generator/backend/internal/auth"
	"github.com/tainalo2/devis_generator/backend/internal/authstore"
)

type apiKeyDTO struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Prefix     string   `json:"prefix"`
	Scopes     []string `json:"scopes"`
	CreatedAt  string   `json:"created_at"`
	LastUsedAt *string  `json:"last_used_at,omitempty"`
	RevokedAt  *string  `json:"revoked_at,omitempty"`
}

func apiKeyToDTO(k authstore.APIKey) apiKeyDTO {
	dto := apiKeyDTO{
		ID:        k.ID.String(),
		Name:      k.Name,
		Prefix:    k.KeyPrefix,
		Scopes:    k.Scopes,
		CreatedAt: k.CreatedAt.UTC().Format(time.RFC3339),
	}
	if k.LastUsed != nil {
		s := k.LastUsed.UTC().Format(time.RFC3339)
		dto.LastUsedAt = &s
	}
	if k.RevokedAt != nil {
		s := k.RevokedAt.UTC().Format(time.RFC3339)
		dto.RevokedAt = &s
	}
	return dto
}

func (s *Server) handleListAPIKeys(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "Non authentifié.")
		return
	}
	if s.store == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "Service indisponible.")
		return
	}
	keys, err := s.store.ListAPIKeys(r.Context(), u.ID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}
	out := make([]apiKeyDTO, 0, len(keys))
	for _, k := range keys {
		out = append(out, apiKeyToDTO(k))
	}
	writeJSON(w, http.StatusOK, map[string]any{"api_keys": out})
}

func (s *Server) handleCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "Non authentifié.")
		return
	}
	if s.store == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "Service indisponible.")
		return
	}
	var body struct {
		Name   string   `json:"name"`
		Scopes []string `json:"scopes"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "Requête invalide.")
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		writeAPIError(w, http.StatusBadRequest, "Nom requis.")
		return
	}
	scopes := body.Scopes
	if len(scopes) == 0 {
		scopes = []string{"read"}
	}
	for _, sc := range scopes {
		if sc != "read" && sc != "write" {
			writeAPIError(w, http.StatusBadRequest, "Scope invalide.")
			return
		}
	}
	raw, hash, err := auth.NewToken(32)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}
	prefix := raw
	if len(prefix) > 8 {
		prefix = prefix[:8]
	}
	k, err := s.store.CreateAPIKey(r.Context(), u.ID, name, hash, prefix, scopes)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"api_key": apiKeyToDTO(k),
		"key":     raw,
	})
}

func (s *Server) handleRevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "Non authentifié.")
		return
	}
	if s.store == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "Service indisponible.")
		return
	}
	id, err := uuid.Parse(r.URL.Query().Get("id"))
	// chi path param fallback
	if err != nil {
		// try path value
	}
	_ = id
	// Use chi.URLParam via r path? We'll read from path param "id"
	pid := r.PathValue("id")
	parsed, err := uuid.Parse(pid)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "Identifiant invalide.")
		return
	}
	if err := s.store.RevokeAPIKey(r.Context(), u.ID, parsed); err != nil {
		if err == authstore.ErrNotFound {
			writeAPIError(w, http.StatusNotFound, "Clé introuvable.")
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

var _ = context.Background
