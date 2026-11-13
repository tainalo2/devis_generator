package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/tainalo2/devis_generator/backend/internal/bankcoordsvault"
	"github.com/tainalo2/devis_generator/backend/internal/paymentsecrets"
	"github.com/tainalo2/devis_generator/backend/internal/testhelpers"
)

func TestHandleCreateBankCoordinates(t *testing.T) {
	ctx := context.Background()
	pool := testhelpers.NewTestPool(t)
	defer pool.Close()

	testhelpers.RunMigrations(t, pool)

	box := testhelpers.NewTestBox(t)
	vaultRepo := paymentsecrets.NewPGRepository(pool)
	vault := paymentsecrets.NewVault(vaultRepo, box)
	coordRepo := bankcoordsvault.NewRepository(pool, vault)

	userID := uuid.New()
	testhelpers.CreateTestUser(t, pool, userID)

	server := &Server{
		bankCoordRepo: coordRepo,
	}

	reqBody := CreateBankCoordinatesRequest{
		AccountHolderName: "John Doe",
		IBAN:              "FR1420041010050500013M02606",
		BIC:               "BNPAFRPP",
		BankName:          "BNP Paribas",
	}

	data, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/bank-coordinates", bytes.NewReader(data))
	req = req.WithContext(context.WithValue(req.Context(), "user_id", userID))

	w := httptest.NewRecorder()
	server.handleCreateBankCoordinates(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp BankCoordinatesResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.AccountHolderName != "John Doe" {
		t.Errorf("account holder name mismatch: %s", resp.AccountHolderName)
	}
	if resp.BankName != "BNP Paribas" {
		t.Errorf("bank name mismatch: %s", resp.BankName)
	}
}

func TestHandleGetBankCoordinates(t *testing.T) {
	ctx := context.Background()
	pool := testhelpers.NewTestPool(t)
	defer pool.Close()

	testhelpers.RunMigrations(t, pool)

	box := testhelpers.NewTestBox(t)
	vaultRepo := paymentsecrets.NewPGRepository(pool)
	vault := paymentsecrets.NewVault(vaultRepo, box)
	coordRepo := bankcoordsvault.NewRepository(pool, vault)

	userID := uuid.New()
	testhelpers.CreateTestUser(t, pool, userID)

	actor := paymentsecrets.Actor{UserID: userID, RequestID: "test"}
	coordRepo.CreateOrUpdate(ctx, actor, userID,
		"Jane Doe",
		"DE89370400440532013000",
		"COBADEFF",
		"Commerzbank",
	)

	server := &Server{
		bankCoordRepo: coordRepo,
	}

	req := httptest.NewRequest("GET", "/api/v1/bank-coordinates", nil)
	req = req.WithContext(context.WithValue(req.Context(), "user_id", userID))

	w := httptest.NewRecorder()
	server.handleGetBankCoordinates(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp BankCoordinatesDetailResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.AccountHolderName != "Jane Doe" {
		t.Errorf("account holder name mismatch: %s", resp.AccountHolderName)
	}
	if resp.IBAN != "DE89370400440532013000" {
		t.Errorf("IBAN mismatch: %s", resp.IBAN)
	}
	if resp.BIC != "COBADEFF" {
		t.Errorf("BIC mismatch: %s", resp.BIC)
	}
}

func TestHandleDeleteBankCoordinates(t *testing.T) {
	ctx := context.Background()
	pool := testhelpers.NewTestPool(t)
	defer pool.Close()

	testhelpers.RunMigrations(t, pool)

	box := testhelpers.NewTestBox(t)
	vaultRepo := paymentsecrets.NewPGRepository(pool)
	vault := paymentsecrets.NewVault(vaultRepo, box)
	coordRepo := bankcoordsvault.NewRepository(pool, vault)

	userID := uuid.New()
	testhelpers.CreateTestUser(t, pool, userID)

	actor := paymentsecrets.Actor{UserID: userID, RequestID: "test"}
	coordRepo.CreateOrUpdate(ctx, actor, userID,
		"Test User",
		"FR1420041010050500013M02606",
		"",
		"",
	)

	server := &Server{
		bankCoordRepo: coordRepo,
	}

	req := httptest.NewRequest("DELETE", "/api/v1/bank-coordinates", nil)
	req = req.WithContext(context.WithValue(req.Context(), "user_id", userID))

	w := httptest.NewRecorder()
	server.handleDeleteBankCoordinates(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", w.Code)
	}

	// Verify deletion
	exists, _ := coordRepo.Exists(ctx, userID)
	if exists {
		t.Error("coordinates should be deleted")
	}
}

func TestHandleUnauthorized(t *testing.T) {
	server := &Server{
		bankCoordRepo: nil,
	}

	req := httptest.NewRequest("GET", "/api/v1/bank-coordinates", nil)
	// No user_id in context

	w := httptest.NewRecorder()
	server.handleGetBankCoordinates(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestHandleInvalidIBANRequest(t *testing.T) {
	server := &Server{
		bankCoordRepo: nil,
	}

	reqBody := CreateBankCoordinatesRequest{
		AccountHolderName: "John",
		IBAN:              "INVALID",
		BIC:               "",
		BankName:          "",
	}

	data, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/bank-coordinates", bytes.NewReader(data))

	userID := uuid.New()
	req = req.WithContext(context.WithValue(req.Context(), "user_id", userID))

	w := httptest.NewRecorder()
	server.handleCreateBankCoordinates(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}
