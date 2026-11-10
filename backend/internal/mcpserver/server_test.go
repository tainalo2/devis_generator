package mcpserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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
	var createBody map[string]any
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && r.URL.Path == "/documents/quote-id/link" {
			_, _ = w.Write([]byte(`{"link":"https://example.test/d/stable"}`))
			return
		}
		if r.Method != http.MethodPost || r.URL.Path != "/documents" {
			t.Fatalf("request=%s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&createBody); err != nil {
			t.Fatal(err)
		}
		if createBody["kind"] != "quote" || createBody["client_id"] != "client-id" || createBody["currency"] != "EUR" || createBody["vat_regime"] != "standard" {
			t.Fatalf("unexpected body: %#v", createBody)
		}
		if _, leaked := createBody["client"]; leaked {
			t.Fatalf("MCP-only client field leaked to API: %#v", createBody)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"document":{"id":"quote-id","kind":"quote"}}`))
	}))
	defer api.Close()

	s := &Server{APIURL: api.URL, Client: api.Client(), Now: func() time.Time {
		return time.Date(2026, time.November, 10, 15, 0, 0, 0, time.UTC)
	}}
	res, err := s.callTool("create_devis", json.RawMessage(`{"client":"client-id","lignes":[{"description":"Conseil","quantity_milli":1000,"unit_price_cents":50000,"tax_rate_bps":2000}],"devise":"EUR","delai_validite_jours":"30","idempotency_key":"quote-2026-001"}`))
	if err != nil {
		t.Fatal(err)
	}
	if res["status"] != "created" || res["id"] != "quote-id" || res["link"] != "https://example.test/d/stable" {
		t.Fatalf("unexpected result: %#v", res)
	}
	if createBody["idempotency_key"] != "quote-2026-001" || createBody["issue_date"] != "2026-11-10" || createBody["due_date"] != "2026-12-10" {
		t.Fatalf("idempotency and validity not applied: %#v", createBody)
	}
}

func TestCreateFactureAppliesExplicitDueDateAndIdempotency(t *testing.T) {
	var createBody map[string]any
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/documents":
			if err := json.NewDecoder(r.Body).Decode(&createBody); err != nil {
				t.Fatal(err)
			}
			_, _ = w.Write([]byte(`{"document":{"id":"invoice-id"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/documents/invoice-id/link":
			_, _ = w.Write([]byte(`{"link":"https://example.test/d/invoice"}`))
		default:
			t.Fatalf("request=%s %s", r.Method, r.URL.Path)
		}
	}))
	defer api.Close()

	s := &Server{APIURL: api.URL, Client: api.Client(), Now: func() time.Time {
		return time.Date(2026, time.November, 10, 15, 0, 0, 0, time.UTC)
	}}
	res, err := s.callTool("create_facture", json.RawMessage(`{"client":"client-id","lignes":[{"description":"Conseil"}],"date_echeance":"2027-01-15","idempotency_key":"invoice-2026-001"}`))
	if err != nil {
		t.Fatal(err)
	}
	if res["status"] != "created" || res["id"] != "invoice-id" {
		t.Fatalf("unexpected result: %#v", res)
	}
	if createBody["issue_date"] != "2026-11-10" || createBody["due_date"] != "2027-01-15" || createBody["idempotency_key"] != "invoice-2026-001" {
		t.Fatalf("invoice dates and idempotency not applied: %#v", createBody)
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
	res, err := s.callTool("create_acompte", json.RawMessage(`{"devis_id":"quote-id","pourcentage":"30","idempotency_key":"deposit-2026-001"}`))
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

func TestCreateRequiresIdempotencyKey(t *testing.T) {
	s := &Server{APIURL: "http://unused", Client: &http.Client{}}
	res, err := s.callTool("create_devis", json.RawMessage(`{"client":"c","lignes":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	if res["status"] != "clarification" {
		t.Fatalf("expected clarification, got %#v", res)
	}
	missing, _ := res["missing"].([]string)
	if len(missing) != 1 || missing[0] != "idempotency_key" {
		t.Fatalf("unexpected missing fields: %#v", res)
	}
	for _, tool := range toolDefs()[:3] {
		schema := tool["inputSchema"].(map[string]any)
		required := schema["required"].([]string)
		found := false
		for _, field := range required {
			found = found || field == "idempotency_key"
		}
		if !found {
			t.Fatalf("tool %s does not require idempotency_key: %#v", tool["name"], required)
		}
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
