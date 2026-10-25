package legal

import (
	"context"
	"testing"
)

func TestNoopProvider(t *testing.T) {
	p := NoopProvider{}
	if p.Name() != "noop" {
		t.Fatalf("Name() = %q, want noop", p.Name())
	}
	if _, err := p.Fetch(context.Background(), "vat.standard"); err != ErrNotConfigured {
		t.Fatalf("Fetch() error = %v, want ErrNotConfigured", err)
	}
}

func TestServiceRequiresStore(t *testing.T) {
	s := &Service{Provider: NoopProvider{}}
	if _, err := s.Get(context.Background(), "vat.standard"); err != ErrNotConfigured {
		t.Fatalf("Get() error = %v, want ErrNotConfigured", err)
	}
}
