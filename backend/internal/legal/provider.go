// Package legal defines the contract for fetching official French legal
// variables (VAT rates, franchise thresholds, penalty rates, company data via
// SIREN, etc.). Concrete providers and official data sources will be wired later;
// values are expected to be persisted in PostgreSQL with source + fetched_at.
package legal

import (
	"context"
	"encoding/json"
	"time"
)

// Variable is a cached legal value retrieved from an official source.
type Variable struct {
	Key       string
	Value     json.RawMessage
	Source    string
	FetchedAt time.Time
}

// Provider fetches legal variables from an upstream official source.
// Implementations must be safe for concurrent use.
type Provider interface {
	// Fetch retrieves the current value for key from the official source.
	// It must not write to the database; caching is the caller's responsibility.
	Fetch(ctx context.Context, key string) (Variable, error)

	// Name returns a stable identifier for this provider (used as source metadata).
	Name() string
}

// Store persists and reads cached legal variables.
type Store interface {
	Get(ctx context.Context, key string) (Variable, error)
	Upsert(ctx context.Context, v Variable) error
}

// Service coordinates fetch + cache. Skeleton only — no official sources yet.
type Service struct {
	Provider Provider
	Store    Store
}

// Get returns a cached value, or fetches and caches it when missing/stale logic
// is added later. For now it only reads from the store.
func (s *Service) Get(ctx context.Context, key string) (Variable, error) {
	if s.Store == nil {
		return Variable{}, ErrNotConfigured
	}
	return s.Store.Get(ctx, key)
}

// ErrNotConfigured is returned when legal dependencies are not wired.
var ErrNotConfigured = errNotConfigured{}

type errNotConfigured struct{}

func (errNotConfigured) Error() string {
	return "legal service is not configured"
}

// NoopProvider is a placeholder until official sources are integrated.
type NoopProvider struct{}

func (NoopProvider) Name() string { return "noop" }

func (NoopProvider) Fetch(context.Context, string) (Variable, error) {
	return Variable{}, ErrNotConfigured
}
