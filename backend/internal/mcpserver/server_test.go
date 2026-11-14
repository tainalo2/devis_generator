package mcpserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tainalo2/devis_generator/backend/internal/billing"
)

func TestInitializeAndToolsList(t *testing.T) {
	s := &Server{APIURL: "http://x", APIKey: "k", Client: &http.Client{}, In: strings.NewReader(""), Out: &bytes.Buffer{}}
	// initialize
	var buf bytes.Buffer
	s.Out = &buf
	s.handle(rpcReq{JSONRPC: "2.0", ID: json.RawMessage("1"), Method: "initialize"})
	// tools/list
	buf.Reset()
	s.handle(rpcReq{JSONRPC: "2.0", ID: json.RawMessage("2"), Method: "tools/list"})
	// call directly
	res, err := s.handle(rpcReq{JSONRPC: "2.0", ID: json.RawMessage("2"), Method: "tools/list"})
	if err != nil {
		t.Fatal(err)
	}
	m := res.(map[string]any)
	tools := m["tools"].([]map[string]any)
	if len(tools) != 5 {
		t.Fatalf("expected 5 tools got %d", len(tools))
	}
	_ = buf
	_ = http.MethodGet
	_ = httptest.NewServer
}

func TestCreateDevisCallsPublicAPIContract(t *testing.T) {
	var createdLines []any
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && r.URL.Path == "/documents/quote-id/link" {
			_, _ = w.Write([]byte(`{"link":"https://example.test/d/stable"}`))
			return
		}
		if r.Method != http.MethodPost || r.URL.Path != "/documents" {
			t.Fatalf("request=%s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["kind"] != "quote" || body["client_id"] != "client-id" || body["currency"] != "EUR" || body["vat_regime"] != "standard" {
			t.Fatalf("unexpected body: %#v", body)
		}
		if _, leaked := body["client"]; leaked {
			t.Fatalf("MCP-only client field leaked to API: %#v", body)
		}
		createdLines = body["lines"].([]any)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"document":{"id":"quote-id","kind":"quote"}}`))
	}))
	defer api.Close()

	s := &Server{APIURL: api.URL, APIKey: "k", Client: api.Client()}
	res, err := s.callTool("create_devis", json.RawMessage(`{"client":"client-id","lignes":[{"description":"Conseil","quantite":"2.5","prix_unitaire_ht":"49.90","tax_rate_bps":2000}],"devise":"EUR","delai_validite_jours":"30"}`))
	if err != nil {
		t.Fatal(err)
	}
	if res["status"] != "created" || res["id"] != "quote-id" || res["link"] != "https://example.test/d/stable" {
		t.Fatalf("unexpected result: %#v", res)
	}
	line := createdLines[0].(map[string]any)
	if line["quantity_milli"] != float64(2500) || line["unit_price_cents"] != float64(4990) {
		t.Fatalf("unexpected converted line: %#v", line)
	}
	if _, leaked := line["quantite"]; leaked {
		t.Fatalf("MCP quantity leaked to API: %#v", line)
	}
	if _, leaked := line["prix_unitaire_ht"]; leaked {
		t.Fatalf("MCP unit price leaked to API: %#v", line)
	}
	calc, err := billing.Compute(billing.RegimeStandard, []billing.LineInput{{
		Description: "Conseil", QuantityMilli: int64(line["quantity_milli"].(float64)),
		UnitPriceCents: int64(line["unit_price_cents"].(float64)), TaxRateBps: 2000,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if calc.SubtotalCents != 12475 || calc.TaxCents != 2495 || calc.TotalCents != 14970 {
		t.Fatalf("billing regression: %#v", calc)
	}
}

func TestConvertDocumentedLinesRoundsHalfAwayFromZero(t *testing.T) {
	lines, err := convertMCPLines([]any{
		map[string]any{"description": "integer", "quantite": "2", "prix_unitaire_ht": "10"},
		map[string]any{"description": "exact documented precision", "quantite": "12.345", "prix_unitaire_ht": "0.1"},
		map[string]any{"description": "quantity rounding", "quantite": "1.2345", "prix_unitaire_ht": "1.005"},
		map[string]any{"description": "negative price rounding", "quantite": "1", "prix_unitaire_ht": "-1.005"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := [][2]int64{{2000, 1000}, {12345, 10}, {1235, 101}, {1000, -101}}
	for i, line := range lines {
		if line["quantity_milli"] != want[i][0] || line["unit_price_cents"] != want[i][1] {
			t.Fatalf("line %d: got quantity=%v price=%v, want %v", i+1, line["quantity_milli"], line["unit_price_cents"], want[i])
		}
	}
}

func TestConvertDocumentedLinesRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name  string
		line  any
		match string
	}{
		{name: "line type", line: "invalid", match: "ligne 1: objet attendu"},
		{name: "quantity type", line: map[string]any{"quantite": 2, "prix_unitaire_ht": "10"}, match: "ligne 1: quantite doit être une chaîne décimale"},
		{name: "quantity format", line: map[string]any{"quantite": "2,5", "prix_unitaire_ht": "10"}, match: "ligne 1: quantite invalide"},
		{name: "zero quantity", line: map[string]any{"quantite": "0.0004", "prix_unitaire_ht": "10"}, match: "ligne 1: quantite arrondie à zéro"},
		{name: "price type", line: map[string]any{"quantite": "1", "prix_unitaire_ht": 10}, match: "ligne 1: prix_unitaire_ht doit être une chaîne décimale"},
		{name: "price format", line: map[string]any{"quantite": "1", "prix_unitaire_ht": "NaN"}, match: "ligne 1: prix_unitaire_ht invalide"},
		{name: "quantity too long", line: map[string]any{"quantite": "0." + strings.Repeat("0", 200) + "1", "prix_unitaire_ht": "10"}, match: "ligne 1: quantite trop longue"},
		{name: "quantity overflow", line: map[string]any{"quantite": "999999999999999999999", "prix_unitaire_ht": "10"}, match: "ligne 1: quantite hors limites"},
		{name: "price overflow", line: map[string]any{"quantite": "1", "prix_unitaire_ht": "999999999999999999999"}, match: "ligne 1: prix_unitaire_ht hors limites"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := convertMCPLines([]any{tt.line})
			if err == nil || !strings.Contains(err.Error(), tt.match) {
				t.Fatalf("error=%v, want containing %q", err, tt.match)
			}
		})
	}
}

func TestCreateAcompteScalesQuoteLines(t *testing.T) {
	var depositBody map[string]any
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/documents/quote-id":
			_, _ = w.Write([]byte(`{"document":{"id":"quote-id","subtotal_cents":100000,"lines":[{"description":"Conseil","quantity_milli":1000,"unit_price_cents":100000,"tax_rate_bps":2000}]}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/documents/deposit-id/link":
			_, _ = w.Write([]byte(`{"link":"https://example.test/d/deposit"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/documents/quote-id/deposit-invoice":
			if err := json.NewDecoder(r.Body).Decode(&depositBody); err != nil {
				t.Fatal(err)
			}
			_, _ = w.Write([]byte(`{"document":{"id":"deposit-id","kind":"deposit_invoice"}}`))
		default:
			t.Fatalf("request=%s %s", r.Method, r.URL.Path)
		}
	}))
	defer api.Close()

	s := &Server{APIURL: api.URL, APIKey: "k", Client: api.Client()}
	res, err := s.callTool("create_acompte", json.RawMessage(`{"devis_id":"quote-id","pourcentage":"30"}`))
	if err != nil {
		t.Fatal(err)
	}
	if res["status"] != "created" || res["id"] != "deposit-id" {
		t.Fatalf("unexpected result: %#v", res)
	}
	lines := depositBody["lines"].([]any)
	line := lines[0].(map[string]any)
	if line["unit_price_cents"] != float64(30000) {
		t.Fatalf("unexpected scaled line: %#v", line)
	}
}

func TestGetDocumentLinkUsesStableLinkEndpoint(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/documents/doc-id/link" {
			t.Fatalf("request=%s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"link":"https://example.test/d/token","permanent":true}`))
	}))
	defer api.Close()
	s := &Server{APIURL: api.URL, APIKey: "k", Client: api.Client()}
	res, err := s.callTool("get_document_link", json.RawMessage(`{"id":"doc-id"}`))
	if err != nil {
		t.Fatal(err)
	}
	if res["link"] != "https://example.test/d/token" || res["status"] != "ok" {
		t.Fatalf("unexpected result: %#v", res)
	}
}

func TestCreateDevisAmbiguity(t *testing.T) {
	s := &Server{APIURL: "http://x", APIKey: "k", Client: &http.Client{}}
	res, err := s.callTool("create_devis", json.RawMessage(`{"client":"c","lignes":[],"valide_jusqu_au":"2026-01-01","delai_validite_jours":"30"}`))
	if err != nil {
		t.Fatal(err)
	}
	if res["status"] != "clarification" {
		t.Fatalf("expected clarification got %v", res["status"])
	}
}

func TestRunStdio(t *testing.T) {
	in := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"ping"}` + "\n")
	var out bytes.Buffer
	s := &Server{APIURL: "http://x", APIKey: "k", Client: &http.Client{}, In: in, Out: &out}
	if err := s.Run(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "result") {
		t.Fatalf("no result: %s", out.String())
	}
}
