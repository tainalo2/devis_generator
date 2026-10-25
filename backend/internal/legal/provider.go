// Package legal defines providers and cache for French/EU legal variables.
package legal

import (
	"context"
	"encoding/json"
	"time"
)

// Variable is a cached legal value retrieved from an official or manual source.
type Variable struct {
	Key       string          `json:"key"`
	Value     json.RawMessage `json:"value"`
	Source    string          `json:"source"`
	FetchedAt time.Time       `json:"fetched_at"`
	Status    string          `json:"status,omitempty"`
}

// StatusEntry is the public status of one legal key.
type StatusEntry struct {
	Key       string          `json:"key"`
	Value     json.RawMessage `json:"value"`
	Source    string          `json:"source"`
	FetchedAt time.Time       `json:"fetched_at"`
	Status    string          `json:"status"`
	Fresh     bool            `json:"fresh"`
	AgeSecs   int64           `json:"age_seconds"`
}

// Provider fetches legal variables from an upstream official source.
type Provider interface {
	Fetch(ctx context.Context, key string) (Variable, error)
	Name() string
}

// Store persists and reads cached legal variables.
type Store interface {
	Get(ctx context.Context, key string) (Variable, error)
	Upsert(ctx context.Context, v Variable) error
	Append(ctx context.Context, v Variable, status, errMsg string) error
	Latest(ctx context.Context, key string) (Variable, string, error) // value + status
	ListStatus(ctx context.Context) ([]StatusEntry, error)
}

// ErrNotConfigured is returned when legal dependencies are not wired.
var ErrNotConfigured = errNotConfigured{}

type errNotConfigured struct{}

func (errNotConfigured) Error() string { return "legal service is not configured" }

// NoopProvider is a placeholder.
type NoopProvider struct{}

func (NoopProvider) Name() string { return "noop" }

func (NoopProvider) Fetch(context.Context, string) (Variable, error) {
	return Variable{}, ErrNotConfigured
}

// StubProvider documents future sources that need credentials (Insee Sirene, Légifrance/PISTE).
// They are listed in Status but never called — no API keys in this repository.
type StubProvider struct {
	SourceName string
}

func (s StubProvider) Name() string {
	if s.SourceName == "" {
		return "stub"
	}
	return s.SourceName
}

func (s StubProvider) Fetch(context.Context, string) (Variable, error) {
	return Variable{}, ErrNotConfigured
}

// OfficialStubs returns placeholder providers for credentialed sources (not wired).
func OfficialStubs() []Provider {
	return []Provider{
		StubProvider{SourceName: "insee.sirene"},
		StubProvider{SourceName: "legifrance.piste"},
	}
}
