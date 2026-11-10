package biz

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func mapReplacementNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

// DocumentLinkTarget is the current document reached from a stable public link.
type DocumentLinkTarget struct {
	ID        uuid.UUID
	Kind      string
	Version   int
	CreatedAt time.Time
	Replaced  bool
}

// ResolveDocumentLink follows a quote replacement chain to its current version.
// Invoice and deposit links resolve to the document itself.
func (s *Store) ResolveDocumentLink(ctx context.Context, id uuid.UUID) (DocumentLinkTarget, error) {
	var target DocumentLinkTarget
	err := s.pool.QueryRow(ctx, `
		WITH RECURSIVE chain AS (
			SELECT id, kind, quote_version, created_at, 0 AS depth
			FROM documents
			WHERE id = $1
			UNION ALL
			SELECT next.id, next.kind, next.quote_version, next.created_at, chain.depth + 1
			FROM documents next
			JOIN chain ON next.replaces_document_id = chain.id
		)
		SELECT id, kind::text, quote_version, created_at, depth > 0
		FROM chain
		ORDER BY depth DESC
		LIMIT 1
	`, id).Scan(&target.ID, &target.Kind, &target.Version, &target.CreatedAt, &target.Replaced)
	if err != nil {
		return DocumentLinkTarget{}, mapReplacementNotFound(err)
	}
	return target, nil
}

// ReplaceQuote creates a new draft from an immutable quote and records the
// predecessor/version relation atomically. A quote can have only one successor.
func (s *Store) ReplaceQuote(ctx context.Context, userID, quoteID uuid.UUID) (Document, error) {
	src, err := s.GetDocument(ctx, userID, quoteID)
	if err != nil {
		return Document{}, err
	}
	if src.Kind != "quote" {
		return Document{}, fmt.Errorf("%w: seul un devis peut être remplacé", ErrInvalidInput)
	}
	if !isImmutable(src.Status) {
		return Document{}, fmt.Errorf("%w: émettez le devis avant de le remplacer", ErrInvalidInput)
	}
	calc, err := computeInput(DocumentInput{VATRegime: src.VATRegime, Lines: src.Lines})
	if err != nil {
		return Document{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Document{}, err
	}
	defer tx.Rollback(ctx)

	var status, kind string
	var version int
	if err := tx.QueryRow(ctx, `
		SELECT status::text, kind::text, quote_version
		FROM documents
		WHERE id=$1 AND user_id=$2
		FOR UPDATE
	`, quoteID, userID).Scan(&status, &kind, &version); err != nil {
		return Document{}, mapReplacementNotFound(err)
	}
	if kind != "quote" || !isImmutable(status) {
		return Document{}, fmt.Errorf("%w: ce devis ne peut pas être remplacé", ErrInvalidInput)
	}
	var successorExists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM documents WHERE replaces_document_id=$1)`, quoteID).Scan(&successorExists); err != nil {
		return Document{}, err
	}
	if successorExists {
		return Document{}, fmt.Errorf("%w: ce devis a déjà été remplacé", ErrInvalidInput)
	}

	var id uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO documents (
			user_id, client_id, kind, status, title, issue_date, due_date, currency, vat_regime,
			notes, payment_terms, related_document_id, replaces_document_id, quote_version,
			subtotal_cents, tax_cents, total_cents
		)
		VALUES ($1,$2,'quote','draft',$3,$4::date,$5::date,$6,$7::vat_regime,
		        $8,$9,$10,$11,$12,$13,$14,$15)
		RETURNING id
	`, userID, src.ClientID, src.Title, src.IssueDate, src.DueDate, src.Currency, src.VATRegime,
		src.Notes, src.PaymentTerms, src.RelatedDocumentID, quoteID, version+1,
		calc.SubtotalCents, calc.TaxCents, calc.TotalCents,
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
