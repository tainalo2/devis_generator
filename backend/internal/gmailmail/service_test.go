package gmailmail

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"net/mail"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/tainalo2/devis_generator/backend/internal/paymentsecrets"
)

type gmailVault struct {
	secret  []byte
	puts    int
	deletes int
	kind    paymentsecrets.Kind
}

func (v *gmailVault) Put(_ context.Context, _ paymentsecrets.Actor, _ uuid.UUID, kind paymentsecrets.Kind, reference string, plaintext []byte) (paymentsecrets.Metadata, error) {
	if kind != paymentsecrets.KindOAuthGmailToken || reference != tokenReference {
		return paymentsecrets.Metadata{}, errors.New("unexpected secret coordinates")
	}
	v.kind = kind
	v.secret = append([]byte(nil), plaintext...)
	v.puts++
	return paymentsecrets.Metadata{}, nil
}

func (v *gmailVault) Reveal(_ context.Context, _ paymentsecrets.Actor, _ uuid.UUID, kind paymentsecrets.Kind, reference string) ([]byte, paymentsecrets.Metadata, error) {
	if kind != paymentsecrets.KindOAuthGmailToken || reference != tokenReference {
		return nil, paymentsecrets.Metadata{}, errors.New("unexpected secret coordinates")
	}
	if len(v.secret) == 0 {
		return nil, paymentsecrets.Metadata{}, paymentsecrets.ErrNotFound
	}
	return append([]byte(nil), v.secret...), paymentsecrets.Metadata{}, nil
}

func (v *gmailVault) Delete(_ context.Context, _ paymentsecrets.Actor, _ uuid.UUID, kind paymentsecrets.Kind, reference string) error {
	if kind != paymentsecrets.KindOAuthGmailToken || reference != tokenReference {
		return errors.New("unexpected secret coordinates")
	}
	if len(v.secret) == 0 {
		return paymentsecrets.ErrNotFound
	}
	v.secret = nil
	v.deletes++
	return nil
}

func gmailTestConfig(serverURL string) Config {
	return Config{
		ClientID: "client-id", ClientSecret: "client-secret",
		RedirectURL:  "https://app.example.test/api/v1/email/gmail/callback",
		AuthorizeURL: serverURL + "/authorize", TokenURL: serverURL + "/token",
		GmailBaseURL: serverURL, StateKey: []byte("state-signing-key-at-least-32-bytes"),
	}
}

func TestAuthorizationURLRequestsOnlyGmailSendAndOfflineAccess(t *testing.T) {
	svc, err := New(gmailTestConfig("https://accounts.example.test"), &gmailVault{}, http.DefaultClient)
	if err != nil {
		t.Fatal(err)
	}
	owner := uuid.New()
	raw, err := svc.AuthorizationURL(owner)
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	if q.Get("client_id") != "client-id" || q.Get("response_type") != "code" {
		t.Fatalf("unexpected authorize query: %s", u.RawQuery)
	}
	if q.Get("scope") != gmailSendScope {
		t.Fatalf("scope = %q", q.Get("scope"))
	}
	if q.Get("access_type") != "offline" || q.Get("prompt") != "consent" || q.Get("include_granted_scopes") != "false" {
		t.Fatalf("offline authorization parameters missing: %s", u.RawQuery)
	}
	for _, forbidden := range []string{"gmail.readonly", "gmail.modify", "mail.google.com"} {
		if strings.Contains(q.Get("scope"), forbidden) {
			t.Fatalf("forbidden scope %s", forbidden)
		}
	}
	if err := svc.ValidateState(q.Get("state"), owner); err != nil {
		t.Fatalf("state: %v", err)
	}
	if err := svc.ValidateState(q.Get("state"), uuid.New()); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("state must reject another user: %v", err)
	}
}

func TestCallbackExchangesCodeAndStoresOnlyRefreshToken(t *testing.T) {
	var tokenForm url.Values
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/token" {
			http.NotFound(w, r)
			return
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		tokenForm = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"access_token":"access","refresh_token":"refresh","token_type":"Bearer","expires_in":3600}`)
	}))
	defer upstream.Close()
	vault := &gmailVault{}
	svc, err := New(gmailTestConfig(upstream.URL), vault, upstream.Client())
	if err != nil {
		t.Fatal(err)
	}
	owner := uuid.New()
	authURL, _ := svc.AuthorizationURL(owner)
	parsed, _ := url.Parse(authURL)
	actor := paymentsecrets.Actor{UserID: owner, RequestID: "callback-request"}
	if err := svc.HandleCallback(context.Background(), actor, owner, parsed.Query().Get("state"), "auth-code"); err != nil {
		t.Fatal(err)
	}
	if tokenForm.Get("grant_type") != "authorization_code" || tokenForm.Get("code") != "auth-code" || tokenForm.Get("redirect_uri") != gmailTestConfig(upstream.URL).RedirectURL {
		t.Fatalf("unexpected token form: %v", tokenForm)
	}
	if string(vault.secret) != "refresh" || vault.kind != paymentsecrets.KindOAuthGmailToken {
		t.Fatalf("stored secret = %q kind=%q", vault.secret, vault.kind)
	}
	if err := svc.HandleCallback(context.Background(), actor, owner, "tampered", "auth-code"); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("tampered state err = %v", err)
	}
}

func TestSendRefreshesAccessTokenAndCallsGmailMessagesSend(t *testing.T) {
	var refreshForm url.Values
	var authorization string
	var requestBody struct {
		Raw string `json:"raw"`
	}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			r.ParseForm()
			refreshForm = r.PostForm
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, `{"access_token":"fresh-access","refresh_token":"rotated-refresh","token_type":"Bearer","expires_in":3600}`)
		case "/gmail/v1/users/me/messages/send":
			authorization = r.Header.Get("Authorization")
			if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
				t.Fatal(err)
			}
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, `{"id":"gmail-message-id"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()
	vault := &gmailVault{secret: []byte("old-refresh")}
	svc, _ := New(gmailTestConfig(upstream.URL), vault, upstream.Client())
	owner := uuid.New()
	message := Message{To: []string{"client@example.test"}, Subject: "Relance facture n° 42", HTML: "<p>Bonjour</p>"}
	if err := svc.Send(context.Background(), paymentsecrets.Actor{UserID: owner}, owner, message); err != nil {
		t.Fatal(err)
	}
	if refreshForm.Get("grant_type") != "refresh_token" || refreshForm.Get("refresh_token") != "old-refresh" {
		t.Fatalf("refresh form: %v", refreshForm)
	}
	if authorization != "Bearer fresh-access" {
		t.Fatalf("authorization = %q", authorization)
	}
	if string(vault.secret) != "rotated-refresh" || vault.puts != 1 {
		t.Fatalf("refresh rotation not stored: %q puts=%d", vault.secret, vault.puts)
	}
	raw, err := base64.RawURLEncoding.DecodeString(requestBody.Raw)
	if err != nil {
		t.Fatalf("decode raw message: %v", err)
	}
	parsed, err := mail.ReadMessage(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatalf("parse raw message: %v\n%s", err, raw)
	}
	decodedSubject, err := new(mime.WordDecoder).DecodeHeader(parsed.Header.Get("Subject"))
	if err != nil {
		t.Fatalf("decode subject: %v", err)
	}
	to, err := mail.ParseAddress(parsed.Header.Get("To"))
	if err != nil {
		t.Fatalf("decode recipient: %v", err)
	}
	if to.Address != "client@example.test" || decodedSubject != "Relance facture n° 42" {
		t.Fatalf("headers: %#v", parsed.Header)
	}
	body, _ := io.ReadAll(parsed.Body)
	if !strings.Contains(string(body), "<p>Bonjour</p>") {
		t.Fatalf("body = %q", body)
	}
}

func TestSendRejectsHeaderInjectionBeforeCallingGoogle(t *testing.T) {
	called := false
	upstream := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	defer upstream.Close()
	svc, _ := New(gmailTestConfig(upstream.URL), &gmailVault{secret: []byte("refresh")}, upstream.Client())
	owner := uuid.New()
	err := svc.Send(context.Background(), paymentsecrets.Actor{UserID: owner}, owner, Message{
		To: []string{"victim@example.test\r\nBcc: attacker@example.test"}, Subject: "Relance", HTML: "body",
	})
	if err == nil {
		t.Fatal("expected invalid recipient error")
	}
	if called {
		t.Fatal("Google must not be called for an invalid message")
	}
}

func TestStatusDisconnectAndStateExpiry(t *testing.T) {
	vault := &gmailVault{secret: []byte("refresh")}
	svc, _ := New(gmailTestConfig("https://example.test"), vault, http.DefaultClient)
	owner := uuid.New()
	actor := paymentsecrets.Actor{UserID: owner}
	connected, err := svc.Status(context.Background(), actor, owner)
	if err != nil || !connected {
		t.Fatalf("status connected=%v err=%v", connected, err)
	}
	svc.now = func() time.Time { return time.Unix(1000, 0) }
	raw, _ := svc.AuthorizationURL(owner)
	parsed, _ := url.Parse(raw)
	state := parsed.Query().Get("state")
	svc.now = func() time.Time { return time.Unix(1000, 0).Add(stateTTL + time.Second) }
	if err := svc.ValidateState(state, owner); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("expired state err=%v", err)
	}
	if err := svc.Disconnect(context.Background(), actor, owner); err != nil {
		t.Fatal(err)
	}
	connected, err = svc.Status(context.Background(), actor, owner)
	if err != nil || connected || vault.deletes != 1 {
		t.Fatalf("after disconnect connected=%v deletes=%d err=%v", connected, vault.deletes, err)
	}
}
