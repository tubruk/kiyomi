package inmemory

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/tubruk/kiyomi/internal/queue"
)

var (
	ErrJobNotFound     = errors.New("job not found")
	ErrHandlerConflict = errors.New("handler already registered for this job type")
)

// Driver implements queue.JobStore, queue.Enqueuer, and queue.Worker interfaces.
type Driver struct {
	mu        sync.RWMutex
	jobs      map[string]*queue.Job
	handlers  map[string]queue.JobHandler
	queueChan chan string
	stopChan  chan struct{}
	wg        sync.WaitGroup
}

// NewDriver initializes a new in-memory queue driver.
func NewDriver(bufferSize int) *Driver {
	return &Driver{
		jobs:      make(map[string]*queue.Job),
		handlers:  make(map[string]queue.JobHandler),
		queueChan: make(chan string, bufferSize),
		stopChan:  make(chan struct{}),
	}
}

// CreateJob saves a new job to the in-memory store.
func (d *Driver) CreateJob(ctx context.Context, job *queue.Job) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if _, exists := d.jobs[job.ID]; exists {
		return fmt.Errorf("job with ID %s already exists", job.ID)
	}

	// Deep copy job
	copiedJob := d.copyJob(job)
	d.jobs[job.ID] = copiedJob
	return nil
}

// GetJob retrieves a job from the in-memory store.
func (d *Driver) GetJob(ctx context.Context, id string) (*queue.Job, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	job, exists := d.jobs[id]
	if !exists {
		return nil, ErrJobNotFound
	}

	return d.copyJob(job), nil
}

// UpdateJob updates an existing job in the in-memory store.
func (d *Driver) UpdateJob(ctx context.Context, job *queue.Job) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if _, exists := d.jobs[job.ID]; !exists {
		return ErrJobNotFound
	}

	d.jobs[job.ID] = d.copyJob(job)
	return nil
}

// DeleteJob deletes a job from the in-memory store.
func (d *Driver) DeleteJob(ctx context.Context, id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if _, exists := d.jobs[id]; !exists {
		return ErrJobNotFound
	}

	delete(d.jobs, id)
	return nil
}

// ListJobs retrieves jobs matching the filter.
func (d *Driver) ListJobs(ctx context.Context, filter queue.JobFilter) ([]*queue.Job, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var result []*queue.Job
	for _, job := range d.jobs {
		if filter.Status != "" && job.Status != filter.Status {
			continue
		}
		if filter.Type != "" && job.Type != filter.Type {
			continue
		}
		matchMetadata := true
		for k, v := range filter.Metadata {
			if jVal, exists := job.Metadata[k]; !exists || jVal != v {
				matchMetadata = false
				break
			}
		}
		if !matchMetadata {
			continue
		}
		result = append(result, d.copyJob(job))
	}
	return result, nil
}

// Enqueue puts the job into the in-memory processing queue.
func (d *Driver) Enqueue(ctx context.Context, job *queue.Job) error {
	d.mu.Lock()
	if _, exists := d.jobs[job.ID]; !exists {
		// Auto-create in store if not present
		copied := d.copyJob(job)
		copied.Status = queue.StatusPending
		d.jobs[job.ID] = copied
	} else {
		d.jobs[job.ID].Status = queue.StatusPending
	}
	d.mu.Unlock()

	select {
	case d.queueChan <- job.ID:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Register maps a handler to a job type.
func (d *Driver) Register(jobType string, handler queue.JobHandler) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if _, exists := d.handlers[jobType]; exists {
		return ErrHandlerConflict
	}
	d.handlers[jobType] = handler
	return nil
}

// SetGroupLimit is a no-op for the in-memory driver. The in-memory driver
// processes jobs serially per registered handler, so per-group throttling is
// not applicable. The method exists to satisfy queue.Worker.
func (d *Driver) SetGroupLimit(_ string, _ int) {}

// Start starts the worker loop to process enqueued jobs.
func (d *Driver) Start(ctx context.Context) error {
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		for {
			select {
			case <-d.stopChan:
				return
			case <-ctx.Done():
				return
			case jobID := <-d.queueChan:
				d.processJob(ctx, jobID)
			}
		}
	}()
	return nil
}

// Stop stops the worker loop and waits for all active jobs to complete.
func (d *Driver) Stop(ctx context.Context) error {
	close(d.stopChan)
	done := make(chan struct{})
	go func() {
		d.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (d *Driver) processJob(ctx context.Context, jobID string) {
	d.mu.RLock()
	job, exists := d.jobs[jobID]
	d.mu.RUnlock()
	if !exists {
		return
	}

	d.mu.RLock()
	handler, exists := d.handlers[job.Type]
	d.mu.RUnlock()

	if !exists {
		d.mu.Lock()
		job.Status = queue.StatusFailed
		job.Error = fmt.Sprintf("no handler registered for job type %s", job.Type)
		now := time.Now()
		job.CompletedAt = &now
		d.mu.Unlock()
		return
	}

	d.mu.Lock()
	job.Status = queue.StatusRunning
	now := time.Now()
	job.StartedAt = &now
	d.mu.Unlock()

	err := handler.Handle(ctx, d.copyJob(job))

	d.mu.Lock()
	defer d.mu.Unlock()
	if err != nil {
		job.Retries++
		permanent := queue.IsPermanent(err)
		if permanent || job.Retries >= job.MaxRetries {
			job.Status = queue.StatusFailed
			job.Error = err.Error()
			compNow := time.Now()
			job.CompletedAt = &compNow
		} else {
			job.Status = queue.StatusPending
			d.mu.Unlock()
			// Re-enqueue
			_ = d.Enqueue(ctx, job)
			d.mu.Lock()
		}
	} else {
		job.Status = queue.StatusCompleted
		compNow := time.Now()
		job.CompletedAt = &compNow
	}
}

func (d *Driver) copyJob(job *queue.Job) *queue.Job {
	if job == nil {
		return nil
	}
	copied := *job
	if job.StartedAt != nil {
		t := *job.StartedAt
		copied.StartedAt = &t
	}
	if job.CompletedAt != nil {
		t := *job.CompletedAt
		copied.CompletedAt = &t
	}
	if job.Metadata != nil {
		copied.Metadata = make(map[string]string)
		for k, v := range job.Metadata {
			copied.Metadata[k] = v
		}
	}
	return &copied
}
