package legal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode"
)

const DefaultVIESURL = "https://ec.europa.eu/taxation_customs/vies/rest-api/check-vat-number"

// VATValidation is the result of a VIES check (possibly from cache).
type VATValidation struct {
	CountryCode string    `json:"country_code"`
	VATNumber   string    `json:"vat_number"`
	Valid       bool      `json:"valid"`
	Name        string    `json:"name,omitempty"`
	Address     string    `json:"address,omitempty"`
	CheckedAt   time.Time `json:"checked_at"`
	Source      string    `json:"source"`
	Cached      bool      `json:"cached"`
}

// VIESClient validates EU VAT numbers via the Commission REST API.
// Repeated checks of the same number are cached and rate-limited (VIES blocks
// rapid re-checks of identical numbers).
type VIESClient struct {
	URL     string
	Client  HTTPClient
	Limiter *RateLimiter
	Retry   RetryPolicy
	CacheFor time.Duration

	mu    sync.Mutex
	cache map[string]cachedVAT
}

type cachedVAT struct {
	v   VATValidation
	exp time.Time
}

func NewVIESClient(httpClient HTTPClient) *VIESClient {
	if httpClient == nil {
		httpClient = NewHTTPClient(10 * time.Second)
	}
	return &VIESClient{
		URL:      DefaultVIESURL,
		Client:   httpClient,
		Limiter:  NewRateLimiter(1500 * time.Millisecond),
		Retry:    RetryPolicy{MaxAttempts: 2, BaseDelay: 500 * time.Millisecond},
		CacheFor: 24 * time.Hour,
		cache:    map[string]cachedVAT{},
	}
}

func (c *VIESClient) Name() string { return "vies.ec.europa.eu" }

// Validate parses a full VAT id (e.g. FR12345678901) or country+number.
func (c *VIESClient) Validate(ctx context.Context, vat string) (VATValidation, error) {
	cc, num, err := ParseVATNumber(vat)
	if err != nil {
		return VATValidation{}, err
	}
	return c.ValidateParts(ctx, cc, num)
}

func (c *VIESClient) ValidateParts(ctx context.Context, country, number string) (VATValidation, error) {
	country = strings.ToUpper(strings.TrimSpace(country))
	number = strings.ToUpper(digitsAndLetters(number))
	if len(country) != 2 || number == "" {
		return VATValidation{}, fmt.Errorf("numéro de TVA invalide")
	}
	key := country + number

	if v, ok := c.getCache(key); ok {
		v.Cached = true
		return v, nil
	}

	// Per-number gate: VIES blocks repeated checks of the same VAT id.
	if err := c.Limiter.Wait(ctx, key); err != nil {
		return VATValidation{}, err
	}
	if err := c.Limiter.Wait(ctx, "global"); err != nil {
		return VATValidation{}, err
	}

	payload, _ := json.Marshal(map[string]string{
		"countryCode": country,
		"vatNumber":   number,
	})

	var out VATValidation
	err := DoWithRetry(ctx, c.Retry, func(ctx context.Context) error {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
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
		var parsed viesResponse
		if err := json.Unmarshal(body, &parsed); err != nil {
			return err
		}
		out = VATValidation{
			CountryCode: country,
			VATNumber:   number,
			Valid:       parsed.Valid,
			Name:        parsed.Name,
			Address:     parsed.Address,
			CheckedAt:   time.Now().UTC(),
			Source:      c.Name(),
			Cached:      false,
		}
		return nil
	})
	if err != nil {
		return VATValidation{}, err
	}
	c.putCache(key, out)
	return out, nil
}

func (c *VIESClient) getCache(key string) (VATValidation, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.cache[key]
	if !ok || time.Now().After(e.exp) {
		return VATValidation{}, false
	}
	return e.v, true
}

func (c *VIESClient) putCache(key string, v VATValidation) {
	ttl := c.CacheFor
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[key] = cachedVAT{v: v, exp: time.Now().Add(ttl)}
}

type viesResponse struct {
	Valid   bool   `json:"valid"`
	Name    string `json:"name"`
	Address string `json:"address"`
}

// ParseVATNumber splits "FR123…" into country + national number.
func ParseVATNumber(raw string) (country, number string, err error) {
	raw = strings.ToUpper(strings.TrimSpace(raw))
	raw = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || r == '.' || r == '-' {
			return -1
		}
		return r
	}, raw)
	if len(raw) < 4 {
		return "", "", fmt.Errorf("numéro de TVA trop court")
	}
	if unicode.IsLetter(rune(raw[0])) && unicode.IsLetter(rune(raw[1])) {
		return raw[:2], raw[2:], nil
	}
	return "", "", fmt.Errorf("préfixe pays manquant (ex. FR…)")
}

func digitsAndLetters(s string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(s) {
		if (r >= '0' && r <= '9') || (r >= 'A' && r <= 'Z') {
			b.WriteRune(r)
		}
	}
	return b.String()
}
