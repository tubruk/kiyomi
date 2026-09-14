package http

import (
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/chickenzord/go-brisk"
)

// RetryTransport wraps an http.RoundTripper and retries requests when transient errors occur.
type RetryTransport struct {
	Base        http.RoundTripper
	MaxAttempts int
	MinBackoff  time.Duration
	MaxBackoff  time.Duration
}

// NewRetryTransport returns a RetryTransport wrapping base. Defaults to 3 attempts, 100ms min backoff, 5s max backoff.
func NewRetryTransport(base http.RoundTripper) *RetryTransport {
	if base == nil {
		base = http.DefaultTransport
	}
	return &RetryTransport{
		Base:        base,
		MaxAttempts: 3,
		MinBackoff:  100 * time.Millisecond,
		MaxBackoff:  5 * time.Second,
	}
}

// IsTransientError reports whether an HTTP error or response status code is transient and eligible for retry.
func IsTransientError(err error, resp *http.Response) bool {
	if err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return true
		}
		var netErr net.Error
		if errors.As(err, &netErr) {
			if netErr.Timeout() {
				return true
			}
		}
		errStr := strings.ToLower(err.Error())
		if strings.Contains(errStr, "connection reset") ||
			strings.Contains(errStr, "broken pipe") ||
			strings.Contains(errStr, "tls handshake timeout") ||
			strings.Contains(errStr, "eof") {
			return true
		}
		return false
	}

	if resp != nil {
		switch resp.StatusCode {
		case http.StatusTooManyRequests, // 429
			http.StatusBadGateway,        // 502
			http.StatusServiceUnavailable, // 503
			http.StatusGatewayTimeout,     // 504
			520, 521, 522, 523, 524:      // Cloudflare transient edge errors
			return true
		}
	}

	return false
}

func parseRetryAfter(header string) (time.Duration, bool) {
	return brisk.ParseRetryAfter(header)
}

func calculateBackoff(attempt int, minBackoff, maxBackoff time.Duration, resp *http.Response) time.Duration {
	cfg := brisk.RetryConfig{
		InitialBackoff:    minBackoff,
		MaxBackoff:        maxBackoff,
		BackoffFactor:     2.0,
		Jitter:            true,
		RespectRetryAfter: true,
	}
	return cfg.CalculateBackoff(attempt, resp)
}

func (r *RetryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := r.Base
	if base == nil {
		base = http.DefaultTransport
	}

	maxAttempts := r.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 3
	}
	maxRetries := maxAttempts - 1
	if maxRetries < 0 {
		maxRetries = 0
	}

	minBackoff := r.MinBackoff
	if minBackoff <= 0 {
		minBackoff = 100 * time.Millisecond
	}
	maxBackoff := r.MaxBackoff
	if maxBackoff <= 0 {
		maxBackoff = 5 * time.Second
	}

	rb := brisk.NewRetryBuilder().
		MaxRetries(maxRetries).
		InitialBackoff(minBackoff).
		MaxBackoff(maxBackoff).
		BackoffFactor(2.0).
		Jitter(true).
		RespectRetryAfter(true).
		When(func(resp *http.Response, err error) bool {
			return IsTransientError(err, resp)
		})

	rt := brisk.NewRetryRoundTripper(base, rb.Config())
	return rt.RoundTrip(req)
}
