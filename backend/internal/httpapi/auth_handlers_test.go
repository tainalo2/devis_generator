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

func TestAPIKeyAuthenticatesAndEnforcesScopes(t *testing.T) {
	user := authstore.User{ID: uuid.New(), Email: "api@example.test"}
	key := authstore.APIKey{ID: uuid.New(), UserID: user.ID, Scopes: []string{"read"}}
	s := &Server{
		apiKeyLookup: func(_ context.Context, hash string) (authstore.APIKey, authstore.User, error) {
			if hash != auth.HashToken("secret-api-key") {
				t.Fatalf("unexpected key hash %q", hash)
			}
			return key, user, nil
		},
	}

	handler := s.withOptionalSession(s.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		authed, ok := UserFromContext(r.Context())
		if !ok || authed.ID != user.ID {
			t.Fatal("API key user missing from context")
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	get := httptest.NewRequest(http.MethodGet, "/api/v1/documents", nil)
	get.Header.Set("Authorization", "Bearer secret-api-key")
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, get)
	if getRec.Code != http.StatusNoContent {
		t.Fatalf("read scope GET status=%d body=%s", getRec.Code, getRec.Body.String())
	}

	post := httptest.NewRequest(http.MethodPost, "/api/v1/documents", nil)
	post.Header.Set("Authorization", "Bearer secret-api-key")
	postRec := httptest.NewRecorder()
	handler.ServeHTTP(postRec, post)
	if postRec.Code != http.StatusForbidden {
		t.Fatalf("read scope POST status=%d want 403", postRec.Code)
	}
}

func TestAuthRateLimiter(t *testing.T) {
	rl := NewRateLimiter(2, time.Minute)
	if !rl.Allow("k") || !rl.Allow("k") || rl.Allow("k") {
		t.Fatal("expected third call denied")
	}
}

func TestLoginRateLimitsByIPAndAccount(t *testing.T) {
	body := `{"email":"limited@example.test","password":"valid-password"}`

	accountLimited := &Server{
		authLimiter:    NewRateLimiter(10, time.Minute),
		accountLimiter: NewRateLimiter(1, time.Minute),
	}
	if !accountLimited.accountLimiter.Allow("login:limited@example.test") {
		t.Fatal("failed to prime account limiter")
	}
	accountReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	accountRec := httptest.NewRecorder()
	accountLimited.handleLogin(accountRec, accountReq)
	if accountRec.Code != http.StatusTooManyRequests {
		t.Fatalf("account tier status=%d body=%s", accountRec.Code, accountRec.Body.String())
	}

	ipLimited := &Server{
		authLimiter:    NewRateLimiter(1, time.Minute),
		accountLimiter: NewRateLimiter(10, time.Minute),
	}
	if !ipLimited.authLimiter.Allow("login:192.0.2.1") {
		t.Fatal("failed to prime IP limiter")
	}
	ipReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	ipRec := httptest.NewRecorder()
	ipLimited.handleLogin(ipRec, ipReq)
	if ipRec.Code != http.StatusTooManyRequests {
		t.Fatalf("IP tier status=%d body=%s", ipRec.Code, ipRec.Body.String())
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
	opts := testOptions()
	opts.Version = "test"
	opts.SessionTTL = time.Hour
	opts.Argon2 = auth.Argon2Params{Time: 1, Memory: 8 * 1024, Threads: 1, KeyLen: 16}
	s := New(opts)
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
	opts := testOptions()
	opts.Version = "test"
	opts.SessionTTL = time.Hour
	opts.Argon2 = auth.Argon2Params{Time: 1, Memory: 8 * 1024, Threads: 1, KeyLen: 16}
	s := New(opts)
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
