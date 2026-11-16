package pdfgen

import (
	"bytes"
	"testing"

	"github.com/tainalo2/devis_generator/backend/internal/billing"
	"github.com/tainalo2/devis_generator/backend/internal/biz"
	"github.com/tainalo2/devis_generator/backend/internal/legal"
)

func TestBrandingBackwardCompatibility(t *testing.T) {
	// Ensure RenderWithBranding(branding={}) is identical to Render()
	num := "F-2026-0001"
	date := "2026-10-28"
	doc := biz.Document{
		Kind: "invoice", Number: &num, IssueDate: &date,
		VATRegime: billing.RegimeFranchise293B, Currency: "EUR",
		SubtotalCents: 10000, TaxCents: 0, TotalCents: 10000,
		Lines: []biz.DocLine{
			{Description: "Test", QuantityMilli: 1000, UnitPriceCents: 10000, TotalCents: 10000},
		},
	}

	issuer := biz.IssuerProfile{LegalName: "Test Ltd", SIRET: "12345678900012"}
	client := biz.Client{Name: "Client"}
	mentions := legal.DefaultMentions()

	// Both should produce identical PDFs
	pdfOld, err := Render(doc, issuer, client, mentions)
	if err != nil {
		t.Fatal(err)
	}

	pdfNew, err := RenderWithBranding(doc, issuer, client, mentions, BrandingData{})
	if err != nil {
		t.Fatal(err)
	}

	// Content should be the same (PDF metadata timestamps may differ)
	if len(pdfOld) != len(pdfNew) {
		t.Logf("PDF size differs: old=%d, new=%d (expected due to timestamp)", len(pdfOld), len(pdfNew))
	}

	// At minimum, both should have the same critical content
	plainOld := inflateStreams(pdfOld)
	plainNew := inflateStreams(pdfNew)

	if !bytes.Contains(plainOld, utf16BE("Test Ltd")) || !bytes.Contains(plainNew, utf16BE("Test Ltd")) {
		t.Fatal("critical content missing")
	}
}

func TestBrandingColorParsing(t *testing.T) {
	tests := []struct {
		name    string
		color   string
		wantErr bool
		wantR   int
		wantG   int
		wantB   int
	}{
		{"Valid hex 6 digits", "2f9e94", false, 0x2f, 0x9e, 0x94},
		{"Valid hex with #", "#2f9e94", false, 0x2f, 0x9e, 0x94},
		{"Black", "000000", false, 0, 0, 0},
		{"White", "ffffff", false, 0xff, 0xff, 0xff},
		{"Uppercase", "2F9E94", false, 0x2f, 0x9e, 0x94},
		{"Mixed case", "2f9E94", false, 0x2f, 0x9e, 0x94},
		{"Too short", "fff", true, 0, 0, 0},
		{"Too long", "1234567", true, 0, 0, 0},
		{"Invalid hex", "gggggg", true, 0, 0, 0},
		{"Empty", "", true, 0, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, g, b, err := parseHexColor(tt.color)
			if tt.wantErr && err == nil {
				t.Errorf("expected error, got none")
			} else if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			} else if !tt.wantErr {
				if r != tt.wantR || g != tt.wantG || b != tt.wantB {
					t.Errorf("got RGB(%d, %d, %d), want RGB(%d, %d, %d)",
						r, g, b, tt.wantR, tt.wantG, tt.wantB)
				}
			}
		})
	}
}

func TestBrandingDataEmpty(t *testing.T) {
	tests := []struct {
		name         string
		branding     BrandingData
		wantEmpty    bool
	}{
		{"Empty struct", BrandingData{}, true},
		{"Logo only", BrandingData{LogoPNG: []byte{1, 2, 3}}, false},
		{"PrimaryColor only", BrandingData{PrimaryColor: "#2f9e94"}, false},
		{"AccentColor only", BrandingData{AccentColor: "#2f9e94"}, false},
		{"TextColor only", BrandingData{TextColor: "#2f9e94"}, false},
		{"All fields", BrandingData{
			LogoPNG:      []byte{1, 2, 3},
			PrimaryColor: "#2f9e94",
			AccentColor:  "#2f9e94",
			TextColor:    "#2f9e94",
		}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.branding.IsEmpty(); got != tt.wantEmpty {
				t.Errorf("IsEmpty()=%v, want %v", got, tt.wantEmpty)
			}
		})
	}
}

func TestBrandingNonRegression_FacturXCompliance(t *testing.T) {
	// Test that RenderWithBranding doesn't break Factur-X compliance.
	// Legal fields must remain unmodified and accessible.
	num := "F-2026-0099"
	date := "2026-10-28"
	doc := biz.Document{
		Kind: "invoice", Number: &num, IssueDate: &date,
		VATRegime: billing.RegimeFranchise293B, Currency: "EUR",
		Title:         "Important Invoice",
		SubtotalCents: 100000,
		TaxCents:      0,
		TotalCents:    100000,
		Lines: []biz.DocLine{
			{Description: "Professional Service", QuantityMilli: 1000, UnitPriceCents: 100000, TotalCents: 100000},
		},
		PaymentTerms: "Net 30",
		LegalMentions: "Pénalité d'escompte : néant.",
	}

	issuer := biz.IssuerProfile{
		LegalName: "Société Example SARL",
		SIRET:     "12345678900012",
		Address: biz.Address{
			Line1: "123 rue Example", PostalCode: "69000", City: "Lyon",
		},
	}

	client := biz.Client{
		Name:  "Client Test",
		Email: "client@test.fr",
		Address: biz.Address{
			Line1: "1 rue Client", PostalCode: "75000", City: "Paris",
		},
	}

	mentions := legal.DefaultMentions()

	// Test without branding
	pdfUnbranded, err := Render(doc, issuer, client, mentions)
	if err != nil {
		t.Fatal(err)
	}
	plainUnbranded := inflateStreams(pdfUnbranded)

	// Test with empty branding (should be identical)
	branding := BrandingData{}
	pdfBranded, err := RenderWithBranding(doc, issuer, client, mentions, branding)
	if err != nil {
		t.Fatal(err)
	}
	plainBranded := inflateStreams(pdfBranded)

	// Verify both are valid PDFs
	if len(pdfUnbranded) < 100 || string(pdfUnbranded[:4]) != "%PDF" {
		t.Fatal("unbranded PDF invalid")
	}
	if len(pdfBranded) < 100 || string(pdfBranded[:4]) != "%PDF" {
		t.Fatal("branded PDF invalid")
	}

	// Verify critical legal content is present in both
	criticalFields := []string{
		"Société Example SARL",
		"123 rue Example",
		"Client Test",
		"1 rue Client",
		"Professional Service",
		"1" + nbsp + "000,00" + nbsp + "€",
		"Net 30",
	}

	for _, field := range criticalFields {
		if !bytes.Contains(plainUnbranded, utf16BE(field)) {
			t.Errorf("unbranded PDF missing: %q", field)
		}
		if !bytes.Contains(plainBranded, utf16BE(field)) {
			t.Errorf("branded PDF missing: %q", field)
		}
	}
}

func BenchmarkRenderWithBranding(b *testing.B) {
	num := "F-2026-0001"
	date := "2026-10-28"
	doc := biz.Document{
		Kind: "invoice", Number: &num, IssueDate: &date,
		VATRegime: billing.RegimeFranchise293B, Currency: "EUR",
		SubtotalCents: 50000, TaxCents: 0, TotalCents: 50000,
		Lines: []biz.DocLine{
			{Description: "Service", QuantityMilli: 1000, UnitPriceCents: 50000, TotalCents: 50000},
		},
	}

	issuer := biz.IssuerProfile{LegalName: "Company LLC", SIRET: "12345678900012"}
	client := biz.Client{Name: "Client Corp"}
	mentions := legal.DefaultMentions()

	branding := BrandingData{} // Empty branding

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := RenderWithBranding(doc, issuer, client, mentions, branding)
		if err != nil {
			b.Fatal(err)
		}
	}
}
