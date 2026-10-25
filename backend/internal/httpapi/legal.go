package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/tainalo2/devis_generator/backend/internal/legal"
)

func (s *Server) handleLegalStatus(w http.ResponseWriter, r *http.Request) {
	if s.legal == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "Sources légales indisponibles.")
		return
	}
	entries, err := s.legal.Status(r.Context())
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Impossible de lire le statut légal.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"variables": entries,
		"sources": []map[string]string{
			{"id": "recherche-entreprises", "label": "API Recherche d'entreprises (DINUM)", "auth": "aucune"},
			{"id": "vies", "label": "VIES (Commission européenne)", "auth": "aucune"},
			{"id": "tedb", "label": "TEDB — taux de TVA UE", "auth": "aucune"},
			{"id": "bce", "label": "BCE SDMX — taux directeur", "auth": "aucune"},
			{"id": "insee.sirene", "label": "Insee Sirene (emplacement)", "auth": "compte requis — non branché"},
			{"id": "legifrance.piste", "label": "Légifrance / PISTE (emplacement)", "auth": "compte requis — non branché"},
			{"id": "manual", "label": "Valeurs surveillées manuellement", "auth": "n/a"},
		},
		"excluded": []string{"API Entreprise (administrations uniquement)"},
	})
}

func (s *Server) handleCompanySearch(w http.ResponseWriter, r *http.Request) {
	if !s.legalGuestLimiter.Allow("search:" + clientIP(r)) {
		writeAPIError(w, http.StatusTooManyRequests, "Trop de recherches. Réessayez plus tard.")
		return
	}
	if s.legal == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "Recherche d'entreprise indisponible.")
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) < 2 {
		writeAPIError(w, http.StatusBadRequest, "Indiquez au moins 2 caractères.")
		return
	}
	limit := 8
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			limit = n
		}
	}
	hits, err := s.legal.SearchCompanies(r.Context(), q, limit)
	if err != nil {
		writeAPIError(w, http.StatusBadGateway, "Recherche upstream indisponible.")
		return
	}
	if hits == nil {
		hits = []legal.CompanyHit{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": hits, "source": "recherche-entreprises.api.gouv.fr"})
}

func (s *Server) handleVATValidate(w http.ResponseWriter, r *http.Request) {
	if !s.legalGuestLimiter.Allow("vat:" + clientIP(r)) {
		writeAPIError(w, http.StatusTooManyRequests, "Trop de vérifications TVA. Réessayez plus tard.")
		return
	}
	if s.legal == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "Validation TVA indisponible.")
		return
	}
	var body struct {
		VATNumber   string `json:"vat_number"`
		CountryCode string `json:"country_code"`
		Number      string `json:"number"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "Requête invalide.")
		return
	}
	vat := strings.TrimSpace(body.VATNumber)
	if vat == "" && body.CountryCode != "" && body.Number != "" {
		vat = body.CountryCode + body.Number
	}
	if vat == "" {
		writeAPIError(w, http.StatusBadRequest, "Numéro de TVA manquant.")
		return
	}
	res, err := s.legal.ValidateVAT(r.Context(), vat)
	if err != nil {
		msg := "Validation VIES impossible."
		if strings.Contains(err.Error(), "TVA") || strings.Contains(err.Error(), "préfixe") {
			writeAPIError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeAPIError(w, http.StatusBadGateway, msg)
		return
	}
	writeJSON(w, http.StatusOK, res)
}
