package microsoftmail

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/tainalo2/devis_generator/backend/internal/paymentsecrets"
)

type memoryVault struct {
	secret  []byte
	puts    int
	deletes int
}

func (v *memoryVault) Put(_ context.Context, _ paymentsecrets.Actor, _ uuid.UUID, kind paymentsecrets.Kind, reference string, plaintext []byte) (paymentsecrets.Metadata, error) {
	if kind != paymentsecrets.KindOAuthMicrosoftToken || reference != tokenReference {
		return paymentsecrets.Metadata{}, errors.New("unexpected secret coordinates")
	}
	v.secret = append([]byte(nil), plaintext...)
	v.puts++
	return paymentsecrets.Metadata{}, nil
}

func (v *memoryVault) Reveal(_ context.Context, _ paymentsecrets.Actor, _ uuid.UUID, _ paymentsecrets.Kind, _ string) ([]byte, paymentsecrets.Metadata, error) {
	if len(v.secret) == 0 {
		return nil, paymentsecrets.Metadata{}, paymentsecrets.ErrNotFound
	}
	return append([]byte(nil), v.secret...), paymentsecrets.Metadata{}, nil
}

func (v *memoryVault) Delete(_ context.Context, _ paymentsecrets.Actor, _ uuid.UUID, _ paymentsecrets.Kind, _ string) error {
	if len(v.secret) == 0 {
		return paymentsecrets.ErrNotFound
	}
	v.secret = nil
	v.deletes++
	return nil
}

func testConfig(serverURL string) Config {
	return Config{
		ClientID: "client-id", ClientSecret: "client-secret",
		RedirectURL:  "https://app.example.test/api/v1/email/microsoft/callback",
		AuthorizeURL: serverURL + "/authorize", TokenURL: serverURL + "/token",
		GraphBaseURL: serverURL, StateKey: []byte("state-signing-key-at-least-32-bytes"),
	}
}

func TestAuthorizationURLUsesOnlyMailSendAndBindsStateToUser(t *testing.T) {
	svc, err := New(testConfig("https://login.example.test"), &memoryVault{}, http.DefaultClient)
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
	if q.Get("client_id") != "client-id" || q.Get("response_type") != "code" || q.Get("response_mode") != "query" {
		t.Fatalf("unexpected authorize query: %s", u.RawQuery)
	}
	if q.Get("scope") != "offline_access https://graph.microsoft.com/Mail.Send" {
		t.Fatalf("scope = %q", q.Get("scope"))
	}
	for _, forbidden := range []string{"Mail.Read", "Mail.ReadWrite", "User.Read"} {
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

func TestCallbackValidatesStateExchangesCodeAndStoresRefreshToken(t *testing.T) {
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
	vault := &memoryVault{}
	svc, err := New(testConfig(upstream.URL), vault, upstream.Client())
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
	if tokenForm.Get("grant_type") != "authorization_code" || tokenForm.Get("code") != "auth-code" {
		t.Fatalf("unexpected token form: %v", tokenForm)
	}
	if string(vault.secret) != "refresh" {
		t.Fatalf("stored secret = %q", vault.secret)
	}
	if err := svc.HandleCallback(context.Background(), actor, owner, "tampered", "auth-code"); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("tampered state err = %v", err)
	}
}

func TestSendRefreshesTokenThenPostsGraphMailAndStoresRotation(t *testing.T) {
	var refreshForm url.Values
	var authorization string
	var graphBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			r.ParseForm()
			refreshForm = r.PostForm
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, `{"access_token":"fresh-access","refresh_token":"rotated-refresh","token_type":"Bearer","expires_in":3600}`)
		case "/v1.0/me/sendMail":
			authorization = r.Header.Get("Authorization")
			if err := json.NewDecoder(r.Body).Decode(&graphBody); err != nil {
				t.Fatal(err)
			}
			w.WriteHeader(http.StatusAccepted)
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()
	vault := &memoryVault{secret: []byte("old-refresh")}
	svc, _ := New(testConfig(upstream.URL), vault, upstream.Client())
	owner := uuid.New()
	msg := Message{To: []string{"client@example.test"}, Subject: "Relance", HTML: "<p>Bonjour</p>"}
	if err := svc.Send(context.Background(), paymentsecrets.Actor{UserID: owner}, owner, msg); err != nil {
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
	message := graphBody["message"].(map[string]any)
	if message["subject"] != "Relance" {
		t.Fatalf("graph body: %#v", graphBody)
	}
}

func TestStatusAndDisconnect(t *testing.T) {
	vault := &memoryVault{secret: []byte("refresh")}
	svc, _ := New(testConfig("https://example.test"), vault, http.DefaultClient)
	owner := uuid.New()
	actor := paymentsecrets.Actor{UserID: owner}
	connected, err := svc.Status(context.Background(), actor, owner)
	if err != nil || !connected {
		t.Fatalf("status connected=%v err=%v", connected, err)
	}
	if err := svc.Disconnect(context.Background(), actor, owner); err != nil {
		t.Fatal(err)
	}
	connected, err = svc.Status(context.Background(), actor, owner)
	if err != nil || connected {
		t.Fatalf("status after disconnect connected=%v err=%v", connected, err)
	}
	if vault.deletes != 1 {
		t.Fatalf("deletes=%d", vault.deletes)
	}
}

func TestStateExpires(t *testing.T) {
	svc, _ := New(testConfig("https://example.test"), &memoryVault{}, http.DefaultClient)
	svc.now = func() time.Time { return time.Unix(1000, 0) }
	owner := uuid.New()
	raw, _ := svc.AuthorizationURL(owner)
	parsed, _ := url.Parse(raw)
	state := parsed.Query().Get("state")
	svc.now = func() time.Time { return time.Unix(1000, 0).Add(stateTTL + time.Second) }
	if err := svc.ValidateState(state, owner); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("expired state err=%v", err)
	}
}
