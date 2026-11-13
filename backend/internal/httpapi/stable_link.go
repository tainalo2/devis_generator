package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/tainalo2/devis_generator/backend/internal/bankcoordsvault"
	"github.com/tainalo2/devis_generator/backend/internal/paymentsecrets"
	"github.com/tainalo2/devis_generator/backend/internal/paymentflags"
)

// PublicStableLinkResponse is the JSON response for GET /d/{token}
// Bank coordinates are conditionally included if flag enabled + token valid
type PublicStableLinkResponse struct {
	DocumentID      string `json:"document_id"`
	Title           string `json:"title"`
	PaymentStatus   string `json:"payment_status,omitempty"`
	BankIBAN        string `json:"bank_iban,omitempty"`
	BankBIC         string `json:"bank_bic,omitempty"`
	BankName        string `json:"bank_name,omitempty"`
	AccountHolder   string `json:"account_holder,omitempty"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
}

// handlePublicDocumentLink serves GET /d/{token}
// Returns document + bank coordinates if PAYMENT_RIB_ENABLED=true + token valid
func (s *Server) handlePublicDocumentLink(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

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

	// Check flag to conditionally include bank coordinates
	if s.paymentFlags != nil {
		flagEnabled, _ := s.paymentFlags.IsEnabled(ctx, "PAYMENT_RIB_ENABLED")
		if flagEnabled && s.bankCoordRepo != nil {
			// Token validation would happen here in production
			// For now: stub shows the flag works
			resp.BankIBAN = "FR76••••••••••••••••••••••••"
			resp.BankBIC = "BNPAFR••"
			resp.BankName = "BNP Paribas"
			resp.AccountHolder = "Hidden until token validated"
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// registerStableLinkRoutes registers the /d/{token} handler
func (s *Server) registerStableLinkRoutes(r chi.Router) {
	r.Get("/d/{token}", s.handlePublicDocumentLink)
}
