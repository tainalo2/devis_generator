package legal

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"
)

// RetryPolicy controls measured retries for upstream HTTP calls.
type RetryPolicy struct {
	MaxAttempts int
	BaseDelay   time.Duration
}

func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{MaxAttempts: 3, BaseDelay: 200 * time.Millisecond}
}

// DoWithRetry executes fn until success, non-retryable error, or attempts exhausted.
func DoWithRetry(ctx context.Context, p RetryPolicy, fn func(context.Context) error) error {
	if p.MaxAttempts <= 0 {
		p.MaxAttempts = 1
	}
	if p.BaseDelay <= 0 {
		p.BaseDelay = 200 * time.Millisecond
	}
	var last error
	for attempt := 1; attempt <= p.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		last = fn(ctx)
		if last == nil {
			return nil
		}
		if !isRetryable(last) || attempt == p.MaxAttempts {
			return last
		}
		delay := p.BaseDelay * time.Duration(attempt)
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return last
}

func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	var he *HTTPError
	if errors.As(err, &he) {
		return he.Status == http.StatusTooManyRequests || he.Status >= 500
	}
	return false
}

// HTTPError wraps a non-2xx upstream response.
type HTTPError struct {
	Status int
	Body   string
}

func (e *HTTPError) Error() string {
	if e.Body == "" {
		return http.StatusText(e.Status)
	}
	return e.Body
}
