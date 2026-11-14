package stripeconnect

import "testing"

func TestValidateOnboardingRejectsUnsupportedCountry(t *testing.T) {
	c := Config{Country: "US", Currency: "EUR", FeeBPS: 100, ComplianceAck: true}
	if err := c.ValidateOnboarding(); err == nil {
		t.Fatal("expected country rejection")
	}
}
func TestValidateChargeRequiresCapabilitiesAndSafeFee(t *testing.T) {
	c := Config{Country: "FR", Currency: "EUR", FeeBPS: 100, ComplianceAck: true}
	acc := &ConnectedAccount{ChargesEnabled: true, PayoutsEnabled: true, Capabilities: map[string]string{"card_payments": "active", "transfers": "active"}}
	if _, err := c.ValidateCharge(acc, 1000, "EUR"); err != nil {
		t.Fatal(err)
	}
	acc.Capabilities["transfers"] = "inactive"
	if _, err := c.ValidateCharge(acc, 1000, "EUR"); err == nil {
		t.Fatal("expected capability rejection")
	}
}
