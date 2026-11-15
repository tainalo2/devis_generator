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
	res, err := s.callTool("create_devis", json.RawMessage(`{"client":"client-id","lignes":[{"description":"Conseil","quantite":"2.5","prix_unitaire_ht":"49.90","tax_rate_bps":2000}],"devise":"EUR","delai_validite_jours":"30","idempotency_key":"devis-1"}`))
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
		{name: "negative quantity", line: map[string]any{"quantite": "-1", "prix_unitaire_ht": "10"}, match: "ligne 1: quantite doit être strictement positive"},
		{name: "negative quantity rounded to zero", line: map[string]any{"quantite": "-0.0004", "prix_unitaire_ht": "10"}, match: "ligne 1: quantite arrondie à zéro"},
		{name: "unknown field", line: map[string]any{"quantite": "1", "prix_unitaire_ht": "10", "quantity_milli": 5}, match: "ligne 1: propriété inconnue"},
		{name: "empty description", line: map[string]any{"description": "", "quantite": "1", "prix_unitaire_ht": "10"}, match: "ligne 1: description"},
		{name: "description type", line: map[string]any{"description": 3, "quantite": "1", "prix_unitaire_ht": "10"}, match: "ligne 1: description"},
		{name: "tax rate not allowed", line: map[string]any{"description": "x", "quantite": "1", "prix_unitaire_ht": "10", "tax_rate_bps": 1234}, match: "ligne 1: tax_rate_bps"},
		{name: "tax rate type", line: map[string]any{"description": "x", "quantite": "1", "prix_unitaire_ht": "10", "tax_rate_bps": "2000"}, match: "ligne 1: tax_rate_bps"},
		{name: "discount out of range", line: map[string]any{"description": "x", "quantite": "1", "prix_unitaire_ht": "10", "discount_bps": 10001}, match: "ligne 1: discount_bps"},
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
	res, err := s.callTool("create_acompte", json.RawMessage(`{"devis_id":"quote-id","pourcentage":"30","idempotency_key":"acompte-1"}`))
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

func TestCreateToolsRejectNegativeQuantityBeforeAPI(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("API must not be called: %s %s", r.Method, r.URL.Path)
	}))
	defer api.Close()
	s := &Server{APIURL: api.URL, APIKey: "k", Client: api.Client()}
	lines := `[{"description":"x","quantite":"-2","prix_unitaire_ht":"10","tax_rate_bps":2000}]`
	for _, tool := range []string{"create_devis", "create_facture"} {
		res, err := s.callTool(tool, json.RawMessage(`{"client":"c","lignes":`+lines+`,"idempotency_key":"k-neg"}`))
		if err != nil {
			t.Fatal(err)
		}
		if res["status"] != "error" || !strings.Contains(res["error"].(string), "strictement positive") {
			t.Fatalf("%s: unexpected result %#v", tool, res)
		}
	}
}

func TestCreditLinesKeepNegativePrices(t *testing.T) {
	lines, err := convertMCPLines([]any{map[string]any{"description": "avoir", "quantite": "1", "prix_unitaire_ht": "-10"}})
	if err != nil || lines[0]["unit_price_cents"] != int64(-1000) {
		t.Fatalf("lines=%#v err=%v", lines, err)
	}
}

func toolSchema(t *testing.T, name string) map[string]any {
	t.Helper()
	s := &Server{}
	res, rerr := s.handle(rpcReq{JSONRPC: "2.0", ID: json.RawMessage("1"), Method: "tools/list"})
	if rerr != nil {
		t.Fatal(rerr)
	}
	// Round-trip through JSON: this is what an MCP client receives.
	raw, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Tools []struct {
			Name        string         `json:"name"`
			InputSchema map[string]any `json:"inputSchema"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, tool := range decoded.Tools {
		if tool.Name == name {
			return tool.InputSchema
		}
	}
	t.Fatalf("tool %s not listed", name)
	return nil
}

func stringSet(v any) map[string]bool {
	out := map[string]bool{}
	for _, item := range v.([]any) {
		out[item.(string)] = true
	}
	return out
}

func TestToolsListLineItemSchema(t *testing.T) {
	for _, tool := range []string{"create_devis", "create_facture"} {
		props := toolSchema(t, tool)["properties"].(map[string]any)
		lignes := props["lignes"].(map[string]any)
		if lignes["type"] != "array" || lignes["maxItems"] != float64(500) {
			t.Fatalf("%s lignes=%#v", tool, lignes)
		}
		items, ok := lignes["items"].(map[string]any)
		if !ok || items["type"] != "object" || items["additionalProperties"] != false {
			t.Fatalf("%s items=%#v", tool, lignes["items"])
		}
		required := stringSet(items["required"])
		for _, name := range []string{"description", "quantite", "prix_unitaire_ht"} {
			if !required[name] {
				t.Fatalf("%s: %s not required: %#v", tool, name, items["required"])
			}
		}
		if len(required) != 3 {
			t.Fatalf("%s: unexpected required %#v", tool, required)
		}
		lineProps := items["properties"].(map[string]any)
		wantTypes := map[string]string{"description": "string", "quantite": "string", "prix_unitaire_ht": "string", "tax_rate_bps": "integer", "discount_bps": "integer"}
		if len(lineProps) != len(wantTypes) {
			t.Fatalf("%s: properties=%#v", tool, lineProps)
		}
		for name, typ := range wantTypes {
			p, ok := lineProps[name].(map[string]any)
			if !ok || p["type"] != typ || p["description"] == "" {
				t.Fatalf("%s: property %s=%#v", tool, name, lineProps[name])
			}
		}
		if lineProps["description"].(map[string]any)["maxLength"] != float64(2000) || lineProps["description"].(map[string]any)["minLength"] != float64(1) {
			t.Fatalf("description bounds: %#v", lineProps["description"])
		}
		for _, name := range []string{"quantite", "prix_unitaire_ht"} {
			if lineProps[name].(map[string]any)["pattern"] == "" {
				t.Fatalf("%s: missing pattern", name)
			}
		}
		rates := lineProps["tax_rate_bps"].(map[string]any)["enum"].([]any)
		if len(rates) != 5 || rates[0] != float64(0) || rates[4] != float64(2000) {
			t.Fatalf("tax_rate_bps enum=%#v", rates)
		}
		discount := lineProps["discount_bps"].(map[string]any)
		if discount["minimum"] != float64(0) || discount["maximum"] != float64(10000) {
			t.Fatalf("discount bounds=%#v", discount)
		}
	}
}

func TestToolsListRequiresIdempotencyKeyOnCreations(t *testing.T) {
	for _, tool := range []string{"create_devis", "create_facture", "create_acompte"} {
		schema := toolSchema(t, tool)
		prop, ok := schema["properties"].(map[string]any)["idempotency_key"].(map[string]any)
		if !ok || prop["type"] != "string" || prop["minLength"] != float64(1) || prop["maxLength"] != float64(255) {
			t.Fatalf("%s idempotency_key=%#v", tool, prop)
		}
		if !stringSet(schema["required"])["idempotency_key"] {
			t.Fatalf("%s: idempotency_key not required: %#v", tool, schema["required"])
		}
	}
	for _, tool := range []string{"get_document_link", "list_documents"} {
		if _, ok := toolSchema(t, tool)["properties"].(map[string]any)["idempotency_key"]; ok {
			t.Fatalf("%s must not expose idempotency_key", tool)
		}
	}
}

func TestCreateToolsRequireIdempotencyKeyBeforeAPI(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("API must not be called: %s %s", r.Method, r.URL.Path)
	}))
	defer api.Close()
	s := &Server{APIURL: api.URL, APIKey: "k", Client: api.Client()}
	line := `[{"description":"x","quantite":"1","prix_unitaire_ht":"10"}]`
	calls := map[string]string{
		"create_devis":   `{"client":"c","lignes":` + line + `}`,
		"create_facture": `{"depuis_devis_id":"q"}`,
		"create_acompte": `{"devis_id":"q","pourcentage":"30"}`,
	}
	for tool, args := range calls {
		for _, key := range []string{``, `,"idempotency_key":""`, `,"idempotency_key":"  "`, `,"idempotency_key":42`, `,"idempotency_key":"` + strings.Repeat("a", 256) + `"`, `,"idempotency_key":"bad key\n"`} {
			res, err := s.callTool(tool, json.RawMessage(args[:len(args)-1]+key+"}"))
			if err != nil {
				t.Fatal(err)
			}
			if res["status"] == "created" || res["status"] == nil {
				t.Fatalf("%s key %q: unexpected result %#v", tool, key, res)
			}
			if res["status"] == "clarification" {
				if !stringSetFromStrings(res["missing"]).has("idempotency_key") {
					t.Fatalf("%s key %q: %#v", tool, key, res)
				}
			}
		}
	}
}

type stringSetValue map[string]bool

func (s stringSetValue) has(k string) bool { return s[k] }

func stringSetFromStrings(v any) stringSetValue {
	out := stringSetValue{}
	if list, ok := v.([]string); ok {
		for _, item := range list {
			out[item] = true
		}
	}
	return out
}

func TestCreateToolsForwardIdempotencyKeyHeader(t *testing.T) {
	type seen struct{ method, path, key string }
	var calls []seen
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, seen{r.Method, r.URL.Path, r.Header.Get("Idempotency-Key")})
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/documents/quote-id":
			_, _ = w.Write([]byte(`{"document":{"id":"quote-id","subtotal_cents":100000,"lines":[{"description":"c","quantity_milli":1000,"unit_price_cents":100000,"tax_rate_bps":2000}]}}`))
		case r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"link":"https://example.test/d/x"}`))
		default:
			_, _ = w.Write([]byte(`{"document":{"id":"new-id"}}`))
		}
	}))
	defer api.Close()
	s := &Server{APIURL: api.URL, APIKey: "k", Client: api.Client()}
	line := `[{"description":"x","quantite":"1","prix_unitaire_ht":"10","tax_rate_bps":2000}]`
	tests := []struct{ tool, args, key, path string }{
		{"create_devis", `{"client":"c","lignes":` + line + `,"idempotency_key":"key-devis"}`, "key-devis", "/documents"},
		{"create_facture", `{"client":"c","lignes":` + line + `,"idempotency_key":"key-facture"}`, "key-facture", "/documents"},
		{"create_facture", `{"depuis_devis_id":"quote-id","idempotency_key":"key-convert"}`, "key-convert", "/documents/quote-id/convert-to-invoice"},
		{"create_acompte", `{"devis_id":"quote-id","pourcentage":"30","idempotency_key":"key-acompte"}`, "key-acompte", "/documents/quote-id/deposit-invoice"},
	}
	for _, tt := range tests {
		calls = nil
		res, err := s.callTool(tt.tool, json.RawMessage(tt.args))
		if err != nil || res["status"] != "created" {
			t.Fatalf("%s: res=%#v err=%v", tt.tool, res, err)
		}
		found := false
		for _, c := range calls {
			if c.method == http.MethodPost {
				found = true
				if c.path != tt.path || c.key != tt.key {
					t.Fatalf("%s: POST %s key=%q, want %s key=%q", tt.tool, c.path, c.key, tt.path, tt.key)
				}
			} else if c.key != "" {
				t.Fatalf("%s: GET %s must not carry an idempotency key", tt.tool, c.path)
			}
		}
		if !found {
			t.Fatalf("%s: no POST seen", tt.tool)
		}
	}
}

func TestRetryReusesSameIdempotencyKey(t *testing.T) {
	var keys []string
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			keys = append(keys, r.Header.Get("Idempotency-Key"))
			_, _ = w.Write([]byte(`{"document":{"id":"same"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"link":"l"}`))
	}))
	defer api.Close()
	s := &Server{APIURL: api.URL, APIKey: "k", Client: api.Client()}
	args := json.RawMessage(`{"client":"c","lignes":[{"description":"x","quantite":"1","prix_unitaire_ht":"10"}],"idempotency_key":"retry-1"}`)
	for i := 0; i < 2; i++ {
		if _, err := s.callTool("create_devis", args); err != nil {
			t.Fatal(err)
		}
	}
	if len(keys) != 2 || keys[0] != "retry-1" || keys[1] != "retry-1" {
		t.Fatalf("keys=%v", keys)
	}
}
