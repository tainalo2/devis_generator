package emailtemplates

import (
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Kind identifies which reminder a template targets.
type Kind string

const (
	QuoteUnsigned  Kind = "quote_unsigned"
	InvoiceUnpaid  Kind = "invoice_unpaid"
)

// Vars are substituted into templates.
type Vars struct {
	ClientName   string
	DocRef       string
	Amount       string
	Link         string
	DueDate      string
}

// Template is a stored email template.
type Template struct {
	ID       string `json:"-"`
	OwnerID  string `json:"-"`
	Kind     Kind   `json:"kind"`
	Subject  string `json:"subject"`
	Body     string `json:"body"`
}

// Default templates in French.
func defaultSubject(k Kind) string {
	switch k {
	case QuoteUnsigned:
		return "Relance : votre devis {{.DocRef}} est en attente de signature"
	case InvoiceUnpaid:
		return "Relance : facture {{.DocRef}} impayée"
	}
	return "Relance"
}

func defaultBody(k Kind) string {
	switch k {
	case QuoteUnsigned:
		return "Bonjour {{.ClientName}},\n\nNous vous relançons concernant le devis {{.DocRef}} d'un montant de {{.Amount}}.\nVous pouvez le consulter et le signer ici : {{.Link}}\n\nCordialement."
	case InvoiceUnpaid:
		return "Bonjour {{.ClientName}},\n\nNous vous relançons concernant la facture {{.DocRef}} d'un montant de {{.Amount}}, échue le {{.DueDate}}.\nVous pouvez la consulter ici : {{.Link}}\n\nCordialement."
	}
	return ""
}

// Store persists templates per owner.
type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

func (s *Store) Get(ctx context.Context, ownerID string, kind Kind) (Template, error) {
	var t Template
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, owner_id::text, kind, subject, body
		FROM email_templates WHERE owner_id=$1 AND kind=$2
	`, ownerID, kind).Scan(&t.ID, &t.OwnerID, &t.Kind, &t.Subject, &t.Body)
	if err != nil {
		// Return default if not found
		return Template{Kind: kind, Subject: defaultSubject(kind), Body: defaultBody(kind)}, nil
	}
	return t, nil
}

func (s *Store) Upsert(ctx context.Context, ownerID string, t Template) (Template, error) {
	var out Template
	err := s.pool.QueryRow(ctx, `
		INSERT INTO email_templates (owner_id, kind, subject, body, updated_at)
		VALUES ($1,$2,$3,$4,NOW())
		ON CONFLICT (owner_id, kind) DO UPDATE SET subject=EXCLUDED.subject, body=EXCLUDED.body, updated_at=NOW()
		RETURNING id::text, owner_id::text, kind, subject, body
	`, ownerID, t.Kind, t.Subject, t.Body).Scan(&out.ID, &out.OwnerID, &out.Kind, &out.Subject, &out.Body)
	if err != nil {
		return Template{}, fmt.Errorf("upsert template: %w", err)
	}
	return out, nil
}

func (s *Store) List(ctx context.Context, ownerID string) ([]Template, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, owner_id::text, kind, subject, body
		FROM email_templates WHERE owner_id=$1 ORDER BY kind
	`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Template
	for rows.Next() {
		var t Template
		if err := rows.Scan(&t.ID, &t.OwnerID, &t.Kind, &t.Subject, &t.Body); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	// Ensure defaults present for both kinds even if not stored
	have := map[Kind]bool{}
	for _, t := range out {
		have[t.Kind] = true
	}
	for _, k := range []Kind{QuoteUnsigned, InvoiceUnpaid} {
		if !have[k] {
			out = append(out, Template{Kind: k, Subject: defaultSubject(k), Body: defaultBody(k)})
		}
	}
	return out, nil
}

// Render applies variables to a template's subject and body.
func Render(t Template, v Vars) (subject, body string, err error) {
	st, err := template.New("subject").Parse(t.Subject)
	if err != nil {
		return "", "", err
	}
	var sb strings.Builder
	if err := st.Execute(&sb, v); err != nil {
		return "", "", err
	}
	subject = sb.String()
	bt, err := template.New("body").Parse(t.Body)
	if err != nil {
		return "", "", err
	}
	var bb strings.Builder
	if err := bt.Execute(&bb, v); err != nil {
		return "", "", err
	}
	body = bb.String()
	return subject, body, nil
}
