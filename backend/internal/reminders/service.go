// Package reminders sends and tracks invoice reminders.
package reminders

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusQueued  Status = "queued"
	StatusSent    Status = "sent"
	StatusFailed  Status = "failed"
	StatusBounced Status = "bounced"
)

func (s Status) Valid() bool {
	return s == StatusQueued || s == StatusSent || s == StatusFailed || s == StatusBounced
}

var (
	ErrNotFound = errors.New("reminder resource not found")
	ErrInvalid  = errors.New("invalid reminder request")
	ErrDelivery = errors.New("reminder delivery failed")
)

type Reminder struct {
	ID         uuid.UUID  `json:"id"`
	OwnerID    uuid.UUID  `json:"-"`
	InvoiceID  uuid.UUID  `json:"invoice_id"`
	TemplateID uuid.UUID  `json:"template_id"`
	Type       string     `json:"type"`
	Recipient  string     `json:"recipient"`
	Status     Status     `json:"status"`
	Error      *string    `json:"error,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	SentAt     *time.Time `json:"sent_at,omitempty"`
}
type Invoice struct {
	ID        uuid.UUID
	OwnerID   uuid.UUID
	Kind      string
	Recipient string
}
type Template struct {
	ID      uuid.UUID
	OwnerID uuid.UUID
	Subject string
	HTML    string
}
type Message struct {
	To      string
	Subject string
	HTML    string
}
type Store interface {
	Prepare(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string, string, time.Time) (Reminder, bool, error)
	MarkSent(context.Context, uuid.UUID, uuid.UUID, time.Time) (Reminder, error)
	MarkFailed(context.Context, uuid.UUID, uuid.UUID, string, time.Time) (Reminder, error)
	Invoice(context.Context, uuid.UUID, uuid.UUID) (Invoice, error)
	Template(context.Context, uuid.UUID, uuid.UUID) (Template, error)
	List(context.Context, uuid.UUID, uuid.UUID, int, int) ([]Reminder, int, error)
}
type Sender interface {
	Send(context.Context, uuid.UUID, Message) error
}
type SenderFunc func(context.Context, uuid.UUID, Message) error

func (f SenderFunc) Send(ctx context.Context, owner uuid.UUID, message Message) error {
	return f(ctx, owner, message)
}

type Service struct {
	store  Store
	sender Sender
	now    func() time.Time
}

func NewService(store Store, sender Sender) *Service {
	return &Service{store: store, sender: sender, now: time.Now}
}
func (s *Service) Send(ctx context.Context, owner, invoiceID, templateID uuid.UUID, key, kind string) (Reminder, error) {
	if owner == uuid.Nil || invoiceID == uuid.Nil || templateID == uuid.Nil || strings.TrimSpace(key) == "" || len(key) > 200 || s.store == nil || s.sender == nil {
		return Reminder{}, ErrInvalid
	}
	inv, err := s.store.Invoice(ctx, owner, invoiceID)
	if err != nil {
		return Reminder{}, err
	}
	if inv.Kind != "invoice" || strings.TrimSpace(inv.Recipient) == "" {
		return Reminder{}, ErrInvalid
	}
	tmpl, err := s.store.Template(ctx, owner, templateID)
	if err != nil {
		return Reminder{}, err
	}
	if kind == "" {
		kind = "manual"
	}
	now := s.now().UTC()
	if kind != "manual" && kind != "scheduled" && kind != "job" {
		return Reminder{}, ErrInvalid
	}
	r, created, err := s.store.Prepare(ctx, owner, invoiceID, templateID, key, kind, now)
	if err != nil {
		return Reminder{}, err
	}
	if !created {
		return r, nil
	}
	msg := Message{To: inv.Recipient, Subject: strings.ReplaceAll(tmpl.Subject, "{{invoice_id}}", invoiceID.String()), HTML: strings.ReplaceAll(tmpl.HTML, "{{invoice_id}}", invoiceID.String())}
	if err := s.sender.Send(ctx, owner, msg); err != nil {
		r, _ = s.store.MarkFailed(ctx, owner, r.ID, "delivery failed", s.now().UTC())
		return r, ErrDelivery
	}
	return s.store.MarkSent(ctx, owner, r.ID, s.now().UTC())
}
func (s *Service) List(ctx context.Context, owner, invoice uuid.UUID, page, size int) ([]Reminder, int, error) {
	page, size = BoundPagination(page, size)
	return s.store.List(ctx, owner, invoice, page, size)
}
func BoundPagination(page, size int) (int, int) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	return page, size
}
