package queue

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ErrPermanent marks an error as non-retryable. Job handlers should return an
// error wrapping this sentinel (via Permanent or fmt.Errorf + %w) when the
// failure will not be fixed by retrying: malformed payload, missing required
// field, 404 from upstream, SSRF block, etc. Workers transition such jobs
// directly to StatusFailed without consuming additional retries.
var ErrPermanent = errors.New("permanent job error")

// Permanent wraps err so it is classified as non-retryable. If err is nil,
// returns nil. Existing error chain is preserved via %w so callers retain
// errors.Is/errors.As introspection.
func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %w", ErrPermanent, err)
}

// IsPermanent reports whether err (or anything in its chain) is a permanent
// error classified via Permanent or wrapping ErrPermanent.
func IsPermanent(err error) bool {
	return errors.Is(err, ErrPermanent)
}

// JobStatus represents the current state of a background job.
type JobStatus string

const (
	StatusPending   JobStatus = "pending"
	StatusRunning   JobStatus = "running"
	StatusCompleted JobStatus = "completed"
	StatusFailed    JobStatus = "failed"
)

// Job represents a background job execution unit.
type Job struct {
	ID               string            `json:"id"`
	ParentID         *string           `json:"parent_id,omitempty"`
	Type             string            `json:"type"`
	Payload          string            `json:"payload"`
	Status           JobStatus         `json:"status"`
	Retries          int               `json:"retries"`
	MaxRetries       int               `json:"max_retries"`
	ConcurrencyGroup string            `json:"concurrency_group"`
	Error            string            `json:"error,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
	StartedAt        *time.Time        `json:"started_at,omitempty"`
	CompletedAt      *time.Time        `json:"completed_at,omitempty"`
	ChildCount       int               `json:"child_count,omitempty"`
	Metadata         map[string]string `json:"metadata"`
}

// JobFilter defines search criteria for listing/querying jobs.
type JobFilter struct {
	Status   JobStatus
	Type     string
	Metadata map[string]string
	ParentID *string
	All      bool
}

// JobHandler defines the function signature for executing a job.
type JobHandler interface {
	Handle(ctx context.Context, job *Job) error
}

// JobHandlerFunc is an adapter to allow the use of ordinary functions as job handlers.
type JobHandlerFunc func(ctx context.Context, job *Job) error

// Handle calls f(ctx, job).
func (f JobHandlerFunc) Handle(ctx context.Context, job *Job) error {
	return f(ctx, job)
}

// CleanupOptions configures retention thresholds and batch sizing for automatic job pruning.
type CleanupOptions struct {
	CompletedTTL time.Duration
	FailedTTL    time.Duration
	BatchSize    int
}

// JobStore defines the storage contract for persisting jobs and metadata.
type JobStore interface {
	CreateJob(ctx context.Context, job *Job) error
	GetJob(ctx context.Context, id string) (*Job, error)
	UpdateJob(ctx context.Context, job *Job) error
	DeleteJob(ctx context.Context, id string) error
	ListJobs(ctx context.Context, filter JobFilter) ([]*Job, error)
	CancelJob(ctx context.Context, id string) error
	CleanupJobs(ctx context.Context, opts CleanupOptions) (int, error)
}

// Enqueuer defines the contract for scheduling jobs.
type Enqueuer interface {
	Enqueue(ctx context.Context, job *Job) error
}

// Worker defines the contract for registering handlers and running the worker loop.
type Worker interface {
	Register(jobType string, handler JobHandler) error
	// SetGroupLimit configures the maximum number of in-flight jobs that share
	// the given concurrency group. Drivers that do not implement per-group
	// throttling may treat this as a no-op.
	SetGroupLimit(group string, limit int)
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}
