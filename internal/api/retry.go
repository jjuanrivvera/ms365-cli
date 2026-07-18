package api

import (
	"context"
	"errors"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"time"
)

// retryBase is the backoff unit for full-jitter waits: random(0, retryBase·2^attempt).
var retryBase = 500 * time.Millisecond

// idempotentMethods are the only methods auto-retried. POST/PATCH are never silently
// retried — a duplicate write is worse than a surfaced failure.
var idempotentMethods = map[string]bool{
	http.MethodGet:     true,
	http.MethodHead:    true,
	http.MethodPut:     true,
	http.MethodDelete:  true,
	http.MethodOptions: true,
}

// sendWithRetry runs send with bounded retries on 429, 5xx, and transient network errors,
// honoring Retry-After (delta-seconds and HTTP-date) before falling back to full-jitter
// exponential backoff. Only idempotent methods retry.
func (c *Client) sendWithRetry(ctx context.Context, method string, send func() (*http.Response, error)) (*http.Response, error) {
	retries := c.maxRetries
	if !idempotentMethods[method] {
		retries = 0
	}
	var resp *http.Response
	var err error
	for attempt := 0; ; attempt++ {
		resp, err = send()
		if err != nil {
			if attempt >= retries || !isTransient(err) {
				return nil, err
			}
		} else if resp.StatusCode != http.StatusTooManyRequests && resp.StatusCode < 500 {
			return resp, nil
		} else if attempt >= retries {
			return resp, nil
		}

		var wait time.Duration
		if resp != nil {
			wait = retryAfter(resp.Header)
			_ = resp.Body.Close()
		}
		if wait == 0 {
			// Full jitter (deliberate design, not a bug): random(0, base·2^n) spreads a
			// thundering herd better than equal or decorrelated jitter.
			wait = time.Duration(rand.Int63n(int64(retryBase) << attempt)) // #nosec G404 -- non-crypto jitter
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(wait):
		}
	}
}

// retryAfter parses the Retry-After header: delta-seconds first, then HTTP-date.
func retryAfter(h http.Header) time.Duration {
	v := h.Get("Retry-After")
	if v == "" {
		return 0
	}
	if secs, err := strconv.Atoi(v); err == nil && secs >= 0 {
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		if d := time.Until(t); d > 0 {
			return d
		}
	}
	return 0
}

// isTransient reports whether a network error is worth retrying (timeouts, refused
// connections mid-flight). Context cancellation is never transient.
func isTransient(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	var opErr *net.OpError
	return errors.As(err, &opErr)
}
