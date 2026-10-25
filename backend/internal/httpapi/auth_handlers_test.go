package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/tainalo2/devis_generator/backend/internal/auth"
	"github.com/tainalo2/devis_generator/backend/internal/authstore"
)

func TestNormalizeEmailAndPassword(t *testing.T) {
	email, err := normalizeEmail("  Alex@Example.COM ")
	if err != nil || email != "alex@example.com" {
		t.Fatalf("email=%q err=%v", email, err)
	}
	if validPassword("short") {
		t.Fatal("short password should be invalid")
	}
	if !validPassword("assezlong1") {
		t.Fatal("expected valid password")
	}
}

func TestAuthRateLimiter(t *testing.T) {
	rl := NewRateLimiter(2, time.Minute)
	if !rl.Allow("k") || !rl.Allow("k") || rl.Allow("k") {
		t.Fatal("expected third call denied")
	}
}

func TestCSRFRejectedWhenSessionPresent(t *testing.T) {
	// Build a server without store (nil) — only exercise CSRF middleware via custom route.
	s := &Server{
		cookieSecure: false,
		sessionTTL:   time.Hour,
		argon2:       auth.Argon2Params{Time: 1, Memory: 8 * 1024, Threads: 1, KeyLen: 16},
		authLimiter:  NewRateLimiter(100, time.Minute),
	}

	var hit bool
	h := s.requireCSRF(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hit = true
		w.WriteHeader(http.StatusNoContent)
	}))

	uid := uuid.New()
	sess := authstore.Session{ID: uuid.New(), UserID: uid, CSRFToken: "csrf-secret"}
	ctx := context.WithValue(context.Background(), ctxSessionKey, sess)
	req := httptest.NewRequest(http.MethodPost, "/x", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden || hit {
		t.Fatalf("expected CSRF forbid, code=%d hit=%v", rec.Code, hit)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/x", nil).WithContext(ctx)
	req2.Header.Set(csrfHeaderName, "csrf-secret")
	req2.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "csrf-secret"})
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusNoContent || !hit {
		t.Fatalf("expected CSRF pass, code=%d hit=%v", rec2.Code, hit)
	}
}

func TestPasswordResetStubNoEnumeration(t *testing.T) {
	s := New(Options{
		Version:    "test",
		SessionTTL: time.Hour,
		Argon2:     auth.Argon2Params{Time: 1, Memory: 8 * 1024, Threads: 1, KeyLen: 16},
	})
	body := bytes.NewBufferString(`{"email":"inconnu@example.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/password-reset/request", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Si un compte correspond") {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestRegisterValidationWithoutStore(t *testing.T) {
	s := New(Options{
		Version:    "test",
		SessionTTL: time.Hour,
		Argon2:     auth.Argon2Params{Time: 1, Memory: 8 * 1024, Threads: 1, KeyLen: 16},
	})
	body, _ := json.Marshal(map[string]string{"email": "bad", "password": "x"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	// Store is nil → would panic on valid input; invalid input should 400 before store.
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
