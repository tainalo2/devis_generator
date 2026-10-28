package legal

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func testClient(fn roundTripFunc) *http.Client {
	return &http.Client{Transport: fn, Timeout: 2 * time.Second}
}

func TestRechercheSearchMock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("q") != "la poste" {
			t.Fatalf("q = %q", r.URL.Query().Get("q"))
		}
		_, _ = w.Write([]byte(`{
			"results":[{
				"siren":"356000000",
				"nom_complet":"LA POSTE",
				"nom_raison_sociale":"LA POSTE",
				"siege":{
					"siret":"35600000000048",
					"adresse":"9 RUE DU COLONEL PIERRE AVIA 75015 PARIS",
					"code_postal":"75015",
					"libelle_commune":"PARIS",
					"activite_principale":"53.10Z"
				}
			}]
		}`))
	}))
	defer srv.Close()

	c := NewRechercheClient(srv.Client())
	c.BaseURL = srv.URL
	c.Limiter = NewRateLimiter(time.Millisecond)
	hits, err := c.Search(context.Background(), "la poste", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].SIREN != "356000000" || hits[0].SIRET == "" {
		t.Fatalf("hits = %+v", hits)
	}
	if !strings.HasPrefix(hits[0].VATNumber, "FR") {
		t.Fatalf("vat = %q", hits[0].VATNumber)
	}
}

func TestRechercheTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(200)
	}))
	defer srv.Close()
	client := &http.Client{Timeout: 20 * time.Millisecond}
	c := NewRechercheClient(client)
	c.BaseURL = srv.URL
	c.Limiter = NewRateLimiter(time.Millisecond)
	c.Retry = RetryPolicy{MaxAttempts: 1, BaseDelay: time.Millisecond}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := c.Search(ctx, "test", 5)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestVIESValidateMock(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"countryCode":"FR"`) {
			t.Fatalf("body = %s", body)
		}
		_, _ = w.Write([]byte(`{"valid":true,"name":"ACME","address":"PARIS"}`))
	}))
	defer srv.Close()

	c := NewVIESClient(srv.Client())
	c.URL = srv.URL
	c.Limiter = NewRateLimiter(time.Millisecond)
	c.Retry = RetryPolicy{MaxAttempts: 1}
	v, err := c.Validate(context.Background(), "FR12345678901")
	if err != nil || !v.Valid || v.Name != "ACME" {
		t.Fatalf("v=%+v err=%v", v, err)
	}
	// Second call must hit cache (no extra HTTP).
	v2, err := c.Validate(context.Background(), "FR12345678901")
	if err != nil || !v2.Cached || calls != 1 {
		t.Fatalf("cache miss: calls=%d cached=%v err=%v", calls, v2.Cached, err)
	}
}

func TestVIESRetryOn500(t *testing.T) {
	n := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		if n == 1 {
			w.WriteHeader(500)
			return
		}
		_, _ = w.Write([]byte(`{"valid":false,"name":"---","address":"---"}`))
	}))
	defer srv.Close()
	c := NewVIESClient(srv.Client())
	c.URL = srv.URL
	c.Limiter = NewRateLimiter(time.Millisecond)
	c.Retry = RetryPolicy{MaxAttempts: 3, BaseDelay: time.Millisecond}
	v, err := c.Validate(context.Background(), "DE122268496")
	if err != nil || v.Valid {
		t.Fatalf("v=%+v err=%v n=%d", v, err, n)
	}
	if n < 2 {
		t.Fatalf("expected retry, n=%d", n)
	}
}

func TestTEDBFetchMock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		soap := string(body)
		if !strings.Contains(soap, tedbMsgNS) || !strings.Contains(soap, tedbTypesNS) {
			t.Fatalf("SOAP namespaces missing: %s", soap)
		}
		if !strings.Contains(soap, "<t:isoCode>FR</t:isoCode>") || !strings.Contains(soap, "<t:situationOn>") {
			t.Fatalf("typed children missing: %s", soap)
		}
		if got := r.Header.Get("SOAPAction"); got != tedbSOAPAction {
			t.Fatalf("SOAPAction=%q", got)
		}
		_, _ = w.Write([]byte(`<?xml version="1.0"?>
		<soap:Envelope><soap:Body>
		  <vatRateResults><type>STANDARD</type><rate><type>DEFAULT</type><value>20.0</value></rate></vatRateResults>
		  <vatRateResults><type>REDUCED</type><rate><type>REDUCED_RATE</type><value>5.5</value></rate></vatRateResults>
		  <vatRateResults><type>REDUCED</type><rate><type>REDUCED_RATE</type><value>10.0</value></rate></vatRateResults>
		</soap:Body></soap:Envelope>`))
	}))
	defer srv.Close()

	p := NewTEDBProvider(srv.Client())
	p.URL = srv.URL
	p.Limiter = NewRateLimiter(time.Millisecond)
	p.Retry = RetryPolicy{MaxAttempts: 1}
	v, err := p.Fetch(context.Background(), KeyTEDBVATRatesFR)
	if err != nil {
		t.Fatal(err)
	}
	var rates VATRates
	if err := json.Unmarshal(v.Value, &rates); err != nil {
		t.Fatal(err)
	}
	if rates.Standard != 20 || len(rates.Reduced) < 2 {
		t.Fatalf("rates=%+v", rates)
	}
}

func TestTEDBParseRealAnonymizedFixture(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "tedb_vat_rates_fr_anon.xml"))
	if err != nil {
		t.Fatal(err)
	}
	rates, err := parseTEDBResponse(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	if rates.Standard != 20 {
		t.Fatalf("standard=%v", rates.Standard)
	}
	for _, want := range []float64{2.1, 5.5, 10} {
		found := false
		for _, got := range rates.Reduced {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing reduced %v in %+v", want, rates.Reduced)
		}
	}
	for _, got := range rates.Reduced {
		if got == 0 {
			t.Fatalf("exempted 0 must be skipped: %+v", rates.Reduced)
		}
	}
	if rates.SituationOn != "2026-07-01" {
		t.Fatalf("situationOn=%q", rates.SituationOn)
	}
}

func TestTEDBParseFaultFixture(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "tedb_fault_xsd.xml"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = parseTEDBResponse(string(raw))
	if err == nil {
		t.Fatal("expected fault error")
	}
	if !strings.Contains(err.Error(), "TEDB") && !strings.Contains(err.Error(), "XSD") && !strings.Contains(err.Error(), "fault") {
		t.Fatalf("err=%v", err)
	}
}

func TestTEDBBuildRequestNamespaces(t *testing.T) {
	soap := buildTEDBRequest("fr", "2026-10-25")
	for _, want := range []string{tedbMsgNS, tedbTypesNS, "<t:isoCode>FR</t:isoCode>", "<t:situationOn>2026-10-25</t:situationOn>"} {
		if !strings.Contains(soap, want) {
			t.Fatalf("missing %q in %s", want, soap)
		}
	}
	// Legacy unqualified children / wrong message NS must not appear as the request NS.
	if strings.Contains(soap, "xmlns:urn=\"urn:ec.europa.eu:taxud:tedb:services:v1:VatRetrievalService\"") {
		t.Fatal("must not bind request element to WSDL targetNamespace VatRetrievalService")
	}
}

func TestTEDBRefreshFailureKeepsVerifiedAndIssuance(t *testing.T) {
	store := NewMemStore()
	_ = store.Upsert(context.Background(), Variable{
		Key: KeyTEDBVATRatesFR,
		Value: json.RawMessage(`{
			"member_state":"FR","situation_on":"2026-01-01",
			"standard_percent":20,"reduced_percent":[5.5,10]
		}`),
		Source: "tedb.ec.europa.eu", FetchedAt: time.Now().UTC(), Status: StatusVerified,
	})
	_ = store.Upsert(context.Background(), Variable{
		Key: KeyManualMentions,
		Value: json.RawMessage(`{
			"franchise_293b":"TVA non applicable, art. 293 B du CGI",
			"flat_indemnity_40e":"Indemnité forfaitaire : 40 €",
			"payment_terms_default":"Paiement à 30 jours.",
			"quote_validity_note":"Devis 15 jours."
		}`),
		Source: "manual-surveillance", FetchedAt: time.Now().UTC(), Status: StatusManual,
	})

	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`<env:Envelope xmlns:env="http://schemas.xmlsoap.org/soap/envelope/"><env:Body><env:Fault><faultstring>TEDB-ERR-2 - Request is not valid</faultstring></env:Fault></env:Body></env:Envelope>`))
	}))
	defer failing.Close()

	tedb := NewTEDBProvider(failing.Client())
	tedb.URL = failing.URL
	tedb.Limiter = NewRateLimiter(time.Millisecond)
	tedb.Retry = RetryPolicy{MaxAttempts: 1}

	svc := &Service{Store: store, TEDB: tedb}
	r := &Refresher{Service: svc, Logger: slogDiscard()}
	if err := r.refreshUnlocked(context.Background()); err != nil {
		t.Fatal(err)
	}
	v, status, err := store.Latest(context.Background(), KeyTEDBVATRatesFR)
	if err != nil || status != StatusVerified {
		t.Fatalf("status=%s err=%v", status, err)
	}
	var rates VATRates
	_ = json.Unmarshal(v.Value, &rates)
	if rates.Standard != 20 {
		t.Fatalf("lost verified TEDB value: %+v", rates)
	}
	m := svc.MentionsForIssue(context.Background())
	if m.Franchise293B == "" {
		t.Fatal("issuance mentions must remain available after TEDB failure")
	}
}

func TestBCEFetchMock(t *testing.T) {
	payload := `{
		"dataSets":[{"series":{"0:0:0:0:0:0":{"observations":{"0":[2.65]}}}}],
		"structure":{"dimensions":{"observation":[{"values":[{"id":"2026-09-16"}]}]}}
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(payload))
	}))
	defer srv.Close()

	p := NewBCEProvider(srv.Client())
	p.URL = srv.URL
	p.Limiter = NewRateLimiter(time.Millisecond)
	v, err := p.Fetch(context.Background(), KeyBCEKeyRate)
	if err != nil {
		t.Fatal(err)
	}
	var rate KeyRate
	_ = json.Unmarshal(v.Value, &rate)
	if rate.AnnualPercent != 2.65 {
		t.Fatalf("rate=%+v", rate)
	}
}

func TestRefreshFallbackKeepsLastVerified(t *testing.T) {
	store := NewMemStore()
	_ = store.Upsert(context.Background(), Variable{
		Key: KeyBCEKeyRate, Value: json.RawMessage(`{"annual_percent":2.5}`),
		Source: "test", FetchedAt: time.Now().UTC(), Status: StatusVerified,
	})

	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
	}))
	defer failing.Close()

	bce := NewBCEProvider(failing.Client())
	bce.URL = failing.URL
	bce.Limiter = NewRateLimiter(time.Millisecond)
	bce.Retry = RetryPolicy{MaxAttempts: 1}

	svc := &Service{Store: store, BCE: bce}
	r := &Refresher{Service: svc, Logger: slogDiscard()}
	if err := r.refreshUnlocked(context.Background()); err != nil {
		t.Fatal(err)
	}
	v, status, err := store.Latest(context.Background(), KeyBCEKeyRate)
	if err != nil || status != StatusVerified {
		t.Fatalf("status=%s err=%v", status, err)
	}
	var rate KeyRate
	_ = json.Unmarshal(v.Value, &rate)
	if rate.AnnualPercent != 2.5 {
		t.Fatalf("lost verified value: %+v", rate)
	}
	entries, _ := store.ListStatus(context.Background())
	var sawError bool
	for _, e := range entries {
		if e.Key == KeyBCEKeyRate && e.Status == StatusError {
			sawError = true
		}
	}
	if !sawError {
		// Last append is error — ListStatus uses latest by key which is the error row.
		// Latest() still returns verified. ListStatus may show error — both OK.
		for _, e := range entries {
			if e.Key == KeyBCEKeyRate {
				t.Logf("list status for bce: %+v", e)
			}
		}
	}
}

func TestMentionsForIssueUsesVerified(t *testing.T) {
	store := NewMemStore()
	_ = store.Upsert(context.Background(), Variable{
		Key: KeyManualMentions,
		Value: json.RawMessage(`{
			"franchise_293b":"TVA non applicable, art. 293 B du CGI",
			"flat_indemnity_40e":"Indemnité forfaitaire : 40 €",
			"payment_terms_default":"Paiement à 30 jours.",
			"quote_validity_note":"Devis 15 jours."
		}`),
		Source: "manual-surveillance", FetchedAt: time.Now().UTC(), Status: StatusManual,
	})
	_ = store.Upsert(context.Background(), Variable{
		Key: KeyLatePenaltyEffective, Value: json.RawMessage(`{"annual_percent":12.5,"rule":"bce_plus_10"}`),
		Source: "derived:bce_plus_10", FetchedAt: time.Now().UTC(), Status: StatusVerified,
	})
	svc := &Service{Store: store}
	m := svc.MentionsForIssue(context.Background())
	if m.Source == DefaultsSource {
		t.Fatalf("expected verified source, got %s", m.Source)
	}
	if m.LatePenaltyRateAnnual != 12.5 {
		t.Fatalf("rate=%v", m.LatePenaltyRateAnnual)
	}
	if !strings.Contains(m.LatePenalty, "12.5") {
		t.Fatalf("penalty text=%q", m.LatePenalty)
	}
}

func TestDeriveLatePenaltyRate(t *testing.T) {
	// 3×3.72=11.16 < BCE+10=12.65 → bce_plus_10
	r, rule := DeriveLatePenaltyRate(3.72, 2.65)
	if rule != "bce_plus_10" || r != 12.65 {
		t.Fatalf("r=%v rule=%s", r, rule)
	}
	// 3×5=15 > BCE+10=12.65 → 3x
	r, rule = DeriveLatePenaltyRate(5.0, 2.65)
	if rule != "3x_legal_interest" || r != 15 {
		t.Fatalf("r=%v rule=%s", r, rule)
	}
	r, rule = DeriveLatePenaltyRate(2.0, 4.0)
	if rule != "bce_plus_10" || r != 14 {
		t.Fatalf("r=%v rule=%s", r, rule)
	}
}

func TestFrenchVATKey(t *testing.T) {
	// Clé = (12 + 3×(SIREN mod 97)) mod 97
	if k := frenchVATKey("732829320"); k != "44" {
		t.Fatalf("key=%s", k)
	}
}

func slogDiscard() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
