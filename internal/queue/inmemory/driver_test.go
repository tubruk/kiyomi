package inmemory

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tubruk/kiyomi/internal/queue"
)

func TestInMemoryStoreCRUD(t *testing.T) {
	driver := NewDriver(10)
	ctx := context.Background()

	job := &queue.Job{
		ID:               "job-1",
		Type:             "test-type",
		Payload:          "test-payload",
		Status:           queue.StatusPending,
		MaxRetries:       3,
		ConcurrencyGroup: "group-a",
		Metadata: map[string]string{
			"key1": "val1",
		},
	}

	// Create
	err := driver.CreateJob(ctx, job)
	require.NoError(t, err)

	// Create duplicate
	err = driver.CreateJob(ctx, job)
	require.Error(t, err)

	// Get
	got, err := driver.GetJob(ctx, "job-1")
	require.NoError(t, err)
	assert.Equal(t, job.ID, got.ID)
	assert.Equal(t, job.Metadata["key1"], got.Metadata["key1"])

	// Get non-existent
	_, err = driver.GetJob(ctx, "job-none")
	require.ErrorIs(t, err, ErrJobNotFound)

	// Update
	job.Status = queue.StatusRunning
	err = driver.UpdateJob(ctx, job)
	require.NoError(t, err)

	got, err = driver.GetJob(ctx, "job-1")
	require.NoError(t, err)
	assert.Equal(t, queue.StatusRunning, got.Status)

	// Update non-existent
	err = driver.UpdateJob(ctx, &queue.Job{ID: "job-none"})
	require.ErrorIs(t, err, ErrJobNotFound)

	// List
	jobs, err := driver.ListJobs(ctx, queue.JobFilter{Status: queue.StatusRunning})
	require.NoError(t, err)
	assert.Len(t, jobs, 1)

	// List with non-matching status
	jobs, err = driver.ListJobs(ctx, queue.JobFilter{Status: queue.StatusPending})
	require.NoError(t, err)
	assert.Len(t, jobs, 0)

	// List with metadata match
	jobs, err = driver.ListJobs(ctx, queue.JobFilter{Metadata: map[string]string{"key1": "val1"}})
	require.NoError(t, err)
	assert.Len(t, jobs, 1)

	// List with metadata non-match
	jobs, err = driver.ListJobs(ctx, queue.JobFilter{Metadata: map[string]string{"key1": "val2"}})
	require.NoError(t, err)
	assert.Len(t, jobs, 0)

	// Delete
	err = driver.DeleteJob(ctx, "job-1")
	require.NoError(t, err)

	_, err = driver.GetJob(ctx, "job-1")
	require.ErrorIs(t, err, ErrJobNotFound)

	// Delete non-existent
	err = driver.DeleteJob(ctx, "job-none")
	require.ErrorIs(t, err, ErrJobNotFound)
}

func TestInMemoryWorker(t *testing.T) {
	driver := NewDriver(10)
	ctx := context.Background()

	var mu sync.Mutex
	processed := make(map[string]bool)

	handler := queue.JobHandlerFunc(func(ctx context.Context, job *queue.Job) error {
		mu.Lock()
		processed[job.ID] = true
		mu.Unlock()
		if job.ID == "job-fail" {
			return errors.New("expected failure")
		}
		return nil
	})

	err := driver.Register("test-task", handler)
	require.NoError(t, err)

	// Try register duplicate
	err = driver.Register("test-task", handler)
	require.ErrorIs(t, err, ErrHandlerConflict)

	err = driver.Start(ctx)
	require.NoError(t, err)

	job1 := &queue.Job{ID: "job-success", Type: "test-task", MaxRetries: 1}
	job2 := &queue.Job{ID: "job-fail", Type: "test-task", MaxRetries: 2}

	err = driver.Enqueue(ctx, job1)
	require.NoError(t, err)
	err = driver.Enqueue(ctx, job2)
	require.NoError(t, err)

	// Wait for processing
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	assert.True(t, processed["job-success"])
	assert.True(t, processed["job-fail"])
	mu.Unlock()

	// Check final statuses
	got1, err := driver.GetJob(ctx, "job-success")
	require.NoError(t, err)
	assert.Equal(t, queue.StatusCompleted, got1.Status)

	got2, err := driver.GetJob(ctx, "job-fail")
	require.NoError(t, err)
	// job-fail had max retries = 2, so it should be pending/failed
	assert.Equal(t, queue.StatusFailed, got2.Status)
	assert.Equal(t, "expected failure", got2.Error)

	err = driver.Stop(ctx)
	require.NoError(t, err)
}
