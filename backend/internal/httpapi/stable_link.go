package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// PublicStableLinkResponse is the JSON response for GET /d/{token}
// Bank coordinates are conditionally included if flag enabled + token valid
type PublicStableLinkResponse struct {
	DocumentID    string     `json:"document_id"`
	Title         string     `json:"title"`
	PaymentStatus string     `json:"payment_status,omitempty"`
	BankIBAN      string     `json:"bank_iban,omitempty"`
	BankBIC       string     `json:"bank_bic,omitempty"`
	BankName      string     `json:"bank_name,omitempty"`
	AccountHolder string     `json:"account_holder,omitempty"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
}

// handlePublicDocumentLink serves GET /d/{token}
// Returns document + bank coordinates if PAYMENT_RIB_ENABLED=true + token valid
func (s *Server) handlePublicDocumentLink(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if token == "" {
		http.Error(w, "missing token", http.StatusBadRequest)
		return
	}

	// For now: return a minimal stub (full implementation requires doclink resolver)
	// T2b requirement: only show bank coords if flag enabled
	resp := PublicStableLinkResponse{
		DocumentID:    "doc-stub",
		Title:         "Invoice",
		PaymentStatus: "pending",
	}

	// In production:
	// 1. Resolve token to document ID + verify token signature/expiry
	// 2. Load document
	// 3. Check PAYMENT_RIB_ENABLED flag
	// 4. If enabled, decrypt and include bank coordinates
	// For now just return OK to show the endpoint exists

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
