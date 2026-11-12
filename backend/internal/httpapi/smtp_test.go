package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/tainalo2/devis_generator/backend/internal/authstore"
	"github.com/tainalo2/devis_generator/backend/internal/paymentsecrets"
	"github.com/tainalo2/devis_generator/backend/internal/smtpconfig"
)

type apiSMTPStore struct {
	config  smtpconfig.Config
	deleted bool
}

func (s *apiSMTPStore) Get(_ context.Context, _ uuid.UUID) (smtpconfig.Config, error) {
	if s.deleted || s.config.OwnerID == uuid.Nil {
		return smtpconfig.Config{}, smtpconfig.ErrNotFound
	}
	return s.config, nil
}
func (s *apiSMTPStore) Upsert(_ context.Context, config smtpconfig.Config) (smtpconfig.Config, error) {
	s.config = config
	return config, nil
}
func (s *apiSMTPStore) Delete(_ context.Context, _ uuid.UUID) error {
	s.deleted = true
	return nil
}

type apiSMTPVault struct{}

func (apiSMTPVault) Put(_ context.Context, _ paymentsecrets.Actor, ownerID uuid.UUID, kind paymentsecrets.Kind, reference string, _ []byte) (paymentsecrets.Metadata, error) {
	return paymentsecrets.Metadata{OwnerID: ownerID, Kind: kind, Reference: reference}, nil
}
func (apiSMTPVault) Delete(_ context.Context, _ paymentsecrets.Actor, _ uuid.UUID, _ paymentsecrets.Kind, _ string) error {
	return nil
}

type apiSMTPTester struct{ calls int }

func (t *apiSMTPTester) Test(_ context.Context, _ smtpconfig.Config, _ string) error {
	t.calls++
	return nil
}

func TestSMTPConfigRoutesCreateGetAndDeleteWithoutReturningPassword(t *testing.T) {
	store := &apiSMTPStore{}
	tester := &apiSMTPTester{}
	service := smtpconfig.NewService(store, apiSMTPVault{}, tester)
	srv := New(Options{SMTP: service})
	user := authstore.User{ID: uuid.New()}

	postBody := `{"host":"smtp.example.com","port":587,"username":"mailer@example.com","password":"super-secret","from_address":"mailer@example.com","tls_mode":"starttls"}`
	post := httptest.NewRequest(http.MethodPost, "/api/v1/email/smtp", strings.NewReader(postBody))
	post.Header.Set("Content-Type", "application/json")
	post = post.WithContext(context.WithValue(post.Context(), ctxUserKey, user))
	postRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(postRec, post)
	if postRec.Code != http.StatusOK {
		t.Fatalf("POST status=%d body=%s", postRec.Code, postRec.Body.String())
	}
	if tester.calls != 1 {
		t.Fatalf("test calls=%d", tester.calls)
	}
	if bytes.Contains(postRec.Body.Bytes(), []byte("super-secret")) || bytes.Contains(postRec.Body.Bytes(), []byte(`"password"`)) {
		t.Fatalf("POST leaked password: %s", postRec.Body.String())
	}

	get := httptest.NewRequest(http.MethodGet, "/api/v1/email/smtp", nil)
	get = get.WithContext(context.WithValue(get.Context(), ctxUserKey, user))
	getRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(getRec, get)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET status=%d body=%s", getRec.Code, getRec.Body.String())
	}
	if bytes.Contains(getRec.Body.Bytes(), []byte("super-secret")) || bytes.Contains(getRec.Body.Bytes(), []byte(`"password"`)) {
		t.Fatalf("GET leaked password: %s", getRec.Body.String())
	}
	var response struct {
		Config smtpconfig.Config `json:"config"`
	}
	if err := json.Unmarshal(getRec.Body.Bytes(), &response); err != nil || !response.Config.PasswordConfigured {
		t.Fatalf("GET response=%s err=%v", getRec.Body.String(), err)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/v1/email/smtp", nil)
	deleteReq = deleteReq.WithContext(context.WithValue(deleteReq.Context(), ctxUserKey, user))
	deleteRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK || !store.deleted {
		t.Fatalf("DELETE status=%d body=%s deleted=%v", deleteRec.Code, deleteRec.Body.String(), store.deleted)
	}
}

func TestSMTPConfigGetReturnsNotFound(t *testing.T) {
	service := smtpconfig.NewService(&apiSMTPStore{}, apiSMTPVault{}, &apiSMTPTester{})
	srv := New(Options{SMTP: service})
	user := authstore.User{ID: uuid.New()}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/email/smtp", nil)
	req = req.WithContext(context.WithValue(req.Context(), ctxUserKey, user))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
