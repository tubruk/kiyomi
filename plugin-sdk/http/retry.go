package http

import (
	"errors"
	"io"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
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
			http.StatusBadGateway,          // 502
			http.StatusServiceUnavailable,   // 503
			http.StatusGatewayTimeout,       // 504
			520, 521, 522, 523, 524:        // Cloudflare transient edge errors
			return true
		}
	}

	return false
}

func parseRetryAfter(header string) (time.Duration, bool) {
	header = strings.TrimSpace(header)
	if header == "" {
		return 0, false
	}

	// Try seconds as integer
	if seconds, err := strconv.Atoi(header); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second, true
	}

	// Try HTTP date formats (RFC1123, RFC850, ANSIC)
	if targetTime, err := http.ParseTime(header); err == nil {
		d := time.Until(targetTime)
		if d > 0 {
			return d, true
		}
		return 0, true
	}

	return 0, false
}

func calculateBackoff(attempt int, minBackoff, maxBackoff time.Duration, resp *http.Response) time.Duration {
	if resp != nil {
		if retryAfterStr := resp.Header.Get("Retry-After"); retryAfterStr != "" {
			if d, ok := parseRetryAfter(retryAfterStr); ok && d > 0 {
				if maxBackoff > 0 && d > maxBackoff {
					return maxBackoff
				}
				return d
			}
		}
	}

	if minBackoff <= 0 {
		minBackoff = 100 * time.Millisecond
	}
	if maxBackoff <= 0 {
		maxBackoff = 5 * time.Second
	}

	// Exponential backoff: minBackoff * 2^(attempt-1)
	multiplier := 1 << (attempt - 1)
	backoff := minBackoff * time.Duration(multiplier)

	// Add jitter (up to 25%)
	jitter := time.Duration(rand.Int63n(int64(backoff/4 + 1)))
	backoff += jitter

	if backoff > maxBackoff {
		backoff = maxBackoff
	}
	return backoff
}

func (r *RetryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	maxAttempts := r.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 3
	}

	var lastResp *http.Response
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if ctxErr := req.Context().Err(); ctxErr != nil {
			if lastResp != nil && lastResp.Body != nil {
				_ = lastResp.Body.Close()
			}
			return nil, ctxErr
		}

		if attempt > 1 && req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				if lastResp != nil && lastResp.Body != nil {
					_ = lastResp.Body.Close()
				}
				return nil, err
			}
			req.Body = body
		}

		resp, err := r.Base.RoundTrip(req)
		if !IsTransientError(err, resp) {
			return resp, err
		}

		lastResp = resp
		lastErr = err

		if attempt == maxAttempts {
			break
		}

		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}

		delay := calculateBackoff(attempt, r.MinBackoff, r.MaxBackoff, resp)
		select {
		case <-req.Context().Done():
			return nil, req.Context().Err()
		case <-time.After(delay):
		}
	}

	return lastResp, lastErr
}
