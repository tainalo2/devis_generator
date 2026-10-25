package legal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const DefaultRechercheBaseURL = "https://recherche-entreprises.api.gouv.fr"

// CompanyHit is a normalized company result for form autofill.
type CompanyHit struct {
	SIREN      string `json:"siren"`
	SIRET      string `json:"siret"`
	Name       string `json:"name"`
	TradeName  string `json:"trade_name,omitempty"`
	Address    string `json:"address,omitempty"`
	PostalCode string `json:"postal_code,omitempty"`
	City       string `json:"city,omitempty"`
	NAF        string `json:"naf,omitempty"`
	VATNumber  string `json:"vat_number,omitempty"`
}

// RechercheClient calls the DINUM Recherche d'entreprises API (no API key).
type RechercheClient struct {
	BaseURL string
	Client  HTTPClient
	Limiter *RateLimiter
	Retry   RetryPolicy
}

func NewRechercheClient(httpClient HTTPClient) *RechercheClient {
	if httpClient == nil {
		httpClient = NewHTTPClient(8 * time.Second)
	}
	return &RechercheClient{
		BaseURL: DefaultRechercheBaseURL,
		Client:  httpClient,
		// API limit ≈ 7 req/s ; stay well under.
		Limiter: NewRateLimiter(200 * time.Millisecond),
		Retry:   DefaultRetryPolicy(),
	}
}

func (c *RechercheClient) Name() string { return "recherche-entreprises.api.gouv.fr" }

// Search returns up to limit company hits for q (name / SIREN / SIRET).
func (c *RechercheClient) Search(ctx context.Context, q string, limit int) ([]CompanyHit, error) {
	q = strings.TrimSpace(q)
	if len(q) < 2 {
		return nil, fmt.Errorf("query too short")
	}
	if limit <= 0 || limit > 25 {
		limit = 10
	}
	if err := c.Limiter.Wait(ctx, "search"); err != nil {
		return nil, err
	}

	base := strings.TrimRight(c.BaseURL, "/")
	u, err := url.Parse(base + "/search")
	if err != nil {
		return nil, err
	}
	qs := u.Query()
	qs.Set("q", q)
	qs.Set("per_page", fmt.Sprintf("%d", limit))
	qs.Set("page", "1")
	u.RawQuery = qs.Encode()

	var hits []CompanyHit
	err = DoWithRetry(ctx, c.Retry, func(ctx context.Context) error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return err
		}
		req.Header.Set("Accept", "application/json")
		res, err := c.Client.Do(req)
		if err != nil {
			return err
		}
		defer res.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
		if res.StatusCode < 200 || res.StatusCode >= 300 {
			return &HTTPError{Status: res.StatusCode, Body: string(body)}
		}
		var parsed rechercheResponse
		if err := json.Unmarshal(body, &parsed); err != nil {
			return err
		}
		hits = make([]CompanyHit, 0, len(parsed.Results))
		for _, r := range parsed.Results {
			hit := CompanyHit{
				SIREN:     r.SIREN,
				Name:      firstNonEmpty(r.NomComplet, r.NomRaisonSociale),
				TradeName: r.Sigle,
			}
			if r.Siege != nil {
				hit.SIRET = r.Siege.SIRET
				hit.Address = firstNonEmpty(r.Siege.GeoAdresse, r.Siege.Adresse)
				hit.PostalCode = r.Siege.CodePostal
				hit.City = firstNonEmpty(r.Siege.LibelleCommune, r.Siege.Commune)
				hit.NAF = r.Siege.ActivitePrincipale
			}
			if hit.SIREN != "" {
				hit.VATNumber = "FR" + frenchVATKey(hit.SIREN) + hit.SIREN
			}
			hits = append(hits, hit)
		}
		return nil
	})
	return hits, err
}

type rechercheResponse struct {
	Results []rechercheResult `json:"results"`
}

type rechercheResult struct {
	SIREN            string          `json:"siren"`
	NomComplet       string          `json:"nom_complet"`
	NomRaisonSociale string          `json:"nom_raison_sociale"`
	Sigle            string          `json:"sigle"`
	Siege            *rechercheSiege `json:"siege"`
}

type rechercheSiege struct {
	SIRET              string `json:"siret"`
	Adresse            string `json:"adresse"`
	GeoAdresse         string `json:"geo_adresse"`
	CodePostal         string `json:"code_postal"`
	Commune            string `json:"commune"`
	LibelleCommune     string `json:"libelle_commune"`
	ActivitePrincipale string `json:"activite_principale"`
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// frenchVATKey computes the 2-digit key for a French intra-community VAT number.
func frenchVATKey(siren string) string {
	siren = digitsOnly(siren)
	if len(siren) != 9 {
		return "00"
	}
	var n int
	for _, c := range siren {
		n = n*10 + int(c-'0')
	}
	key := (12 + 3*(n%97)) % 97
	return fmt.Sprintf("%02d", key)
}

func digitsOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
