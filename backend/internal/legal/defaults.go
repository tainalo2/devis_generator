package legal

import (
	"context"
	"encoding/json"
	"time"
)

// DefaultMentions holds V1 legal copy used until official sources are wired.
// Source label is always "defaults-v1" so frozen document snapshots stay auditable.
const DefaultsSource = "defaults-v1"

// Mentions are French legal strings embedded in issued documents / PDFs.
type Mentions struct {
	Franchise293B         string    `json:"franchise_293b"`
	LatePenalty           string    `json:"late_penalty"`
	FlatIndemnity40EUR    string    `json:"flat_indemnity_40e"`
	PaymentTermsDefault   string    `json:"payment_terms_default"`
	QuoteValidityNote     string    `json:"quote_validity_note"`
	Source                string    `json:"source"`
	FetchedAt             time.Time `json:"fetched_at"`
	LatePenaltyRateAnnual float64   `json:"late_penalty_rate_annual"`
	FlatIndemnityCents    int64     `json:"flat_indemnity_cents"`
}

// DefaultMentions returns the built-in V1 wording (not fetched from official APIs).
func DefaultMentions() Mentions {
	return Mentions{
		Franchise293B: "TVA non applicable, art. 293 B du CGI",
		LatePenalty: "En cas de retard de paiement, des pénalités calculées au taux " +
			"annuel de 10 % seront exigibles (taux minimal légal : 3 fois le taux d'intérêt légal).",
		FlatIndemnity40EUR: "Indemnité forfaitaire pour frais de recouvrement : 40 € " +
			"(art. L441-10 et D441-5 du code de commerce).",
		PaymentTermsDefault:   "Paiement à réception de facture.",
		QuoteValidityNote:     "Devis valable 30 jours sauf mention contraire.",
		Source:                DefaultsSource,
		FetchedAt:             time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		LatePenaltyRateAnnual: 10,
		FlatIndemnityCents:    4000,
	}
}

// DefaultProvider implements Provider with static defaults (no network).
type DefaultProvider struct{}

func (DefaultProvider) Name() string { return DefaultsSource }

func (DefaultProvider) Fetch(_ context.Context, key string) (Variable, error) {
	m := DefaultMentions()
	raw, err := json.Marshal(m)
	if err != nil {
		return Variable{}, err
	}
	if key != "" && key != "mentions.v1" {
		return Variable{}, ErrNotConfigured
	}
	return Variable{
		Key:       "mentions.v1",
		Value:     raw,
		Source:    DefaultsSource,
		FetchedAt: m.FetchedAt,
	}, nil
}
