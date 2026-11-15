package biz

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Opérations couvertes par l'idempotence de création. Chaque opération a son
// propre espace de clés, par compte.
const (
	OpCreateDocument = "create_document"
	OpConvertQuote   = "convert_quote_to_invoice"
	OpCreateDeposit  = "create_deposit_invoice"
	OpCreateCredit   = "create_credit_note"
)

// IdempotencyTTL est la durée de conservation d'une clé : au-delà, la clé est
// purgée (opportunistement à chaque création, et globalement par la tâche périodique
// RunIdempotencyPurgeLoop)
// et peut être réutilisée pour une nouvelle création.
const IdempotencyTTL = 24 * time.Hour

const idempotencyPurgeBatch = 100 // purge opportuniste du compte qui crée

// ErrIdempotencyConflict : la clé a déjà servi avec une requête différente.
var ErrIdempotencyConflict = errors.New("clé d'idempotence déjà utilisée avec une requête différente")

// ErrIdempotencyDocumentDeleted : la clé est encore valide (tombstone conservé
// jusqu'à expiration) mais le document qu'elle a créé a été supprimé depuis.
// Le rejeu ne recrée jamais le document : il faut une nouvelle clé.
var ErrIdempotencyDocumentDeleted = errors.New("le document créé avec cette clé d'idempotence a été supprimé : utilisez une nouvelle clé pour en créer un nouveau")

// Idempotency identifie une création rejouable. La portée complète est
// (compte, Operation, Key) ; RequestHash détecte le changement de requête.
type Idempotency struct {
	Key         string
	Operation   string
	RequestHash string
}

// canonicalLine ne porte que les entrées métier d'une ligne. Les champs
// dérivés ou sans effet (id, position, tax_cents, total_cents) sont exclus :
// ils sont recalculés ou réattribués à la création. L'ordre des champs de la
// struct fixe l'ordre de sérialisation ; l'ordre des lignes est significatif
// (la position est celle du tableau).
type canonicalLine struct {
	Description    string `json:"description"`
	QuantityMilli  int64  `json:"quantity_milli"`
	UnitPriceCents int64  `json:"unit_price_cents"`
	DiscountBps    int    `json:"discount_bps"`
	TaxRateBps     int    `json:"tax_rate_bps"`
}

func canonicalLines(lines []DocLine) []canonicalLine {
	out := make([]canonicalLine, 0, len(lines))
	for _, l := range lines {
		out = append(out, canonicalLine{l.Description, l.QuantityMilli, l.UnitPriceCents, l.DiscountBps, l.TaxRateBps})
	}
	return out
}

// canonicalDocument est la requête de création APRÈS application des défauts
// métier (devise par défaut) : deux requêtes qui produisent le même document
// ont la même représentation.
type canonicalDocument struct {
	ClientID          *uuid.UUID      `json:"client_id"`
	Kind              string          `json:"kind"`
	Title             string          `json:"title"`
	IssueDate         *string         `json:"issue_date"`
	DueDate           *string         `json:"due_date"`
	Currency          string          `json:"currency"`
	VATRegime         string          `json:"vat_regime"`
	Notes             string          `json:"notes"`
	PaymentTerms      string          `json:"payment_terms"`
	RelatedDocumentID *uuid.UUID      `json:"related_document_id"`
	Lines             []canonicalLine `json:"lines"`
}

type canonicalLinked struct {
	OriginID uuid.UUID       `json:"origin_id"`
	Lines    []canonicalLine `json:"lines"`
}

type canonicalConvert struct {
	QuoteID uuid.UUID `json:"quote_id"`
}

func newIdempotency(operation, key string, canonical any) (Idempotency, error) {
	if operation == "" || key == "" {
		return Idempotency{}, fmt.Errorf("%w: clé d'idempotence invalide", ErrInvalidInput)
	}
	raw, err := json.Marshal(canonical)
	if err != nil {
		return Idempotency{}, err
	}
	sum := sha256.Sum256(raw)
	return Idempotency{Key: key, Operation: operation, RequestHash: hex.EncodeToString(sum[:])}, nil
}

// NewDocumentIdempotency hache la requête de création canonique (défauts
// métier appliqués, sans champ dérivé) : SHA-256 d'un JSON à champs ordonnés.
func NewDocumentIdempotency(operation, key string, in DocumentInput) (Idempotency, error) {
	return newIdempotency(operation, key, canonicalDocument{
		ClientID: in.ClientID, Kind: in.Kind, Title: in.Title, IssueDate: in.IssueDate, DueDate: in.DueDate,
		Currency: defaultCurrency(in.Currency), VATRegime: in.VATRegime, Notes: in.Notes, PaymentTerms: in.PaymentTerms,
		RelatedDocumentID: in.RelatedDocumentID, Lines: canonicalLines(in.Lines),
	})
}

// NewConvertIdempotency : la conversion devis -> facture ne dépend que du devis.
func NewConvertIdempotency(key string, quoteID uuid.UUID) (Idempotency, error) {
	return newIdempotency(OpConvertQuote, key, canonicalConvert{QuoteID: quoteID})
}

// NewLinkedIdempotency : acompte ou avoir. Les lignes fournies (vides = copie
// des lignes du document d'origine) et l'origine déterminent le résultat ;
// le type est porté par l'opération.
func NewLinkedIdempotency(operation, key string, originID uuid.UUID, lines []DocLine) (Idempotency, error) {
	return newIdempotency(operation, key, canonicalLinked{OriginID: originID, Lines: canonicalLines(lines)})
}

// CreateDocumentIdempotent crée un document une seule fois par (compte,
// opération, clé). Le booléen vaut true quand le document existait déjà.
func (s *Store) CreateDocumentIdempotent(ctx context.Context, userID uuid.UUID, in DocumentInput, idem Idempotency) (Document, bool, error) {
	return s.createDocument(ctx, userID, in, &idem)
}

// reserveIdempotencyKey insère la clé dans la transaction de création. Une
// requête concurrente portant la même clé attend ici la fin de la transaction
// gagnante (index unique) puis voit la ligne validée. Renvoie found=true avec
// l'identifiant du document déjà créé en cas de rejeu.
func reserveIdempotencyKey(ctx context.Context, tx pgx.Tx, userID uuid.UUID, idem Idempotency) (uuid.UUID, bool, error) {
	// Purge opportuniste (par lot) des clés expirées du compte.
	if _, err := tx.Exec(ctx, `
		WITH doomed AS (
			SELECT user_id, operation, idempotency_key FROM document_idempotency_keys
			WHERE expires_at < NOW() AND user_id = $1
			LIMIT $2 FOR UPDATE SKIP LOCKED
		)
		DELETE FROM document_idempotency_keys k USING doomed d
		WHERE k.user_id = d.user_id AND k.operation = d.operation AND k.idempotency_key = d.idempotency_key
	`, userID, idempotencyPurgeBatch); err != nil {
		return uuid.Nil, false, err
	}
	// Cette clé précise est expirée : elle est libérée (même si le lot ci-dessus ne l'a pas atteinte).
	if _, err := tx.Exec(ctx, `
		DELETE FROM document_idempotency_keys
		WHERE user_id = $1 AND operation = $2 AND idempotency_key = $3 AND expires_at < NOW()
	`, userID, idem.Operation, idem.Key); err != nil {
		return uuid.Nil, false, err
	}
	tag, err := tx.Exec(ctx, `
		INSERT INTO document_idempotency_keys (user_id, operation, idempotency_key, request_hash, expires_at)
		VALUES ($1, $2, $3, $4, NOW() + make_interval(secs => $5))
		ON CONFLICT (user_id, operation, idempotency_key) DO NOTHING
	`, userID, idem.Operation, idem.Key, idem.RequestHash, IdempotencyTTL.Seconds())
	if err != nil {
		return uuid.Nil, false, err
	}
	if tag.RowsAffected() == 1 {
		return uuid.Nil, false, nil
	}
	var hash string
	var docID *uuid.UUID
	if err := tx.QueryRow(ctx, `
		SELECT request_hash, document_id FROM document_idempotency_keys
		WHERE user_id = $1 AND operation = $2 AND idempotency_key = $3
	`, userID, idem.Operation, idem.Key).Scan(&hash, &docID); err != nil {
		return uuid.Nil, false, err
	}
	if hash != idem.RequestHash {
		return uuid.Nil, false, ErrIdempotencyConflict
	}
	if docID == nil {
		return uuid.Nil, false, errors.New("idempotence: création précédente sans document")
	}
	return *docID, true, nil
}

func attachIdempotencyDocument(ctx context.Context, tx pgx.Tx, userID uuid.UUID, idem Idempotency, docID uuid.UUID) error {
	tag, err := tx.Exec(ctx, `
		UPDATE document_idempotency_keys SET document_id = $4
		WHERE user_id = $1 AND operation = $2 AND idempotency_key = $3
	`, userID, idem.Operation, idem.Key, docID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errors.New("idempotence: clé introuvable à la finalisation")
	}
	return nil
}

const (
	// Purge globale : lots de 1000 lignes, au plus 50 lots (50 000 lignes) par
	// appel ; le reliquat est traité au passage suivant.
	idempotencyGlobalPurgeBatch      = 1000
	idempotencyGlobalPurgeMaxBatches = 50
)

// PurgeExpiredIdempotency supprime les clés expirées de TOUS les comptes, par
// lots bornés, et renvoie le nombre de lignes supprimées.
func (s *Store) PurgeExpiredIdempotency(ctx context.Context) (int64, error) {
	return s.purgeExpiredIdempotency(ctx, idempotencyGlobalPurgeBatch, idempotencyGlobalPurgeMaxBatches)
}

func (s *Store) purgeExpiredIdempotency(ctx context.Context, batch, maxBatches int) (int64, error) {
	var total int64
	for i := 0; i < maxBatches; i++ {
		tag, err := s.pool.Exec(ctx, `
			WITH doomed AS (
				SELECT user_id, operation, idempotency_key FROM document_idempotency_keys
				WHERE expires_at < NOW()
				ORDER BY expires_at
				LIMIT $1 FOR UPDATE SKIP LOCKED
			)
			DELETE FROM document_idempotency_keys k USING doomed d
			WHERE k.user_id = d.user_id AND k.operation = d.operation AND k.idempotency_key = d.idempotency_key
		`, batch)
		if err != nil {
			return total, err
		}
		total += tag.RowsAffected()
		if tag.RowsAffected() < int64(batch) {
			break
		}
	}
	return total, nil
}

// IdempotencyPurger est la partie du Store utilisée par la tâche périodique.
type IdempotencyPurger interface {
	PurgeExpiredIdempotency(ctx context.Context) (int64, error)
}

// RunIdempotencyPurgeLoop purge une fois au démarrage puis à chaque tick, et
// s'arrête quand ctx est annulé. Les ticks sont injectables pour les tests ;
// en production, il s'agit de time.NewTicker(...).C. Une erreur est journalisée
// et n'interrompt pas la boucle.
func RunIdempotencyPurgeLoop(ctx context.Context, p IdempotencyPurger, ticks <-chan time.Time, logger *slog.Logger) {
	if logger == nil {
		logger = slog.Default()
	}
	run := func() {
		n, err := p.PurgeExpiredIdempotency(ctx)
		switch {
		case err != nil && ctx.Err() == nil:
			logger.Error("purge des clés d'idempotence", "error", err, "purged", n)
		case n > 0:
			logger.Info("clés d'idempotence expirées purgées", "purged", n)
		}
	}
	run()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticks:
			if ctx.Err() != nil {
				return
			}
			run()
		}
	}
}
