package stripeconnect

import (
	"os"
	"testing"
)

func TestEnabledDisabledByDefault(t *testing.T) {
	os.Unsetenv("PAYMENT_STRIPE_CONNECT_ENABLED")
	cfg := Config{PlatformSecretKey: "sk_test_x", WebhookSecret: "whsec_x", PublicURL: "https://example.com", ComplianceAck: true, Currency: "eur"}
	if ok, _ := cfg.Enabled(); ok {
		t.Fatal("expected disabled by default")
	}
}

func TestEnabledRequiresAll(t *testing.T) {
	os.Setenv("PAYMENT_STRIPE_CONNECT_ENABLED", "true")
	defer os.Unsetenv("PAYMENT_STRIPE_CONNECT_ENABLED")
	cfg := Config{PlatformSecretKey: "sk_test_x", WebhookSecret: "whsec_x", PublicURL: "https://example.com", ComplianceAck: true, Currency: "eur"}
	ok, reason := cfg.Enabled()
	if !ok {
		t.Fatalf("expected enabled, got %s", reason)
	}
	// missing compliance
	cfg2 := cfg
	cfg2.ComplianceAck = false
	if ok, _ := cfg2.Enabled(); ok {
		t.Fatal("expected disabled without compliance ack")
	}
}

func TestComputeFee(t *testing.T) {
	cfg := Config{FeeBPS: 250, FeeFixedCents: 30, FeeCapCents: 500, Currency: "eur"}
	fee, err := cfg.ComputeFee(10000, "eur")
	if err != nil {
		t.Fatal(err)
	}
	// 2.5% of 10000 = 250 + 30 = 280
	if fee != 280 {
		t.Fatalf("got %d want 280", fee)
	}
	// cap
	fee2, _ := cfg.ComputeFee(100000, "eur")
	if fee2 != 500 {
		t.Fatalf("cap: got %d want 500", fee2)
	}
	// low amount
	fee3, _ := cfg.ComputeFee(100, "eur")
	if fee3 < 0 {
		t.Fatal("negative")
	}
	// currency mismatch
	if _, err := cfg.ComputeFee(1000, "usd"); err == nil {
		t.Fatal("expected currency mismatch error")
	}
}