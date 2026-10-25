package db

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestWithRetrySucceedsAfterFailures(t *testing.T) {
	attempts := 0
	err := withRetry(context.Background(), "test-op", RetryOptions{
		MaxAttempts:    5,
		InitialBackoff: time.Millisecond,
		MaxBackoff:     2 * time.Millisecond,
	}, func(context.Context) error {
		attempts++
		if attempts < 3 {
			return errors.New("not yet")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("withRetry: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
}

func TestWithRetryExhausted(t *testing.T) {
	err := withRetry(context.Background(), "test-op", RetryOptions{
		MaxAttempts:    2,
		InitialBackoff: time.Millisecond,
		MaxBackoff:     time.Millisecond,
	}, func(context.Context) error {
		return errors.New("always")
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWithRetryContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := withRetry(ctx, "test-op", RetryOptions{MaxAttempts: 3, InitialBackoff: time.Millisecond}, func(context.Context) error {
		return errors.New("always")
	})
	if err == nil {
		t.Fatal("expected error")
	}
}
