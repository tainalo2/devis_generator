package payments

import (
	"encoding/json"
	"github.com/tainalo2/devis_generator/backend/internal/paymentbyo"
	"strings"
	"testing"
)

func TestConfigViewNeverSerializesPlaintextFields(t *testing.T) {
	b, err := json.Marshal(ConfigView{Provider: paymentbyo.ProviderStripe, Env: paymentbyo.EnvLive, Masked: map[string]string{"stripe_api_key": "•••• 1234"}})
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, name := range []string{"stripe_secret_key", "stripe_webhook_secret", "client_secret"} {
		if strings.Contains(s, name) {
			t.Fatalf("response leaked secret field %q", name)
		}
	}
}
