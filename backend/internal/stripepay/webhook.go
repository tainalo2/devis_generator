package stripepay

import (
	"fmt"
	"io"
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/webhook"
)

// VerifyWebhook reads the request body and verifies the Stripe signature using
// the configured webhook secret. It returns the parsed event on success.
func VerifyWebhook(r *http.Request, secret string) (stripe.Event, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return stripe.Event{}, fmt.Errorf("read body: %w", err)
	}
	sig := r.Header.Get("Stripe-Signature")
	evt, err := webhook.ConstructEvent(body, sig, secret)
	if err != nil {
		return stripe.Event{}, fmt.Errorf("verify signature: %w", err)
	}
	return evt, nil
}
