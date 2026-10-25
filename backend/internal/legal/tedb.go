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

const DefaultTEDBURL = "https://ec.europa.eu/taxation_customs/tedb/ws/VatRetrievalService"

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
		req.Header.Set("SOAPAction", "urn:ec.europa.eu:taxud:tedb:services:v1:VatRetrievalService/RetrieveVatRates")
		req.Header.Set("Accept", "text/xml")
		res, err := p.Client.Do(req)
		if err != nil {
			return err
		}
		defer res.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(res.Body, 2<<20))
		if res.StatusCode < 200 || res.StatusCode >= 300 {
			return &HTTPError{Status: res.StatusCode, Body: truncate(string(body), 500)}
		}
		parsed, err := parseTEDBResponse(string(body))
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
	rates.SituationOn = situationOn
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

func buildTEDBRequest(iso, situationOn string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:urn="urn:ec.europa.eu:taxud:tedb:services:v1:VatRetrievalService">
  <soapenv:Header/>
  <soapenv:Body>
    <urn:retrieveVatRatesReqMsg>
      <memberStates><isoCode>%s</isoCode></memberStates>
      <situationOn>%s</situationOn>
    </urn:retrieveVatRatesReqMsg>
  </soapenv:Body>
</soapenv:Envelope>`, iso, situationOn)
}

var (
	reType    = regexp.MustCompile(`(?i)<type>\s*(STANDARD|REDUCED)\s*</type>`)
	reRateVal = regexp.MustCompile(`(?i)<rate[^>]*>[\s\S]*?<value>\s*([0-9]+(?:\.[0-9]+)?)\s*</value>`)
	reSimple  = regexp.MustCompile(`(?i)<(?:rate|value)>\s*([0-9]+(?:\.[0-9]+)?)\s*</(?:rate|value)>`)
)

func parseTEDBResponse(xmlBody string) (VATRates, error) {
	// Prefer structured blocks; fall back to any numeric rate tags.
	var out VATRates
	blocks := strings.Split(xmlBody, "vatRateResults")
	for _, block := range blocks {
		tm := reType.FindStringSubmatch(block)
		vm := reRateVal.FindStringSubmatch(block)
		if len(vm) < 2 {
			continue
		}
		v, err := strconv.ParseFloat(vm[1], 64)
		if err != nil {
			continue
		}
		kind := ""
		if len(tm) >= 2 {
			kind = strings.ToUpper(tm[1])
		}
		switch kind {
		case "STANDARD":
			if out.Standard == 0 || v > out.Standard {
				out.Standard = v
			}
		case "REDUCED":
			out.Reduced = appendUniqueFloat(out.Reduced, v)
		default:
			if v >= 15 {
				out.Standard = v
			} else {
				out.Reduced = appendUniqueFloat(out.Reduced, v)
			}
		}
	}
	if out.Standard == 0 {
		for _, m := range reSimple.FindAllStringSubmatch(xmlBody, -1) {
			if len(m) < 2 {
				continue
			}
			v, err := strconv.ParseFloat(m[1], 64)
			if err != nil {
				continue
			}
			if v >= 15 && (out.Standard == 0 || v > out.Standard) {
				out.Standard = v
			} else if v > 0 && v < 15 {
				out.Reduced = appendUniqueFloat(out.Reduced, v)
			}
		}
	}
	if out.Standard == 0 && len(out.Reduced) == 0 {
		return VATRates{}, fmt.Errorf("aucune taux TVA trouvé dans la réponse TEDB")
	}
	if out.Standard == 0 {
		out.Standard = 20 // FR fallback only if SOAP omitted STANDARD but had reduced
		out.Note = "taux standard absent de la réponse ; 20 % utilisé comme repère FR"
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
