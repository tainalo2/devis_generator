package stripepay

import (
	"fmt"

	stripe "github.com/stripe/stripe-go/v81"
	stripeclient "github.com/stripe/stripe-go/v81/client"
)

// CreateCheckoutSession creates a Stripe Checkout Session for a one-off payment.
// In test mode, use card 4242 4242 4242 4242 (success) or 4000 0000 0000 9995 (decline).
func CreateCheckoutSession(cfg Config, amountCents int64, currency, successURL, cancelURL string) (*stripe.CheckoutSession, error) {
	client := stripeclient.New(cfg.SecretKey, nil)
	params := &stripe.CheckoutSessionParams{
		Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL: stripe.String(successURL),
		CancelURL:  stripe.String(cancelURL),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String(currency),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String("Devis payment"),
					},
					UnitAmount: stripe.Int64(amountCents),
				},
				Quantity: stripe.Int64(1),
			},
		},
	}
	if cfg.ConnectAccountID != "" {
		params.PaymentIntentData = &stripe.CheckoutSessionPaymentIntentDataParams{
			OnBehalfOf:   stripe.String(cfg.ConnectAccountID),
			TransferData: &stripe.CheckoutSessionPaymentIntentDataTransferDataParams{Destination: stripe.String(cfg.ConnectAccountID)},
		}
	}
	s, err := client.CheckoutSessions.New(params)
	if err != nil {
		return nil, fmt.Errorf("create checkout session: %w", err)
	}
	return s, nil
}
