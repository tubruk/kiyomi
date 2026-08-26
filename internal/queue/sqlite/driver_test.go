package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tubruk/kiyomi/internal/queue"
)

func TestSQLiteStoreCRUD(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "jobs.db")

	driver, err := NewDriver(dbPath, 0)
	require.NoError(t, err)
	defer driver.Close()

	ctx := context.Background()

	job := &queue.Job{
		ID:               "job-1",
		Type:             "test-type",
		Payload:          "test-payload",
		Status:           queue.StatusPending,
		MaxRetries:       3,
		ConcurrencyGroup: "group-a",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		Metadata: map[string]string{
			"key1": "val1",
			"key2": "val2",
		},
	}

	// Create
	err = driver.CreateJob(ctx, job)
	require.NoError(t, err)

	// Create duplicate
	err = driver.CreateJob(ctx, job)
	require.Error(t, err)

	// Get
	got, err := driver.GetJob(ctx, "job-1")
	require.NoError(t, err)
	assert.Equal(t, job.ID, got.ID)
	assert.Equal(t, job.Type, got.Type)
	assert.Equal(t, job.Payload, got.Payload)
	assert.Equal(t, job.Status, got.Status)
	assert.Equal(t, job.MaxRetries, got.MaxRetries)
	assert.Equal(t, job.ConcurrencyGroup, got.ConcurrencyGroup)
	assert.Equal(t, "val1", got.Metadata["key1"])
	assert.Equal(t, "val2", got.Metadata["key2"])

	// Get non-existent
	_, err = driver.GetJob(ctx, "job-none")
	require.ErrorIs(t, err, ErrJobNotFound)

	// Update
	job.Status = queue.StatusRunning
	now := time.Now()
	job.StartedAt = &now
	job.Metadata["key1"] = "updated-val"
	err = driver.UpdateJob(ctx, job)
	require.NoError(t, err)

	got, err = driver.GetJob(ctx, "job-1")
	require.NoError(t, err)
	assert.Equal(t, queue.StatusRunning, got.Status)
	assert.Equal(t, "updated-val", got.Metadata["key1"])
	assert.NotNil(t, got.StartedAt)

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
	jobs, err = driver.ListJobs(ctx, queue.JobFilter{Metadata: map[string]string{"key2": "val2"}})
	require.NoError(t, err)
	assert.Len(t, jobs, 1)

	// List with metadata non-match
	jobs, err = driver.ListJobs(ctx, queue.JobFilter{Metadata: map[string]string{"key2": "val3"}})
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

func TestSQLiteWorkerAndConcurrencyLimit(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "jobs.db")

	driver, err := NewDriver(dbPath, 0)
	require.NoError(t, err)
	defer driver.Close()

	ctx := context.Background()

	var mu sync.Mutex
	processed := make(map[string]int)

	handler := queue.JobHandlerFunc(func(ctx context.Context, job *queue.Job) error {
		mu.Lock()
		processed[job.ID]++
		mu.Unlock()
		if job.ID == "job-fail" {
			return errors.New("expected failure")
		}
		// If job-slow, simulate some duration
		if job.ID == "job-slow-1" || job.ID == "job-slow-2" {
			time.Sleep(50 * time.Millisecond)
		}
		return nil
	})

	err = driver.Register("test-task", handler)
	require.NoError(t, err)

	// Set concurrency group limit
	driver.SetGroupLimit("slow-group", 1)

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
	assert.Equal(t, 1, processed["job-success"])
	assert.Equal(t, 2, processed["job-fail"]) // 1 initial + 1 retry = 2
	mu.Unlock()

	// Check concurrency limit enqueuing two slow jobs in same group
	slow1 := &queue.Job{ID: "job-slow-1", Type: "test-task", ConcurrencyGroup: "slow-group", MaxRetries: 1}
	slow2 := &queue.Job{ID: "job-slow-2", Type: "test-task", ConcurrencyGroup: "slow-group", MaxRetries: 1}

	err = driver.Enqueue(ctx, slow1)
	require.NoError(t, err)
	err = driver.Enqueue(ctx, slow2)
	require.NoError(t, err)

	// Wait a brief moment: because limit is 1, slow1 should be running/completed, but slow2 should not be done yet
	time.Sleep(20 * time.Millisecond)
	mu.Lock()
	assert.Equal(t, 1, processed["job-slow-1"])
	assert.Equal(t, 0, processed["job-slow-2"])
	mu.Unlock()

	// Wait for slow1 to finish, which triggers poller for slow2
	time.Sleep(60 * time.Millisecond)
	mu.Lock()
	assert.Equal(t, 1, processed["job-slow-2"])
	mu.Unlock()

	err = driver.Stop(ctx)
	require.NoError(t, err)
}

func TestSQLiteGlobalConcurrencyLimit(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "jobs_global_limit.db")

	// Set maxConcurrency = 2
	driver, err := NewDriver(dbPath, 2)
	require.NoError(t, err)
	defer driver.Close()

	ctx := context.Background()

	var mu sync.Mutex
	processed := make(map[string]int)

	handler := queue.JobHandlerFunc(func(ctx context.Context, job *queue.Job) error {
		mu.Lock()
		processed[job.ID]++
		mu.Unlock()
		time.Sleep(50 * time.Millisecond)
		return nil
	})

	err = driver.Register("test-task", handler)
	require.NoError(t, err)

	err = driver.Start(ctx)
	require.NoError(t, err)

	job1 := &queue.Job{ID: "job-1", Type: "test-task", MaxRetries: 1}
	job2 := &queue.Job{ID: "job-2", Type: "test-task", MaxRetries: 1}
	job3 := &queue.Job{ID: "job-3", Type: "test-task", MaxRetries: 1}

	err = driver.Enqueue(ctx, job1)
	require.NoError(t, err)
	err = driver.Enqueue(ctx, job2)
	require.NoError(t, err)
	err = driver.Enqueue(ctx, job3)
	require.NoError(t, err)

	// Wait a brief moment: because limit is 2, only job-1 and job-2 should start running, job-3 must wait
	time.Sleep(20 * time.Millisecond)
	mu.Lock()
	assert.Equal(t, 1, processed["job-1"])
	assert.Equal(t, 1, processed["job-2"])
	assert.Equal(t, 0, processed["job-3"])
	mu.Unlock()

	// Wait for job-1 and job-2 to finish, triggering poller for job-3
	time.Sleep(60 * time.Millisecond)
	mu.Lock()
	assert.Equal(t, 1, processed["job-3"])
	mu.Unlock()

	err = driver.Stop(ctx)
	require.NoError(t, err)
}

func TestSQLiteListJobsOrdering(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "jobs_ordering.db")

	driver, err := NewDriver(dbPath, 0)
	require.NoError(t, err)
	defer driver.Close()

	ctx := context.Background()

	baseTime := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)

	job1 := &queue.Job{
		ID:        "job-old",
		Type:      "test-type",
		Payload:   "old",
		Status:    queue.StatusPending,
		CreatedAt: baseTime,
		UpdatedAt: baseTime,
	}
	job2 := &queue.Job{
		ID:        "job-new",
		Type:      "test-type",
		Payload:   "new",
		Status:    queue.StatusPending,
		CreatedAt: baseTime.Add(1 * time.Hour),
		UpdatedAt: baseTime.Add(1 * time.Hour),
	}

	err = driver.CreateJob(ctx, job1)
	require.NoError(t, err)
	err = driver.CreateJob(ctx, job2)
	require.NoError(t, err)

	jobs, err := driver.ListJobs(ctx, queue.JobFilter{})
	require.NoError(t, err)
	require.Len(t, jobs, 2)
	assert.Equal(t, "job-new", jobs[0].ID)
	assert.Equal(t, "job-old", jobs[1].ID)
}
