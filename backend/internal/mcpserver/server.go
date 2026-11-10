package mcpserver

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type Server struct {
	APIURL string
	APIKey string
	Client *http.Client
	In     io.Reader
	Out    io.Writer
	Now    func() time.Time
}

func New() *Server {
	return &Server{
		APIURL: strings.TrimRight(os.Getenv("VIBED_API_URL"), "/"),
		APIKey: os.Getenv("VIBED_API_KEY"),
		Client: &http.Client{},
		In:     os.Stdin,
		Out:    os.Stdout,
	}
}

type rpcReq struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type rpcResp struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcErr         `json:"error,omitempty"`
}

type rpcErr struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (s *Server) Run() error {
	sc := bufio.NewScanner(s.In)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var req rpcReq
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			s.write(rpcResp{JSONRPC: "2.0", Error: &rpcErr{Code: -32700, Message: "parse error"}})
			continue
		}
		if req.JSONRPC != "2.0" {
			s.write(rpcResp{JSONRPC: "2.0", ID: req.ID, Error: &rpcErr{Code: -32600, Message: "invalid request"}})
			continue
		}
		if len(req.ID) == 0 {
			// notification
			continue
		}
		res, rerr := s.handle(req)
		resp := rpcResp{JSONRPC: "2.0", ID: req.ID}
		if rerr != nil {
			resp.Error = rerr
		} else {
			resp.Result = res
		}
		s.write(resp)
	}
	return sc.Err()
}

func (s *Server) write(v any) {
	b, _ := json.Marshal(v)
	s.Out.Write(append(b, '\n'))
}

func (s *Server) handle(req rpcReq) (any, *rpcErr) {
	switch req.Method {
	case "initialize":
		return map[string]any{
			"protocolVersion": "2025-03-26",
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]string{"name": "devis-mcp", "version": "dev"},
		}, nil
	case "ping":
		return map[string]any{}, nil
	case "tools/list":
		return map[string]any{"tools": toolDefs()}, nil
	case "tools/call":
		var p struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return nil, &rpcErr{Code: -32602, Message: "invalid params"}
		}
		res, err := s.callTool(p.Name, p.Arguments)
		if err != nil {
			return map[string]any{"isError": true, "content": []any{map[string]string{"type": "text", "text": err.Error()}}}, nil
		}
		b, _ := json.Marshal(res)
		return map[string]any{"isError": false, "content": []any{map[string]string{"type": "text", "text": string(b)}}}, nil
	default:
		return nil, &rpcErr{Code: -32601, Message: "method not found"}
	}
}

func toolDefs() []map[string]any {
	mk := func(name, desc string, props map[string]any, req []string) map[string]any {
		return map[string]any{
			"name":        name,
			"description": desc,
			"inputSchema": map[string]any{"type": "object", "properties": props, "required": req},
		}
	}
	str := func(d string) map[string]any { return map[string]any{"type": "string", "description": d} }
	return []map[string]any{
		mk("create_devis", "Créer un devis", map[string]any{
			"client": str("client"), "lignes": map[string]any{"type": "array"}, "devise": str("devise ISO"), "valide_jusqu_au": str("date"), "delai_validite_jours": str("jours"), "idempotency_key": str("clé unique de création"),
		}, []string{"client", "lignes", "idempotency_key"}),
		mk("create_facture", "Créer une facture", map[string]any{
			"depuis_devis_id": str("id devis"), "client": str("client"), "lignes": map[string]any{"type": "array"},
			"date_echeance": str("date"), "delai_paiement_jours": str("jours"), "idempotency_key": str("clé unique de création"),
		}, []string{"idempotency_key"}),
		mk("create_acompte", "Créer un acompte", map[string]any{
			"devis_id": str("id devis"), "pourcentage": str("pct"), "montant_ht": str("montant"), "idempotency_key": str("clé unique de création"),
		}, []string{"devis_id", "idempotency_key"}),
		mk("get_document_link", "Lien direct d'un document", map[string]any{"id": str("id document")}, []string{"id"}),
		mk("list_documents", "Lister documents", map[string]any{"type": str("type"), "cursor": str("cursor")}, []string{}),
	}
}

func (s *Server) callTool(name string, args json.RawMessage) (map[string]any, error) {
	switch name {
	case "create_devis":
		return s.handleCreate("devis", args)
	case "create_facture":
		return s.handleCreate("facture", args)
	case "create_acompte":
		return s.handleCreate("acompte", args)
	case "get_document_link":
		return s.handleGetLink(args)
	case "list_documents":
		return s.handleList(args)
	default:
		return nil, fmt.Errorf("unknown tool %s", name)
	}
}

func (s *Server) doAPI(method, path string, body any) (map[string]any, error) {
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req, _ := http.NewRequest(method, s.APIURL+path, &buf)
	req.Header.Set("Authorization", "Bearer "+s.APIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out map[string]any
	json.NewDecoder(resp.Body).Decode(&out)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("api error %d: %v", resp.StatusCode, out)
	}
	return out, nil
}

func parseArgs(raw json.RawMessage) map[string]any {
	var m map[string]any
	json.Unmarshal(raw, &m)
	if m == nil {
		m = map[string]any{}
	}
	return m
}

func (s *Server) handleCreate(kind string, args json.RawMessage) (map[string]any, error) {
	a := parseArgs(args)
	if key, _ := a["idempotency_key"].(string); strings.TrimSpace(key) == "" {
		return map[string]any{"status": "clarification", "missing": []string{"idempotency_key"}, "alternatives": []string{}}, nil
	}
	// ambiguity checks
	switch kind {
	case "devis":
		if _, ok1 := a["valide_jusqu_au"]; ok1 {
			if _, ok2 := a["delai_validite_jours"]; ok2 {
				return map[string]any{"status": "clarification", "missing": []string{}, "alternatives": []string{"valide_jusqu_au", "delai_validite_jours"}}, nil
			}
		}
	case "facture":
		hasDevis := a["depuis_devis_id"] != nil
		hasClientLines := a["client"] != nil && a["lignes"] != nil
		if hasDevis == hasClientLines {
			// both or neither -> ambiguity
			if !(hasDevis || hasClientLines) {
				return map[string]any{"status": "clarification", "missing": []string{"depuis_devis_id or client+lignes"}, "alternatives": []string{}}, nil
			}
			return map[string]any{"status": "clarification", "missing": []string{}, "alternatives": []string{"depuis_devis_id", "client+lignes"}}, nil
		}
		if _, ok1 := a["date_echeance"]; ok1 {
			if _, ok2 := a["delai_paiement_jours"]; ok2 {
				return map[string]any{"status": "clarification", "missing": []string{}, "alternatives": []string{"date_echeance", "delai_paiement_jours"}}, nil
			}
		}
	case "acompte":
		if a["devis_id"] == nil {
			return map[string]any{"status": "clarification", "missing": []string{"devis_id"}, "alternatives": []string{}}, nil
		}
		hasPct := a["pourcentage"] != nil
		hasAmt := a["montant_ht"] != nil
		if hasPct == hasAmt {
			if !hasPct {
				return map[string]any{"status": "clarification", "missing": []string{"pourcentage or montant_ht"}, "alternatives": []string{}}, nil
			}
			return map[string]any{"status": "clarification", "missing": []string{}, "alternatives": []string{"pourcentage", "montant_ht"}}, nil
		}
	}
	var path string
	var body map[string]any
	switch kind {
	case "devis":
		path = "/documents"
		body = map[string]any{
			"kind":            "quote",
			"client_id":       a["client"],
			"lines":           a["lignes"],
			"currency":        valueOr(a, "devise", "EUR"),
			"vat_regime":      valueOr(a, "regime_tva", "standard"),
			"title":           valueOr(a, "titre", "Devis"),
			"idempotency_key": a["idempotency_key"],
		}
		body["issue_date"] = s.now().Format("2006-01-02")
		if due, ok := a["valide_jusqu_au"]; ok {
			body["due_date"] = due
		} else if delay, ok := a["delai_validite_jours"].(string); ok && delay != "" {
			days, err := strconv.Atoi(delay)
			if err != nil || days < 0 {
				return map[string]any{"status": "error", "error": "délai de validité invalide"}, nil
			}
			body["due_date"] = s.now().AddDate(0, 0, days).Format("2006-01-02")
		}
	case "facture":
		if quoteID, ok := a["depuis_devis_id"].(string); ok && quoteID != "" {
			path = "/documents/" + quoteID + "/convert-to-invoice"
			body = map[string]any{}
		} else {
			path = "/documents"
			body = map[string]any{
				"kind":            "invoice",
				"client_id":       a["client"],
				"lines":           a["lignes"],
				"currency":        valueOr(a, "devise", "EUR"),
				"vat_regime":      valueOr(a, "regime_tva", "standard"),
				"title":           valueOr(a, "titre", "Facture"),
				"idempotency_key": a["idempotency_key"],
			}
		}
		body["issue_date"] = s.now().Format("2006-01-02")
		if due, ok := a["date_echeance"]; ok {
			body["due_date"] = due
		} else if delay, ok := a["delai_paiement_jours"].(string); ok && delay != "" {
			days, err := strconv.Atoi(delay)
			if err != nil || days < 0 {
				return map[string]any{"status": "error", "error": "délai de paiement invalide"}, nil
			}
			body["due_date"] = s.now().AddDate(0, 0, days).Format("2006-01-02")
		}
		body["idempotency_key"] = a["idempotency_key"]
	case "acompte":
		return s.createDeposit(a), nil
	default:
		return map[string]any{"status": "error", "error": "création non prise en charge"}, nil
	}
	out, err := s.doAPI("POST", path, body)
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}, nil
	}
	return s.documentResult("created", out), nil
}

func (s *Server) createDeposit(args map[string]any) map[string]any {
	quoteID, _ := args["devis_id"].(string)
	out, err := s.doAPI(http.MethodGet, "/documents/"+quoteID, nil)
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}
	}
	document, ok := out["document"].(map[string]any)
	if !ok {
		return map[string]any{"status": "error", "error": "réponse document invalide"}
	}
	factor, err := depositFactor(args, document)
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}
	}
	rawLines, ok := document["lines"].([]any)
	if !ok || len(rawLines) == 0 {
		return map[string]any{"status": "error", "error": "devis sans lignes"}
	}
	lines := make([]any, 0, len(rawLines))
	for _, raw := range rawLines {
		line, ok := raw.(map[string]any)
		if !ok {
			return map[string]any{"status": "error", "error": "ligne de devis invalide"}
		}
		copyLine := make(map[string]any, len(line))
		for key, value := range line {
			copyLine[key] = value
		}
		price, ok := copyLine["unit_price_cents"].(float64)
		if !ok {
			return map[string]any{"status": "error", "error": "prix de ligne invalide"}
		}
		copyLine["unit_price_cents"] = math.Round(price * factor)
		lines = append(lines, copyLine)
	}
	created, err := s.doAPI(http.MethodPost, "/documents/"+quoteID+"/deposit-invoice", map[string]any{"lines": lines})
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}
	}
	return s.documentResult("created", created)
}

func depositFactor(args, document map[string]any) (float64, error) {
	if percentage, ok := args["pourcentage"].(string); ok && percentage != "" {
		value, err := strconv.ParseFloat(percentage, 64)
		if err != nil || value <= 0 || value > 100 {
			return 0, fmt.Errorf("pourcentage invalide")
		}
		return value / 100, nil
	}
	amount, _ := args["montant_ht"].(string)
	value, err := strconv.ParseFloat(amount, 64)
	subtotal, ok := document["subtotal_cents"].(float64)
	if err != nil || value <= 0 || !ok || subtotal <= 0 {
		return 0, fmt.Errorf("montant HT invalide")
	}
	return math.Round(value*100) / subtotal, nil
}

func valueOr(values map[string]any, key string, fallback any) any {
	if value, ok := values[key]; ok && value != nil && value != "" {
		return value
	}
	return fallback
}

func (s *Server) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Server) documentResult(status string, out map[string]any) map[string]any {
	id, _ := out["id"].(string)
	if id == "" {
		if d, ok := out["document"].(map[string]any); ok {
			id, _ = d["id"].(string)
		}
	}
	link := ""
	if id != "" {
		link = s.APIURL + "/documents/" + id
		if stable, err := s.doAPI(http.MethodGet, "/documents/"+id+"/link", nil); err == nil {
			if stableLink, ok := stable["link"].(string); ok && stableLink != "" {
				link = stableLink
			}
		}
	}
	return map[string]any{"status": status, "id": id, "link": link}
}

func (s *Server) handleGetLink(args json.RawMessage) (map[string]any, error) {
	a := parseArgs(args)
	id, _ := a["id"].(string)
	out, err := s.doAPI("GET", "/documents/"+id+"/link", nil)
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}, nil
	}
	link, _ := out["link"].(string)
	result := map[string]any{"status": "ok", "id": id, "link": link}
	for _, key := range []string{"permanent", "expires_at", "message"} {
		if value, ok := out[key]; ok {
			result[key] = value
		}
	}
	return result, nil
}

func (s *Server) handleList(args json.RawMessage) (map[string]any, error) {
	a := parseArgs(args)
	path := "/documents"
	qs := ""
	if t, ok := a["type"].(string); ok {
		qs = "?type=" + t
	}
	if c, ok := a["cursor"].(string); ok && c != "" {
		sep := "?"
		if qs != "" {
			sep = "&"
		}
		qs += sep + "cursor=" + c
	}
	out, err := s.doAPI("GET", path+qs, nil)
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}, nil
	}
	return map[string]any{"status": "ok", "result": out}, nil
}
