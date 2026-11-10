package stripeconnect

import (
	"fmt"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/checkout/session"
)

// CreateDirectChargeCheckout creates a Checkout Session as a direct charge on the
// connected account, with application_fee_amount computed server-side.
func CreateDirectChargeCheckout(cfg Config, connectedAccountID string, amountCents, feeAmount int64, currency, successURL, cancelURL string) (*stripe.CheckoutSession, error) {
	Init(cfg)
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
	params.PaymentIntentData = &stripe.CheckoutSessionPaymentIntentDataParams{
		ApplicationFeeAmount: stripe.Int64(feeAmount),
	}
	// run in connected account context
	params.SetStripeAccount(connectedAccountID)
	s, err := session.New(params)
	if err != nil {
		return nil, fmt.Errorf("create checkout session: %w", err)
	}
	return s, nil
}