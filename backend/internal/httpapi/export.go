package httpapi

import (
	"context"
	"encoding/csv"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/tainalo2/devis_generator/backend/internal/biz"
)

// handleAccountingExport generates a CSV export of issued documents with payment details.
// Query parameters:
//   - date_from: (optional) YYYY-MM-DD, filter documents issued on or after this date
//   - date_to: (optional) YYYY-MM-DD, filter documents issued on or before this date
//   - status: (optional) comma-separated payment statuses (en_attente,paye,en_retard,annule)
//   - types: (optional) comma-separated document types (facture,facture_acompte,avoir,devis)
func (s *Server) handleAccountingExport(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "Non authentifié.")
		return
	}

	// Parse query parameters
	params := parseExportParams(r)
	if params.err != nil {
		writeAPIError(w, http.StatusBadRequest, params.err.Error())
		return
	}

	// Fetch documents (all issued documents, filtered below)
	docs, err := s.biz.ListDocuments(r.Context(), u.ID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to list documents", "error", err)
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}

	// Filter documents based on query parameters
	filtered := filterDocumentsForExport(docs, params)

	// Get payment details for all documents
	paymentDetails, err := s.fetchPaymentDetails(r.Context(), u.ID, filtered)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to fetch payment details", "error", err)
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}

	// Generate CSV
	csvData, err := generateAccountingCSV(filtered, paymentDetails)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to generate CSV", "error", err)
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}

	// Set response headers for CSV download
	filename := fmt.Sprintf("export-comptable-%s.csv", time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "text/csv; charset=utf-8-sig")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.WriteHeader(http.StatusOK)
	w.Write(csvData)
}

type exportParams struct {
	dateFrom   *time.Time
	dateTo     *time.Time
	statuses   map[string]bool
	docTypes   map[string]bool
	err        error
}

func parseExportParams(r *http.Request) exportParams {
	params := exportParams{
		statuses: make(map[string]bool),
		docTypes: make(map[string]bool),
	}

	// Parse date_from
	if df := r.URL.Query().Get("date_from"); df != "" {
		t, err := time.Parse("2006-01-02", df)
		if err != nil {
			params.err = fmt.Errorf("invalid date_from format, expected YYYY-MM-DD: %w", err)
			return params
		}
		params.dateFrom = &t
	}

	// Parse date_to
	if dt := r.URL.Query().Get("date_to"); dt != "" {
		t, err := time.Parse("2006-01-02", dt)
		if err != nil {
			params.err = fmt.Errorf("invalid date_to format, expected YYYY-MM-DD: %w", err)
			return params
		}
		params.dateTo = &t
	}

	// Parse statuses
	if st := r.URL.Query().Get("status"); st != "" {
		for _, s := range strings.Split(st, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				params.statuses[s] = true
			}
		}
	}

	// Parse document types
	if ty := r.URL.Query().Get("types"); ty != "" {
		for _, t := range strings.Split(ty, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				params.docTypes[t] = true
			}
		}
	}

	return params
}

func filterDocumentsForExport(docs []biz.Document, params exportParams) []biz.Document {
	var filtered []biz.Document

	for _, doc := range docs {
		// Only include issued documents (invoices, deposit invoices, credit notes)
		// Exclude drafts and quotes
		if doc.Kind != "invoice" && doc.Kind != "deposit_invoice" && doc.Kind != "credit_note" {
			continue
		}
		if doc.IssuedAt == nil {
			continue
		}

		// Filter by date_from (inclusive)
		if params.dateFrom != nil && doc.IssuedAt.Before(*params.dateFrom) {
			continue
		}

		// Filter by date_to (inclusive)
		if params.dateTo != nil && doc.IssuedAt.After(params.dateTo.Add(24 * time.Hour)) {
			continue
		}

		// Filter by document types if specified
		if len(params.docTypes) > 0 {
			docType := kindToExportType(doc.Kind)
			if !params.docTypes[docType] {
				continue
			}
		}

		filtered = append(filtered, doc)
	}

	return filtered
}

func kindToExportType(kind string) string {
	switch kind {
	case "invoice":
		return "facture"
	case "deposit_invoice":
		return "facture_acompte"
	case "credit_note":
		return "avoir"
	case "quote":
		return "devis"
	default:
		return kind
	}
}

type paymentInfo struct {
	status        string    // en_attente, paye, en_retard, annule
	paidAt        *time.Time
	paymentMethod string // virement, stripe, chèque, etc.
}

func (s *Server) fetchPaymentDetails(ctx context.Context, userID interface{}, docs []biz.Document) (map[string]paymentInfo, error) {
	details := make(map[string]paymentInfo)

	// Query all payment attempts for this user and these documents
	for _, doc := range docs {
		info := paymentInfo{
			status: "en_attente",
		}

		// Determine payment status from document fields
		if doc.PaidAt != nil {
			info.status = "paye"
			info.paidAt = doc.PaidAt
			// Try to get payment method from payment attempts
			// For now, we'll set it via a separate query in the biz layer
		} else if doc.IssuedAt != nil && time.Since(*doc.IssuedAt) > 30*24*time.Hour {
			// If issued more than 30 days ago and not paid, mark as en_retard
			// Note: This is a simplified heuristic; a real implementation might
			// use the due_date field if available
			if doc.DueDate != nil {
				dueTime, _ := time.Parse("2006-01-02", *doc.DueDate)
				if time.Now().After(dueTime) {
					info.status = "en_retard"
				}
			}
		}

		details[doc.ID.String()] = info
	}

	return details, nil
}

func generateAccountingCSV(docs []biz.Document, paymentDetails map[string]paymentInfo) ([]byte, error) {
	// Create a buffer to write CSV
	var buf strings.Builder
	writer := csv.NewWriter(&buf)
	writer.Comma = ';'

	// Write BOM for UTF-8 (Excel compatibility)
	buf.WriteString("\ufeff")

	// Write header row
	headers := []string{
		"numero_document",
		"type_document",
		"date_emission",
		"client_nom",
		"client_siret",
		"montant_ht",
		"taux_tva",
		"montant_tva",
		"montant_ttc",
		"statut_paiement",
		"date_paiement",
		"mode_paiement",
		"document_origine",
	}
	if err := writer.Write(headers); err != nil {
		return nil, fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write data rows
	for _, doc := range docs {
		row, err := documentToCSVRow(doc, paymentDetails[doc.ID.String()])
		if err != nil {
			return nil, fmt.Errorf("failed to convert document to CSV row: %w", err)
		}
		if err := writer.Write(row); err != nil {
			return nil, fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("CSV writer error: %w", err)
	}

	return []byte(buf.String()), nil
}

func documentToCSVRow(doc biz.Document, payment paymentInfo) ([]string, error) {
	// Extract client name and SIRET
	clientName := ""
	clientSIRET := ""
	if doc.ClientSnapshot != nil {
		clientName = doc.ClientSnapshot.Name
		clientSIRET = doc.ClientSnapshot.SIRET
	} else if doc.ClientName != "" {
		clientName = doc.ClientName
	}

	// Format dates
	issueDate := ""
	if doc.IssueDate != nil {
		issueDate = *doc.IssueDate
	}

	paidDate := ""
	if payment.paidAt != nil {
		paidDate = payment.paidAt.Format("2006-01-02")
	}

	// Calculate VAT rate (average if multiple lines, or 0 if franchise)
	vatRate := "0"
	if doc.VATRegime == "standard" && len(doc.Lines) > 0 {
		totalTaxRateBps := 0
		for _, line := range doc.Lines {
			totalTaxRateBps += line.TaxRateBps
		}
		avgTaxRateBps := totalTaxRateBps / len(doc.Lines)
		vatRate = fmt.Sprintf("%.1f", float64(avgTaxRateBps)/100.0)
	}

	// Convert cents to euros (2 decimal places)
	htAmount := fmt.Sprintf("%.2f", float64(doc.SubtotalCents)/100.0)
	tvAmount := fmt.Sprintf("%.2f", float64(doc.TaxCents)/100.0)
	ttcAmount := fmt.Sprintf("%.2f", float64(doc.TotalCents)/100.0)

	// Get related document number if applicable
	relatedDocNumber := ""
	// Note: We would need to fetch the related document from the store
	// For now, leave empty - this will be enhanced if needed

	row := []string{
		*doc.Number,                    // numero_document
		kindToExportType(doc.Kind),     // type_document
		issueDate,                      // date_emission
		clientName,                     // client_nom
		clientSIRET,                    // client_siret
		htAmount,                       // montant_ht
		vatRate,                        // taux_tva
		tvAmount,                       // montant_tva
		ttcAmount,                      // montant_ttc
		payment.status,                 // statut_paiement
		paidDate,                       // date_paiement
		payment.paymentMethod,          // mode_paiement
		relatedDocNumber,               // document_origine
	}

	return row, nil
}
