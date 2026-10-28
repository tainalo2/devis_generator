package facturx

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/tainalo2/devis_generator/backend/internal/billing"
	"github.com/tainalo2/devis_generator/backend/internal/biz"
	"github.com/tainalo2/devis_generator/backend/internal/legal"
)

func TestBuildInvoiceMultiRate(t *testing.T) {
	doc, issuer, client := sample(t, "invoice", billing.RegimeStandard, []biz.DocLine{
		{Description: "Conseil", QuantityMilli: 1000, UnitPriceCents: 10_000, TaxRateBps: 2000},
		{Description: "Hébergement", QuantityMilli: 2000, UnitPriceCents: 5_000, DiscountBps: 1000, TaxRateBps: 550},
	})
	client.Address.Country = ""
	raw := mustBuild(t, doc, issuer, client)
	mustWellFormed(t, raw)

	// 2 × 50 € − 10 % = 90 HT à 5,5 % ; 100 HT à 20 %.
	requireOrder(t, raw,
		GuidelineEN16931,
		"<ram:ID>F-2026-0001</ram:ID>",
		"<ram:TypeCode>380</ram:TypeCode>",
		`format="102">20260315<`,
		"Objet : Prestation mars",
		"Merci.",
		"<ram:Name>Conseil</ram:Name>",
		`<ram:BilledQuantity unitCode="C62">1</ram:BilledQuantity>`,
		"<ram:CategoryCode>S</ram:CategoryCode>",
		"<ram:RateApplicablePercent>20.00</ram:RateApplicablePercent>",
		"<ram:LineTotalAmount>100.00</ram:LineTotalAmount>",
		"<ram:Name>Hébergement</ram:Name>",
		"<ram:ChargeAmount>50.00</ram:ChargeAmount>",
		"<udt:Indicator>false</udt:Indicator>",
		"<ram:ActualAmount>5.00</ram:ActualAmount>",
		"<ram:ChargeAmount>45.00</ram:ChargeAmount>",
		`<ram:BilledQuantity unitCode="C62">2</ram:BilledQuantity>`,
		"<ram:RateApplicablePercent>5.50</ram:RateApplicablePercent>",
		"<ram:LineTotalAmount>90.00</ram:LineTotalAmount>",
		`<ram:GlobalID schemeID="0009">12345678900012</ram:GlobalID>`,
		`<ram:ID schemeID="0002">123456789</ram:ID>`,
		"<ram:TradingBusinessName>Martin Studio</ram:TradingBusinessName>",
		"<ram:CountryID>FR</ram:CountryID>",
		`<ram:ID schemeID="VA">FR12123456789</ram:ID>`,
		"<ram:Name>Client SAS</ram:Name>",
		"<ram:CountryID>FR</ram:CountryID>",
		"<ram:InvoiceCurrencyCode>EUR</ram:InvoiceCurrencyCode>",
		"<ram:TypeCode>30</ram:TypeCode>",
		"<ram:IBANID>FR7630006000011234567890189</ram:IBANID>",
		"<ram:CalculatedAmount>4.95</ram:CalculatedAmount>",
		"<ram:BasisAmount>90.00</ram:BasisAmount>",
		"<ram:CategoryCode>S</ram:CategoryCode>",
		"<ram:RateApplicablePercent>5.50</ram:RateApplicablePercent>",
		"<ram:CalculatedAmount>20.00</ram:CalculatedAmount>",
		"<ram:BasisAmount>100.00</ram:BasisAmount>",
		"<ram:RateApplicablePercent>20.00</ram:RateApplicablePercent>",
		"<ram:Description>Paiement à 30 jours.</ram:Description>",
		`format="102">20260415<`,
		"<ram:LineTotalAmount>190.00</ram:LineTotalAmount>",
		"<ram:TaxBasisTotalAmount>190.00</ram:TaxBasisTotalAmount>",
		`<ram:TaxTotalAmount currencyID="EUR">24.95</ram:TaxTotalAmount>`,
		"<ram:GrandTotalAmount>214.95</ram:GrandTotalAmount>",
		"<ram:DuePayableAmount>214.95</ram:DuePayableAmount>",
	)
}

func TestBuildFranchise293B(t *testing.T) {
	doc, issuer, client := sample(t, "invoice", billing.RegimeFranchise293B, []biz.DocLine{
		{Description: "Atelier", QuantityMilli: 1500, UnitPriceCents: 8_000, TaxRateBps: 2000},
	})
	issuer.VATNumber = ""
	doc.LegalMentions = "Mention propre au document."
	raw := mustBuild(t, doc, issuer, client)

	requireOrder(t, raw,
		"<ram:TypeCode>380</ram:TypeCode>",
		"Mention propre au document.",
		"<ram:CategoryCode>E</ram:CategoryCode>",
		"<ram:RateApplicablePercent>0.00</ram:RateApplicablePercent>",
		"<ram:LineTotalAmount>120.00</ram:LineTotalAmount>",
		"<ram:CalculatedAmount>0.00</ram:CalculatedAmount>",
		"<ram:ExemptionReason>"+legal.DefaultMentions().Franchise293B+"</ram:ExemptionReason>",
		"<ram:BasisAmount>120.00</ram:BasisAmount>",
		"<ram:CategoryCode>E</ram:CategoryCode>",
		"<ram:GrandTotalAmount>120.00</ram:GrandTotalAmount>",
	)
	if strings.Contains(raw, `schemeID="VA"`) {
		t.Fatal("la franchise sans n° de TVA ne doit pas émettre d'identifiant TVA")
	}
}

func TestBuildCreditNoteAndDeposit(t *testing.T) {
	lines := []biz.DocLine{
		{Description: "Acompte chantier", QuantityMilli: 1000, UnitPriceCents: 20_000, TaxRateBps: 2000},
	}
	credit, issuer, client := sample(t, "credit_note", billing.RegimeStandard, lines)
	credit.Number = strPtr("A-2026-0003")
	raw := mustBuild(t, credit, issuer, client)
	requireOrder(t, raw,
		"<ram:ID>A-2026-0003</ram:ID>",
		"<ram:TypeCode>381</ram:TypeCode>",
		"<ram:GrandTotalAmount>240.00</ram:GrandTotalAmount>",
	)

	deposit, _, _ := sample(t, "deposit_invoice", billing.RegimeStandard, lines)
	deposit.Number = strPtr("FA-2026-0002")
	raw = mustBuild(t, deposit, issuer, client)
	requireOrder(t, raw,
		"<ram:ID>FA-2026-0002</ram:ID>",
		"<ram:TypeCode>386</ram:TypeCode>",
	)
}

func TestBuildZeroRateIsCategoryZ(t *testing.T) {
	doc, issuer, client := sample(t, "invoice", billing.RegimeStandard, []biz.DocLine{
		{Description: "Exonéré", QuantityMilli: 1000, UnitPriceCents: 5_000, TaxRateBps: 0},
	})
	issuer.VATNumber = ""
	raw := mustBuild(t, doc, issuer, client)
	if strings.Contains(raw, "<ram:CategoryCode>S</ram:CategoryCode>") || strings.Contains(raw, "<ram:ExemptionReason>") {
		t.Fatalf("un taux 0 hors franchise est la catégorie Z, sans motif d'exonération\n%s", raw)
	}
	requireOrder(t, raw, "<ram:CategoryCode>Z</ram:CategoryCode>", "<ram:RateApplicablePercent>0.00</ram:RateApplicablePercent>")
}

func TestBuildSIRETDerivesSIREN(t *testing.T) {
	doc, issuer, client := sample(t, "invoice", billing.RegimeStandard, []biz.DocLine{
		{Description: "Ligne", QuantityMilli: 1000, UnitPriceCents: 1_000, TaxRateBps: 2000},
	})
	issuer.SIREN = ""
	issuer.SIRET = "732 829 320 00074"
	client.Address.Country = "Belgique"
	raw := mustBuild(t, doc, issuer, client)
	requireOrder(t, raw,
		`<ram:GlobalID schemeID="0009">73282932000074</ram:GlobalID>`,
		`<ram:ID schemeID="0002">732829320</ram:ID>`,
		"<ram:CountryID>BE</ram:CountryID>",
	)
}

func TestBuildRejectsIncomplete(t *testing.T) {
	base, issuer, client := sample(t, "invoice", billing.RegimeStandard, []biz.DocLine{
		{Description: "Ligne", QuantityMilli: 1000, UnitPriceCents: 1_000, TaxRateBps: 2000},
	})

	clone := func(doc biz.Document) biz.Document {
		doc.Lines = append([]biz.DocLine(nil), doc.Lines...)
		return doc
	}
	quote := clone(base)
	quote.Kind = "quote"
	blank := " "
	noDate := clone(base)
	noDate.IssueDate = nil
	badDate := clone(base)
	badDate.IssueDate = strPtr("15/03/2026")
	noNumber := clone(base)
	noNumber.Number = nil
	emptyNumber := clone(base)
	emptyNumber.Number = &blank
	noSeller := issuer
	noSeller.LegalName = " "
	noBuyer := client
	noBuyer.Name = ""
	noLines := clone(base)
	noLines.Lines = nil
	noDesc := clone(base)
	noDesc.Lines = []biz.DocLine{{Description: " ", QuantityMilli: 1000, UnitPriceCents: 1000, TaxRateBps: 2000, TotalCents: base.Lines[0].TotalCents}}
	zeroQty := clone(base)
	zeroQty.Lines = []biz.DocLine{{Description: "Ligne", QuantityMilli: 0, UnitPriceCents: 1000, TaxRateBps: 2000}}
	badTotals := clone(base)
	badTotals.TotalCents++
	badLine := clone(base)
	badLine.Lines[0].TotalCents++
	noVAT := issuer
	noVAT.VATNumber = ""
	badCountry := client
	badCountry.Address.Country = "Narnia"
	badKind := base
	badKind.Kind = "draft"

	cases := []struct {
		name   string
		doc    biz.Document
		issuer biz.IssuerProfile
		client biz.Client
		want   error
		sub    string
	}{
		{name: "devis", doc: quote, issuer: issuer, client: client, want: errQuote},
		{name: "type inconnu", doc: badKind, issuer: issuer, client: client, want: errKind},
		{name: "numéro absent", doc: noNumber, issuer: issuer, client: client, want: errNumber},
		{name: "numéro vide", doc: emptyNumber, issuer: issuer, client: client, want: errNumber},
		{name: "date absente", doc: noDate, issuer: issuer, client: client, want: errDate},
		{name: "date invalide", doc: badDate, issuer: issuer, client: client, sub: "date d'émission invalide"},
		{name: "vendeur", doc: base, issuer: noSeller, client: client, want: errSeller},
		{name: "acheteur", doc: base, issuer: issuer, client: noBuyer, want: errBuyer},
		{name: "sans ligne", doc: noLines, issuer: issuer, client: client, want: errNoLines},
		{name: "désignation", doc: noDesc, issuer: issuer, client: client, sub: "ligne 1 : désignation manquante"},
		{name: "quantité", doc: zeroQty, issuer: issuer, client: client, sub: "ligne 1 : quantité nulle"},
		{name: "totaux", doc: badTotals, issuer: issuer, client: client, want: errTotals},
		{name: "total de ligne", doc: badLine, issuer: issuer, client: client, sub: "ligne 1 : total HT incohérent"},
		{name: "tva vendeur", doc: base, issuer: noVAT, client: client, want: errSellerVAT},
		{name: "pays", doc: base, issuer: issuer, client: badCountry, sub: "pays inconnu"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Build(tc.doc, tc.issuer, tc.client)
			if err == nil {
				t.Fatal("erreur attendue")
			}
			if tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, attendu %v", err, tc.want)
			}
			if tc.sub != "" && !strings.Contains(err.Error(), tc.sub) {
				t.Fatalf("err = %q, doit contenir %q", err, tc.sub)
			}
		})
	}
}

func sample(t *testing.T, kind, regime string, lines []biz.DocLine) (biz.Document, biz.IssuerProfile, biz.Client) {
	t.Helper()
	res, err := billing.Compute(regime, lineInputs(lines))
	if err != nil {
		t.Fatal(err)
	}
	for i := range lines {
		lines[i].TotalCents = res.Lines[i].TotalHTCents
	}
	issuer := biz.IssuerProfile{
		LegalName: "Atelier Martin",
		TradeName: "Martin Studio",
		Email:     "factures@martin.fr",
		Phone:     "01 23 45 67 89",
		SIRET:     "12345678900012",
		VATNumber: "FR12 123456789",
		Address:   biz.Address{Line1: "1 rue de la Paix", PostalCode: "75002", City: "Paris", Country: "France"},
		IBAN:      "FR76 3000 6000 0112 3456 7890 189",
	}
	client := biz.Client{
		Name:    "Client SAS",
		Email:   "achat@client.fr",
		SIREN:   "552100554",
		Address: biz.Address{Line1: "10 avenue", PostalCode: "69001", City: "Lyon", Country: "France"},
	}
	return biz.Document{
		Kind:          kind,
		Number:        strPtr("F-2026-0001"),
		Title:         "Prestation mars",
		IssueDate:     strPtr("2026-03-15"),
		DueDate:       strPtr("2026-04-15"),
		Currency:      "eur",
		VATRegime:     regime,
		Notes:         "Merci.",
		PaymentTerms:  "Paiement à 30 jours.",
		Lines:         lines,
		SubtotalCents: res.SubtotalCents,
		TaxCents:      res.TaxCents,
		TotalCents:    res.TotalCents,
	}, issuer, client
}

func mustBuild(t *testing.T, doc biz.Document, issuer biz.IssuerProfile, client biz.Client) string {
	t.Helper()
	raw, err := Build(doc, issuer, client)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(raw, []byte(xml.Header)) {
		t.Fatal("en-tête XML UTF-8 manquant")
	}
	return string(raw)
}

func mustWellFormed(t *testing.T, raw string) {
	t.Helper()
	dec := xml.NewDecoder(strings.NewReader(raw))
	for {
		_, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return
		}
		if err != nil {
			t.Fatalf("XML mal formé : %v\n%s", err, raw)
		}
	}
}

func requireOrder(t *testing.T, raw string, parts ...string) {
	t.Helper()
	pos := 0
	for _, p := range parts {
		i := strings.Index(raw[pos:], p)
		if i < 0 {
			t.Fatalf("fragment absent (%q) après l'offset %d\n%s", p, pos, raw)
		}
		pos += i + len(p)
	}
}

func strPtr(s string) *string { return &s }
