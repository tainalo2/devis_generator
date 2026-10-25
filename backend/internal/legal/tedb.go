package legal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// DefaultTEDBURL is the public TEDB SOAP endpoint (no API key).
const DefaultTEDBURL = "https://ec.europa.eu/taxation_customs/tedb/ws/VatRetrievalService"

// SOAP / XSD namespaces (WSDL targetNamespace vs message/types schemas diverge):
//   - WSDL / SOAPAction:  urn:...:v1:VatRetrievalService
//   - Request element:    urn:...:v1:IVatRetrievalService
//   - Child elements:     urn:...:v1:IVatRetrievalService:types (elementFormDefault=qualified)
const (
	tedbSOAPAction = "urn:ec.europa.eu:taxud:tedb:services:v1:VatRetrievalService/RetrieveVatRates"
	tedbMsgNS      = "urn:ec.europa.eu:taxud:tedb:services:v1:IVatRetrievalService"
	tedbTypesNS    = "urn:ec.europa.eu:taxud:tedb:services:v1:IVatRetrievalService:types"
)

// TEDBProvider fetches FR VAT rates from the Commission TEDB SOAP service.
type TEDBProvider struct {
	URL     string
	Client  HTTPClient
	Limiter *RateLimiter
	Retry   RetryPolicy
}

func NewTEDBProvider(httpClient HTTPClient) *TEDBProvider {
	if httpClient == nil {
		httpClient = NewHTTPClient(15 * time.Second)
	}
	return &TEDBProvider{
		URL:     DefaultTEDBURL,
		Client:  httpClient,
		Limiter: NewRateLimiter(2 * time.Second),
		Retry:   DefaultRetryPolicy(),
	}
}

func (p *TEDBProvider) Name() string { return "tedb.ec.europa.eu" }

func (p *TEDBProvider) Fetch(ctx context.Context, key string) (Variable, error) {
	if key != "" && key != KeyTEDBVATRatesFR {
		return Variable{}, ErrNotConfigured
	}
	if err := p.Limiter.Wait(ctx, "tedb"); err != nil {
		return Variable{}, err
	}

	situationOn := time.Now().UTC().Format("2006-01-02")
	soap := buildTEDBRequest("FR", situationOn)

	var rates VATRates
	err := DoWithRetry(ctx, p.Retry, func(ctx context.Context) error {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.URL, bytes.NewReader([]byte(soap)))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "text/xml; charset=utf-8")
		req.Header.Set("SOAPAction", tedbSOAPAction)
		req.Header.Set("Accept", "text/xml")
		res, err := p.Client.Do(req)
		if err != nil {
			return err
		}
		defer res.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(res.Body, 2<<20))
		text := string(body)
		if fault := tedbFaultMessage(text); fault != "" {
			status := res.StatusCode
			if status < 400 {
				status = http.StatusBadRequest
			}
			return &HTTPError{Status: status, Body: fault}
		}
		if res.StatusCode < 200 || res.StatusCode >= 300 {
			return &HTTPError{Status: res.StatusCode, Body: truncate(text, 500)}
		}
		parsed, err := parseTEDBResponse(text)
		if err != nil {
			return err
		}
		rates = parsed
		return nil
	})
	if err != nil {
		return Variable{}, err
	}
	rates.MemberState = "FR"
	if rates.SituationOn == "" {
		rates.SituationOn = situationOn
	}
	raw, err := json.Marshal(rates)
	if err != nil {
		return Variable{}, err
	}
	return Variable{
		Key:       KeyTEDBVATRatesFR,
		Value:     raw,
		Source:    p.Name(),
		FetchedAt: time.Now().UTC(),
		Status:    StatusVerified,
	}, nil
}

// VATRates is the cached shape for tedb.vat_rates.fr.
type VATRates struct {
	MemberState string    `json:"member_state"`
	SituationOn string    `json:"situation_on"`
	Standard    float64   `json:"standard_percent"`
	Reduced     []float64 `json:"reduced_percent"`
	Note        string    `json:"note,omitempty"`
}

// buildTEDBRequest builds a document/literal SOAP envelope that validates against
// VatRetrievalServiceMessage.xsd + VatRetrievalServiceType.xsd.
func buildTEDBRequest(iso, situationOn string) string {
	iso = strings.ToUpper(strings.TrimSpace(iso))
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:urn="%s" xmlns:t="%s">
  <soapenv:Header/>
  <soapenv:Body>
    <urn:retrieveVatRatesReqMsg>
      <t:memberStates>
        <t:isoCode>%s</t:isoCode>
      </t:memberStates>
      <t:situationOn>%s</t:situationOn>
    </urn:retrieveVatRatesReqMsg>
  </soapenv:Body>
</soapenv:Envelope>`, tedbMsgNS, tedbTypesNS, iso, situationOn)
}

var (
	reFaultString = regexp.MustCompile(`(?is)<faultstring>\s*([^<]+)\s*</faultstring>`)
	reFaultDesc   = regexp.MustCompile(`(?is)<(?:\w+:)?description>\s*([^<]+)\s*</(?:\w+:)?description>`)
	reResultBlock = regexp.MustCompile(`(?is)<(?:\w+:)?vatRateResults\b[^>]*>[\s\S]*?</(?:\w+:)?vatRateResults>`)
	reOuterType   = regexp.MustCompile(`(?is)<(?:\w+:)?type>\s*(STANDARD|REDUCED)\s*</(?:\w+:)?type>`)
	reRateInner   = regexp.MustCompile(`(?is)<(?:\w+:)?rate\b[^>]*>\s*<(?:\w+:)?type>\s*([^<]+)\s*</(?:\w+:)?type>\s*(?:<(?:\w+:)?value>\s*([0-9]+(?:\.[0-9]+)?)\s*</(?:\w+:)?value>)?`)
	reSituationOn = regexp.MustCompile(`(?is)<(?:\w+:)?situationOn>\s*([^<]+)\s*</(?:\w+:)?situationOn>`)
)

func tedbFaultMessage(xmlBody string) string {
	if !strings.Contains(strings.ToLower(xmlBody), "fault") {
		return ""
	}
	if !strings.Contains(xmlBody, "Fault") && !strings.Contains(strings.ToLower(xmlBody), ":fault") {
		return ""
	}
	parts := make([]string, 0, 2)
	if m := reFaultString.FindStringSubmatch(xmlBody); len(m) >= 2 {
		parts = append(parts, strings.TrimSpace(m[1]))
	}
	if m := reFaultDesc.FindStringSubmatch(xmlBody); len(m) >= 2 {
		parts = append(parts, strings.TrimSpace(m[1]))
	}
	if len(parts) == 0 {
		if strings.Contains(xmlBody, "Fault") {
			return truncate(xmlBody, 300)
		}
		return ""
	}
	return strings.Join(parts, " — ")
}

func parseTEDBResponse(xmlBody string) (VATRates, error) {
	if fault := tedbFaultMessage(xmlBody); fault != "" {
		return VATRates{}, fmt.Errorf("TEDB SOAP fault: %s", fault)
	}

	var out VATRates
	if m := reSituationOn.FindStringSubmatch(xmlBody); len(m) >= 2 {
		out.SituationOn = strings.TrimSpace(m[1])
		// Keep YYYY-MM-DD when TEDB returns xs:date with offset (e.g. 2026-07-01+02:00).
		if len(out.SituationOn) >= 10 && out.SituationOn[4] == '-' && out.SituationOn[7] == '-' {
			out.SituationOn = out.SituationOn[:10]
		}
	}

	for _, block := range reResultBlock.FindAllString(xmlBody, -1) {
		tm := reOuterType.FindStringSubmatch(block)
		rm := reRateInner.FindStringSubmatch(block)
		if len(tm) < 2 || len(rm) < 2 {
			continue
		}
		kind := strings.ToUpper(strings.TrimSpace(tm[1]))
		rateKind := strings.ToUpper(strings.TrimSpace(rm[1]))
		switch rateKind {
		case "EXEMPTED", "NOT_APPLICABLE", "OUT_OF_SCOPE":
			continue
		}
		if len(rm) < 3 || strings.TrimSpace(rm[2]) == "" {
			continue
		}
		v, err := strconv.ParseFloat(rm[2], 64)
		if err != nil || v <= 0 {
			continue
		}
		switch kind {
		case "STANDARD":
			// Prefer DEFAULT; otherwise keep the highest STANDARD value seen.
			if rateKind == "DEFAULT" || out.Standard == 0 || v > out.Standard {
				out.Standard = v
			}
		case "REDUCED":
			out.Reduced = appendUniqueFloat(out.Reduced, v)
		}
	}

	if out.Standard == 0 && len(out.Reduced) == 0 {
		return VATRates{}, fmt.Errorf("aucune taux TVA trouvé dans la réponse TEDB")
	}
	if out.Standard == 0 {
		out.Standard = 20
		out.Note = "taux standard absent de la réponse ; 20 % utilisé comme repère FR (repli documenté)"
	}
	return out, nil
}

func appendUniqueFloat(in []float64, v float64) []float64 {
	for _, x := range in {
		if x == v {
			return in
		}
	}
	return append(in, v)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
