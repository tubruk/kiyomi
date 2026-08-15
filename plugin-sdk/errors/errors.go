package errors

import (
	"errors"
	"fmt"
	"time"
)

var (
	// ErrNotFound is returned when a requested manga, chapter, page, or resource does not exist.
	ErrNotFound = errors.New("resource not found")

	// ErrRateLimited is returned when upstream requests are being throttled.
	ErrRateLimited = errors.New("rate limited by provider")

	// ErrCloudflareBlocked is returned when a Cloudflare challenge/block is detected.
	ErrCloudflareBlocked = errors.New("blocked by cloudflare protection")

	// ErrAuthRequired is returned when an unauthenticated request requires credentials.
	ErrAuthRequired = errors.New("authentication required")

	// ErrInvalidCredentials is returned when login or token refresh fails due to bad credentials.
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrNotImplemented is returned when an optional capability or method is not supported.
	ErrNotImplemented = errors.New("not implemented")

	// ErrInvalidInput is returned when arguments or parameters are malformed.
	ErrInvalidInput = errors.New("invalid input")
)

// RateLimitError provides structured rate limit details including retry hints.
type RateLimitError struct {
	RetryAfter time.Duration
	Message    string
}

func (e *RateLimitError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("rate limited by provider (retry after %s): %s", e.RetryAfter, e.Message)
	}
	if e.Message != "" {
		return fmt.Sprintf("rate limited by provider: %s", e.Message)
	}
	return ErrRateLimited.Error()
}

func (e *RateLimitError) Unwrap() error {
	return ErrRateLimited
}

// NewRateLimitError creates a structured RateLimitError wrapping ErrRateLimited.
func NewRateLimitError(retryAfter time.Duration, message string) error {
	return &RateLimitError{
		RetryAfter: retryAfter,
		Message:    message,
	}
}

// IsNotFound checks if err is or wraps ErrNotFound.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsRateLimited checks if err is or wraps ErrRateLimited.
func IsRateLimited(err error) bool {
	return errors.Is(err, ErrRateLimited)
}

// IsCloudflareBlocked checks if err is or wraps ErrCloudflareBlocked.
func IsCloudflareBlocked(err error) bool {
	return errors.Is(err, ErrCloudflareBlocked)
}

// IsAuthRequired checks if err is or wraps ErrAuthRequired.
func IsAuthRequired(err error) bool {
	return errors.Is(err, ErrAuthRequired)
}

// IsInvalidCredentials checks if err is or wraps ErrInvalidCredentials.
func IsInvalidCredentials(err error) bool {
	return errors.Is(err, ErrInvalidCredentials)
}

// IsNotImplemented checks if err is or wraps ErrNotImplemented.
func IsNotImplemented(err error) bool {
	return errors.Is(err, ErrNotImplemented)
}
