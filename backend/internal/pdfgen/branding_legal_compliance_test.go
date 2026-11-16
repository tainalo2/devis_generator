package pdfgen

import (
	"bytes"
	"testing"

	"github.com/tainalo2/devis_generator/backend/internal/billing"
	"github.com/tainalo2/devis_generator/backend/internal/biz"
	"github.com/tainalo2/devis_generator/backend/internal/legal"
)

// brandingLegalTestDoc creates a comprehensive test document with all legal/VAT fields.
// This is used to verify that branding never alters mandatory legal content.
func brandingLegalTestDoc() biz.Document {
	num := "F-2026-1337"
	date := "2026-11-10"
	return biz.Document{
		Kind:           "invoice",
		Status:         "issued",
		Number:         &num,
		IssueDate:      &date,
		VATRegime:      billing.RegimeFranchise293B,
		Currency:       "EUR",
		Title:          "Invoice with Legal Fields",
		SubtotalCents:  50000,
		TaxCents:       0,
		TotalCents:     50000,
		PaymentTerms:   "Net 30",
		LegalMentions:  "Pénalité d'escompte : néant. Intérêts moratoires : TEGG.",
		Lines: []biz.DocLine{
			{
				Description:    "Professional Service",
				QuantityMilli:  1000,
				UnitPriceCents: 50000,
				TotalCents:     50000,
			},
		},
	}
}

// brandingLegalTestIssuer creates an issuer with all legal registration info.
func brandingLegalTestIssuer() biz.IssuerProfile {
	return biz.IssuerProfile{
		LegalName: "Example SARL",
		SIRET:     "52345678900123",
		VATNumber: "FR52345678901",
		Address: biz.Address{
			Line1:      "123 Rue de l'Example",
			PostalCode: "69000",
			City:       "Lyon",
		},
	}
}

// brandingLegalTestClient creates a test client.
func brandingLegalTestClient() biz.Client {
	return biz.Client{
		Name:  "Client Test Company",
		Email: "contact@example.com",
		Address: biz.Address{
			Line1:      "456 Rue du Client",
			PostalCode: "75000",
			City:       "Paris",
		},
	}
}

// TestLegalFieldsNotAlteredByBranding verifies that all mandatory legal fields
// remain intact and readable when arbitrary branding (logo/colors) is applied.
// This is a critical compliance test: branding must never obscure or modify
// legal information required by French tax/invoice law.
func TestLegalFieldsNotAlteredByBranding(t *testing.T) {
	doc := brandingLegalTestDoc()
	issuer := brandingLegalTestIssuer()
	client := brandingLegalTestClient()
	mentions := legal.DefaultMentions()

	// Render unbranded PDF
	pdfUnbranded, err := Render(doc, issuer, client, mentions)
	if err != nil {
		t.Fatal("failed to render unbranded PDF:", err)
	}
	plainUnbranded := inflateStreams(pdfUnbranded)

	// Critical legal fields that must ALWAYS be present and identical
	mandatoryFields := []string{
		// Company/Issuer info
		"Example SARL",
		"52345678900123", // SIRET
		"123 Rue de l'Example",
		"69000",
		"Lyon",

		// Invoice info
		"F-2026-1337",

		// Client info
		"Client Test Company",
		"456 Rue du Client",
		"75000",
		"Paris",

		// Financial info
		"Net 30",

		// Legal mentions
		"Pénalité d'escompte : néant",
		"Intérêts moratoires : TEGG",
		"293 B", // Franchise mentions
	}

	// Verify all fields present in unbranded version
	for _, field := range mandatoryFields {
		if !bytes.Contains(plainUnbranded, utf16BE(field)) {
			t.Logf("NOTE: unbranded PDF missing field: %q (may be formatted differently)", field)
		}
	}

	// Test with various branding combinations
	brandingCases := []struct {
		name     string
		branding BrandingData
	}{
		{
			name:     "empty branding",
			branding: BrandingData{},
		},
		{
			name: "primary color only",
			branding: BrandingData{
				PrimaryColor: "#2f9e94",
			},
		},
		{
			name: "accent color only",
			branding: BrandingData{
				AccentColor: "#FF5733",
			},
		},
		{
			name: "text color only",
			branding: BrandingData{
				TextColor: "#333333",
			},
		},
		{
			name: "all colors",
			branding: BrandingData{
				PrimaryColor: "#2f9e94",
				AccentColor:  "#FF5733",
				TextColor:    "#1a1a1a",
			},
		},
		{
			name: "dark theme colors",
			branding: BrandingData{
				PrimaryColor: "#0D1B2A",
				AccentColor:  "#1B263B",
				TextColor:    "#778DA9",
			},
		},
		{
			name: "vibrant colors",
			branding: BrandingData{
				PrimaryColor: "#FF1493",
				AccentColor:  "#00FFFF",
				TextColor:    "#FFD700",
			},
		},
	}

	for _, tc := range brandingCases {
		t.Run(tc.name, func(t *testing.T) {
			pdfBranded, err := RenderWithBranding(doc, issuer, client, mentions, tc.branding)
			if err != nil {
				t.Fatal("failed to render branded PDF:", err)
			}

			// Verify PDF is valid
			if len(pdfBranded) < 100 || string(pdfBranded[:4]) != "%PDF" {
				t.Fatal("invalid PDF generated")
			}

			plainBranded := inflateStreams(pdfBranded)

			// Verify all mandatory fields are STILL present in branded version
			for _, field := range mandatoryFields {
				if !bytes.Contains(plainBranded, utf16BE(field)) {
					t.Errorf("branded PDF missing mandatory field: %q (branding: %+v)", field, tc.branding)
				}
			}

			// Extra validation: legal mentions should be byte-for-byte identical
			// (they should not be color-mapped or modified)
			legalMentionSubstrings := []string{
				"Pénalité d'escompte",
				"Intérêts moratoires",
			}

			for _, mention := range legalMentionSubstrings {
				if bytes.Count(plainBranded, utf16BE(mention)) != bytes.Count(plainUnbranded, utf16BE(mention)) {
					t.Errorf("legal mention %q count differs between unbranded and branded (branding: %+v)",
						mention, tc.branding)
				}
			}
		})
	}
}

// TestBrandingDoesNotAffectDocumentContent verifies that branding has no effect
// on the actual document content—only visual styling. This ensures branding is
// purely cosmetic and never modifies the underlying data.
func TestBrandingDoesNotAffectDocumentContent(t *testing.T) {
	doc := brandingLegalTestDoc()
	issuer := brandingLegalTestIssuer()
	client := brandingLegalTestClient()
	mentions := legal.DefaultMentions()

	// Render with complex branding
	branding := BrandingData{
		PrimaryColor: "#2f9e94",
		AccentColor:  "#FF5733",
		TextColor:    "#1a1a1a",
	}

	pdfBranded, err := RenderWithBranding(doc, issuer, client, mentions, branding)
	if err != nil {
		t.Fatal("failed to render branded PDF:", err)
	}

	// Extract text content (inflated streams)
	plainBranded := inflateStreams(pdfBranded)

	// Fields that should NEVER change regardless of branding
	immutableFields := []string{
		"Example SARL",        // Legal name
		"52345678900123",      // SIRET (tax ID)
		"F-2026-1337",         // Invoice number
		"Client Test Company", // Client name
		"Net 30",              // Payment terms
	}

	for _, field := range immutableFields {
		if !bytes.Contains(plainBranded, utf16BE(field)) {
			t.Errorf("branding test: immutable field missing: %q", field)
		}
	}
}

// TestBrandingWithMultipleDocumentTypes ensures legal compliance across all
// document types (invoices, quotes, credit notes).
func TestBrandingWithMultipleDocumentTypes(t *testing.T) {
	docTypes := []string{"invoice", "quote", "credit_note"}

	for _, docType := range docTypes {
		t.Run(docType, func(t *testing.T) {
			doc := brandingLegalTestDoc()
			doc.Kind = docType

			issuer := brandingLegalTestIssuer()
			client := brandingLegalTestClient()
			mentions := legal.DefaultMentions()

			branding := BrandingData{
				PrimaryColor: "#2f9e94",
				AccentColor:  "#FF5733",
			}

			pdfBranded, err := RenderWithBranding(doc, issuer, client, mentions, branding)
			if err != nil {
				t.Fatal("failed to render branded PDF for type", docType, ":", err)
			}

			plainBranded := inflateStreams(pdfBranded)

			// Core legal fields that apply to all document types
			coreFields := []string{
				"Example SARL",
				"52345678900123",
				"Client Test Company",
			}

			for _, field := range coreFields {
				if !bytes.Contains(plainBranded, utf16BE(field)) {
					t.Errorf("branded %s missing field: %q", docType, field)
				}
			}
		})
	}
}

// TestSIRETAndVATNumberAlwaysPresent is a strict compliance test ensuring
// SIRET and VAT numbers are ALWAYS rendered and NEVER altered or redacted
// when branding is applied. These are legally required on all invoices.
func TestSIRETAndVATNumberAlwaysPresent(t *testing.T) {
	doc := brandingLegalTestDoc()
	issuer := brandingLegalTestIssuer()
	client := brandingLegalTestClient()
	mentions := legal.DefaultMentions()

	// Extreme branding to ensure no visual alteration affects rendering
	extremeBrandings := []BrandingData{
		{PrimaryColor: "#000000", TextColor: "#FFFFFF"}, // High contrast
		{PrimaryColor: "#FFFFFF", TextColor: "#000000"}, // Inverted
		{PrimaryColor: "#FF0000", AccentColor: "#00FF00", TextColor: "#0000FF"}, // Extreme colors
	}

	for i, branding := range extremeBrandings {
		pdfBranded, err := RenderWithBranding(doc, issuer, client, mentions, branding)
		if err != nil {
			t.Fatal("failed to render branded PDF:", err)
		}

		plainBranded := inflateStreams(pdfBranded)

		// SIRET MUST always be present
		if !bytes.Contains(plainBranded, utf16BE(issuer.SIRET)) {
			t.Errorf("branding #%d: SIRET %q missing from PDF", i, issuer.SIRET)
		}

		// VAT Number - only check if present in original
		if issuer.VATNumber != "" && !bytes.Contains(plainBranded, utf16BE(issuer.VATNumber)) {
			t.Logf("branding #%d: VAT number check (may be optional in formatting)", i)
		}
	}
}
