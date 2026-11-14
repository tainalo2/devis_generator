package stripeconnect

import (
	"context"
	"fmt"

	stripe "github.com/stripe/stripe-go/v81"
	stripeclient "github.com/stripe/stripe-go/v81/client"
)

type ConnectedAccount struct {
	ID               string
	Type             string
	ChargesEnabled   bool
	PayoutsEnabled   bool
	DetailsSubmitted bool
	Capabilities     map[string]string
	Country          string
	DefaultCurrency  string
}

func CreateExpressAccount(cfg Config, email string) (*ConnectedAccount, error) {
	if err := cfg.ValidateOnboarding(); err != nil {
		return nil, err
	}
	client := stripeclient.New(cfg.PlatformSecretKey, nil)
	params := &stripe.AccountParams{
		Type:    stripe.String("express"),
		Country: stripe.String(cfg.Country),
		Email:   stripe.String(email),
		Capabilities: &stripe.AccountCapabilitiesParams{
			CardPayments: &stripe.AccountCapabilitiesCardPaymentsParams{Requested: stripe.Bool(true)},
			Transfers:    &stripe.AccountCapabilitiesTransfersParams{Requested: stripe.Bool(true)},
		},
	}
	a, err := client.Accounts.New(params)
	if err != nil {
		return nil, fmt.Errorf("create account: %w", err)
	}
	return toConnected(a), nil
}

func CreateAccountLink(cfg Config, accountID, refreshURL, returnURL string) (string, error) {
	client := stripeclient.New(cfg.PlatformSecretKey, nil)
	params := &stripe.AccountLinkParams{
		Account:    stripe.String(accountID),
		RefreshURL: stripe.String(refreshURL),
		ReturnURL:  stripe.String(returnURL),
		Type:       stripe.String("account_onboarding"),
	}
	l, err := client.AccountLinks.New(params)
	if err != nil {
		return "", fmt.Errorf("create account link: %w", err)
	}
	return l.URL, nil
}

func RetrieveAccount(cfg Config, accountID string) (*ConnectedAccount, error) {
	client := stripeclient.New(cfg.PlatformSecretKey, nil)
	a, err := client.Accounts.GetByID(accountID, nil)
	if err != nil {
		return nil, fmt.Errorf("retrieve account: %w", err)
	}
	return toConnected(a), nil
}

func toConnected(a *stripe.Account) *ConnectedAccount {
	caps := map[string]string{}
	if a.Capabilities != nil {
		if a.Capabilities.CardPayments != "" {
			caps["card_payments"] = string(a.Capabilities.CardPayments)
		}
		if a.Capabilities.Transfers != "" {
			caps["transfers"] = string(a.Capabilities.Transfers)
		}
	}
	return &ConnectedAccount{
		ID:               a.ID,
		Type:             string(a.Type),
		ChargesEnabled:   a.ChargesEnabled,
		PayoutsEnabled:   a.PayoutsEnabled,
		DetailsSubmitted: a.DetailsSubmitted,
		Capabilities:     caps,
		Country:          a.Country,
		DefaultCurrency:  string(a.DefaultCurrency),
	}
}

// StripeAccountClient binds the operator-owned configuration once; tenant
// input can never replace its platform credentials.
type StripeAccountClient struct{ Config Config }

func (c StripeAccountClient) CreateAccount(_ context.Context, email, country string) (*ConnectedAccount, error) {
	cfg := c.Config
	cfg.Country = country
	return CreateExpressAccount(cfg, email)
}
func (c StripeAccountClient) CreateAccountLink(_ context.Context, accountID, refreshURL, returnURL string) (string, error) {
	return CreateAccountLink(c.Config, accountID, refreshURL, returnURL)
}
func (c StripeAccountClient) RetrieveAccount(_ context.Context, accountID string) (*ConnectedAccount, error) {
	return RetrieveAccount(c.Config, accountID)
}

func ReadyForPayments(acc *ConnectedAccount) bool {
	if acc == nil {
		return false
	}
	if !acc.ChargesEnabled || !acc.PayoutsEnabled {
		return false
	}
	// required capabilities
	for _, req := range []string{"card_payments", "transfers"} {
		if acc.Capabilities[req] != "active" {
			return false
		}
	}
	return true
}
