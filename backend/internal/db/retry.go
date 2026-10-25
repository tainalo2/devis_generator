package db

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// RetryOptions controls startup retries for database connectivity / migrations.
type RetryOptions struct {
	MaxAttempts    int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
}

// DefaultRetryOptions is used when the database is not yet reachable at boot.
func DefaultRetryOptions() RetryOptions {
	return RetryOptions{
		MaxAttempts:    30,
		InitialBackoff: time.Second,
		MaxBackoff:     10 * time.Second,
	}
}

func (o RetryOptions) normalized() RetryOptions {
	if o.MaxAttempts < 1 {
		o.MaxAttempts = 1
	}
	if o.InitialBackoff <= 0 {
		o.InitialBackoff = time.Second
	}
	if o.MaxBackoff < o.InitialBackoff {
		o.MaxBackoff = o.InitialBackoff
	}
	return o
}

// withRetry runs fn until it succeeds or attempts / context are exhausted.
func withRetry(ctx context.Context, op string, opts RetryOptions, fn func(context.Context) error) error {
	opts = opts.normalized()
	backoff := opts.InitialBackoff
	var lastErr error

	for attempt := 1; attempt <= opts.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			if lastErr != nil {
				return fmt.Errorf("%s: %w (last error: %v)", op, err, lastErr)
			}
			return fmt.Errorf("%s: %w", op, err)
		}

		err := fn(ctx)
		if err == nil {
			if attempt > 1 {
				slog.Info(op+" succeeded after retry", "attempt", attempt)
			}
			return nil
		}
		lastErr = err
		if attempt == opts.MaxAttempts {
			break
		}

		slog.Warn(op+" failed, retrying", "attempt", attempt, "backoff", backoff.String(), "error", err)

		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("%s: %w (last error: %v)", op, ctx.Err(), lastErr)
		case <-timer.C:
		}

		backoff *= 2
		if backoff > opts.MaxBackoff {
			backoff = opts.MaxBackoff
		}
	}

	return fmt.Errorf("%s: exhausted %d attempts: %w", op, opts.MaxAttempts, lastErr)
}
