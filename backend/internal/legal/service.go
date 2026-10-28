package legal

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
)

// Service orchestrates providers, cache and live lookups (search / VIES).
type Service struct {
	Store     Store
	Provider  Provider // optional fallback (defaults-v1)
	Recherche *RechercheClient
	VIES      *VIESClient
	TEDB      *TEDBProvider
	BCE       *BCEProvider
	Stubs     []Provider // Insee / PISTE placeholders
	Logger    *slog.Logger
}

// Get returns a cached variable, falling back to Provider then ErrNotConfigured.
func (s *Service) Get(ctx context.Context, key string) (Variable, error) {
	if s == nil || s.Store == nil {
		if s != nil && s.Provider != nil {
			return s.Provider.Fetch(ctx, key)
		}
		return Variable{}, ErrNotConfigured
	}
	v, status, err := s.Store.Latest(ctx, key)
	if err == nil {
		v.Status = status
		return v, nil
	}
	if s.Provider != nil {
		return s.Provider.Fetch(ctx, key)
	}
	return Variable{}, err
}

// Status lists all known legal keys with freshness.
func (s *Service) Status(ctx context.Context) ([]StatusEntry, error) {
	if s == nil || s.Store == nil {
		return nil, ErrNotConfigured
	}
	return s.Store.ListStatus(ctx)
}

// SearchCompanies proxies DINUM Recherche d'entreprises.
func (s *Service) SearchCompanies(ctx context.Context, q string, limit int) ([]CompanyHit, error) {
	if s == nil || s.Recherche == nil {
		return nil, ErrNotConfigured
	}
	return s.Recherche.Search(ctx, q, limit)
}

// ValidateVAT proxies VIES with cache + rate limit.
func (s *Service) ValidateVAT(ctx context.Context, vat string) (VATValidation, error) {
	if s == nil || s.VIES == nil {
		return VATValidation{}, ErrNotConfigured
	}
	return s.VIES.Validate(ctx, vat)
}

// MentionsForIssue builds Mentions from verified/manual cache, else defaults-v1.
func (s *Service) MentionsForIssue(ctx context.Context) Mentions {
	base := DefaultMentions()
	if s == nil || s.Store == nil {
		return base
	}

	if v, status, err := s.Store.Latest(ctx, KeyManualMentions); err == nil {
		var m struct {
			Franchise293B       string `json:"franchise_293b"`
			LatePenaltyTemplate string `json:"late_penalty_template"`
			FlatIndemnity40EUR  string `json:"flat_indemnity_40e"`
			PaymentTermsDefault string `json:"payment_terms_default"`
			QuoteValidityNote   string `json:"quote_validity_note"`
		}
		if json.Unmarshal(v.Value, &m) == nil {
			if m.Franchise293B != "" {
				base.Franchise293B = m.Franchise293B
			}
			if m.FlatIndemnity40EUR != "" {
				base.FlatIndemnity40EUR = m.FlatIndemnity40EUR
			}
			if m.PaymentTermsDefault != "" {
				base.PaymentTermsDefault = m.PaymentTermsDefault
			}
			if m.QuoteValidityNote != "" {
				base.QuoteValidityNote = m.QuoteValidityNote
			}
			base.Source = v.Source
			base.FetchedAt = v.FetchedAt
			_ = status
		}
	}

	if v, _, err := s.Store.Latest(ctx, KeyFlatIndemnity); err == nil {
		var fi struct {
			Cents int64 `json:"cents"`
		}
		if json.Unmarshal(v.Value, &fi) == nil && fi.Cents > 0 {
			base.FlatIndemnityCents = fi.Cents
		}
	}

	rate := base.LatePenaltyRateAnnual
	rateSource := base.Source
	rateAt := base.FetchedAt

	if v, _, err := s.Store.Latest(ctx, KeyLatePenaltyEffective); err == nil {
		var d struct {
			AnnualPercent float64 `json:"annual_percent"`
		}
		if json.Unmarshal(v.Value, &d) == nil && d.AnnualPercent > 0 {
			rate = d.AnnualPercent
			rateSource = v.Source
			rateAt = v.FetchedAt
		}
	} else if v, _, err := s.Store.Latest(ctx, KeyLegalInterestRate); err == nil {
		var li struct {
			AnnualPercent float64 `json:"annual_percent"`
		}
		if json.Unmarshal(v.Value, &li) == nil && li.AnnualPercent > 0 {
			rate = math.Round(li.AnnualPercent*3*100) / 100
			rateSource = v.Source
			rateAt = v.FetchedAt
		}
	}

	base.LatePenaltyRateAnnual = rate
	base.LatePenalty = fmt.Sprintf(
		"En cas de retard de paiement, des pénalités calculées au taux annuel de %g %% seront exigibles "+
			"(taux minimal légal : 3 fois le taux d'intérêt légal ; le taux BCE majoré peut s'appliquer selon le contrat).",
		rate,
	)
	if rateSource != DefaultsSource {
		base.Source = rateSource
		base.FetchedAt = rateAt
	}

	// Prefer composite source label when we left defaults entirely.
	if base.Source == DefaultsSource {
		if v, _, err := s.Store.Latest(ctx, KeyManualMentions); err == nil && v.Source != "" {
			base.Source = v.Source
			base.FetchedAt = v.FetchedAt
		}
	}
	return base
}

// DeriveLatePenaltyRate computes max(3×legal interest, BCE+10) when inputs exist.
// Rule documented in DECISIONS.md — must be validated legally.
func DeriveLatePenaltyRate(legalInterest, bceRate float64) (float64, string) {
	threeTimes := legalInterest * 3
	bcePlus10 := bceRate + 10
	if legalInterest <= 0 && bceRate <= 0 {
		return 0, ""
	}
	if legalInterest <= 0 {
		return round2(bcePlus10), "bce_plus_10"
	}
	if bceRate <= 0 {
		return round2(threeTimes), "3x_legal_interest"
	}
	if threeTimes >= bcePlus10 {
		return round2(threeTimes), "3x_legal_interest"
	}
	return round2(bcePlus10), "bce_plus_10"
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
