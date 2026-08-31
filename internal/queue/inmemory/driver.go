package inmemory

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
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
	stopOnce  sync.Once
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

	copied := d.copyJob(job)
	var count int
	for _, other := range d.jobs {
		if other.ParentID != nil && *other.ParentID == id {
			count++
		}
	}
	copied.ChildCount = count

	return copied, nil
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

// DeleteJob deletes a job and its descendant child jobs from the in-memory store.
func (d *Driver) DeleteJob(ctx context.Context, id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if _, exists := d.jobs[id]; !exists {
		return ErrJobNotFound
	}

	visited := make(map[string]bool)
	d.deleteRecursive(id, visited)
	return nil
}

func (d *Driver) deleteRecursive(id string, visited map[string]bool) {
	if visited[id] {
		return
	}
	visited[id] = true

	var childIDs []string
	for _, j := range d.jobs {
		if j.ParentID != nil && *j.ParentID == id {
			childIDs = append(childIDs, j.ID)
		}
	}

	delete(d.jobs, id)

	for _, childID := range childIDs {
		d.deleteRecursive(childID, visited)
	}
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
		if filter.ParentID != nil {
			if job.ParentID == nil || *job.ParentID != *filter.ParentID {
				continue
			}
		} else if !filter.All {
			if job.ParentID != nil && *job.ParentID != "" {
				continue
			}
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

		copied := d.copyJob(job)
		var count int
		for _, other := range d.jobs {
			if other.ParentID != nil && *other.ParentID == job.ID {
				count++
			}
		}
		copied.ChildCount = count

		result = append(result, copied)
	}

	sort.Slice(result, func(i, j int) bool {
		if !result[i].CreatedAt.Equal(result[j].CreatedAt) {
			return result[i].CreatedAt.Before(result[j].CreatedAt)
		}
		piStr, okI := result[i].Metadata["page_index"]
		pjStr, okJ := result[j].Metadata["page_index"]
		if okI && okJ {
			pi, errI := strconv.Atoi(piStr)
			pj, errJ := strconv.Atoi(pjStr)
			if errI == nil && errJ == nil && pi != pj {
				return pi < pj
			}
		}
		return result[i].ID < result[j].ID
	})

	return result, nil
}

// CancelJob recursively cancels a job and all its pending or running descendants.
func (d *Driver) CancelJob(ctx context.Context, id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	root, exists := d.jobs[id]
	if !exists {
		return ErrJobNotFound
	}

	toCancel := []string{root.ID}
	visited := make(map[string]bool)

	for len(toCancel) > 0 {
		currID := toCancel[0]
		toCancel = toCancel[1:]

		if visited[currID] {
			continue
		}
		visited[currID] = true

		if job, ok := d.jobs[currID]; ok {
			if job.Status == queue.StatusPending || job.Status == queue.StatusRunning {
				job.Status = queue.StatusFailed
				job.Error = "job cancelled by user"
				now := time.Now()
				job.UpdatedAt = now
				job.CompletedAt = &now
			}
		}

		for _, j := range d.jobs {
			if j.ParentID != nil && *j.ParentID == currID {
				toCancel = append(toCancel, j.ID)
			}
		}
	}

	return nil
}

// CleanupJobs prunes expired completed and failed jobs from the in-memory store.
func (d *Driver) CleanupJobs(ctx context.Context, opts queue.CleanupOptions) (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if opts.BatchSize <= 0 {
		opts.BatchSize = 500
	}

	completedCutoff := time.Now().Add(-opts.CompletedTTL)
	failedCutoff := time.Now().Add(-opts.FailedTTL)

	var rootIDs []string
	for _, j := range d.jobs {
		if j.ParentID == nil || *j.ParentID == "" {
			rootIDs = append(rootIDs, j.ID)
		}
	}

	deletedCount := 0

	for _, rootID := range rootIDs {
		if deletedCount >= opts.BatchSize {
			break
		}

		rootJob, exists := d.jobs[rootID]
		if !exists {
			continue
		}

		treeJobs := []*queue.Job{rootJob}
		queueList := []string{rootID}
		treeVisited := map[string]bool{rootID: true}

		for len(queueList) > 0 {
			currID := queueList[0]
			queueList = queueList[1:]

			for _, j := range d.jobs {
				if j.ParentID != nil && *j.ParentID == currID && !treeVisited[j.ID] {
					treeVisited[j.ID] = true
					treeJobs = append(treeJobs, j)
					queueList = append(queueList, j.ID)
				}
			}
		}

		hasActive := false
		hasFailed := false
		var maxTime time.Time

		for _, j := range treeJobs {
			if j.Status == queue.StatusPending || j.Status == queue.StatusRunning {
				hasActive = true
				break
			}
			if j.Status == queue.StatusFailed {
				hasFailed = true
			}
			jTime := j.UpdatedAt
			if j.CompletedAt != nil {
				jTime = *j.CompletedAt
			}
			if maxTime.IsZero() || jTime.After(maxTime) {
				maxTime = jTime
			}
		}

		if hasActive {
			continue
		}

		eligible := false
		if !hasFailed {
			if maxTime.Before(completedCutoff) || maxTime.Equal(completedCutoff) {
				eligible = true
			}
		} else {
			if maxTime.Before(failedCutoff) || maxTime.Equal(failedCutoff) {
				eligible = true
			}
		}

		if eligible {
			deleteVisited := make(map[string]bool)
			d.deleteRecursive(rootID, deleteVisited)
			deletedCount++
		}
	}

	return deletedCount, nil
}

// Enqueue puts the job into the in-memory processing queue.
func (d *Driver) Enqueue(ctx context.Context, job *queue.Job) error {
	select {
	case <-d.stopChan:
		return errors.New("queue driver is stopped")
	default:
	}

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
	case <-d.stopChan:
		return errors.New("queue driver is stopped")
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
	d.stopOnce.Do(func() {
		close(d.stopChan)
	})
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
	if job.Status == queue.StatusFailed || job.Status == queue.StatusCompleted {
		return
	}

	d.mu.RLock()
	handler, exists := d.handlers[job.Type]
	d.mu.RUnlock()

	if !exists {
		d.mu.Lock()
		if job.Status != queue.StatusFailed && job.Status != queue.StatusCompleted {
			job.Status = queue.StatusFailed
			job.Error = fmt.Sprintf("no handler registered for job type %s", job.Type)
			now := time.Now()
			job.CompletedAt = &now
		}
		d.mu.Unlock()
		return
	}

	d.mu.Lock()
	if job.Status != queue.StatusPending {
		d.mu.Unlock()
		return
	}
	job.Status = queue.StatusRunning
	now := time.Now()
	job.StartedAt = &now
	d.mu.Unlock()

	var handleErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				handleErr = fmt.Errorf("panic in job handler: %v", r)
			}
		}()
		handleErr = handler.Handle(ctx, d.copyJob(job))
	}()

	d.mu.Lock()
	if job.Status != queue.StatusRunning {
		d.mu.Unlock()
		return
	}

	var shouldReenqueue bool
	if handleErr != nil {
		job.Retries++
		permanent := queue.IsPermanent(handleErr)
		if permanent || job.Retries >= job.MaxRetries {
			job.Status = queue.StatusFailed
			job.Error = handleErr.Error()
			compNow := time.Now()
			job.CompletedAt = &compNow
		} else {
			job.Status = queue.StatusPending
			shouldReenqueue = true
		}
	} else {
		job.Status = queue.StatusCompleted
		compNow := time.Now()
		job.CompletedAt = &compNow
	}
	d.mu.Unlock()

	if shouldReenqueue {
		_ = d.Enqueue(ctx, job)
	}
}

func (d *Driver) copyJob(job *queue.Job) *queue.Job {
	if job == nil {
		return nil
	}
	copied := *job
	if job.ParentID != nil {
		pID := *job.ParentID
		copied.ParentID = &pID
	}
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
