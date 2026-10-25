package biz

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/tainalo2/devis_generator/backend/internal/legal"
)

type Address struct {
	Line1      string `json:"line1"`
	Line2      string `json:"line2"`
	PostalCode string `json:"postal_code"`
	City       string `json:"city"`
	Country    string `json:"country"`
}

type Client struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Address   Address   `json:"address"`
	SIREN     string    `json:"siren"`
	SIRET     string    `json:"siret"`
	VATNumber string    `json:"vat_number"`
	IBAN      string    `json:"iban,omitempty"` // decrypted for owner only
	Notes     string    `json:"notes"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type IssuerProfile struct {
	LegalName        string    `json:"legal_name"`
	TradeName        string    `json:"trade_name"`
	Email            string    `json:"email"`
	Phone            string    `json:"phone"`
	SIRET            string    `json:"siret"`
	SIREN            string    `json:"siren"`
	VATNumber        string    `json:"vat_number"`
	APECode          string    `json:"ape_code"`
	RCS              string    `json:"rcs"`
	Address          Address   `json:"address"`
	IBAN             string    `json:"iban,omitempty"`
	DefaultVATRegime string    `json:"default_vat_regime"`
	PaymentTerms     string    `json:"payment_terms"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type DocLine struct {
	ID             uuid.UUID `json:"id,omitempty"`
	Position       int       `json:"position"`
	Description    string    `json:"description"`
	QuantityMilli  int64     `json:"quantity_milli"`
	UnitPriceCents int64     `json:"unit_price_cents"`
	DiscountBps    int       `json:"discount_bps"`
	TaxRateBps     int       `json:"tax_rate_bps"`
	TaxCents       int64     `json:"tax_cents"`
	TotalCents     int64     `json:"total_cents"` // HT line
}

type Document struct {
	ID                uuid.UUID       `json:"id"`
	ClientID          *uuid.UUID      `json:"client_id"`
	Kind              string          `json:"kind"`
	Status            string          `json:"status"`
	Number            *string         `json:"number"`
	Title             string          `json:"title"`
	IssueDate         *string         `json:"issue_date"` // YYYY-MM-DD
	DueDate           *string         `json:"due_date"`
	Currency          string          `json:"currency"`
	VATRegime         string          `json:"vat_regime"`
	Notes             string          `json:"notes"`
	PaymentTerms      string          `json:"payment_terms"`
	LegalMentions     string          `json:"legal_mentions"`
	RelatedDocumentID *uuid.UUID      `json:"related_document_id"`
	SubtotalCents     int64           `json:"subtotal_cents"`
	TaxCents          int64           `json:"tax_cents"`
	TotalCents        int64           `json:"total_cents"`
	IssuedAt          *time.Time      `json:"issued_at"`
	IssuerSnapshot    json.RawMessage `json:"issuer_snapshot,omitempty"`
	LegalSnapshot     json.RawMessage `json:"legal_snapshot,omitempty"`
	Lines             []DocLine       `json:"lines"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

// GuestParty is a client/issuer identity for ephemeral guest PDF (no UUID required).
type GuestParty struct {
	Name      string  `json:"name"`
	Email     string  `json:"email"`
	Address   Address `json:"address"`
	SIREN     string  `json:"siren"`
	SIRET     string  `json:"siret"`
	VATNumber string  `json:"vat_number"`
	Notes     string  `json:"notes"`
}

// GuestPDFRequest is the payload for ephemeral guest PDF generation.
type GuestPDFRequest struct {
	Kind         string          `json:"kind"`
	VATRegime    string          `json:"vat_regime"`
	Title        string          `json:"title"`
	IssueDate    string          `json:"issue_date"`
	DueDate      string          `json:"due_date"`
	Currency     string          `json:"currency"`
	Notes        string          `json:"notes"`
	PaymentTerms string          `json:"payment_terms"`
	Issuer       IssuerProfile   `json:"issuer"`
	Client       GuestParty      `json:"client"`
	Lines        []DocLine       `json:"lines"`
	Number       string          `json:"number"`
	Mentions     *legal.Mentions `json:"mentions,omitempty"`
}
