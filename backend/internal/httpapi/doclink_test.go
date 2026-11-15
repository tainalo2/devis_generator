package httpapi

import (
	"bytes"
	"compress/zlib"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"github.com/tainalo2/devis_generator/backend/internal/biz"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/tainalo2/devis_generator/backend/internal/doclink"
)

func TestPublicDocumentPageIsServedBySPA(t *testing.T) {
	staticDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(staticDir, "index.html"), []byte("<!doctype html><div id=\"root\">app</div>"), 0o600); err != nil {
		t.Fatal(err)
	}
	opts := testOptions()
	opts.StaticDir = staticDir
	srv := New(opts)

	req := httptest.NewRequest(http.MethodGet, "/d/signed-token", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `id="root"`) {
		t.Fatalf("body = %q, want SPA index", rec.Body.String())
	}
}

const testIBAN = "FR7630006000011234567890189"

type fakePublicDocs struct {
	docs    map[uuid.UUID]biz.Document
	issuer  biz.IssuerProfile
	ownerID uuid.UUID
}

func (f *fakePublicDocs) GetDocumentPublic(_ context.Context, id uuid.UUID) (biz.Document, uuid.UUID, error) {
	d, ok := f.docs[id]
	if !ok {
		return biz.Document{}, uuid.Nil, biz.ErrNotFound
	}
	return d, f.ownerID, nil
}

func (f *fakePublicDocs) GetIssuer(context.Context, uuid.UUID) (biz.IssuerProfile, error) {
	return f.issuer, nil
}

func (f *fakePublicDocs) GetClient(context.Context, uuid.UUID, uuid.UUID) (biz.Client, error) {
	return biz.Client{Name: "Client SAS"}, nil
}

func testInvoice(t *testing.T, num string) biz.Document {
	t.Helper()
	date := "2026-10-25"
	snap, err := json.Marshal(biz.IssuerProfile{LegalName: "Snapshot AE", SIRET: "12345678900012", IBAN: testIBAN})
	if err != nil {
		t.Fatal(err)
	}
	return biz.Document{
		ID: uuid.New(), Kind: doclink.KindInvoice, Status: "issued", Number: &num, IssueDate: &date,
		VATRegime: "franchise_293b", Currency: "EUR", PaymentTerms: "À réception",
		SubtotalCents: 10000, TotalCents: 10000, CreatedAt: time.Now(),
		IssuerSnapshot: snap,
		Lines:          []biz.DocLine{{Description: "Prestation", QuantityMilli: 1000, UnitPriceCents: 10000, TotalCents: 10000}},
	}
}

func publicFixture(t *testing.T) (*Server, Options, *fakePublicDocs, biz.Document, biz.Document) {
	t.Helper()
	a, b := testInvoice(t, "F-2026-0001"), testInvoice(t, "F-2026-0002")
	f := &fakePublicDocs{
		docs:   map[uuid.UUID]biz.Document{a.ID: a, b.ID: b},
		issuer: biz.IssuerProfile{LegalName: "Profil AE", SIRET: "12345678900012", IBAN: testIBAN},
	}
	opts := testOptions()
	opts.PublicDocs = f
	return New(opts), opts, f, a, b
}

func getPDF(srv *Server, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/documents/"+token+"/pdf", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	return rec
}

func tokenFor(t *testing.T, opts Options, id uuid.UUID, kind string, issued time.Time) string {
	t.Helper()
	tok, err := doclink.Generate(opts.DocLinkSecret, id.String(), kind, 1, issued)
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func utf16BEBytes(s string) []byte {
	var b bytes.Buffer
	for _, r := range s {
		b.WriteByte(byte(r >> 8))
		b.WriteByte(byte(r))
	}
	return b.Bytes()
}

// pdfPlain returns the raw PDF plus every inflated stream.
func pdfPlain(pdf []byte) []byte {
	var out bytes.Buffer
	out.Write(pdf)
	rest := pdf
	for {
		i := bytes.Index(rest, []byte("stream\n"))
		if i < 0 {
			break
		}
		rest = rest[i+len("stream\n"):]
		j := bytes.Index(rest, []byte("endstream"))
		if j < 0 {
			break
		}
		if zr, err := zlib.NewReader(bytes.NewReader(rest[:j])); err == nil {
			if dec, _ := io.ReadAll(zr); len(dec) > 0 {
				out.Write(dec)
			}
		}
		rest = rest[j:]
	}
	return out.Bytes()
}

func TestPublicDocumentPDFSuccessHeadersAndNoIBANLeak(t *testing.T) {
	srv, opts, _, a, _ := publicFixture(t)
	rec := getPDF(srv, tokenFor(t, opts, a.ID, a.Kind, a.CreatedAt))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/pdf" {
		t.Fatalf("Content-Type = %q", ct)
	}
	if cd := rec.Header().Get("Content-Disposition"); !strings.HasPrefix(cd, "inline") || !strings.Contains(cd, "F-2026-0001.pdf") {
		t.Fatalf("Content-Disposition = %q", cd)
	}
	cc := rec.Header().Get("Cache-Control")
	if !strings.Contains(cc, "private") || !strings.Contains(cc, "no-store") {
		t.Fatalf("Cache-Control = %q", cc)
	}
	pdf := rec.Body.Bytes()
	if !bytes.HasPrefix(pdf, []byte("%PDF")) {
		t.Fatal("not a PDF")
	}
	plain := pdfPlain(pdf)
	// control: the detector must see ordinary document text in UTF-16BE
	if !bytes.Contains(plain, utf16BEBytes("Prestation")) {
		t.Fatal("control failed: cannot read text from PDF streams; leak check would be blind")
	}
	spaced := "FR76 3000 6000 0112 3456 7890 189"
	for _, secret := range []string{testIBAN, spaced, "30006000011234567890189"} {
		if bytes.Contains(plain, []byte(secret)) || bytes.Contains(plain, utf16BEBytes(secret)) {
			t.Fatalf("IBAN leaked in public PDF: %q", secret)
		}
	}
}

func TestPublicDocumentPDFIsolationAndFailuresAreIdentical(t *testing.T) {
	srv, opts, f, a, b := publicFixture(t)
	good := tokenFor(t, opts, a.ID, a.Kind, a.CreatedAt)

	// token of A serves A, never B
	recA := getPDF(srv, good)
	recB := getPDF(srv, tokenFor(t, opts, b.ID, b.Kind, b.CreatedAt))
	if recA.Code != 200 || recB.Code != 200 {
		t.Fatalf("A=%d B=%d", recA.Code, recB.Code)
	}
	if !strings.Contains(recA.Header().Get("Content-Disposition"), "F-2026-0001") ||
		!strings.Contains(recB.Header().Get("Content-Disposition"), "F-2026-0002") {
		t.Fatal("token served the wrong document")
	}

	// tampered DocID (payload swapped to B, signature of A)
	parts := strings.Split(good, ".")
	payloadB := base64.RawURLEncoding.EncodeToString([]byte(`{"d":"` + b.ID.String() + `","k":"invoice","v":1,"i":` + "0" + `}`))
	tamperedDoc := payloadB + "." + parts[1]
	// tampered signature
	tamperedSig := parts[0] + "." + base64.RawURLEncoding.EncodeToString([]byte("forged-signature"))

	expiredQuote := testInvoice(t, "D-1")
	expiredQuote.Kind = doclink.KindQuote
	f.docs[expiredQuote.ID] = expiredQuote
	expired := tokenFor(t, opts, expiredQuote.ID, doclink.KindQuote, time.Now().Add(-doclink.QuoteTTL-time.Minute))

	cases := map[string]string{
		"invalid":        "not-a-token",
		"tampered docid": tamperedDoc,
		"tampered sig":   tamperedSig,
		"expired":        expired,
		"absent":         tokenFor(t, opts, uuid.New(), doclink.KindInvoice, time.Now()),
		"kind mismatch":  tokenFor(t, opts, a.ID, doclink.KindQuote, a.CreatedAt),
	}
	var wantCode int
	var wantBody string
	for name, tok := range cases {
		rec := getPDF(srv, tok)
		if wantBody == "" {
			wantCode, wantBody = rec.Code, rec.Body.String()
			if wantCode != http.StatusNotFound {
				t.Fatalf("%s: status = %d, want 404", name, wantCode)
			}
		}
		if rec.Code != wantCode || rec.Body.String() != wantBody {
			t.Fatalf("%s: response differs: %d %q vs %d %q", name, rec.Code, rec.Body.String(), wantCode, wantBody)
		}
		if got := rec.Header().Get("Content-Type"); got != "application/json" && !strings.HasPrefix(got, "application/json") {
			t.Fatalf("%s: Content-Type = %q", name, got)
		}
		if cc := rec.Header().Get("Cache-Control"); cc != "private, no-store" {
			t.Fatalf("%s: Cache-Control = %q", name, cc)
		}
	}
	for _, leak := range []string{"expir", "invalid"} {
		if strings.Contains(strings.ToLower(wantBody), leak) {
			t.Fatalf("generic body reveals cause: %q", wantBody)
		}
	}
}

func TestPublicDocumentPageCSPAllowsBlobObjectOnlyThere(t *testing.T) {
	srv, _, _, _, _ := publicFixture(t)
	get := func(path string) string {
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		return rec.Header().Get("Content-Security-Policy")
	}
	page := get("/d/signed-token")
	if !strings.Contains(page, "object-src blob:") || strings.Contains(page, "object-src 'none'") {
		t.Fatalf("/d/{token} CSP = %q, want object-src blob:", page)
	}
	if want := strings.Replace(contentSecurityPolicy, "object-src 'none'", "object-src blob:; frame-src blob:", 1); page != want {
		t.Fatalf("/d/{token} CSP differs beyond object-src: %q", page)
	}
	for _, p := range []string{"/", "/d", "/d/", "/d/a/b", "/login", "/api/v1/healthz", "/api/v1/public/documents/x/pdf"} {
		if csp := get(p); !strings.Contains(csp, "object-src 'none'") || strings.Contains(csp, "blob:") && !strings.Contains(csp, "img-src 'self' data: blob:") {
			t.Fatalf("%s CSP = %q, want object-src 'none'", p, csp)
		}
		if csp := get(p); csp != contentSecurityPolicy {
			t.Fatalf("%s CSP = %q, want global policy", p, csp)
		}
	}
}

type countingPublicDocs struct {
	*fakePublicDocs
	docCalls, other int
	failWith        error
}

func (c *countingPublicDocs) GetDocumentPublic(ctx context.Context, id uuid.UUID) (biz.Document, uuid.UUID, error) {
	c.docCalls++
	if c.failWith != nil {
		return biz.Document{}, uuid.Nil, c.failWith
	}
	return c.fakePublicDocs.GetDocumentPublic(ctx, id)
}

func (c *countingPublicDocs) GetIssuer(ctx context.Context, id uuid.UUID) (biz.IssuerProfile, error) {
	c.other++
	return c.fakePublicDocs.GetIssuer(ctx, id)
}

func TestPublicDocumentLinkStoreWorkIsUniformForValidSignatures(t *testing.T) {
	srv, opts, f, a, _ := publicFixture(t)
	c := &countingPublicDocs{fakePublicDocs: f}
	srv.publicDocs = c
	expiredQuote := testInvoice(t, "D-1")
	expiredQuote.Kind = doclink.KindQuote
	f.docs[expiredQuote.ID] = expiredQuote

	parts := strings.Split(tokenFor(t, opts, a.ID, a.Kind, a.CreatedAt), ".")
	bad := map[string]string{
		"garbage":    "not-a-token",
		"forged sig": parts[0] + "." + base64.RawURLEncoding.EncodeToString([]byte("forged")),
	}
	for name, tok := range bad {
		c.docCalls, c.other = 0, 0
		if rec := getPDF(srv, tok); rec.Code != http.StatusNotFound {
			t.Fatalf("%s: status %d", name, rec.Code)
		}
		if c.docCalls != 0 || c.other != 0 {
			t.Fatalf("%s: store called (%d,%d), want none", name, c.docCalls, c.other)
		}
	}

	failing := map[string]string{
		"expired":       tokenFor(t, opts, expiredQuote.ID, doclink.KindQuote, time.Now().Add(-doclink.QuoteTTL-time.Minute)),
		"absent":        tokenFor(t, opts, uuid.New(), doclink.KindInvoice, time.Now()),
		"kind mismatch": tokenFor(t, opts, a.ID, doclink.KindQuote, a.CreatedAt),
	}
	for name, tok := range failing {
		c.docCalls, c.other = 0, 0
		if rec := getPDF(srv, tok); rec.Code != http.StatusNotFound {
			t.Fatalf("%s: status %d", name, rec.Code)
		}
		if c.docCalls != 1 || c.other != 0 {
			t.Fatalf("%s: store calls doc=%d other=%d, want doc=1 other=0", name, c.docCalls, c.other)
		}
	}
	c.docCalls, c.other = 0, 0
	if rec := getPDF(srv, tokenFor(t, opts, a.ID, a.Kind, a.CreatedAt)); rec.Code != http.StatusOK {
		t.Fatalf("valid: status %d", rec.Code)
	}
	if c.docCalls != 1 {
		t.Fatalf("valid: GetDocumentPublic calls = %d, want 1", c.docCalls)
	}
}

func TestPublicDocumentLinkNoStoreOnInternalErrorsAndRateLimit(t *testing.T) {
	srv, opts, f, a, _ := publicFixture(t)
	srv.publicDocs = &countingPublicDocs{fakePublicDocs: f, failWith: context.DeadlineExceeded}
	rec := getPDF(srv, tokenFor(t, opts, a.ID, a.Kind, a.CreatedAt))
	if rec.Code != http.StatusInternalServerError || rec.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("500 path: %d cache-control %q", rec.Code, rec.Header().Get("Cache-Control"))
	}

	srv.publicDocs = f
	srv.publicLinkLimiter = NewRateLimiter(2, time.Hour)
	for i := 0; i < 2; i++ {
		if rec := getPDF(srv, "not-a-token"); rec.Code != http.StatusNotFound {
			t.Fatalf("request %d: %d", i, rec.Code)
		}
	}
	rec = getPDF(srv, "not-a-token")
	if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("limited: %d cache-control %q", rec.Code, rec.Header().Get("Cache-Control"))
	}
}

func TestRequestSchemeUsesForwardedProtoFromTLSProxy(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.test", nil)
	req.Header.Set("X-Forwarded-Proto", "https")

	if got := requestScheme(req); got != "https" {
		t.Fatalf("requestScheme() = %q, want https", got)
	}
}

func TestRequestSchemeFallsBackToDirectTLS(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.test", nil)
	req.TLS = &tls.ConnectionState{}

	if got := requestScheme(req); got != "https" {
		t.Fatalf("requestScheme() = %q, want https", got)
	}
}

func TestRequestSchemePrioritizesForwardedProtoOverTLS(t *testing.T) {
	req := httptest.NewRequest("GET", "https://example.test", nil)
	req.TLS = &tls.ConnectionState{}
	req.Header.Set("X-Forwarded-Proto", "http")

	if got := requestScheme(req); got != "http" {
		t.Fatalf("requestScheme() = %q, want http", got)
	}
}
