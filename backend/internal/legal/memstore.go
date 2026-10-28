package legal

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

// MemStore is an in-memory Store for unit tests (no Postgres).
type MemStore struct {
	mu       sync.Mutex
	history  []Variable
	statuses []string
	errors   []string
	FreshFor time.Duration
}

func NewMemStore() *MemStore {
	return &MemStore{FreshFor: 7 * 24 * time.Hour}
}

func (s *MemStore) Get(ctx context.Context, key string) (Variable, error) {
	v, _, err := s.Latest(ctx, key)
	return v, err
}

func (s *MemStore) Latest(_ context.Context, key string) (Variable, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := len(s.history) - 1; i >= 0; i-- {
		st := s.statuses[i]
		if s.history[i].Key != key {
			continue
		}
		if st == StatusVerified || st == StatusManual || st == StatusStale {
			v := s.history[i]
			v.Status = st
			return v, st, nil
		}
	}
	return Variable{}, "", ErrNotConfigured
}

func (s *MemStore) Upsert(ctx context.Context, v Variable) error {
	status := v.Status
	if status == "" {
		status = StatusVerified
	}
	return s.Append(ctx, v, status, "")
}

func (s *MemStore) Append(_ context.Context, v Variable, status, errMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if status == "" {
		status = StatusVerified
	}
	if v.FetchedAt.IsZero() {
		v.FetchedAt = time.Now().UTC()
	}
	if len(v.Value) == 0 {
		v.Value = json.RawMessage(`{}`)
	}
	s.history = append(s.history, v)
	s.statuses = append(s.statuses, status)
	s.errors = append(s.errors, errMsg)
	return nil
}

func (s *MemStore) ListStatus(_ context.Context) ([]StatusEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	freshFor := s.FreshFor
	if freshFor <= 0 {
		freshFor = 7 * 24 * time.Hour
	}
	latest := map[string]int{}
	for i, v := range s.history {
		latest[v.Key] = i
	}
	now := time.Now().UTC()
	out := make([]StatusEntry, 0, len(latest))
	for _, i := range latest {
		v := s.history[i]
		st := s.statuses[i]
		age := now.Sub(v.FetchedAt)
		e := StatusEntry{
			Key: v.Key, Value: v.Value, Source: v.Source, FetchedAt: v.FetchedAt,
			Status: st, AgeSecs: int64(age.Seconds()),
			Fresh: (st == StatusVerified && age <= freshFor) || st == StatusManual,
		}
		out = append(out, e)
	}
	return out, nil
}
