package biz

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
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
// purgée (opportunistement à chaque création, ou via PurgeExpiredIdempotency)
// et peut être réutilisée pour une nouvelle création.
const IdempotencyTTL = 24 * time.Hour

const idempotencyPurgeBatch = 100

// ErrIdempotencyConflict : la clé a déjà servi avec une requête différente.
var ErrIdempotencyConflict = errors.New("clé d'idempotence déjà utilisée avec une requête différente")

// Idempotency identifie une création rejouable. La portée complète est
// (compte, Operation, Key) ; RequestHash détecte le changement de requête.
type Idempotency struct {
	Key         string
	Operation   string
	RequestHash string
}

// NewIdempotency calcule le SHA-256 de la représentation JSON du payload
// (les structs et map[string]any sont sérialisés de façon déterministe).
func NewIdempotency(operation, key string, payload any) (Idempotency, error) {
	if operation == "" || key == "" {
		return Idempotency{}, fmt.Errorf("%w: clé d'idempotence invalide", ErrInvalidInput)
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return Idempotency{}, err
	}
	sum := sha256.Sum256(raw)
	return Idempotency{Key: key, Operation: operation, RequestHash: hex.EncodeToString(sum[:])}, nil
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

// PurgeExpiredIdempotency supprime les clés expirées de tous les comptes et
// renvoie le nombre de lignes supprimées.
func (s *Store) PurgeExpiredIdempotency(ctx context.Context) (int64, error) {
	tag, err := s.pool.Exec(ctx, `DELETE FROM document_idempotency_keys WHERE expires_at < NOW()`)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
