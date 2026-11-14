package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/tainalo2/devis_generator/backend/internal/authstore"
)

func TestRequireSessionRejectsAPIKeyAuthentication(t *testing.T) {
	user := authstore.User{ID: uuid.New(), Email: "api@example.test"}
	key := authstore.APIKey{ID: uuid.New(), UserID: user.ID, Scopes: []string{"read", "write"}}
	s := &Server{}
	hit := false
	handler := s.requireSession(func(w http.ResponseWriter, _ *http.Request) {
		hit = true
		w.WriteHeader(http.StatusNoContent)
	})

	ctx := context.WithValue(context.Background(), ctxUserKey, user)
	ctx = context.WithValue(ctx, ctxAPIKeyKey, key)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/api-keys", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized || hit {
		t.Fatalf("API key auth status=%d hit=%v, want 401 and no handler call", rec.Code, hit)
	}

	sessionCtx := context.WithValue(context.Background(), ctxUserKey, user)
	sessionCtx = context.WithValue(sessionCtx, ctxSessionKey, authstore.Session{ID: uuid.New(), UserID: user.ID})
	sessionReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/api-keys", nil).WithContext(sessionCtx)
	sessionRec := httptest.NewRecorder()
	handler.ServeHTTP(sessionRec, sessionReq)
	if sessionRec.Code != http.StatusNoContent || !hit {
		t.Fatalf("session auth status=%d hit=%v, want 204 and handler call", sessionRec.Code, hit)
	}
}

func TestAPIKeyToDTOIncludesLastUsedAt(t *testing.T) {
	lastUsed := time.Date(2026, time.November, 14, 12, 34, 56, 0, time.FixedZone("CET", 60*60))
	key := authstore.APIKey{
		ID:        uuid.New(),
		Name:      "MCP préproduction",
		KeyPrefix: "abcdefgh",
		Scopes:    []string{"read", "write"},
		CreatedAt: time.Date(2026, time.November, 13, 8, 0, 0, 0, time.UTC),
		LastUsed:  &lastUsed,
	}

	dto := apiKeyToDTO(key)

	if dto.LastUsedAt == nil {
		t.Fatal("expected last_used_at to be exposed")
	}
	if got, want := *dto.LastUsedAt, "2026-11-14T11:34:56Z"; got != want {
		t.Fatalf("last_used_at = %q, want %q", got, want)
	}
}
