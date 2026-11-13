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
	StatusSending Status = "sending"
	StatusSent    Status = "sent"
	StatusFailed  Status = "failed"
	StatusBounced Status = "bounced"
	StatusSkipped Status = "skipped"
)

func (s Status) Valid() bool {
	return s == StatusQueued || s == StatusSending || s == StatusSent || s == StatusFailed || s == StatusBounced || s == StatusSkipped
}

var (
	ErrNotFound        = errors.New("reminder resource not found")
	ErrInvalid         = errors.New("invalid reminder request")
	ErrDelivery        = errors.New("reminder delivery failed")
	ErrRateLimited     = errors.New("hourly reminder limit reached")
	ErrDocumentChanged = errors.New("document changed since reminder was prepared")
	ErrDeliveryMode    = errors.New("email delivery mode is not configured")
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
	LeaseUntil *time.Time `json:"lease_until,omitempty"`
	Attempts   int        `json:"attempt_count"`
}
type Invoice struct {
	ID        uuid.UUID
	OwnerID   uuid.UUID
	Kind      string
	Status    string
	Recipient string
	UpdatedAt time.Time
}
type Template struct {
	ID        uuid.UUID
	OwnerID   uuid.UUID
	Subject   string
	HTML      string
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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
	MarkSkipped(context.Context, uuid.UUID, uuid.UUID, string, time.Time) (Reminder, error)
	LockForDelivery(context.Context, uuid.UUID, uuid.UUID, time.Time, string) (func(), error)
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

type DeliveryMode string

const (
	DeliveryModeGmail     DeliveryMode = "gmail"
	DeliveryModeMicrosoft DeliveryMode = "microsoft"
	DeliveryModeSMTP      DeliveryMode = "smtp"
)

type ModeStore interface {
	Mode(context.Context, uuid.UUID) (DeliveryMode, error)
}

type ProviderRouter struct {
	modes   ModeStore
	senders map[DeliveryMode]Sender
}

func NewProviderRouter(modes ModeStore, senders map[DeliveryMode]Sender) *ProviderRouter {
	return &ProviderRouter{modes: modes, senders: senders}
}

func (r *ProviderRouter) Send(ctx context.Context, owner uuid.UUID, message Message) error {
	if r == nil || r.modes == nil {
		return ErrDeliveryMode
	}
	mode, err := r.modes.Mode(ctx, owner)
	if err != nil {
		return err
	}
	sender := r.senders[mode]
	if sender == nil {
		return ErrDeliveryMode
	}
	return sender.Send(ctx, owner, message)
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
	if (inv.Kind != "invoice" && inv.Kind != "quote") || inv.Status != "sent" || strings.TrimSpace(inv.Recipient) == "" {
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
	release, err := s.store.LockForDelivery(ctx, owner, invoiceID, inv.UpdatedAt, inv.Recipient)
	if err != nil {
		r, _ = s.store.MarkSkipped(ctx, owner, r.ID, "document changed", s.now().UTC())
		return r, err
	}
	if err := s.sender.Send(ctx, owner, msg); err != nil {
		release()
		r, _ = s.store.MarkFailed(ctx, owner, r.ID, "delivery failed", s.now().UTC())
		return r, ErrDelivery
	}
	release()
	return s.store.MarkSent(ctx, owner, r.ID, s.now().UTC())
}
func (s *Service) List(ctx context.Context, owner, invoice uuid.UUID, page, size int) ([]Reminder, int, error) {
	page, size = BoundPagination(page, size)
	return s.store.List(ctx, owner, invoice, page, size)
}
func (s *Service) Mode(ctx context.Context, owner uuid.UUID) (DeliveryMode, error) {
	store, ok := s.store.(ModeStore)
	if !ok {
		return "", ErrDeliveryMode
	}
	return store.Mode(ctx, owner)
}
func (s *Service) SetMode(ctx context.Context, owner uuid.UUID, mode DeliveryMode) error {
	store, ok := s.store.(interface {
		SetMode(context.Context, uuid.UUID, DeliveryMode) error
	})
	if !ok {
		return ErrDeliveryMode
	}
	return store.SetMode(ctx, owner, mode)
}

type TemplateStore interface {
	CreateTemplate(context.Context, uuid.UUID, Template) (Template, error)
	ListTemplates(context.Context, uuid.UUID) ([]Template, error)
	UpdateTemplate(context.Context, uuid.UUID, Template) (Template, error)
	DeleteTemplate(context.Context, uuid.UUID, uuid.UUID) error
}

func validTemplate(t Template) bool {
	return strings.TrimSpace(t.Name) != "" && len(t.Name) <= 100 && strings.TrimSpace(t.Subject) != "" && len(t.Subject) <= 200 && strings.TrimSpace(t.HTML) != "" && len(t.HTML) <= 100000
}
func (s *Service) CreateTemplate(ctx context.Context, owner uuid.UUID, t Template) (Template, error) {
	if owner == uuid.Nil || !validTemplate(t) {
		return Template{}, ErrInvalid
	}
	return s.store.(TemplateStore).CreateTemplate(ctx, owner, t)
}
func (s *Service) ListTemplates(ctx context.Context, owner uuid.UUID) ([]Template, error) {
	if owner == uuid.Nil {
		return nil, ErrInvalid
	}
	return s.store.(TemplateStore).ListTemplates(ctx, owner)
}
func (s *Service) UpdateTemplate(ctx context.Context, owner uuid.UUID, t Template) (Template, error) {
	if owner == uuid.Nil || t.ID == uuid.Nil || !validTemplate(t) {
		return Template{}, ErrInvalid
	}
	return s.store.(TemplateStore).UpdateTemplate(ctx, owner, t)
}
func (s *Service) DeleteTemplate(ctx context.Context, owner, id uuid.UUID) error {
	if owner == uuid.Nil || id == uuid.Nil {
		return ErrInvalid
	}
	return s.store.(TemplateStore).DeleteTemplate(ctx, owner, id)
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
