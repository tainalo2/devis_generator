package httpapi

import (
	"errors"
	"net/http"
	"regexp"

	"github.com/tainalo2/devis_generator/backend/internal/biz"
)

// idempotencyKeyPattern : 1 à 255 caractères ASCII sûrs (UUID, ULID, slug...).
// Aucun espace : la valeur n'est ni tronquée ni normalisée.
var idempotencyKeyPattern = regexp.MustCompile(`^[A-Za-z0-9._:/-]{1,255}$`)

var errInvalidIdempotencyKey = errors.New("Idempotency-Key invalide (1 à 255 caractères parmi A-Z a-z 0-9 . _ : / -).")

// idempotencyKeyFromRequest lit l'en-tête Idempotency-Key. present=false
// signale l'absence de l'en-tête : la création n'est alors pas dédupliquée
// (comportement historique de l'API, utilisé par l'interface web).
func idempotencyKeyFromRequest(r *http.Request) (key string, present bool, err error) {
	values := r.Header.Values("Idempotency-Key")
	if len(values) == 0 {
		return "", false, nil
	}
	if len(values) > 1 || !idempotencyKeyPattern.MatchString(values[0]) {
		return "", true, errInvalidIdempotencyKey
	}
	return values[0], true, nil
}

// writeCreated répond 201 pour une création, 200 + Idempotent-Replayed: true
// pour le rejeu d'une création déjà effectuée.
func writeCreated(w http.ResponseWriter, d biz.Document, replayed bool) {
	status := http.StatusCreated
	if replayed {
		status = http.StatusOK
		w.Header().Set("Idempotent-Replayed", "true")
	}
	writeJSON(w, status, map[string]any{"document": d})
}
