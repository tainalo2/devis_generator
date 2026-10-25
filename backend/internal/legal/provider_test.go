package legal

import (
	"context"
	"testing"
	"time"
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

func TestStubProviders(t *testing.T) {
	stubs := OfficialStubs()
	if len(stubs) < 2 {
		t.Fatalf("stubs=%d", len(stubs))
	}
	for _, s := range stubs {
		if _, err := s.Fetch(context.Background(), "any"); err != ErrNotConfigured {
			t.Fatalf("%s: %v", s.Name(), err)
		}
	}
}

func TestRateLimiterAllow(t *testing.T) {
	rl := NewRateLimiter(time.Hour)
	if !rl.Allow("a") {
		t.Fatal("first allow")
	}
	if rl.Allow("a") {
		t.Fatal("second should block")
	}
}
