package reminders

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

type memoryStore struct {
	mu            sync.Mutex
	items         map[string]Reminder
	template      Template
	invoice       Invoice
	prepareErr    error
	revalidateErr error
	revalidations int
}

func (m *memoryStore) Prepare(_ context.Context, owner, invoice, template uuid.UUID, key, kind string, now time.Time) (Reminder, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.prepareErr != nil {
		return Reminder{}, false, m.prepareErr
	}
	scope := owner.String() + "/" + invoice.String() + "/" + template.String() + "/" + kind + "/" + key
	if x, ok := m.items[scope]; ok {
		return x, false, nil
	}
	r := Reminder{ID: uuid.New(), OwnerID: owner, InvoiceID: invoice, TemplateID: template, Type: kind, Recipient: m.invoice.Recipient, Status: StatusQueued, CreatedAt: now}
	m.items[scope] = r
	return r, true, nil
}
func (m *memoryStore) LockForDelivery(_ context.Context, owner, invoice uuid.UUID, expectedUpdatedAt time.Time, expectedRecipient string) (func(), error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.revalidations++
	if m.revalidateErr != nil {
		return nil, m.revalidateErr
	}
	return func() {}, nil
}
func (m *memoryStore) MarkSent(_ context.Context, owner, id uuid.UUID, now time.Time) (Reminder, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, r := range m.items {
		if r.ID == id && r.OwnerID == owner {
			r.Status = StatusSent
			r.SentAt = &now
			m.items[k] = r
			return r, nil
		}
	}
	return Reminder{}, ErrNotFound
}
func (m *memoryStore) MarkFailed(_ context.Context, owner, id uuid.UUID, msg string, now time.Time) (Reminder, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, r := range m.items {
		if r.ID == id && r.OwnerID == owner {
			r.Status = StatusFailed
			r.Error = &msg
			m.items[k] = r
			return r, nil
		}
	}
	return Reminder{}, ErrNotFound
}
func (m *memoryStore) MarkSkipped(_ context.Context, owner, id uuid.UUID, msg string, now time.Time) (Reminder, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, r := range m.items {
		if r.ID == id && r.OwnerID == owner {
			r.Status = StatusSkipped
			r.Error = &msg
			m.items[k] = r
			return r, nil
		}
	}
	return Reminder{}, ErrNotFound
}
func (m *memoryStore) Invoice(_ context.Context, owner, id uuid.UUID) (Invoice, error) {
	if m.invoice.OwnerID != owner || m.invoice.ID != id {
		return Invoice{}, ErrNotFound
	}
	return m.invoice, nil
}
func (m *memoryStore) Template(_ context.Context, owner, id uuid.UUID) (Template, error) {
	if m.template.OwnerID != owner || m.template.ID != id {
		return Template{}, ErrNotFound
	}
	return m.template, nil
}
func (m *memoryStore) List(context.Context, uuid.UUID, uuid.UUID, int, int) ([]Reminder, int, error) {
	return nil, 0, nil
}

type fakeSender struct {
	mu    sync.Mutex
	calls int
	err   error
}

func (f *fakeSender) Send(context.Context, uuid.UUID, Message) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	return f.err
}
func TestSendManualIsIdempotent(t *testing.T) {
	owner, inv, tmpl := uuid.New(), uuid.New(), uuid.New()
	st := &memoryStore{items: map[string]Reminder{}, invoice: Invoice{ID: inv, OwnerID: owner, Kind: "invoice", Status: "sent", Recipient: "client@example.com", UpdatedAt: time.Now()}, template: Template{ID: tmpl, OwnerID: owner, Subject: "Invoice {{invoice_id}}", HTML: "Hello"}}
	sender := &fakeSender{}
	svc := NewService(st, sender)
	first, err := svc.Send(context.Background(), owner, inv, tmpl, "same-key", "manual")
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.Send(context.Background(), owner, inv, tmpl, "same-key", "manual")
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID || sender.calls != 1 {
		t.Fatalf("ids %s/%s calls=%d", first.ID, second.ID, sender.calls)
	}
	if first.Status != StatusSent {
		t.Fatalf("status=%s", first.Status)
	}
}
func TestSendFailureIsRecordedWithoutSecret(t *testing.T) {
	owner, inv, tmpl := uuid.New(), uuid.New(), uuid.New()
	st := &memoryStore{items: map[string]Reminder{}, invoice: Invoice{ID: inv, OwnerID: owner, Kind: "invoice", Status: "sent", Recipient: "x@y.test", UpdatedAt: time.Now()}, template: Template{ID: tmpl, OwnerID: owner, Subject: "s", HTML: "h"}}
	sender := &fakeSender{err: errors.New("provider token=SECRET")}
	r, err := NewService(st, sender).Send(context.Background(), owner, inv, tmpl, "k", "job")
	if err == nil || r.Status != StatusFailed || r.Error == nil || *r.Error != "delivery failed" {
		t.Fatalf("r=%+v err=%v", r, err)
	}
}
func TestPaginationBounds(t *testing.T) {
	if page, size := BoundPagination(-1, 1000); page != 1 || size != 100 {
		t.Fatalf("%d %d", page, size)
	}
	if page, size := BoundPagination(2, 0); page != 2 || size != 20 {
		t.Fatalf("%d %d", page, size)
	}
}
func TestStatusesIncludeBounced(t *testing.T) {
	if !StatusBounced.Valid() || Status("unknown").Valid() {
		t.Fatal("status validation")
	}
}

func TestSendStopsWhenHourlyAccountLimitIsReached(t *testing.T) {
	owner, inv, tmpl := uuid.New(), uuid.New(), uuid.New()
	st := &memoryStore{items: map[string]Reminder{}, invoice: Invoice{ID: inv, OwnerID: owner, Kind: "invoice", Status: "sent", Recipient: "client@example.com", UpdatedAt: time.Now()}, template: Template{ID: tmpl, OwnerID: owner, Subject: "s", HTML: "h"}, prepareErr: ErrRateLimited}
	sender := &fakeSender{}
	_, err := NewService(st, sender).Send(context.Background(), owner, inv, tmpl, "rate-key", "job")
	if !errors.Is(err, ErrRateLimited) || sender.calls != 0 {
		t.Fatalf("err=%v calls=%d", err, sender.calls)
	}
}

func TestSendRevalidatesDocumentImmediatelyBeforeDelivery(t *testing.T) {
	owner, inv, tmpl := uuid.New(), uuid.New(), uuid.New()
	st := &memoryStore{items: map[string]Reminder{}, invoice: Invoice{ID: inv, OwnerID: owner, Kind: "invoice", Status: "sent", Recipient: "client@example.com", UpdatedAt: time.Now()}, template: Template{ID: tmpl, OwnerID: owner, Subject: "s", HTML: "h"}, revalidateErr: ErrDocumentChanged}
	sender := &fakeSender{}
	r, err := NewService(st, sender).Send(context.Background(), owner, inv, tmpl, "changed-key", "scheduled")
	if !errors.Is(err, ErrDocumentChanged) || sender.calls != 0 || st.revalidations != 1 {
		t.Fatalf("reminder=%+v err=%v calls=%d revalidations=%d", r, err, sender.calls, st.revalidations)
	}
	if r.Status != StatusSkipped {
		t.Fatalf("status=%s", r.Status)
	}
}

type fakeModeStore struct{ mode DeliveryMode }

func (s fakeModeStore) Mode(context.Context, uuid.UUID) (DeliveryMode, error) { return s.mode, nil }

func TestProviderRouterUsesConfiguredAccountMode(t *testing.T) {
	owner := uuid.New()
	gmail, microsoft, smtp := &fakeSender{}, &fakeSender{}, &fakeSender{}
	router := NewProviderRouter(fakeModeStore{mode: DeliveryModeMicrosoft}, map[DeliveryMode]Sender{DeliveryModeGmail: gmail, DeliveryModeMicrosoft: microsoft, DeliveryModeSMTP: smtp})
	if err := router.Send(context.Background(), owner, Message{To: "client@example.com", Subject: "s"}); err != nil {
		t.Fatal(err)
	}
	if gmail.calls != 0 || microsoft.calls != 1 || smtp.calls != 0 {
		t.Fatalf("gmail=%d microsoft=%d smtp=%d", gmail.calls, microsoft.calls, smtp.calls)
	}
}

func TestSameIdempotencyKeyWithDifferentTemplateIsIndependent(t *testing.T) {
	owner, inv, firstTemplate, secondTemplate := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	st := &memoryStore{items: map[string]Reminder{}, invoice: Invoice{ID: inv, OwnerID: owner, Kind: "invoice", Status: "sent", Recipient: "client@example.com", UpdatedAt: time.Now()}, template: Template{ID: firstTemplate, OwnerID: owner, Subject: "s", HTML: "h"}}
	sender := &fakeSender{}
	svc := NewService(st, sender)
	if _, err := svc.Send(context.Background(), owner, inv, firstTemplate, "shared", "manual"); err != nil {
		t.Fatal(err)
	}
	st.template.ID = secondTemplate
	if _, err := svc.Send(context.Background(), owner, inv, secondTemplate, "shared", "manual"); err != nil {
		t.Fatal(err)
	}
	if sender.calls != 2 {
		t.Fatalf("calls=%d", sender.calls)
	}
}
func TestProviderRouterNeverFallsBack(t *testing.T) {
	gmail := &fakeSender{}
	router := NewProviderRouter(fakeModeStore{mode: DeliveryModeMicrosoft}, map[DeliveryMode]Sender{DeliveryModeGmail: gmail})
	if err := router.Send(context.Background(), uuid.New(), Message{To: "x@example.com"}); !errors.Is(err, ErrDeliveryMode) {
		t.Fatalf("err=%v", err)
	}
	if gmail.calls != 0 {
		t.Fatalf("unexpected fallback calls=%d", gmail.calls)
	}
}
