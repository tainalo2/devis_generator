package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/tainalo2/devis_generator/backend/internal/biz"
	"github.com/tainalo2/devis_generator/backend/internal/doclink"
)

// fakeSigner reproduces the store contract (single signature, signable states) so the
// handlers are exercised through the real router, not by calling them directly.
type fakeSigner struct {
	mu    sync.Mutex
	docs  map[uuid.UUID]biz.Document
	sigs  map[uuid.UUID]biz.Signature
	calls int
}

func (f *fakeSigner) SignQuote(_ context.Context, id uuid.UUID, in biz.SignInput) (biz.Signature, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	d, ok := f.docs[id]
	if !ok {
		return biz.Signature{}, biz.ErrNotFound
	}
	if _, signed := f.sigs[id]; signed {
		return biz.Signature{}, biz.ErrAlreadySigned
	}
	if d.Kind != "quote" || (d.Status != "issued" && d.Status != "sent") {
		return biz.Signature{}, biz.ErrNotSignable
	}
	if len(strings.TrimSpace(in.Name)) < 2 {
		return biz.Signature{}, biz.ErrInvalidInput
	}
	sig := biz.Signature{DocumentID: id, SignedAt: time.Now().UTC(), DocumentHash: strings.Repeat("a", 64), SignerName: in.Name, SignerIP: in.IP, SignerUserAgent: in.UserAgent}
	f.sigs[id] = sig
	d.Status = "accepted"
	f.docs[id] = d
	return sig, nil
}

func (f *fakeSigner) GetSignaturePublic(_ context.Context, id uuid.UUID) (biz.Signature, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if s, ok := f.sigs[id]; ok {
		return s, nil
	}
	return biz.Signature{}, biz.ErrNotFound
}

func signFixture(t *testing.T) (*Server, Options, *fakeSigner, biz.Document, biz.Document) {
	t.Helper()
	quote := testInvoice(t, "D-2026-0001")
	quote.Kind = doclink.KindQuote
	invoice := testInvoice(t, "F-2026-0001")
	pub := &fakePublicDocs{docs: map[uuid.UUID]biz.Document{quote.ID: quote, invoice.ID: invoice}, issuer: biz.IssuerProfile{LegalName: "AE"}}
	sg := &fakeSigner{docs: pub.docs, sigs: map[uuid.UUID]biz.Signature{}}
	opts := testOptions()
	opts.PublicDocs = pub
	opts.PublicSigner = sg
	return New(opts), opts, sg, quote, invoice
}

func doSign(srv *Server, token, body string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/public/documents/"+token+"/sign", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "198.51.100.9:4242"
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	return rec
}

func getSigState(srv *Server, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/documents/"+token+"/signature", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	return rec
}

func TestSignRouteSignsQuoteAndCapturesServerSideEvidence(t *testing.T) {
	srv, opts, sg, quote, _ := signFixture(t)
	tok := tokenFor(t, opts, quote.ID, doclink.KindQuote, quote.CreatedAt)
	rec := doSign(srv, tok, `{"name":"Jeanne Client"}`, map[string]string{"User-Agent": "TestUA/1.0"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body)
	}
	if rec.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("Cache-Control=%q", rec.Header().Get("Cache-Control"))
	}
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["signer_name"] != "Jeanne Client" || out["signed_at"] == nil {
		t.Fatalf("réponse: %v", out)
	}
	for _, leaked := range []string{"signer_ip", "signer_user_agent", "document_hash"} {
		if _, ok := out[leaked]; ok {
			t.Fatalf("la réponse publique ne doit pas exposer %s", leaked)
		}
	}
	stored := sg.sigs[quote.ID]
	if stored.SignerIP != "198.51.100.9" || stored.SignerUserAgent != "TestUA/1.0" {
		t.Fatalf("IP/UA doivent venir du serveur: %+v", stored)
	}
}

func TestSignRouteIgnoresClientSuppliedEvidence(t *testing.T) {
	srv, opts, sg, quote, _ := signFixture(t)
	tok := tokenFor(t, opts, quote.ID, doclink.KindQuote, quote.CreatedAt)
	rec := doSign(srv, tok, `{"name":"Jeanne Client","signer_ip":"1.2.3.4","document_hash":"x"}`, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("champs inconnus doivent être refusés: %d", rec.Code)
	}
	if sg.calls != 0 {
		t.Fatal("le store ne doit pas être appelé")
	}
}

func TestSignRouteSecondSignatureIs409(t *testing.T) {
	srv, opts, _, quote, _ := signFixture(t)
	tok := tokenFor(t, opts, quote.ID, doclink.KindQuote, quote.CreatedAt)
	if rec := doSign(srv, tok, `{"name":"Jeanne Client"}`, nil); rec.Code != http.StatusCreated {
		t.Fatalf("première signature: %d", rec.Code)
	}
	rec := doSign(srv, tok, `{"name":"Autre Personne"}`, nil)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status=%d, want 409", rec.Code)
	}
}

func TestSignRouteInvalidTokensAreUniform404(t *testing.T) {
	srv, opts, sg, quote, invoice := signFixture(t)
	good := tokenFor(t, opts, quote.ID, doclink.KindQuote, quote.CreatedAt)
	cases := map[string]string{
		"jeton falsifié": good + "x",
		"jeton vide":     "abc.def",
		"autre secret": func() string {
			tk, _ := doclink.Generate([]byte("autre-secret"), quote.ID.String(), doclink.KindQuote, 1, quote.CreatedAt)
			return tk
		}(),
		"document inconnu": tokenFor(t, opts, uuid.New(), doclink.KindQuote, time.Now()),
		"kind incohérent":  tokenFor(t, opts, quote.ID, doclink.KindInvoice, quote.CreatedAt),
		"devis expiré":     tokenFor(t, opts, quote.ID, doclink.KindQuote, time.Now().Add(-doclink.QuoteTTL-time.Hour)),
	}
	_ = invoice
	var ref string
	for name, tok := range cases {
		rec := doSign(srv, tok, `{"name":"Jeanne Client"}`, nil)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s: status=%d, want 404", name, rec.Code)
		}
		if ref == "" {
			ref = rec.Body.String()
		}
		if rec.Body.String() != ref {
			t.Fatalf("%s: corps différent (oracle): %q vs %q", name, rec.Body, ref)
		}
	}
	if sg.calls != 0 {
		t.Fatalf("aucune signature ne doit être tentée avec un jeton invalide (calls=%d)", sg.calls)
	}
}

func TestSignRouteInvoiceIsNotSignable(t *testing.T) {
	srv, opts, _, _, invoice := signFixture(t)
	tok := tokenFor(t, opts, invoice.ID, doclink.KindInvoice, invoice.CreatedAt)
	if rec := doSign(srv, tok, `{"name":"Jeanne Client"}`, nil); rec.Code != http.StatusConflict {
		t.Fatalf("status=%d, want 409", rec.Code)
	}
}

func TestSignRouteValidatesName(t *testing.T) {
	srv, opts, _, quote, _ := signFixture(t)
	tok := tokenFor(t, opts, quote.ID, doclink.KindQuote, quote.CreatedAt)
	for _, body := range []string{`{"name":""}`, `{"name":" "}`, `{"name":"A"}`, `{}`, `not json`, `{"name":123}`} {
		if rec := doSign(srv, tok, body, nil); rec.Code != http.StatusBadRequest {
			t.Fatalf("body=%s status=%d, want 400", body, rec.Code)
		}
	}
}

func TestSignRouteRejectsCrossOriginAndNonJSON(t *testing.T) {
	srv, opts, sg, quote, _ := signFixture(t)
	tok := tokenFor(t, opts, quote.ID, doclink.KindQuote, quote.CreatedAt)
	if rec := doSign(srv, tok, `{"name":"Jeanne Client"}`, map[string]string{"Origin": "https://evil.example"}); rec.Code != http.StatusForbidden {
		t.Fatalf("cross-origin: %d", rec.Code)
	}
	if rec := doSign(srv, tok, `{"name":"Jeanne Client"}`, map[string]string{"Content-Type": "text/plain"}); rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("non JSON: %d", rec.Code)
	}
	if sg.calls != 0 {
		t.Fatal("aucune signature ne doit passer")
	}
}

func TestSignRouteRateLimited(t *testing.T) {
	srv, opts, sg, quote, _ := signFixture(t)
	tok := tokenFor(t, opts, quote.ID, doclink.KindQuote, quote.CreatedAt)
	var last int
	for i := 0; i < 15; i++ {
		last = doSign(srv, tok, `{"name":"Jeanne Client"}`, nil).Code
	}
	if last != http.StatusTooManyRequests {
		t.Fatalf("dernier statut=%d, want 429", last)
	}
	if sg.calls > 10 {
		t.Fatalf("le limiteur doit couper avant le store (calls=%d)", sg.calls)
	}
}

func TestSignatureStateRouteReportsSignableThenSigned(t *testing.T) {
	srv, opts, _, quote, invoice := signFixture(t)
	tok := tokenFor(t, opts, quote.ID, doclink.KindQuote, quote.CreatedAt)
	rec := getSigState(srv, tok)
	var st map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &st)
	if rec.Code != 200 || st["signable"] != true || st["signed"] != false {
		t.Fatalf("avant: %d %v", rec.Code, st)
	}
	doSign(srv, tok, `{"name":"Jeanne Client"}`, nil)
	rec = getSigState(srv, tok)
	st = map[string]any{}
	_ = json.Unmarshal(rec.Body.Bytes(), &st)
	if st["signed"] != true || st["signable"] != false || st["signer_name"] != "Jeanne Client" || st["signed_at"] == nil {
		t.Fatalf("après: %v", st)
	}
	if _, ok := st["signer_ip"]; ok {
		t.Fatal("IP exposée publiquement")
	}
	itok := tokenFor(t, opts, invoice.ID, doclink.KindInvoice, invoice.CreatedAt)
	rec = getSigState(srv, itok)
	st = map[string]any{}
	_ = json.Unmarshal(rec.Body.Bytes(), &st)
	if rec.Code != 200 || st["signable"] != false {
		t.Fatalf("facture: %d %v", rec.Code, st)
	}
	if rec := getSigState(srv, tok+"x"); rec.Code != http.StatusNotFound {
		t.Fatalf("jeton invalide: %d", rec.Code)
	}
}

func TestSignatureProofRouteRequiresAuthentication(t *testing.T) {
	srv, _, _, quote, _ := signFixture(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/documents/"+quote.ID.String()+"/signature", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want 401", rec.Code)
	}
}
