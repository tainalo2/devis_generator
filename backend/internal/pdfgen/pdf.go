// Package pdfgen renders Devis Generator : VIBED documents to PDF.
// Library: github.com/phpdave11/gofpdf (MIT — compatible with Apache-2.0).
package pdfgen

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/phpdave11/gofpdf"
	"github.com/tainalo2/devis_generator/backend/internal/billing"
	"github.com/tainalo2/devis_generator/backend/internal/biz"
	"github.com/tainalo2/devis_generator/backend/internal/legal"
)

// Render builds a simple A4 PDF for a document + parties.
func Render(doc biz.Document, issuer biz.IssuerProfile, client biz.Client, mentions legal.Mentions) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetTitle(docTitle(doc), false)
	pdf.SetAuthor("Devis Generator : VIBED", false)
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 16)
	pdf.Cell(0, 10, docTitle(doc))
	pdf.Ln(12)

	pdf.SetFont("Helvetica", "", 10)
	if doc.Number != nil && *doc.Number != "" {
		pdf.Cell(0, 6, "N° "+*doc.Number)
		pdf.Ln(6)
	}
	if doc.IssueDate != nil {
		pdf.Cell(0, 6, "Date : "+*doc.IssueDate)
		pdf.Ln(6)
	}

	pdf.Ln(4)
	writeParty(pdf, "Émetteur", issuer.LegalName, issuer.TradeName, issuer.Address, issuer.SIRET, issuer.Email, issuer.VATNumber)
	pdf.Ln(4)
	writeParty(pdf, "Client", client.Name, "", client.Address, client.SIRET, client.Email, client.VATNumber)

	pdf.Ln(6)
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(80, 7, "Désignation", "1", 0, "", false, 0, "")
	pdf.CellFormat(25, 7, "Qté", "1", 0, "R", false, 0, "")
	pdf.CellFormat(30, 7, "P.U. HT", "1", 0, "R", false, 0, "")
	pdf.CellFormat(25, 7, "TVA", "1", 0, "R", false, 0, "")
	pdf.CellFormat(30, 7, "Total HT", "1", 1, "R", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)

	for _, l := range doc.Lines {
		qty := float64(l.QuantityMilli) / 1000.0
		pdf.CellFormat(80, 6, truncate(l.Description, 48), "1", 0, "", false, 0, "")
		pdf.CellFormat(25, 6, fmt.Sprintf("%.3f", qty), "1", 0, "R", false, 0, "")
		pdf.CellFormat(30, 6, formatEUR(l.UnitPriceCents), "1", 0, "R", false, 0, "")
		pdf.CellFormat(25, 6, formatRate(l.TaxRateBps), "1", 0, "R", false, 0, "")
		pdf.CellFormat(30, 6, formatEUR(l.TotalCents), "1", 1, "R", false, 0, "")
	}

	pdf.Ln(4)
	pdf.CellFormat(160, 6, "Total HT", "", 0, "R", false, 0, "")
	pdf.CellFormat(30, 6, formatEUR(doc.SubtotalCents), "", 1, "R", false, 0, "")
	pdf.CellFormat(160, 6, "Total TVA", "", 0, "R", false, 0, "")
	pdf.CellFormat(30, 6, formatEUR(doc.TaxCents), "", 1, "R", false, 0, "")
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(160, 7, "Total TTC", "", 0, "R", false, 0, "")
	pdf.CellFormat(30, 7, formatEUR(doc.TotalCents), "", 1, "R", false, 0, "")

	pdf.Ln(8)
	pdf.SetFont("Helvetica", "", 8)
	if doc.VATRegime == billing.RegimeFranchise293B {
		pdf.MultiCell(0, 4, mentions.Franchise293B, "", "", false)
	}
	if doc.Kind == "invoice" || doc.Kind == "deposit_invoice" || doc.Kind == "credit_note" {
		terms := doc.PaymentTerms
		if terms == "" {
			terms = mentions.PaymentTermsDefault
		}
		pdf.MultiCell(0, 4, "Conditions de paiement : "+terms, "", "", false)
		pdf.MultiCell(0, 4, mentions.LatePenalty, "", "", false)
		pdf.MultiCell(0, 4, mentions.FlatIndemnity40EUR, "", "", false)
	}
	if doc.Kind == "quote" {
		pdf.MultiCell(0, 4, mentions.QuoteValidityNote, "", "", false)
	}
	if doc.LegalMentions != "" {
		pdf.Ln(2)
		pdf.MultiCell(0, 4, doc.LegalMentions, "", "", false)
	}
	if doc.Notes != "" {
		pdf.Ln(2)
		pdf.SetFont("Helvetica", "I", 8)
		pdf.MultiCell(0, 4, "Notes : "+doc.Notes, "", "", false)
	}
	pdf.Ln(6)
	pdf.SetFont("Helvetica", "", 7)
	pdf.MultiCell(0, 3, "Document généré par Devis Generator : VIBED — mentions légales source "+mentions.Source, "", "", false)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeParty(pdf *gofpdf.Fpdf, label, name, trade string, addr biz.Address, siret, email, vat string) {
	pdf.SetFont("Helvetica", "B", 10)
	pdf.Cell(0, 6, label)
	pdf.Ln(5)
	pdf.SetFont("Helvetica", "", 9)
	if name != "" {
		pdf.Cell(0, 5, name)
		pdf.Ln(4)
	}
	if trade != "" && trade != name {
		pdf.Cell(0, 5, trade)
		pdf.Ln(4)
	}
	line := strings.TrimSpace(strings.Join([]string{addr.Line1, addr.Line2}, " "))
	if line != "" {
		pdf.Cell(0, 5, line)
		pdf.Ln(4)
	}
	city := strings.TrimSpace(strings.Join([]string{addr.PostalCode, addr.City}, " "))
	if city != "" {
		pdf.Cell(0, 5, city)
		pdf.Ln(4)
	}
	if siret != "" {
		pdf.Cell(0, 5, "SIRET : "+siret)
		pdf.Ln(4)
	}
	if vat != "" {
		pdf.Cell(0, 5, "N° TVA : "+vat)
		pdf.Ln(4)
	}
	if email != "" {
		pdf.Cell(0, 5, email)
		pdf.Ln(4)
	}
}

func docTitle(doc biz.Document) string {
	switch doc.Kind {
	case "quote":
		return "Devis"
	case "invoice":
		return "Facture"
	case "deposit_invoice":
		return "Facture d'acompte"
	case "credit_note":
		return "Avoir"
	default:
		return "Document"
	}
}

func formatEUR(cents int64) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	return fmt.Sprintf("%s%d,%02d €", sign, cents/100, cents%100)
}

func formatRate(bps int) string {
	return fmt.Sprintf("%d,%02d %%", bps/100, bps%100)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
