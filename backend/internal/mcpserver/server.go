package mcpserver

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/big"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/tainalo2/devis_generator/backend/internal/billing"
)

type Server struct {
	APIURL string
	APIKey string
	Client *http.Client
	In     io.Reader
	Out    io.Writer
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

const (
	decimalSchemaPattern = `^[+-]?([0-9]+(\.[0-9]*)?|\.[0-9]+)$`
	// positiveDecimalSchemaPattern : quantite n'accepte aucun signe négatif ;
	// un « + » explicite reste toléré comme par la conversion. Zéro est
	// syntaxiquement valide mais refusé à l'exécution (strictement positive).
	positiveDecimalSchemaPattern = `^\+?([0-9]+(\.[0-9]*)?|\.[0-9]+)$`
	maxDecimalLength             = 128
	idempotencyKeySchemaPattern  = `^[A-Za-z0-9._:/-]{1,255}$`
)

var idempotencyKeyPattern = regexp.MustCompile(idempotencyKeySchemaPattern)

func idempotencyKeyProperty() map[string]any {
	return map[string]any{
		"type": "string", "minLength": 1, "maxLength": 255, "pattern": idempotencyKeySchemaPattern,
		"description": "Clé d'idempotence obligatoire (1 à 255 caractères : A-Z a-z 0-9 . _ : / -). " +
			"Réutiliser la MÊME clé pour retenter une création sans la dupliquer ; une clé neuve (ex. UUID) pour une nouvelle création. " +
			"Même clé + paramètres différents = erreur de conflit. Les clés sont conservées 24 h.",
	}
}

// linesSchema décrit le format public d'une ligne ; il doit rester aligné sur
// convertMCPLines et sur billing.Compute.
func linesSchema() map[string]any {
	taxRates := make([]any, 0, len(billing.AllowedTaxRatesBPS))
	for _, rate := range billing.AllowedTaxRatesBPS {
		taxRates = append(taxRates, rate)
	}
	return map[string]any{
		"type":        "array",
		"maxItems":    billing.MaxLines,
		"description": "Lignes du document, au format MCP public (montants et quantités en chaînes décimales).",
		"items": map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"required":             []string{"description", "quantite", "prix_unitaire_ht"},
			"properties": map[string]any{
				"description": map[string]any{
					"type": "string", "minLength": 1, "maxLength": billing.MaxDescriptionLen,
					"description": "Désignation de la ligne (1 à 2000 caractères).",
				},
				"quantite": map[string]any{
					"type": "string", "pattern": positiveDecimalSchemaPattern, "maxLength": maxDecimalLength,
					"description": "Quantité en unités, chaîne décimale STRICTEMENT positive, sans signe négatif (ex. \"2.5\"). Le motif accepte \"0\" syntaxiquement mais la valeur est refusée à l'exécution. Arrondie au millième, demis éloignés de zéro ; une valeur arrondie à zéro est refusée.",
				},
				"prix_unitaire_ht": map[string]any{
					"type": "string", "pattern": decimalSchemaPattern, "maxLength": maxDecimalLength,
					"description": "Prix unitaire HT en euros, chaîne décimale (ex. \"49.90\"). Arrondi au centime, demis éloignés de zéro ; un prix négatif est autorisé (ligne de crédit).",
				},
				"tax_rate_bps": map[string]any{
					"type": "integer", "enum": taxRates, "default": 0,
					"description": "Taux de TVA en points de base : 0, 210, 550, 1000 ou 2000 (défaut 0). Ignoré (forcé à 0) en régime franchise_293b : omis, 0 ou toute autre valeur y donnent le même document et la même empreinte d'idempotence.",
				},
				"discount_bps": map[string]any{
					"type": "integer", "minimum": 0, "maximum": 10000, "default": 0,
					"description": "Remise de ligne en points de base, de 0 à 10000 (défaut 0).",
				},
			},
		},
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
			"client": str("client"), "lignes": linesSchema(), "devise": str("devise ISO"), "valide_jusqu_au": str("date"), "delai_validite_jours": str("jours"),
			"idempotency_key": idempotencyKeyProperty(),
		}, []string{"client", "lignes", "idempotency_key"}),
		mk("create_facture", "Créer une facture", map[string]any{
			"depuis_devis_id": str("id devis"), "client": str("client"), "lignes": linesSchema(),
			"date_echeance": str("date"), "delai_paiement_jours": str("jours"),
			"idempotency_key": idempotencyKeyProperty(),
		}, []string{"idempotency_key"}),
		mk("create_acompte", "Créer un acompte", map[string]any{
			"devis_id": str("id devis"), "pourcentage": str("pct"), "montant_ht": str("montant"),
			"idempotency_key": idempotencyKeyProperty(),
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
	return s.doAPIWithHeaders(method, path, body, nil)
}

func (s *Server) doAPIWithHeaders(method, path string, body any, headers map[string]string) (map[string]any, error) {
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req, _ := http.NewRequest(method, s.APIURL+path, &buf)
	req.Header.Set("Authorization", "Bearer "+s.APIKey)
	req.Header.Set("Content-Type", "application/json")
	for name, value := range headers {
		req.Header.Set(name, value)
	}
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

var decimalPattern = regexp.MustCompile(`^[+-]?(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+)$`)

// scaledDecimal converts a documented decimal string to an integer using
// commercial rounding: an exact half is rounded away from zero.
func scaledDecimal(value, field string, scale int64) (int64, error) {
	if len(value) > maxDecimalLength {
		return 0, fmt.Errorf("%s trop longue (%d caractères maximum)", field, maxDecimalLength)
	}
	if !decimalPattern.MatchString(value) {
		return 0, fmt.Errorf("%s invalide: chaîne décimale attendue (ex. \"2.5\")", field)
	}
	rational, ok := new(big.Rat).SetString(value)
	if !ok {
		return 0, fmt.Errorf("%s invalide: chaîne décimale attendue (ex. \"2.5\")", field)
	}
	scaled := new(big.Rat).Mul(rational, big.NewRat(scale, 1))
	numerator := new(big.Int).Set(scaled.Num())
	negative := numerator.Sign() < 0
	numerator.Abs(numerator)
	denominator := scaled.Denom()
	quotient, remainder := new(big.Int), new(big.Int)
	quotient.QuoRem(numerator, denominator, remainder)
	if remainder.Lsh(remainder, 1).Cmp(denominator) >= 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	if negative {
		quotient.Neg(quotient)
	}
	if !quotient.IsInt64() {
		return 0, fmt.Errorf("%s hors limites", field)
	}
	return quotient.Int64(), nil
}

var mcpLineFields = map[string]bool{
	"description": true, "quantite": true, "prix_unitaire_ht": true, "tax_rate_bps": true, "discount_bps": true,
}

// boundedInt lit un entier JSON (décodé en float64) dans [min,max].
func boundedInt(line map[string]any, field string, line1 int, allowed func(int) bool, hint string) (int, bool, error) {
	raw, ok := line[field]
	if !ok {
		return 0, false, nil
	}
	number, ok := raw.(float64)
	if !ok || number != math.Trunc(number) || number < math.MinInt32 || number > math.MaxInt32 || !allowed(int(number)) {
		return 0, false, fmt.Errorf("ligne %d: %s invalide (%s)", line1, field, hint)
	}
	return int(number), true, nil
}

// convertMCPLines valide les lignes publiques et les convertit au format API.
// Les quantités doivent être strictement positives (les prix négatifs restent
// permis pour les lignes de crédit). Toute propriété inconnue est refusée.
func convertMCPLines(rawLines []any) ([]map[string]any, error) {
	if len(rawLines) > billing.MaxLines {
		return nil, fmt.Errorf("lignes: trop de lignes (%d maximum)", billing.MaxLines)
	}
	lines := make([]map[string]any, 0, len(rawLines))
	for i, raw := range rawLines {
		line, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("ligne %d: objet attendu", i+1)
		}
		quantity, ok := line["quantite"].(string)
		if !ok {
			return nil, fmt.Errorf("ligne %d: quantite doit être une chaîne décimale", i+1)
		}
		quantityMilli, err := scaledDecimal(quantity, "quantite", 1000)
		if err != nil {
			return nil, fmt.Errorf("ligne %d: %w", i+1, err)
		}
		if quantityMilli == 0 {
			return nil, fmt.Errorf("ligne %d: quantite arrondie à zéro", i+1)
		}
		if quantityMilli < 0 {
			return nil, fmt.Errorf("ligne %d: quantite doit être strictement positive", i+1)
		}
		unitPrice, ok := line["prix_unitaire_ht"].(string)
		if !ok {
			return nil, fmt.Errorf("ligne %d: prix_unitaire_ht doit être une chaîne décimale", i+1)
		}
		unitPriceCents, err := scaledDecimal(unitPrice, "prix_unitaire_ht", 100)
		if err != nil {
			return nil, fmt.Errorf("ligne %d: %w", i+1, err)
		}
		for key := range line {
			if !mcpLineFields[key] {
				return nil, fmt.Errorf("ligne %d: propriété inconnue %q", i+1, key)
			}
		}
		description, ok := line["description"].(string)
		if !ok || strings.TrimSpace(description) == "" || utf8.RuneCountInString(description) > billing.MaxDescriptionLen {
			return nil, fmt.Errorf("ligne %d: description requise (chaîne de 1 à %d caractères)", i+1, billing.MaxDescriptionLen)
		}
		converted := map[string]any{
			"description":      description,
			"quantity_milli":   quantityMilli,
			"unit_price_cents": unitPriceCents,
		}
		taxAllowed := func(v int) bool {
			for _, rate := range billing.AllowedTaxRatesBPS {
				if rate == v {
					return true
				}
			}
			return false
		}
		if v, present, err := boundedInt(line, "tax_rate_bps", i+1, taxAllowed, "0, 210, 550, 1000 ou 2000"); err != nil {
			return nil, err
		} else if present {
			converted["tax_rate_bps"] = v
		}
		if v, present, err := boundedInt(line, "discount_bps", i+1, func(v int) bool { return v >= 0 && v <= 10000 }, "entier de 0 à 10000"); err != nil {
			return nil, err
		} else if present {
			converted["discount_bps"] = v
		}
		lines = append(lines, converted)
	}
	return lines, nil
}

// idempotencyKey extrait la clé obligatoire des créations. Une clé absente ou
// vide donne ok=false sans erreur (demande de clarification) ; une clé présente
// mais invalide donne une erreur.
func idempotencyKey(args map[string]any) (key string, ok bool, err error) {
	raw, present := args["idempotency_key"]
	if !present || raw == nil || raw == "" {
		return "", false, nil
	}
	key, isString := raw.(string)
	if !isString || !idempotencyKeyPattern.MatchString(key) {
		return "", false, fmt.Errorf("idempotency_key invalide (1 à 255 caractères parmi A-Z a-z 0-9 . _ : / -)")
	}
	return key, true, nil
}

func (s *Server) handleCreate(kind string, args json.RawMessage) (map[string]any, error) {
	a := parseArgs(args)
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
	key, hasKey, err := idempotencyKey(a)
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}, nil
	}
	if !hasKey {
		return map[string]any{"status": "clarification", "missing": []string{"idempotency_key"}, "alternatives": []string{}}, nil
	}
	var path string
	var body map[string]any
	switch kind {
	case "devis":
		lines, err := convertCreateLines(a["lignes"])
		if err != nil {
			return map[string]any{"status": "error", "error": err.Error()}, nil
		}
		path = "/documents"
		body = map[string]any{
			"kind":       "quote",
			"client_id":  a["client"],
			"lines":      lines,
			"currency":   valueOr(a, "devise", "EUR"),
			"vat_regime": valueOr(a, "regime_tva", "standard"),
			"title":      valueOr(a, "titre", "Devis"),
		}
		if due, ok := a["valide_jusqu_au"]; ok {
			body["due_date"] = due
		}
	case "facture":
		if quoteID, ok := a["depuis_devis_id"].(string); ok && quoteID != "" {
			path = "/documents/" + quoteID + "/convert-to-invoice"
			body = map[string]any{}
		} else {
			lines, err := convertCreateLines(a["lignes"])
			if err != nil {
				return map[string]any{"status": "error", "error": err.Error()}, nil
			}
			path = "/documents"
			body = map[string]any{
				"kind":       "invoice",
				"client_id":  a["client"],
				"lines":      lines,
				"currency":   valueOr(a, "devise", "EUR"),
				"vat_regime": valueOr(a, "regime_tva", "standard"),
				"title":      valueOr(a, "titre", "Facture"),
			}
		}
	case "acompte":
		return s.createDeposit(a, key), nil
	default:
		return map[string]any{"status": "error", "error": "création non prise en charge"}, nil
	}
	out, err := s.doAPIWithHeaders("POST", path, body, map[string]string{"Idempotency-Key": key})
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}, nil
	}
	return s.documentResult("created", out), nil
}

func convertCreateLines(value any) ([]map[string]any, error) {
	rawLines, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("lignes doit être un tableau")
	}
	return convertMCPLines(rawLines)
}

func (s *Server) createDeposit(args map[string]any, idempotencyKey string) map[string]any {
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
	created, err := s.doAPIWithHeaders(http.MethodPost, "/documents/"+quoteID+"/deposit-invoice", map[string]any{"lines": lines}, map[string]string{"Idempotency-Key": idempotencyKey})
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
