package legal

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"
)

// HTTPClient is the subset used by providers (mockable).
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// NewHTTPClient returns a client with timeouts suitable for official APIs.
func NewHTTPClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          20,
			IdleConnTimeout:       60 * time.Second,
			TLSHandshakeTimeout:   5 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			ResponseHeaderTimeout: timeout,
		},
	}
}

// RateLimiter is a simple min-interval gate per key (in-process).
type RateLimiter struct {
	mu       sync.Mutex
	interval time.Duration
	last     map[string]time.Time
}

func NewRateLimiter(minInterval time.Duration) *RateLimiter {
	if minInterval <= 0 {
		minInterval = 200 * time.Millisecond
	}
	return &RateLimiter{interval: minInterval, last: map[string]time.Time{}}
}

// Wait blocks until the key may proceed or ctx is cancelled.
func (r *RateLimiter) Wait(ctx context.Context, key string) error {
	for {
		r.mu.Lock()
		last := r.last[key]
		wait := r.interval - time.Since(last)
		if wait <= 0 {
			r.last[key] = time.Now()
			r.mu.Unlock()
			return nil
		}
		r.mu.Unlock()
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

// Allow reports whether a call may proceed immediately (no wait).
func (r *RateLimiter) Allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if time.Since(r.last[key]) < r.interval {
		return false
	}
	r.last[key] = time.Now()
	return true
}
