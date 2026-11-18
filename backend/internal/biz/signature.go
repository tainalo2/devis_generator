package biz

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	// ErrAlreadySigned : le document porte déjà une signature (une seule, définitive).
	ErrAlreadySigned = errors.New("document already signed")
	// ErrNotSignable : seul un devis émis ou envoyé, non remplacé, peut être signé.
	ErrNotSignable = errors.New("document is not signable")
)

const (
	signerNameMin = 2
	signerNameMax = 200
	maxSignerIP   = 64
	maxSignerUA   = 512
)

// SignInput : ce que le signataire déclare (nom) et ce que le serveur capte (IP, user-agent).
type SignInput struct {
	Name      string
	IP        string
	UserAgent string
}

// Signature est la preuve d'une signature électronique simple (non qualifiée).
type Signature struct {
	ID              uuid.UUID `json:"-"`
	DocumentID      uuid.UUID `json:"document_id"`
	SignedAt        time.Time `json:"signed_at"`
	DocumentHash    string    `json:"document_hash"`
	SignerName      string    `json:"signer_name"`
	SignerIP        string    `json:"signer_ip"`
	SignerUserAgent string    `json:"signer_user_agent"`
}

// SignatureProof est la preuve consultable par l'émetteur, avec le contrôle d'intégrité :
// le hash est recalculé sur le contenu actuel du document et comparé à celui figé à la signature.
type SignatureProof struct {
	Signature
	IntegrityOK bool `json:"integrity_ok"`
}

type signedLine struct {
	Position       int    `json:"position"`
	Description    string `json:"description"`
	QuantityMilli  int64  `json:"quantity_milli"`
	UnitPriceCents int64  `json:"unit_price_cents"`
	DiscountBps    int    `json:"discount_bps"`
	TaxRateBps     int    `json:"tax_rate_bps"`
	TaxCents       int64  `json:"tax_cents"`
	TotalCents     int64  `json:"total_cents"`
}

type signedContent struct {
	ID             uuid.UUID       `json:"id"`
	Kind           string          `json:"kind"`
	Number         *string         `json:"number"`
	Title          string          `json:"title"`
	IssueDate      *string         `json:"issue_date"`
	DueDate        *string         `json:"due_date"`
	Currency       string          `json:"currency"`
	VATRegime      string          `json:"vat_regime"`
	Notes          string          `json:"notes"`
	PaymentTerms   string          `json:"payment_terms"`
	LegalMentions  string          `json:"legal_mentions"`
	SubtotalCents  int64           `json:"subtotal_cents"`
	TaxCents       int64           `json:"tax_cents"`
	TotalCents     int64           `json:"total_cents"`
	IssuedAt       *time.Time      `json:"issued_at"`
	IssuerSnapshot json.RawMessage `json:"issuer_snapshot"`
	ClientSnapshot *ClientSnapshot `json:"client_snapshot"`
	LegalSnapshot  json.RawMessage `json:"legal_snapshot"`
	Lines          []signedLine    `json:"lines"`
}

// ContentHash retourne le SHA-256 (hex) du contenu figé du document : identité, montants,
// lignes et instantanés émetteur/client/mentions légales. Le statut, les dates de suivi et
// les identifiants de ligne n'en font pas partie : seul le contenu qui engage les parties.
func ContentHash(d Document) (string, error) {
	c := signedContent{
		ID: d.ID, Kind: d.Kind, Number: d.Number, Title: d.Title, IssueDate: d.IssueDate, DueDate: d.DueDate,
		Currency: d.Currency, VATRegime: d.VATRegime, Notes: d.Notes, PaymentTerms: d.PaymentTerms,
		LegalMentions: d.LegalMentions, SubtotalCents: d.SubtotalCents, TaxCents: d.TaxCents, TotalCents: d.TotalCents,
		IssuedAt: d.IssuedAt, IssuerSnapshot: d.IssuerSnapshot, ClientSnapshot: d.ClientSnapshot, LegalSnapshot: d.LegalSnapshot,
		Lines: make([]signedLine, 0, len(d.Lines)),
	}
	if c.IssuedAt != nil {
		u := c.IssuedAt.UTC().Truncate(time.Microsecond)
		c.IssuedAt = &u
	}
	for _, l := range d.Lines {
		c.Lines = append(c.Lines, signedLine{l.Position, l.Description, l.QuantityMilli, l.UnitPriceCents, l.DiscountBps, l.TaxRateBps, l.TaxCents, l.TotalCents})
	}
	raw, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func truncateRunes(s string, max int) string {
	s = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, s)
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	return string([]rune(s)[:max])
}

// SignQuote enregistre la signature d'un devis émis ou envoyé et le passe en « accepté ».
// Tout se joue dans une transaction qui verrouille la ligne du document : deux signatures
// concurrentes ne peuvent pas réussir toutes les deux (la seconde reçoit ErrAlreadySigned).
// Le hash porte sur le contenu figé, relu côté serveur ; le client n'en fournit aucun.
func (s *Store) SignQuote(ctx context.Context, docID uuid.UUID, in SignInput) (Signature, error) {
	name := strings.TrimSpace(in.Name)
	if n := utf8.RuneCountInString(name); n < signerNameMin || n > signerNameMax {
		return Signature{}, fmt.Errorf("%w: nom du signataire invalide", ErrInvalidInput)
	}
	// Le contenu est lu AVANT d'ouvrir la transaction : lire pendant qu'on tient une connexion
	// épuiserait le pool sous charge (chaque signature concurrente attendrait une seconde connexion).
	// Le contenu d'un document émis est immuable ; on revérifie sous verrou qu'il s'agit bien
	// de la même émission (issued_at) pour qu'une modification entre-temps ne passe pas.
	doc, _, err := s.GetDocumentPublic(ctx, docID)
	if err != nil {
		return Signature{}, err
	}
	hash, err := ContentHash(doc)
	if err != nil {
		return Signature{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Signature{}, err
	}
	defer tx.Rollback(ctx)

	var kind, status string
	var issuedAt *time.Time
	err = tx.QueryRow(ctx, `SELECT kind::text, status::text, issued_at FROM documents WHERE id = $1 FOR UPDATE`, docID).Scan(&kind, &status, &issuedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Signature{}, ErrNotFound
	}
	if err != nil {
		return Signature{}, err
	}
	var signed, replaced bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM document_signatures WHERE document_id = $1),
	                                   EXISTS(SELECT 1 FROM documents WHERE replaces_document_id = $1)`, docID).Scan(&signed, &replaced); err != nil {
		return Signature{}, err
	}
	if signed {
		return Signature{}, ErrAlreadySigned
	}
	if kind != "quote" || (status != "issued" && status != "sent") || replaced {
		return Signature{}, ErrNotSignable
	}
	if issuedAt == nil || doc.IssuedAt == nil || !issuedAt.Equal(*doc.IssuedAt) {
		return Signature{}, ErrNotSignable
	}
	sig := Signature{
		DocumentID: docID, DocumentHash: hash, SignerName: name,
		SignerIP: truncateRunes(in.IP, maxSignerIP), SignerUserAgent: truncateRunes(in.UserAgent, maxSignerUA),
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO document_signatures (document_id, document_hash, signer_name, signer_ip, signer_user_agent)
		VALUES ($1, $2, $3, $4, $5) RETURNING id, signed_at`,
		docID, hash, sig.SignerName, sig.SignerIP, sig.SignerUserAgent).Scan(&sig.ID, &sig.SignedAt); err != nil {
		return Signature{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE documents SET status = 'accepted', accepted_at = $2, updated_at = NOW() WHERE id = $1`, docID, sig.SignedAt); err != nil {
		return Signature{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Signature{}, err
	}
	return sig, nil
}

// GetSignaturePublic retourne la signature d'un document ou ErrNotFound s'il n'est pas signé.
func (s *Store) GetSignaturePublic(ctx context.Context, docID uuid.UUID) (Signature, error) {
	var sig Signature
	err := s.pool.QueryRow(ctx, `
		SELECT id, document_id, signed_at, document_hash, signer_name, signer_ip, signer_user_agent
		FROM document_signatures WHERE document_id = $1`, docID).
		Scan(&sig.ID, &sig.DocumentID, &sig.SignedAt, &sig.DocumentHash, &sig.SignerName, &sig.SignerIP, &sig.SignerUserAgent)
	if errors.Is(err, pgx.ErrNoRows) {
		return Signature{}, ErrNotFound
	}
	return sig, err
}

// GetSignatureProof retourne la preuve au propriétaire du document, avec contrôle d'intégrité.
func (s *Store) GetSignatureProof(ctx context.Context, userID, docID uuid.UUID) (SignatureProof, error) {
	doc, err := s.GetDocument(ctx, userID, docID)
	if err != nil {
		return SignatureProof{}, err
	}
	sig, err := s.GetSignaturePublic(ctx, docID)
	if err != nil {
		return SignatureProof{}, err
	}
	hash, err := ContentHash(doc)
	if err != nil {
		return SignatureProof{}, err
	}
	return SignatureProof{Signature: sig, IntegrityOK: hash == sig.DocumentHash}, nil
}
