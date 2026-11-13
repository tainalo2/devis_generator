package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/tainalo2/devis_generator/backend/internal/bankcoord"
	"github.com/tainalo2/devis_generator/backend/internal/bankcoordsvault"
	"github.com/tainalo2/devis_generator/backend/internal/paymentsecrets"
)

// CreateBankCoordinatesRequest is the payload for creating/updating bank coordinates.
type CreateBankCoordinatesRequest struct {
	AccountHolderName string `json:"account_holder_name"`
	IBAN              string `json:"iban"`
	BIC               string `json:"bic,omitempty"`
	BankName          string `json:"bank_name,omitempty"`
}

// BankCoordinatesResponse is the safe response (no plaintext secrets).
type BankCoordinatesResponse struct {
	ID                string `json:"id"`
	UserID            string `json:"user_id"`
	AccountHolderName string `json:"account_holder_name"`
	BankName          string `json:"bank_name,omitempty"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
}

// BankCoordinatesDetailResponse includes plaintext IBAN/BIC (audit logged).
type BankCoordinatesDetailResponse struct {
	BankCoordinatesResponse
	IBAN string `json:"iban"`
	BIC  string `json:"bic,omitempty"`
}

// handleCreateBankCoordinates creates or updates bank coordinates for the authenticated user.
// POST /api/v1/bank-coordinates
func (s *Server) handleCreateBankCoordinates(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := s.extractAuthenticatedUserID(r)
	if !ok {
		s.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateBankCoordinatesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	actor := paymentsecrets.Actor{
		UserID:    userID,
		RequestID: s.extractRequestID(r),
	}

	coords, err := s.bankCoordRepo.CreateOrUpdate(ctx, actor, userID,
		req.AccountHolderName, req.IBAN, req.BIC, req.BankName)

	if err != nil {
		if errors.Is(err, bankcoord.ErrInvalidIBAN) {
			s.writeError(w, http.StatusBadRequest, "invalid IBAN format")
			return
		}
		if errors.Is(err, bankcoord.ErrInvalidBIC) {
			s.writeError(w, http.StatusBadRequest, "invalid BIC format")
			return
		}
		if errors.Is(err, bankcoord.ErrMissingName) {
			s.writeError(w, http.StatusBadRequest, "account holder name required")
			return
		}
		s.writeError(w, http.StatusInternalServerError, "failed to save bank coordinates")
		return
	}

	resp := BankCoordinatesResponse{
		ID:                coords.ID.String(),
		UserID:            coords.UserID.String(),
		AccountHolderName: coords.AccountHolderName,
		BankName:          coords.BankName,
		CreatedAt:         coords.CreatedAt,
		UpdatedAt:         coords.UpdatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// handleGetBankCoordinates retrieves bank coordinates with plaintext IBAN/BIC (audit logged).
// GET /api/v1/bank-coordinates
func (s *Server) handleGetBankCoordinates(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := s.extractAuthenticatedUserID(r)
	if !ok {
		s.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	actor := paymentsecrets.Actor{
		UserID:    userID,
		RequestID: s.extractRequestID(r),
	}

	name, iban, bic, bankName, err := s.bankCoordRepo.Read(ctx, actor, userID)
	if err != nil {
		if errors.Is(err, bankcoordsvault.ErrNotFound) {
			s.writeError(w, http.StatusNotFound, "bank coordinates not found")
			return
		}
		s.writeError(w, http.StatusInternalServerError, "failed to retrieve bank coordinates")
		return
	}

	// Get metadata for timestamps
	metadata, err := s.bankCoordRepo.GetMetadata(ctx, userID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to retrieve metadata")
		return
	}

	resp := BankCoordinatesDetailResponse{
		BankCoordinatesResponse: BankCoordinatesResponse{
			ID:                metadata.ID.String(),
			UserID:            metadata.UserID.String(),
			AccountHolderName: name,
			BankName:          bankName,
			CreatedAt:         metadata.CreatedAt,
			UpdatedAt:         metadata.UpdatedAt,
		},
		IBAN: iban,
		BIC:  bic,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// handleGetBankCoordinatesMetadata retrieves bank coordinates metadata without plaintext.
// GET /api/v1/bank-coordinates/metadata
func (s *Server) handleGetBankCoordinatesMetadata(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := s.extractAuthenticatedUserID(r)
	if !ok {
		s.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	metadata, err := s.bankCoordRepo.GetMetadata(ctx, userID)
	if err != nil {
		if errors.Is(err, bankcoordsvault.ErrNotFound) {
			s.writeError(w, http.StatusNotFound, "bank coordinates not found")
			return
		}
		s.writeError(w, http.StatusInternalServerError, "failed to retrieve metadata")
		return
	}

	resp := BankCoordinatesResponse{
		ID:                metadata.ID.String(),
		UserID:            metadata.UserID.String(),
		AccountHolderName: metadata.AccountHolderName,
		BankName:          metadata.BankName,
		CreatedAt:         metadata.CreatedAt,
		UpdatedAt:         metadata.UpdatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// handleDeleteBankCoordinates deletes bank coordinates for the authenticated user.
// DELETE /api/v1/bank-coordinates
func (s *Server) handleDeleteBankCoordinates(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := s.extractAuthenticatedUserID(r)
	if !ok {
		s.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	actor := paymentsecrets.Actor{
		UserID:    userID,
		RequestID: s.extractRequestID(r),
	}

	err := s.bankCoordRepo.Delete(ctx, actor, userID)
	if err != nil {
		if errors.Is(err, bankcoordsvault.ErrNotFound) {
			s.writeError(w, http.StatusNotFound, "bank coordinates not found")
			return
		}
		s.writeError(w, http.StatusInternalServerError, "failed to delete bank coordinates")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helper to extract authenticated user ID from context (assumes auth middleware sets it).
func (s *Server) extractAuthenticatedUserID(r *http.Request) (uuid.UUID, bool) {
	// In real implementation, this would read from session/JWT context.
	// For now, this is a placeholder that the actual server implementation must fill.
	val := r.Context().Value("user_id")
	if val == nil {
		return uuid.UUID{}, false
	}
	userID, ok := val.(uuid.UUID)
	return userID, ok
}

// Helper to extract request ID from headers for audit logging.
func (s *Server) extractRequestID(r *http.Request) string {
	if id := r.Header.Get("X-Request-ID"); id != "" {
		return id
	}
	return uuid.New().String()
}
