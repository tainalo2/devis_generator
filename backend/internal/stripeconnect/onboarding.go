package stripeconnect

import (
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
}

func CreateExpressAccount(cfg Config, email string) (*ConnectedAccount, error) {
	client := stripeclient.New(cfg.PlatformSecretKey, nil)
	params := &stripe.AccountParams{
		Type:    stripe.String("express"),
		Country: stripe.String("FR"),
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
	}
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
