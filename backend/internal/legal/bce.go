package legal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// DefaultBCEURL is the ECB SDMX series for the main refinancing rate (MRR fixed).
const DefaultBCEURL = "https://data-api.ecb.europa.eu/service/data/FM/B.U2.EUR.4F.KR.MRR_FR.LEV?lastNObservations=1&format=jsondata"

// BCEProvider fetches the ECB key interest rate via SDMX JSON.
type BCEProvider struct {
	URL     string
	Client  HTTPClient
	Limiter *RateLimiter
	Retry   RetryPolicy
}

func NewBCEProvider(httpClient HTTPClient) *BCEProvider {
	if httpClient == nil {
		httpClient = NewHTTPClient(10 * time.Second)
	}
	return &BCEProvider{
		URL:     DefaultBCEURL,
		Client:  httpClient,
		Limiter: NewRateLimiter(time.Second),
		Retry:   DefaultRetryPolicy(),
	}
}

func (p *BCEProvider) Name() string { return "data-api.ecb.europa.eu" }

func (p *BCEProvider) Fetch(ctx context.Context, key string) (Variable, error) {
	if key != "" && key != KeyBCEKeyRate {
		return Variable{}, ErrNotConfigured
	}
	if err := p.Limiter.Wait(ctx, "bce"); err != nil {
		return Variable{}, err
	}

	var rate KeyRate
	err := DoWithRetry(ctx, p.Retry, func(ctx context.Context) error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.URL, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Accept", "application/json")
		res, err := p.Client.Do(req)
		if err != nil {
			return err
		}
		defer res.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(res.Body, 2<<20))
		if res.StatusCode < 200 || res.StatusCode >= 300 {
			return &HTTPError{Status: res.StatusCode, Body: truncate(string(body), 500)}
		}
		parsed, err := parseBCEJSON(body)
		if err != nil {
			return err
		}
		rate = parsed
		return nil
	})
	if err != nil {
		return Variable{}, err
	}
	raw, err := json.Marshal(rate)
	if err != nil {
		return Variable{}, err
	}
	return Variable{
		Key:       KeyBCEKeyRate,
		Value:     raw,
		Source:    p.Name(),
		FetchedAt: time.Now().UTC(),
		Status:    StatusVerified,
	}, nil
}

// KeyRate is the cached shape for bce.key_rate.
type KeyRate struct {
	AnnualPercent float64 `json:"annual_percent"`
	Period        string  `json:"period,omitempty"`
	Series        string  `json:"series"`
	Note          string  `json:"note,omitempty"`
}

func parseBCEJSON(body []byte) (KeyRate, error) {
	var root struct {
		DataSets []struct {
			Series map[string]struct {
				Observations map[string][]any `json:"observations"`
			} `json:"series"`
		} `json:"dataSets"`
		Structure struct {
			Dimensions struct {
				Observation []struct {
					Values []struct {
						ID string `json:"id"`
					} `json:"values"`
				} `json:"observation"`
			} `json:"dimensions"`
		} `json:"structure"`
	}
	if err := json.Unmarshal(body, &root); err != nil {
		return KeyRate{}, err
	}
	if len(root.DataSets) == 0 {
		return KeyRate{}, fmt.Errorf("réponse BCE sans dataSets")
	}
	for _, series := range root.DataSets[0].Series {
		for idx, obs := range series.Observations {
			if len(obs) == 0 {
				continue
			}
			var pct float64
			switch v := obs[0].(type) {
			case float64:
				pct = v
			case string:
				f, err := strconv.ParseFloat(v, 64)
				if err != nil {
					continue
				}
				pct = f
			default:
				continue
			}
			period := idx
			if i, err := strconv.Atoi(idx); err == nil {
				obsDims := root.Structure.Dimensions.Observation
				if len(obsDims) > 0 && i >= 0 && i < len(obsDims[0].Values) {
					period = obsDims[0].Values[i].ID
				}
			}
			return KeyRate{
				AnnualPercent: pct,
				Period:        period,
				Series:        "FM.B.U2.EUR.4F.KR.MRR_FR.LEV",
				Note:          "Taux directeur BCE (opérations principales de refinancement)",
			}, nil
		}
	}
	return KeyRate{}, fmt.Errorf("aucune observation dans la réponse BCE")
}
