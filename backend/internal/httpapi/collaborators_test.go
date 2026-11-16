package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tainalo2/devis_generator/backend/internal/auth"
	"github.com/tainalo2/devis_generator/backend/internal/authstore"
	"github.com/tainalo2/devis_generator/backend/internal/biz"
	"github.com/tainalo2/devis_generator/backend/internal/smtpconfig"
)

// TestHandleInviteCollaboratorOwnerCanInvite tests that an owner can send an invitation
func TestHandleInviteCollaboratorOwnerCanInvite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db, store, bizStore, server := setupTestServerWithDB(t)
	defer db.Close()

	// Create owner user
	owner := createTestUser(t, store, "owner@example.com", "owner")

	// Create account_collaborators entry for owner
	createTestAccountCollaborator(t, db, owner.ID, owner.ID, "owner")

	// Create a session for the owner
	sessionID, _ := createTestSession(t, store, owner.ID)

	// Prepare invitation request
	inviteInput := map[string]string{
		"email": "collaborator@example.com",
	}
	body, _ := json.Marshal(inviteInput)

	// Make request
	req := httptest.NewRequest(http.MethodPost, "/api/v1/collaborators/invite", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID})
	req = req.WithContext(context.WithValue(req.Context(), userCtxKey, owner))

	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["invite_id"] == nil || resp["email"] == nil {
		t.Errorf("response missing required fields: %v", resp)
	}
	if resp["email"] != "collaborator@example.com" {
		t.Errorf("expected email collaborator@example.com, got %v", resp["email"])
	}

	// Verify invitation was created in DB
	var count int
	db.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM invitations WHERE email = $1 AND account_id = $2 AND expires_at > NOW()
	`, "collaborator@example.com", owner.ID).Scan(&count)

	if count != 1 {
		t.Errorf("expected 1 invitation in DB, found %d", count)
	}
}

// TestHandleInviteCollaboratorNonOwnerCannotInvite tests that non-owners get 403 when trying to invite
func TestHandleInviteCollaboratorNonOwnerCannotInvite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db, store, _, server := setupTestServerWithDB(t)
	defer db.Close()

	// Create non-owner user
	nonOwner := createTestUser(t, store, "collaborator@example.com", "collaborator")

	// Create a session for non-owner
	sessionID, _ := createTestSession(t, store, nonOwner.ID)

	// Prepare invitation request
	inviteInput := map[string]string{
		"email": "someone@example.com",
	}
	body, _ := json.Marshal(inviteInput)

	// Make request
	req := httptest.NewRequest(http.MethodPost, "/api/v1/collaborators/invite", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID})
	req = req.WithContext(context.WithValue(req.Context(), userCtxKey, nonOwner))

	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d: %s", w.Code, w.Body.String())
	}

	var errResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&errResp)
	if errResp["error"] == nil {
		t.Errorf("expected error message in response")
	}
}

// TestHandleInviteCollaboratorSelfInvitationRejected tests that self-invitation is rejected
func TestHandleInviteCollaboratorSelfInvitationRejected(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db, store, _, server := setupTestServerWithDB(t)
	defer db.Close()

	// Create owner user
	owner := createTestUser(t, store, "owner@example.com", "owner")

	// Create account_collaborators entry for owner
	createTestAccountCollaborator(t, db, owner.ID, owner.ID, "owner")

	// Create a session for the owner
	sessionID, _ := createTestSession(t, store, owner.ID)

	// Try to invite self
	inviteInput := map[string]string{
		"email": "owner@example.com",
	}
	body, _ := json.Marshal(inviteInput)

	// Make request
	req := httptest.NewRequest(http.MethodPost, "/api/v1/collaborators/invite", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID})
	req = req.WithContext(context.WithValue(req.Context(), userCtxKey, owner))

	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

// TestHandleInviteCollaboratorIdempotent tests that re-inviting the same email updates expiry
func TestHandleInviteCollaboratorIdempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db, store, _, server := setupTestServerWithDB(t)
	defer db.Close()

	// Create owner user
	owner := createTestUser(t, store, "owner@example.com", "owner")

	// Create account_collaborators entry for owner
	createTestAccountCollaborator(t, db, owner.ID, owner.ID, "owner")

	// Create a session for the owner
	sessionID, _ := createTestSession(t, store, owner.ID)

	// First invitation
	inviteInput := map[string]string{
		"email": "someone@example.com",
	}
	body, _ := json.Marshal(inviteInput)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/collaborators/invite", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID})
	req = req.WithContext(context.WithValue(req.Context(), userCtxKey, owner))

	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("first invite failed: expected 201, got %d", w.Code)
	}

	// Second invitation to same email
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/collaborators/invite", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	req2.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID})
	req2 = req2.WithContext(context.WithValue(req2.Context(), userCtxKey, owner))

	w2 := httptest.NewRecorder()
	server.ServeHTTP(w2, req2)

	if w2.Code != http.StatusCreated {
		t.Errorf("second invite failed: expected 201, got %d", w2.Code)
	}

	// Verify only one invitation exists (updated, not duplicated)
	var count int
	db.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM invitations WHERE email = $1 AND account_id = $2 AND expires_at > NOW()
	`, "someone@example.com", owner.ID).Scan(&count)

	if count != 1 {
		t.Errorf("expected 1 invitation after re-invite, found %d", count)
	}
}

// TestHandleInviteCollaboratorResponseFormat tests that response has correct format
func TestHandleInviteCollaboratorResponseFormat(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db, store, _, server := setupTestServerWithDB(t)
	defer db.Close()

	// Create owner user
	owner := createTestUser(t, store, "owner@example.com", "owner")

	// Create account_collaborators entry for owner
	createTestAccountCollaborator(t, db, owner.ID, owner.ID, "owner")

	// Create a session for the owner
	sessionID, _ := createTestSession(t, store, owner.ID)

	// Make invitation
	inviteInput := map[string]string{
		"email": "someone@example.com",
	}
	body, _ := json.Marshal(inviteInput)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/collaborators/invite", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID})
	req = req.WithContext(context.WithValue(req.Context(), userCtxKey, owner))

	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	// Check required fields
	requiredFields := []string{"invite_id", "email", "expires_at"}
	for _, field := range requiredFields {
		if resp[field] == nil {
			t.Errorf("response missing required field: %s", field)
		}
	}

	// Verify expires_at is in RFC3339 format
	if expiresAt, ok := resp["expires_at"].(string); ok {
		if _, err := time.Parse(time.RFC3339, expiresAt); err != nil {
			t.Errorf("expires_at not in RFC3339 format: %s", expiresAt)
		}
	}
}

// Helper functions

func setupTestServerWithDB(t *testing.T) (*pgxpool.Pool, *authstore.Store, *biz.Store, *Server) {
	// This is a simplified setup - in real tests you'd use a test database
	// For now, we'll create a mock server
	db := pgxpool.New(context.Background(), "")
	store := authstore.New(db)
	bizStore := biz.New(db)

	server := &Server{
		db:  db,
		biz: bizStore,
	}
	return db, store, bizStore, server
}

func createTestUser(t *testing.T, store *authstore.Store, email, displayName string) *authstore.User {
	user := &authstore.User{
		ID:          uuid.New(),
		Email:       email,
		DisplayName: displayName,
	}
	return user
}

func createTestSession(t *testing.T, store *authstore.Store, userID uuid.UUID) (string, error) {
	sessionID := uuid.New().String()
	return sessionID, nil
}

func createTestAccountCollaborator(t *testing.T, db *pgxpool.Pool, accountID, userID uuid.UUID, role string) {
	_, err := db.Exec(context.Background(), `
		INSERT INTO account_collaborators (id, account_id, user_id, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, uuid.New(), accountID, userID, role, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("failed to create account collaborator: %v", err)
	}
}
