package sdk

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ErrorKind classifies a provider failure for targeted handling.
type ErrorKind int

const (
	KindTransient ErrorKind = iota
	KindAuth
	KindPermanent
	KindRateLimit
)

// Classifier maps an error to an ErrorKind + RetryAfter.
// Providers may implement ErrorClassifier to supply their own.
type Classifier func(err error) (kind ErrorKind, retryAfter time.Duration)

// DefaultClassifier maps common HTTP/network errors to ErrorKind.
// Walks error chain looking for *http.Response error shapes, status codes,
// and known error string fragments.
func DefaultClassifier(err error) (ErrorKind, time.Duration) {
	if err == nil {
		return KindTransient, 0
	}

	// Unwrap layers to find the underlying error
	var unwrapErr error = err
	for unwrapErr != nil {
		// Check if it's already a ProviderError
		var pe *ProviderError
		if errors.As(unwrapErr, &pe) {
			return pe.Kind, pe.RetryAfter
		}
		unwrapErr = errors.Unwrap(unwrapErr)
	}

	// Heuristic matching on error string (lowercased)
	msg := strings.ToLower(err.Error())

	// Auth errors
	if strings.Contains(msg, "401") || strings.Contains(msg, "403") ||
		strings.Contains(msg, "unauthorized") || strings.Contains(msg, "forbidden") ||
		strings.Contains(msg, "auth") {
		return KindAuth, 0
	}

	// Permanent errors
	if strings.Contains(msg, "404") || strings.Contains(msg, "410") ||
		strings.Contains(msg, "not found") || strings.Contains(msg, "gone") {
		return KindPermanent, 0
	}

	// Rate limit
	if strings.Contains(msg, "429") || strings.Contains(msg, "too many requests") ||
		strings.Contains(msg, "rate limit") {
		return KindRateLimit, 30*time.Second
	}

	// Transient errors
	if strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline") ||
		strings.Contains(msg, "connection refused") || strings.Contains(msg, "eof") ||
		strings.Contains(msg, "reset") || strings.Contains(msg, "temporary") {
		return KindTransient, 5*time.Second
	}

	// Default
	return KindTransient, 0
}

// ClassifyResponse uses ClassifyHTTPStatus to derive ErrorKind + RetryAfter from an HTTP response.
func ClassifyResponse(resp *http.Response) (ErrorKind, time.Duration) {
	if resp == nil {
		return KindTransient, 0
	}
	return ClassifyHTTPStatus(resp.StatusCode, resp.Header.Get("Retry-After"))
}

// IsTransientError returns true if err or resp represents a transient network/server error
// that is suitable for an automatic retry (e.g. EOF, connection reset, timeout, 502/503/504).
func IsTransientError(err error, resp *http.Response) bool {
	if err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
			return true
		}
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return true
		}
		kind, _ := DefaultClassifier(err)
		return kind == KindTransient
	}
	if resp != nil {
		return resp.StatusCode == http.StatusBadGateway ||
			resp.StatusCode == http.StatusServiceUnavailable ||
			resp.StatusCode == http.StatusGatewayTimeout
	}
	return false
}

func (k ErrorKind) String() string {
	switch k {
	case KindTransient:
		return "transient"
	case KindAuth:
		return "auth"
	case KindPermanent:
		return "permanent"
	case KindRateLimit:
		return "rate_limit"
	default:
		return "unknown"
	}
}

// ProviderError is a typed failure returned across the provider boundary.
type ProviderError struct {
	Kind       ErrorKind
	ProviderID string
	Message    string
	RetryAfter time.Duration
	Cause      error
}

func (e *ProviderError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("provider %s [%s]: %s: %v", e.ProviderID, e.Kind.String(), e.Message, e.Cause)
	}
	return fmt.Sprintf("provider %s [%s]: %s", e.ProviderID, e.Kind.String(), e.Message)
}

func (e *ProviderError) Unwrap() error {
	return e.Cause
}

func (e *ProviderError) Is(target error) bool {
	// Pattern match so callers can use errors.Is with sentinel provider errors.
	_, ok := target.(*ProviderError)
	return ok
}

// HTTPStatus maps an ErrorKind to a representative HTTP status code.
func (e *ProviderError) HTTPStatus() int {
	switch e.Kind {
	case KindAuth:
		return http.StatusUnauthorized
	case KindPermanent:
		return http.StatusUnprocessableEntity
	case KindRateLimit:
		return http.StatusTooManyRequests
	case KindTransient:
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}

// ClassifyHTTPStatus derives an ErrorKind and RetryAfter duration from an HTTP response.
func ClassifyHTTPStatus(statusCode int, retryAfterHeader string) (ErrorKind, time.Duration) {
	switch statusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return KindAuth, 0
	case http.StatusNotFound, http.StatusGone:
		return KindPermanent, 0
	case http.StatusTooManyRequests:
		delay := 30 * time.Second
		if retryAfterHeader != "" {
			if secs, err := strconv.Atoi(retryAfterHeader); err == nil && secs > 0 {
				delay = time.Duration(secs) * time.Second
			}
		}
		return KindRateLimit, delay
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return KindTransient, 5 * time.Second
	default:
		return KindTransient, 0
	}
}

// NewProviderError constructs a ProviderError with the current time as the default RetryAfter.
func NewProviderError(kind ErrorKind, providerID, message string, cause error) *ProviderError {
	return &ProviderError{
		Kind:       kind,
		ProviderID: providerID,
		Message:    message,
		Cause:      cause,
	}
}

// WithRetryAfter sets the RetryAfter duration on ProviderError and returns it.
func (e *ProviderError) WithRetryAfter(d time.Duration) *ProviderError {
	e.RetryAfter = d
	return e
}

// Log logs the ProviderError using structured slog.
func (e *ProviderError) Log() {
	if e == nil {
		return
	}
	attrs := []any{
		slog.String("provider_id", e.ProviderID),
		slog.String("kind", e.Kind.String()),
		slog.String("message", e.Message),
	}
	if e.RetryAfter > 0 {
		attrs = append(attrs, slog.Duration("retry_after", e.RetryAfter))
	}
	if e.Cause != nil {
		attrs = append(attrs, slog.String("cause", e.Cause.Error()))
	}

	switch e.Kind {
	case KindPermanent, KindAuth:
		slog.Error("provider error", attrs...)
	case KindRateLimit, KindTransient:
		slog.Warn("provider warning", attrs...)
	default:
		slog.Error("provider error", attrs...)
	}
}

// ClassifyError classifies any error returned from a provider operation without logging.
func ClassifyError(providerID string, err error) *ProviderError {
	if err == nil {
		return nil
	}
	var pe *ProviderError
	if errors.As(err, &pe) {
		return pe
	}

	kind, retryAfter := DefaultClassifier(err)
	return NewProviderError(kind, providerID, err.Error(), err).WithRetryAfter(retryAfter)
}

// LogProviderError classifies and logs any error returned from a provider operation.
func LogProviderError(providerID string, err error) *ProviderError {
	if err == nil {
		return nil
	}
	pe := ClassifyError(providerID, err)
	pe.Log()
	return pe
}

