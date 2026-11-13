package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/tainalo2/devis_generator/backend/internal/authstore"
	"github.com/tainalo2/devis_generator/backend/internal/microsoftmail"
	"github.com/tainalo2/devis_generator/backend/internal/paymentsecrets"
)

type fakeMicrosoftMail struct {
	authorizationURL string
	callbackState    string
	callbackCode     string
	sent             microsoftmail.Message
	connected        bool
	disconnected     bool
}

func (f *fakeMicrosoftMail) AuthorizationURL(uuid.UUID) (string, error) {
	return f.authorizationURL, nil
}
func (f *fakeMicrosoftMail) HandleCallback(_ context.Context, _ paymentsecrets.Actor, _ uuid.UUID, state, code string) error {
	f.callbackState, f.callbackCode = state, code
	f.connected = true
	return nil
}
func (f *fakeMicrosoftMail) Send(_ context.Context, _ paymentsecrets.Actor, _ uuid.UUID, message microsoftmail.Message) error {
	f.sent = message
	return nil
}
func (f *fakeMicrosoftMail) Status(context.Context, paymentsecrets.Actor, uuid.UUID) (bool, error) {
	return f.connected, nil
}
func (f *fakeMicrosoftMail) Disconnect(context.Context, paymentsecrets.Actor, uuid.UUID) error {
	f.connected = false
	f.disconnected = true
	return nil
}

func microsoftRequest(method, target string, body []byte, userID uuid.UUID) *http.Request {
	req := httptest.NewRequest(method, target, bytes.NewReader(body))
	return req.WithContext(context.WithValue(req.Context(), ctxUserKey, authstore.User{ID: userID, CreatedAt: time.Now()}))
}

func TestMicrosoftOAuthHandlers(t *testing.T) {
	owner := uuid.New()
	mail := &fakeMicrosoftMail{authorizationURL: "https://login.example.test/authorize", connected: true}
	s := &Server{microsoftMail: mail}

	rec := httptest.NewRecorder()
	s.handleMicrosoftAuthorize(rec, microsoftRequest(http.MethodGet, "/", nil, owner))
	if rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte(mail.authorizationURL)) {
		t.Fatalf("authorize: %d %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	s.handleMicrosoftCallback(rec, microsoftRequest(http.MethodGet, "/?state=signed&code=oauth-code", nil, owner))
	if rec.Code != http.StatusOK || mail.callbackState != "signed" || mail.callbackCode != "oauth-code" {
		t.Fatalf("callback: %d %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	s.handleMicrosoftStatus(rec, microsoftRequest(http.MethodGet, "/", nil, owner))
	if rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte(`"connected":true`)) {
		t.Fatalf("status: %d %s", rec.Code, rec.Body.String())
	}

	payload, _ := json.Marshal(microsoftmail.Message{To: []string{"client@example.test"}, Subject: "Relance", HTML: "<p>Bonjour</p>"})
	rec = httptest.NewRecorder()
	s.handleMicrosoftSend(rec, microsoftRequest(http.MethodPost, "/", payload, owner))
	if rec.Code != http.StatusAccepted || mail.sent.Subject != "Relance" {
		t.Fatalf("send: %d %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	s.handleMicrosoftDisconnect(rec, microsoftRequest(http.MethodDelete, "/", nil, owner))
	if rec.Code != http.StatusOK || !mail.disconnected {
		t.Fatalf("disconnect: %d %s", rec.Code, rec.Body.String())
	}
}

func TestMicrosoftRoutesRequireAuthentication(t *testing.T) {
	s := New(Options{MicrosoftMail: &fakeMicrosoftMail{authorizationURL: "https://example.test"}})
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/email/microsoft/authorize"},
		{http.MethodGet, "/api/v1/email/microsoft/callback?state=x&code=y"},
		{http.MethodGet, "/api/v1/email/microsoft/status"},
		{http.MethodPost, "/api/v1/email/microsoft/send"},
		{http.MethodDelete, "/api/v1/email/microsoft"},
	} {
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, nil))
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s status=%d body=%s", tc.method, tc.path, rec.Code, rec.Body.String())
		}
	}
}
