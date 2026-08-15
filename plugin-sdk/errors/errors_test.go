package errors

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSentinelErrors(t *testing.T) {
	assert.True(t, IsNotFound(ErrNotFound))
	assert.True(t, IsNotFound(fmt.Errorf("wrapped: %w", ErrNotFound)))
	assert.False(t, IsNotFound(ErrRateLimited))

	assert.True(t, IsRateLimited(ErrRateLimited))
	assert.True(t, IsRateLimited(fmt.Errorf("wrapped: %w", ErrRateLimited)))
	assert.False(t, IsRateLimited(ErrNotFound))

	assert.True(t, IsCloudflareBlocked(ErrCloudflareBlocked))
	assert.True(t, IsCloudflareBlocked(fmt.Errorf("wrapped: %w", ErrCloudflareBlocked)))
	assert.False(t, IsCloudflareBlocked(ErrNotFound))

	assert.True(t, IsAuthRequired(ErrAuthRequired))
	assert.True(t, IsAuthRequired(fmt.Errorf("wrapped: %w", ErrAuthRequired)))

	assert.True(t, IsInvalidCredentials(ErrInvalidCredentials))
	assert.True(t, IsInvalidCredentials(fmt.Errorf("wrapped: %w", ErrInvalidCredentials)))

	assert.True(t, IsNotImplemented(ErrNotImplemented))
	assert.True(t, IsNotImplemented(fmt.Errorf("wrapped: %w", ErrNotImplemented)))
}

func TestRateLimitError(t *testing.T) {
	err := NewRateLimitError(5*time.Second, "too many requests")
	assert.Error(t, err)
	assert.True(t, IsRateLimited(err))
	assert.Contains(t, err.Error(), "retry after 5s")
	assert.Contains(t, err.Error(), "too many requests")

	simpleErr := NewRateLimitError(0, "exceeded")
	assert.Contains(t, simpleErr.Error(), "exceeded")

	emptyErr := NewRateLimitError(0, "")
	assert.Equal(t, ErrRateLimited.Error(), emptyErr.Error())
}
