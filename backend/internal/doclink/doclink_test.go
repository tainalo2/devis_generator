package doclink

import (
	"testing"
	"time"
)

func TestQuoteExpiresAfterNinetyDays(t *testing.T) {
	now := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
	token, err := Generate([]byte("secret"), "quote-id", KindQuote, 1, now)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := Verify([]byte("secret"), token)
	if err != nil {
		t.Fatal(err)
	}
	if Expired(payload, now.Add(QuoteTTL)) {
		t.Fatal("quote must remain valid through the 90-day boundary")
	}
	if !Expired(payload, now.Add(QuoteTTL+time.Second)) {
		t.Fatal("quote must expire after 90 days")
	}
}

func TestInvoiceAndDepositLinksArePermanent(t *testing.T) {
	now := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
	for _, kind := range []string{KindInvoice, KindDeposit} {
		token, err := Generate([]byte("secret"), kind+"-id", kind, 1, now)
		if err != nil {
			t.Fatal(err)
		}
		payload, err := Verify([]byte("secret"), token)
		if err != nil {
			t.Fatal(err)
		}
		if Expired(payload, now.Add(20*365*24*time.Hour)) {
			t.Fatalf("%s link unexpectedly expired", kind)
		}
	}
}

func TestTamperedTokenIsRejected(t *testing.T) {
	token, err := Generate([]byte("secret"), "invoice-id", KindInvoice, 1, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Verify([]byte("secret"), token+"x"); err == nil {
		t.Fatal("tampered token must be rejected")
	}
}
