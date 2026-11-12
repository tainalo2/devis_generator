package pdfgen

import (
	"bytes"
	"strings"
	"testing"

	"github.com/tainalo2/devis_generator/backend/internal/biz"
	"github.com/tainalo2/devis_generator/backend/internal/legal"
)

// TestNoBankCoordsInPDF ensures IBAN/BIC/RIB never leak into generated PDF bytes.
func TestNoBankCoordsInPDF(t *testing.T) {
	num := "F-2026-9999"
	date := "2026-10-25"
	doc := biz.Document{
		Kind: "invoice", Status: "issued", Number: &num, IssueDate: &date,
		VATRegime: "franchise", Currency: "EUR",
		SubtotalCents: 10000, TaxCents: 0, TotalCents: 10000,
		Lines: []biz.DocLine{{Description: "Prestation", QuantityMilli: 1000, UnitPriceCents: 10000, TotalCents: 10000}},
	}
	iban := "FR7630006000011234567890189"
	bic := "BNPAFRPPXXX"
	pdf, err := Render(doc, biz.IssuerProfile{
		LegalName: "AE Exemple", SIRET: "12345678900012",
		IBAN: iban, // should not appear
	}, biz.Client{Name: "Client SAS"}, legal.DefaultMentions())
	if err != nil {
		t.Fatal(err)
	}
	plain := inflateStreams(pdf)
	for _, secret := range []string{iban, bic} {
		if bytes.Contains(plain, []byte(secret)) {
			t.Fatalf("bank coord leaked into PDF text: %s", secret)
		}
		if bytes.Contains(pdf, []byte(secret)) {
			t.Fatalf("bank coord leaked into PDF bytes: %s", secret)
		}
	}
	// also ensure no spaces variant
	if bytes.Contains(plain, []byte("FR76 3000")) {
		t.Fatal("iban spaced leaked")
	}
	_ = strings.Contains
}
