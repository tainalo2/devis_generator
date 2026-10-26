package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/tainalo2/devis_generator/backend/internal/billing"
	"github.com/tainalo2/devis_generator/backend/internal/legal"
)

type DocumentInput struct {
	ClientID          *uuid.UUID `json:"client_id"`
	Kind              string     `json:"kind"`
	Title             string     `json:"title"`
	IssueDate         *string    `json:"issue_date"`
	DueDate           *string    `json:"due_date"`
	Currency          string     `json:"currency"`
	VATRegime         string     `json:"vat_regime"`
	Notes             string     `json:"notes"`
	PaymentTerms      string     `json:"payment_terms"`
	RelatedDocumentID *uuid.UUID `json:"related_document_id"`
	Lines             []DocLine  `json:"lines"`
}

func (s *Store) ListDocuments(ctx context.Context, userID uuid.UUID) ([]Document, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, client_id, kind::text, status::text, number, COALESCE(title,''),
		       to_char(issue_date, 'YYYY-MM-DD'), to_char(due_date, 'YYYY-MM-DD'),
		       currency, vat_regime::text, COALESCE(notes,''), COALESCE(payment_terms,''),
		       COALESCE(legal_mentions,''), related_document_id,
		       subtotal_cents, tax_cents, total_cents, issued_at, sent_at, accepted_at, paid_at, created_at, updated_at,
		       COALESCE(client_snapshot->>'name', '')
		FROM documents WHERE user_id = $1 ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Document{}
	for rows.Next() {
		d, err := scanDocHeader(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

type scannable interface {
	Scan(dest ...any) error
}

func scanDocHeader(row scannable) (Document, error) {
	var d Document
	var issue, due *string
	err := row.Scan(
		&d.ID, &d.ClientID, &d.Kind, &d.Status, &d.Number, &d.Title,
		&issue, &due, &d.Currency, &d.VATRegime, &d.Notes, &d.PaymentTerms,
		&d.LegalMentions, &d.RelatedDocumentID,
		&d.SubtotalCents, &d.TaxCents, &d.TotalCents, &d.IssuedAt, &d.SentAt, &d.AcceptedAt, &d.PaidAt, &d.CreatedAt, &d.UpdatedAt,
		&d.ClientName,
	)
	if err != nil {
		return Document{}, err
	}
	d.IssueDate = issue
	d.DueDate = due
	return d, nil
}

func (s *Store) GetDocument(ctx context.Context, userID, id uuid.UUID) (Document, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, client_id, kind::text, status::text, number, COALESCE(title,''),
		       to_char(issue_date, 'YYYY-MM-DD'), to_char(due_date, 'YYYY-MM-DD'),
		       currency, vat_regime::text, COALESCE(notes,''), COALESCE(payment_terms,''),
		       COALESCE(legal_mentions,''), related_document_id,
		       subtotal_cents, tax_cents, total_cents, issued_at, sent_at, accepted_at, paid_at, issuer_snapshot, legal_snapshot,
		       created_at, updated_at, client_snapshot
		FROM documents WHERE id = $1 AND user_id = $2
	`, id, userID)
	var d Document
	var issue, due *string
	var clientSnap []byte
	err := row.Scan(
		&d.ID, &d.ClientID, &d.Kind, &d.Status, &d.Number, &d.Title,
		&issue, &due, &d.Currency, &d.VATRegime, &d.Notes, &d.PaymentTerms,
		&d.LegalMentions, &d.RelatedDocumentID,
		&d.SubtotalCents, &d.TaxCents, &d.TotalCents, &d.IssuedAt, &d.SentAt, &d.AcceptedAt, &d.PaidAt, &d.IssuerSnapshot, &d.LegalSnapshot,
		&d.CreatedAt, &d.UpdatedAt, &clientSnap,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Document{}, ErrNotFound
	}
	if err != nil {
		return Document{}, err
	}
	d.IssueDate = issue
	d.DueDate = due
	// L'instantané est stocké chiffré ; le propriétaire le reçoit en clair.
	if len(d.IssuerSnapshot) > 0 {
		snap, err := s.openIssuerSnapshot(d.IssuerSnapshot)
		if err != nil {
			return Document{}, err
		}
		if d.IssuerSnapshot, err = json.Marshal(snap); err != nil {
			return Document{}, err
		}
	}
	if len(clientSnap) > 0 {
		c, err := s.openClientSnapshot(clientSnap)
		if err != nil {
			return Document{}, err
		}
		d.ClientSnapshot = &c
		d.ClientName = c.Name
	}
	lines, err := s.loadLines(ctx, d.ID)
	if err != nil {
		return Document{}, err
	}
	d.Lines = lines
	return d, nil
}

func (s *Store) loadLines(ctx context.Context, docID uuid.UUID) ([]DocLine, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, position, description, quantity_milli, unit_price_cents, discount_bps, tax_rate_bps, tax_cents, total_cents
		FROM document_lines WHERE document_id = $1 ORDER BY position, created_at
	`, docID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	lines := []DocLine{}
	for rows.Next() {
		var line DocLine
		if err := rows.Scan(&line.ID, &line.Position, &line.Description, &line.QuantityMilli, &line.UnitPriceCents, &line.DiscountBps, &line.TaxRateBps, &line.TaxCents, &line.TotalCents); err != nil {
			return nil, err
		}
		lines = append(lines, line)
	}
	return lines, rows.Err()
}

func (s *Store) CreateDocument(ctx context.Context, userID uuid.UUID, in DocumentInput) (Document, error) {
	if err := validateDocInput(in, false); err != nil {
		return Document{}, err
	}
	calc, err := computeInput(in)
	if err != nil {
		return Document{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Document{}, err
	}
	defer tx.Rollback(ctx)

	var id uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO documents (user_id, client_id, kind, status, title, issue_date, due_date, currency, vat_regime,
		                       notes, payment_terms, related_document_id, subtotal_cents, tax_cents, total_cents)
		VALUES ($1,$2,$3::document_kind,'draft',$4,$5::date,$6::date,$7,$8::vat_regime,$9,$10,$11,$12,$13,$14)
		RETURNING id
	`, userID, in.ClientID, in.Kind, in.Title, in.IssueDate, in.DueDate, defaultCurrency(in.Currency), in.VATRegime,
		in.Notes, in.PaymentTerms, in.RelatedDocumentID, calc.SubtotalCents, calc.TaxCents, calc.TotalCents,
	).Scan(&id)
	if err != nil {
		return Document{}, err
	}
	if err := insertLines(ctx, tx, id, calc); err != nil {
		return Document{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Document{}, err
	}
	return s.GetDocument(ctx, userID, id)
}

func (s *Store) UpdateDocument(ctx context.Context, userID, id uuid.UUID, in DocumentInput) (Document, error) {
	cur, err := s.GetDocument(ctx, userID, id)
	if err != nil {
		return Document{}, err
	}
	if isImmutable(cur.Status) {
		return Document{}, ErrImmutable
	}
	if err := validateDocInput(in, false); err != nil {
		return Document{}, err
	}
	calc, err := computeInput(in)
	if err != nil {
		return Document{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Document{}, err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		UPDATE documents SET client_id=$1, kind=$2::document_kind, title=$3, issue_date=$4::date, due_date=$5::date,
		                     currency=$6, vat_regime=$7::vat_regime, notes=$8, payment_terms=$9,
		                     related_document_id=$10, subtotal_cents=$11, tax_cents=$12, total_cents=$13, updated_at=NOW()
		WHERE id=$14 AND user_id=$15 AND status = 'draft'
	`, in.ClientID, in.Kind, in.Title, in.IssueDate, in.DueDate, defaultCurrency(in.Currency), in.VATRegime,
		in.Notes, in.PaymentTerms, in.RelatedDocumentID, calc.SubtotalCents, calc.TaxCents, calc.TotalCents, id, userID)
	if err != nil {
		return Document{}, err
	}
	if tag.RowsAffected() == 0 {
		return Document{}, ErrImmutable
	}
	if _, err := tx.Exec(ctx, `DELETE FROM document_lines WHERE document_id = $1`, id); err != nil {
		return Document{}, err
	}
	if err := insertLines(ctx, tx, id, calc); err != nil {
		return Document{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Document{}, err
	}
	return s.GetDocument(ctx, userID, id)
}

func (s *Store) DeleteDocument(ctx context.Context, userID, id uuid.UUID) error {
	cur, err := s.GetDocument(ctx, userID, id)
	if err != nil {
		return err
	}
	if isImmutable(cur.Status) {
		return ErrImmutable
	}
	tag, err := s.pool.Exec(ctx, `DELETE FROM documents WHERE id=$1 AND user_id=$2 AND status='draft'`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) IssueDocument(ctx context.Context, userID, id uuid.UUID) (Document, error) {
	cur, err := s.GetDocument(ctx, userID, id)
	if err != nil {
		return Document{}, err
	}
	if isImmutable(cur.Status) {
		return Document{}, ErrImmutable
	}
	if len(cur.Lines) == 0 {
		return Document{}, fmt.Errorf("%w: document sans lignes", ErrInvalidInput)
	}

	issuer, err := s.GetIssuer(ctx, userID)
	if err != nil {
		return Document{}, err
	}
	mentions := s.resolveMentions(ctx)
	issuerJSON, err := s.sealIssuerSnapshot(issuer)
	if err != nil {
		return Document{}, err
	}
	// Le client est figé avec le document : la fiche peut ensuite changer ou disparaître.
	var clientJSON []byte
	if cur.ClientID != nil {
		c, err := s.GetClient(ctx, userID, *cur.ClientID)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return Document{}, err
		}
		if err == nil {
			if clientJSON, err = s.sealClientSnapshot(snapshotOf(c)); err != nil {
				return Document{}, err
			}
		}
	}
	legalJSON, _ := json.Marshal(mentions)

	issueDate := time.Now()
	if cur.IssueDate != nil && *cur.IssueDate != "" {
		if t, err := time.Parse("2006-01-02", *cur.IssueDate); err == nil {
			issueDate = t
		}
	}
	year := issueDate.Year()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Document{}, err
	}
	defer tx.Rollback(ctx)

	// Lock document row
	var status string
	if err := tx.QueryRow(ctx, `SELECT status::text FROM documents WHERE id=$1 AND user_id=$2 FOR UPDATE`, id, userID).Scan(&status); err != nil {
		return Document{}, err
	}
	if isImmutable(status) {
		return Document{}, ErrImmutable
	}

	number, err := nextNumber(ctx, tx, userID.String(), cur.Kind, year)
	if err != nil {
		return Document{}, err
	}
	legalMentions := buildLegalMentions(cur.VATRegime, mentions)

	_, err = tx.Exec(ctx, `
		UPDATE documents SET status='issued', number=$1, issue_date=$2::date, issued_at=NOW(),
		                     issuer_snapshot=$3, legal_snapshot=$4, legal_mentions=$5, client_snapshot=$6, updated_at=NOW()
		WHERE id=$7 AND user_id=$8
	`, number, issueDate.Format("2006-01-02"), issuerJSON, legalJSON, legalMentions, clientJSON, id, userID)
	if err != nil {
		return Document{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Document{}, err
	}
	return s.GetDocument(ctx, userID, id)
}

func (s *Store) ConvertQuoteToInvoice(ctx context.Context, userID, quoteID uuid.UUID) (Document, error) {
	src, err := s.GetDocument(ctx, userID, quoteID)
	if err != nil {
		return Document{}, err
	}
	if src.Kind != "quote" {
		return Document{}, fmt.Errorf("%w: seul un devis peut être converti", ErrInvalidInput)
	}
	in := DocumentInput{
		ClientID:          src.ClientID,
		Kind:              "invoice",
		Title:             src.Title,
		Currency:          src.Currency,
		VATRegime:         src.VATRegime,
		Notes:             src.Notes,
		PaymentTerms:      src.PaymentTerms,
		RelatedDocumentID: &src.ID,
		Lines:             src.Lines,
	}
	return s.CreateDocument(ctx, userID, in)
}

func (s *Store) CreateLinkedDocument(ctx context.Context, userID, originID uuid.UUID, kind string, lines []DocLine) (Document, error) {
	src, err := s.GetDocument(ctx, userID, originID)
	if err != nil {
		return Document{}, err
	}
	if kind != "deposit_invoice" && kind != "credit_note" {
		return Document{}, ErrInvalidInput
	}
	if kind == "credit_note" && src.Kind != "invoice" && src.Kind != "deposit_invoice" {
		return Document{}, fmt.Errorf("%w: un avoir doit être lié à une facture", ErrInvalidInput)
	}
	useLines := lines
	if len(useLines) == 0 {
		useLines = src.Lines
		if kind == "credit_note" {
			for i := range useLines {
				useLines[i].UnitPriceCents = -abs64(useLines[i].UnitPriceCents)
			}
		}
	}
	in := DocumentInput{
		ClientID:          src.ClientID,
		Kind:              kind,
		Title:             src.Title,
		Currency:          src.Currency,
		VATRegime:         src.VATRegime,
		Notes:             src.Notes,
		PaymentTerms:      src.PaymentTerms,
		RelatedDocumentID: &src.ID,
		Lines:             useLines,
	}
	return s.CreateDocument(ctx, userID, in)
}

func nextNumber(ctx context.Context, tx pgx.Tx, issuerKey, kind string, year int) (string, error) {
	prefix, err := billing.PrefixForKind(kind)
	if err != nil {
		return "", err
	}
	var seq int
	err = tx.QueryRow(ctx, `
		INSERT INTO document_number_sequences (issuer_key, kind, year, prefix, last_value, updated_at)
		VALUES ($1, $2::document_kind, $3, $4, 1, NOW())
		ON CONFLICT (issuer_key, kind, year)
		DO UPDATE SET last_value = document_number_sequences.last_value + 1, updated_at = NOW()
		RETURNING last_value
	`, issuerKey, kind, year, prefix).Scan(&seq)
	if err != nil {
		return "", err
	}
	return billing.FormatNumber(prefix, year, seq), nil
}

func insertLines(ctx context.Context, tx pgx.Tx, docID uuid.UUID, calc billing.Result) error {
	for i, l := range calc.Lines {
		tax := int64(0)
		// proportional display tax on line for UI (optional); totals use buckets
		if calc.VatRegime == billing.RegimeStandard && l.TaxRateBps > 0 {
			tax = billingRoundLineTax(l.TotalHTCents, l.TaxRateBps)
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO document_lines (document_id, position, description, quantity, quantity_milli, unit_price_cents,
			                            discount_bps, tax_rate_bps, tax_cents, total_cents)
			VALUES ($1,$2,$3,$4::numeric,$5,$6,$7,$8,$9,$10)
		`, docID, i, l.Description, fmt.Sprintf("%.3f", float64(l.QuantityMilli)/1000.0), l.QuantityMilli,
			l.UnitPriceCents, l.DiscountBps, l.TaxRateBps, tax, l.TotalHTCents)
		if err != nil {
			return err
		}
	}
	return nil
}

func billingRoundLineTax(ht int64, rateBps int) int64 {
	return roundHalfUpLocal(ht*int64(rateBps), 10000)
}

func roundHalfUpLocal(numer, denom int64) int64 {
	if denom < 0 {
		numer = -numer
		denom = -denom
	}
	if numer >= 0 {
		return (numer + denom/2) / denom
	}
	return -((-numer + denom/2) / denom)
}

func computeInput(in DocumentInput) (billing.Result, error) {
	inputs := make([]billing.LineInput, 0, len(in.Lines))
	for _, l := range in.Lines {
		inputs = append(inputs, billing.LineInput{
			Description:    l.Description,
			QuantityMilli:  l.QuantityMilli,
			UnitPriceCents: l.UnitPriceCents,
			DiscountBps:    l.DiscountBps,
			TaxRateBps:     l.TaxRateBps,
		})
	}
	return billing.Compute(in.VATRegime, inputs)
}

func validateDocInput(in DocumentInput, requireLines bool) error {
	switch in.Kind {
	case "quote", "invoice", "deposit_invoice", "credit_note":
	default:
		return fmt.Errorf("%w: type invalide", ErrInvalidInput)
	}
	if in.VATRegime != billing.RegimeFranchise293B && in.VATRegime != billing.RegimeStandard {
		return fmt.Errorf("%w: régime TVA invalide", ErrInvalidInput)
	}
	if requireLines && len(in.Lines) == 0 {
		return fmt.Errorf("%w: lignes requises", ErrInvalidInput)
	}
	for _, l := range in.Lines {
		if l.QuantityMilli == 0 {
			return fmt.Errorf("%w: quantité invalide", ErrInvalidInput)
		}
	}
	return nil
}

func isImmutable(status string) bool {
	return status == "issued" || status == "sent" || status == "paid" || status == "accepted" || status == "rejected"
}

func defaultCurrency(c string) string {
	if strings.TrimSpace(c) == "" {
		return "EUR"
	}
	return c
}

func buildLegalMentions(regime string, m legal.Mentions) string {
	parts := []string{}
	if regime == billing.RegimeFranchise293B {
		parts = append(parts, m.Franchise293B)
	}
	parts = append(parts, m.LatePenalty, m.FlatIndemnity40EUR)
	return strings.Join(parts, "\n")
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

// StatusTransitionAllowed indique si un document émis peut passer au statut demandé.
// Le contenu reste figé : seul le suivi (envoi, acceptation, paiement) évolue.
func StatusTransitionAllowed(kind, from, to string) bool {
	if from == "draft" || from == to {
		return false
	}
	switch to {
	case "sent":
		return from == "issued"
	case "accepted", "rejected":
		return kind == "quote" && (from == "issued" || from == "sent")
	case "paid":
		return (kind == "invoice" || kind == "deposit_invoice") && (from == "issued" || from == "sent")
	}
	return false
}

func (s *Store) UpdateDocumentStatus(ctx context.Context, userID, id uuid.UUID, to string) (Document, error) {
	cur, err := s.GetDocument(ctx, userID, id)
	if err != nil {
		return Document{}, err
	}
	if !StatusTransitionAllowed(cur.Kind, cur.Status, to) {
		return Document{}, fmt.Errorf("%w: transition de statut impossible", ErrInvalidInput)
	}
	col := map[string]string{"sent": "sent_at", "accepted": "accepted_at", "rejected": "accepted_at", "paid": "paid_at"}[to]
	// col provient d'une table fixe ci-dessus : pas d'injection possible.
	tag, err := s.pool.Exec(ctx, `
		UPDATE documents SET status=$1::document_status, `+col+`=NOW(), updated_at=NOW()
		WHERE id=$2 AND user_id=$3 AND status::text=$4
	`, to, id, userID, cur.Status)
	if err != nil {
		return Document{}, err
	}
	if tag.RowsAffected() == 0 {
		return Document{}, fmt.Errorf("%w: statut modifié entre-temps", ErrInvalidInput)
	}
	return s.GetDocument(ctx, userID, id)
}
