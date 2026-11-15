package pdfgen

import (
	"bytes"
	"testing"

	"github.com/tainalo2/devis_generator/backend/internal/biz"
	"github.com/tainalo2/devis_generator/backend/internal/legal"
)

func ribTestDoc() biz.Document {
	num := "F-2026-9999"
	date := "2026-10-25"
	return biz.Document{
		Kind: "invoice", Status: "issued", Number: &num, IssueDate: &date,
		VATRegime: "franchise", Currency: "EUR",
		SubtotalCents: 10000, TotalCents: 10000,
		Lines: []biz.DocLine{{Description: "Prestation", QuantityMilli: 1000, UnitPriceCents: 10000, TotalCents: 10000}},
	}
}

// The renderer uses an Identity-H Unicode font, so text is stored as UTF-16BE
// glyph strings: searching ASCII bytes alone cannot detect a leak. This test
// proves the helper is discriminating (the IBAN of a non-redacted issuer IS
// found) and that an issuer without IBAN produces a PDF free of it.
func TestRibLeakDetectionIsDiscriminating(t *testing.T) {
	iban := "FR7630006000011234567890189"
	render := func(i string) []byte {
		pdf, err := Render(ribTestDoc(), biz.IssuerProfile{LegalName: "AE Exemple", SIRET: "12345678900012", IBAN: i},
			biz.Client{Name: "Client SAS"}, legal.DefaultMentions())
		if err != nil {
			t.Fatal(err)
		}
		return inflateStreams(pdf)
	}
	if !bytes.Contains(render(iban), utf16BE(iban)) {
		t.Fatal("control failed: IBAN given to Render should be detectable in UTF-16BE; detector is blind")
	}
	plain := render("")
	for _, secret := range []string{iban, "BNPAFRPPXXX", "FR76 3000"} {
		if bytes.Contains(plain, []byte(secret)) || bytes.Contains(plain, utf16BE(secret)) {
			t.Fatalf("bank coord leaked into PDF: %s", secret)
		}
	}
}
