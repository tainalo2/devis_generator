package pdfgen

import (
	"testing"

	"github.com/tainalo2/devis_generator/backend/internal/billing"
	"github.com/tainalo2/devis_generator/backend/internal/biz"
	"github.com/tainalo2/devis_generator/backend/internal/legal"
)

func TestRenderPDFMinimal(t *testing.T) {
	num := "F-2026-0001"
	date := "2026-10-25"
	doc := biz.Document{
		Kind: "invoice", Status: "issued", Number: &num, IssueDate: &date,
		VATRegime: billing.RegimeFranchise293B, Currency: "EUR",
		SubtotalCents: 10000, TaxCents: 0, TotalCents: 10000,
		Lines: []biz.DocLine{{
			Description: "Prestation", QuantityMilli: 1000, UnitPriceCents: 10000, TotalCents: 10000,
		}},
		PaymentTerms: "À réception",
	}
	pdf, err := Render(doc, biz.IssuerProfile{LegalName: "AE Exemple", SIRET: "12345678900012"},
		biz.Client{Name: "Client SAS"}, legal.DefaultMentions())
	if err != nil {
		t.Fatal(err)
	}
	if len(pdf) < 100 || string(pdf[:4]) != "%PDF" {
		t.Fatalf("not a pdf: len=%d", len(pdf))
	}
}
