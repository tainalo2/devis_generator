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
	"github.com/tainalo2/devis_generator/backend/internal/gmailmail"
	"github.com/tainalo2/devis_generator/backend/internal/paymentsecrets"
)

type fakeGmailMail struct {
	authorizationURL string
	callbackState    string
	callbackCode     string
	sent             gmailmail.Message
	connected        bool
	disconnected     bool
}

func (f *fakeGmailMail) AuthorizationURL(uuid.UUID) (string, error) {
	return f.authorizationURL, nil
}

func (f *fakeGmailMail) HandleCallback(_ context.Context, _ paymentsecrets.Actor, _ uuid.UUID, state, code string) error {
	f.callbackState, f.callbackCode = state, code
	f.connected = true
	return nil
}

func (f *fakeGmailMail) Send(_ context.Context, _ paymentsecrets.Actor, _ uuid.UUID, message gmailmail.Message) error {
	f.sent = message
	return nil
}

func (f *fakeGmailMail) Status(context.Context, paymentsecrets.Actor, uuid.UUID) (bool, error) {
	return f.connected, nil
}

func (f *fakeGmailMail) Disconnect(context.Context, paymentsecrets.Actor, uuid.UUID) error {
	f.connected = false
	f.disconnected = true
	return nil
}

func gmailRequest(method, target string, body []byte, userID uuid.UUID) *http.Request {
	req := httptest.NewRequest(method, target, bytes.NewReader(body))
	return req.WithContext(context.WithValue(req.Context(), ctxUserKey, authstore.User{ID: userID, CreatedAt: time.Now()}))
}

func TestGmailOAuthHandlers(t *testing.T) {
	owner := uuid.New()
	gmail := &fakeGmailMail{authorizationURL: "https://accounts.example.test/authorize", connected: true}
	s := &Server{gmailMail: gmail}

	rec := httptest.NewRecorder()
	s.handleGmailAuthorize(rec, gmailRequest(http.MethodGet, "/", nil, owner))
	if rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte(gmail.authorizationURL)) {
		t.Fatalf("authorize: %d %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	s.handleGmailCallback(rec, gmailRequest(http.MethodGet, "/?state=signed&code=oauth-code", nil, owner))
	if rec.Code != http.StatusOK || gmail.callbackState != "signed" || gmail.callbackCode != "oauth-code" {
		t.Fatalf("callback: %d %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	s.handleGmailStatus(rec, gmailRequest(http.MethodGet, "/", nil, owner))
	if rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte(`"connected":true`)) {
		t.Fatalf("status: %d %s", rec.Code, rec.Body.String())
	}

	payload, _ := json.Marshal(gmailmail.Message{To: []string{"client@example.test"}, Subject: "Relance", HTML: "<p>Bonjour</p>"})
	rec = httptest.NewRecorder()
	s.handleGmailSend(rec, gmailRequest(http.MethodPost, "/", payload, owner))
	if rec.Code != http.StatusAccepted || gmail.sent.Subject != "Relance" {
		t.Fatalf("send: %d %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	s.handleGmailDisconnect(rec, gmailRequest(http.MethodDelete, "/", nil, owner))
	if rec.Code != http.StatusOK || !gmail.disconnected {
		t.Fatalf("disconnect: %d %s", rec.Code, rec.Body.String())
	}
}

func TestGmailRoutesRequireAuthentication(t *testing.T) {
	opts := testOptions()
	opts.GmailMail = &fakeGmailMail{authorizationURL: "https://example.test"}
	s := New(opts)
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/email/gmail/authorize"},
		{http.MethodGet, "/api/v1/email/gmail/callback?state=x&code=y"},
		{http.MethodGet, "/api/v1/email/gmail/status"},
		{http.MethodPost, "/api/v1/email/gmail/send"},
		{http.MethodDelete, "/api/v1/email/gmail"},
	} {
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, nil))
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s status=%d body=%s", tc.method, tc.path, rec.Code, rec.Body.String())
		}
	}
}
