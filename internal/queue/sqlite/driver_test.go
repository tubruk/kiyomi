package sqlite

import (
	"context"
	"database/sql"
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
	assert.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return processed["job-success"] == 1 && processed["job-fail"] == 2
	}, 3*time.Second, 20*time.Millisecond)

	// Check concurrency limit enqueuing two slow jobs in same group
	slow1 := &queue.Job{ID: "job-slow-1", Type: "test-task", ConcurrencyGroup: "slow-group", MaxRetries: 1}
	slow2 := &queue.Job{ID: "job-slow-2", Type: "test-task", ConcurrencyGroup: "slow-group", MaxRetries: 1}

	err = driver.Enqueue(ctx, slow1)
	require.NoError(t, err)
	err = driver.Enqueue(ctx, slow2)
	require.NoError(t, err)

	// Wait for slow1 to run while slow2 waits
	assert.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return processed["job-slow-1"] >= 1
	}, 3*time.Second, 20*time.Millisecond)

	// Wait for slow1 to finish, which triggers poller for slow2
	assert.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return processed["job-slow-2"] == 1
	}, 3*time.Second, 20*time.Millisecond)

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

	// Wait for job-1 and job-2 to be picked up
	assert.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return processed["job-1"] == 1 && processed["job-2"] == 1
	}, 3*time.Second, 20*time.Millisecond)

	// Wait for job-3 to finish once concurrency frees up
	assert.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return processed["job-3"] == 1
	}, 3*time.Second, 20*time.Millisecond)

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

func TestSQLiteAutoMigration(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "jobs_legacy.db")

	// 1. Create a legacy table without parent_id column
	legacyDSN := "file:" + dbPath + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys=ON"
	rawDB, err := sql.Open("sqlite", legacyDSN)
	require.NoError(t, err)

	legacySchema := `
	CREATE TABLE jobs (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		payload TEXT NOT NULL,
		status TEXT NOT NULL,
		retries INTEGER NOT NULL,
		max_retries INTEGER NOT NULL,
		concurrency_group TEXT NOT NULL,
		error TEXT,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		started_at TEXT,
		completed_at TEXT
	);
	CREATE TABLE job_metadata (
		job_id TEXT NOT NULL,
		key TEXT NOT NULL,
		value TEXT NOT NULL,
		PRIMARY KEY (job_id, key),
		FOREIGN KEY (job_id) REFERENCES jobs (id) ON DELETE CASCADE
	);
	CREATE INDEX idx_jobs_status ON jobs(status);
	`
	_, err = rawDB.Exec(legacySchema)
	require.NoError(t, err)

	now := time.Now().Format(time.RFC3339Nano)
	_, err = rawDB.Exec(`INSERT INTO jobs (id, type, payload, status, retries, max_retries, concurrency_group, created_at, updated_at)
		VALUES ('legacy-job-1', 'pull_manga', '{}', 'pending', 0, 3, 'pull:test', ?, ?)`, now, now)
	require.NoError(t, err)
	require.NoError(t, rawDB.Close())

	// 2. Open with NewDriver and ensure migration happens automatically
	driver, err := NewDriver(dbPath, 0)
	require.NoError(t, err)
	defer driver.Close()

	ctx := context.Background()

	// Verify legacy job exists and has nil ParentID and 0 ChildCount
	job, err := driver.GetJob(ctx, "legacy-job-1")
	require.NoError(t, err)
	assert.Equal(t, "legacy-job-1", job.ID)
	assert.Nil(t, job.ParentID)
	assert.Equal(t, 0, job.ChildCount)

	// Verify we can create a child job referencing the legacy job
	parentID := "legacy-job-1"
	childJob := &queue.Job{
		ID:               "child-job-1",
		ParentID:         &parentID,
		Type:             "pull_chapter",
		Payload:          "{}",
		Status:           queue.StatusPending,
		MaxRetries:       3,
		ConcurrencyGroup: "pull:test",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	err = driver.CreateJob(ctx, childJob)
	require.NoError(t, err)

	// Verify legacy job now has ChildCount == 1
	job, err = driver.GetJob(ctx, "legacy-job-1")
	require.NoError(t, err)
	assert.Equal(t, 1, job.ChildCount)

	// Verify child job has parent_id set
	gotChild, err := driver.GetJob(ctx, "child-job-1")
	require.NoError(t, err)
	require.NotNil(t, gotChild.ParentID)
	assert.Equal(t, "legacy-job-1", *gotChild.ParentID)
}

func TestSQLiteParentChildHierarchyAndFiltering(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "jobs_hierarchy.db")

	driver, err := NewDriver(dbPath, 0)
	require.NoError(t, err)
	defer driver.Close()

	ctx := context.Background()
	now := time.Now()

	rootID := "job-root"
	rootJob := &queue.Job{
		ID:               rootID,
		Type:             "pull_manga",
		Payload:          "{}",
		Status:           queue.StatusPending,
		MaxRetries:       3,
		ConcurrencyGroup: "pull:test",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	require.NoError(t, driver.CreateJob(ctx, rootJob))

	child1ID := "job-child-1"
	child1 := &queue.Job{
		ID:               child1ID,
		ParentID:         &rootID,
		Type:             "pull_chapter",
		Payload:          "{}",
		Status:           queue.StatusPending,
		MaxRetries:       3,
		ConcurrencyGroup: "pull:test",
		CreatedAt:        now.Add(1 * time.Second),
		UpdatedAt:        now.Add(1 * time.Second),
	}
	require.NoError(t, driver.CreateJob(ctx, child1))

	child2ID := "job-child-2"
	child2 := &queue.Job{
		ID:               child2ID,
		ParentID:         &rootID,
		Type:             "pull_cover",
		Payload:          "{}",
		Status:           queue.StatusPending,
		MaxRetries:       3,
		ConcurrencyGroup: "pull:cover",
		CreatedAt:        now.Add(2 * time.Second),
		UpdatedAt:        now.Add(2 * time.Second),
	}
	require.NoError(t, driver.CreateJob(ctx, child2))

	grandchildID := "job-grandchild-1"
	grandchild := &queue.Job{
		ID:               grandchildID,
		ParentID:         &child1ID,
		Type:             "pull_page",
		Payload:          "{}",
		Status:           queue.StatusPending,
		MaxRetries:       3,
		ConcurrencyGroup: "pull:test",
		CreatedAt:        now.Add(3 * time.Second),
		UpdatedAt:        now.Add(3 * time.Second),
	}
	require.NoError(t, driver.CreateJob(ctx, grandchild))

	// 1. Default ListJobs should return ONLY top-level jobs (rootJob)
	topLevelJobs, err := driver.ListJobs(ctx, queue.JobFilter{})
	require.NoError(t, err)
	require.Len(t, topLevelJobs, 1)
	assert.Equal(t, rootID, topLevelJobs[0].ID)
	assert.Equal(t, 2, topLevelJobs[0].ChildCount)

	// 2. Filter by ParentID = rootID should return child1 and child2
	children, err := driver.ListJobs(ctx, queue.JobFilter{ParentID: &rootID})
	require.NoError(t, err)
	require.Len(t, children, 2)
	for _, ch := range children {
		if ch.ID == child1ID {
			assert.Equal(t, 1, ch.ChildCount)
		} else if ch.ID == child2ID {
			assert.Equal(t, 0, ch.ChildCount)
		}
	}

	// 3. Filter by ParentID = child1ID should return grandchild
	grandchildren, err := driver.ListJobs(ctx, queue.JobFilter{ParentID: &child1ID})
	require.NoError(t, err)
	require.Len(t, grandchildren, 1)
	assert.Equal(t, grandchildID, grandchildren[0].ID)
	assert.Equal(t, 0, grandchildren[0].ChildCount)

	// 4. Filter with All = true should return all 4 jobs
	allJobs, err := driver.ListJobs(ctx, queue.JobFilter{All: true})
	require.NoError(t, err)
	require.Len(t, allJobs, 4)
}

func TestSQLiteRecursiveCancelJob(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "jobs_cancel.db")

	driver, err := NewDriver(dbPath, 0)
	require.NoError(t, err)
	defer driver.Close()

	ctx := context.Background()
	now := time.Now()

	rootID := "job-root-cancel"
	childID := "job-child-cancel"
	grandchildID := "job-grandchild-cancel"
	unrelatedID := "job-unrelated"

	require.NoError(t, driver.CreateJob(ctx, &queue.Job{
		ID:               rootID,
		Type:             "pull_manga",
		Payload:          "{}",
		Status:           queue.StatusPending,
		MaxRetries:       3,
		ConcurrencyGroup: "pull:test",
		CreatedAt:        now,
		UpdatedAt:        now,
	}))

	require.NoError(t, driver.CreateJob(ctx, &queue.Job{
		ID:               childID,
		ParentID:         &rootID,
		Type:             "pull_chapter",
		Payload:          "{}",
		Status:           queue.StatusRunning,
		MaxRetries:       3,
		ConcurrencyGroup: "pull:test",
		CreatedAt:        now,
		UpdatedAt:        now,
	}))

	require.NoError(t, driver.CreateJob(ctx, &queue.Job{
		ID:               grandchildID,
		ParentID:         &childID,
		Type:             "pull_page",
		Payload:          "{}",
		Status:           queue.StatusPending,
		MaxRetries:       3,
		ConcurrencyGroup: "pull:test",
		CreatedAt:        now,
		UpdatedAt:        now,
	}))

	require.NoError(t, driver.CreateJob(ctx, &queue.Job{
		ID:               unrelatedID,
		Type:             "pull_manga",
		Payload:          "{}",
		Status:           queue.StatusPending,
		MaxRetries:       3,
		ConcurrencyGroup: "pull:test",
		CreatedAt:        now,
		UpdatedAt:        now,
	}))

	// Cancel non-existent job
	err = driver.CancelJob(ctx, "non-existent")
	require.ErrorIs(t, err, ErrJobNotFound)

	// Cancel root job
	err = driver.CancelJob(ctx, rootID)
	require.NoError(t, err)

	// Check root job
	jRoot, err := driver.GetJob(ctx, rootID)
	require.NoError(t, err)
	assert.Equal(t, queue.StatusFailed, jRoot.Status)
	assert.Equal(t, "job cancelled by user", jRoot.Error)
	assert.NotNil(t, jRoot.CompletedAt)

	// Check child job
	jChild, err := driver.GetJob(ctx, childID)
	require.NoError(t, err)
	assert.Equal(t, queue.StatusFailed, jChild.Status)
	assert.Equal(t, "job cancelled by user", jChild.Error)
	assert.NotNil(t, jChild.CompletedAt)

	// Check grandchild job
	jGrandchild, err := driver.GetJob(ctx, grandchildID)
	require.NoError(t, err)
	assert.Equal(t, queue.StatusFailed, jGrandchild.Status)
	assert.Equal(t, "job cancelled by user", jGrandchild.Error)
	assert.NotNil(t, jGrandchild.CompletedAt)

	// Check unrelated job
	jUnrelated, err := driver.GetJob(ctx, unrelatedID)
	require.NoError(t, err)
	assert.Equal(t, queue.StatusPending, jUnrelated.Status)
	assert.Empty(t, jUnrelated.Error)
	assert.Nil(t, jUnrelated.CompletedAt)
}

func TestSQLiteHandlerPanicRecoveryAndConcurrency(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "jobs_panic.db")

	driver, err := NewDriver(dbPath, 1) // maxConcurrency = 1
	require.NoError(t, err)
	defer driver.Close()

	ctx := context.Background()

	var mu sync.Mutex
	processed := make(map[string]bool)

	err = driver.Register("panic-task", queue.JobHandlerFunc(func(ctx context.Context, job *queue.Job) error {
		mu.Lock()
		processed[job.ID] = true
		mu.Unlock()
		panic("handler panic error")
	}))
	require.NoError(t, err)

	err = driver.Register("normal-task", queue.JobHandlerFunc(func(ctx context.Context, job *queue.Job) error {
		mu.Lock()
		processed[job.ID] = true
		mu.Unlock()
		return nil
	}))
	require.NoError(t, err)

	err = driver.Start(ctx)
	require.NoError(t, err)

	panicJob := &queue.Job{
		ID:         "job-panic-1",
		Type:       "panic-task",
		Status:     queue.StatusPending,
		MaxRetries: 1,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	err = driver.Enqueue(ctx, panicJob)
	require.NoError(t, err)

	// Wait for panicJob to be processed
	assert.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return processed["job-panic-1"]
	}, 3*time.Second, 20*time.Millisecond)

	// Check job status is failed and error contains panic message
	assert.Eventually(t, func() bool {
		j, err := driver.GetJob(ctx, "job-panic-1")
		if err != nil {
			return false
		}
		return j.Status == queue.StatusFailed && j.Error == "panic in job handler: handler panic error"
	}, 3*time.Second, 20*time.Millisecond)

	// Verify that concurrency was not leaked by enqueuing a normal job with maxConcurrency=1
	normalJob := &queue.Job{
		ID:         "job-normal-1",
		Type:       "normal-task",
		Status:     queue.StatusPending,
		MaxRetries: 1,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	err = driver.Enqueue(ctx, normalJob)
	require.NoError(t, err)

	assert.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return processed["job-normal-1"]
	}, 3*time.Second, 20*time.Millisecond)

	jNormal, err := driver.GetJob(ctx, "job-normal-1")
	require.NoError(t, err)
	assert.Equal(t, queue.StatusCompleted, jNormal.Status)

	err = driver.Stop(ctx)
	require.NoError(t, err)
}

func TestSQLiteCancelJobWhileRunningNotOverwritten(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "jobs_cancel_running.db")

	driver, err := NewDriver(dbPath, 0)
	require.NoError(t, err)
	defer driver.Close()

	ctx := context.Background()

	runningChan := make(chan struct{})
	releaseChan := make(chan struct{})

	err = driver.Register("slow-cancel", queue.JobHandlerFunc(func(ctx context.Context, job *queue.Job) error {
		close(runningChan)
		<-releaseChan
		return nil
	}))
	require.NoError(t, err)

	err = driver.Start(ctx)
	require.NoError(t, err)

	job := &queue.Job{
		ID:         "job-cancel-running",
		Type:       "slow-cancel",
		Status:     queue.StatusPending,
		MaxRetries: 1,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	err = driver.Enqueue(ctx, job)
	require.NoError(t, err)

	// Wait until handler is running
	select {
	case <-runningChan:
	case <-time.After(3 * time.Second):
		t.Fatal("handler did not start in time")
	}

	// Cancel job while it is running
	err = driver.CancelJob(ctx, job.ID)
	require.NoError(t, err)

	j, err := driver.GetJob(ctx, job.ID)
	require.NoError(t, err)
	assert.Equal(t, queue.StatusFailed, j.Status)
	assert.Equal(t, "job cancelled by user", j.Error)

	// Allow handler to finish normally
	close(releaseChan)

	// Wait a bit to ensure finalizeJob ran
	time.Sleep(100 * time.Millisecond)

	// Verify job is STILL failed and error was NOT overwritten to completed
	jAfter, err := driver.GetJob(ctx, job.ID)
	require.NoError(t, err)
	assert.Equal(t, queue.StatusFailed, jAfter.Status)
	assert.Equal(t, "job cancelled by user", jAfter.Error)

	err = driver.Stop(ctx)
	require.NoError(t, err)
}

func TestSQLiteDeleteJobCascade(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "jobs_delete_cascade.db")

	driver, err := NewDriver(dbPath, 0)
	require.NoError(t, err)
	defer driver.Close()

	ctx := context.Background()
	now := time.Now()

	rootID := "job-root-del"
	childID := "job-child-del"
	grandchildID := "job-grandchild-del"
	otherID := "job-other-del"

	require.NoError(t, driver.CreateJob(ctx, &queue.Job{
		ID:        rootID,
		Type:      "pull_manga",
		Payload:   "{}",
		Status:    queue.StatusPending,
		CreatedAt: now,
		UpdatedAt: now,
		Metadata:  map[string]string{"manga_id": "1"},
	}))

	require.NoError(t, driver.CreateJob(ctx, &queue.Job{
		ID:        childID,
		ParentID:  &rootID,
		Type:      "pull_chapter",
		Payload:   "{}",
		Status:    queue.StatusPending,
		CreatedAt: now,
		UpdatedAt: now,
		Metadata:  map[string]string{"chapter_id": "10"},
	}))

	require.NoError(t, driver.CreateJob(ctx, &queue.Job{
		ID:        grandchildID,
		ParentID:  &childID,
		Type:      "pull_page",
		Payload:   "{}",
		Status:    queue.StatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}))

	require.NoError(t, driver.CreateJob(ctx, &queue.Job{
		ID:        otherID,
		Type:      "pull_manga",
		Payload:   "{}",
		Status:    queue.StatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}))

	// Delete root job
	err = driver.DeleteJob(ctx, rootID)
	require.NoError(t, err)

	// Verify root, child, and grandchild are gone
	_, err = driver.GetJob(ctx, rootID)
	require.ErrorIs(t, err, ErrJobNotFound)

	_, err = driver.GetJob(ctx, childID)
	require.ErrorIs(t, err, ErrJobNotFound)

	_, err = driver.GetJob(ctx, grandchildID)
	require.ErrorIs(t, err, ErrJobNotFound)

	// Verify other job is untouched
	otherJob, err := driver.GetJob(ctx, otherID)
	require.NoError(t, err)
	assert.Equal(t, otherID, otherJob.ID)

	// Verify deleting non-existent job returns ErrJobNotFound
	err = driver.DeleteJob(ctx, "non-existent")
	require.ErrorIs(t, err, ErrJobNotFound)
}

func TestSQLiteMultipleStop(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "jobs_multistop.db")

	driver, err := NewDriver(dbPath, 0)
	require.NoError(t, err)
	defer driver.Close()

	ctx := context.Background()

	err = driver.Start(ctx)
	require.NoError(t, err)

	// Call Stop concurrently and multiple times sequentially
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = driver.Stop(ctx)
		}()
	}
	wg.Wait()

	// Calling Stop again sequentially should not panic
	err = driver.Stop(ctx)
	require.NoError(t, err)
}

func TestSQLiteCleanupJobs_StandaloneCompleted(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "jobs_cleanup_completed.db")

	driver, err := NewDriver(dbPath, 0)
	require.NoError(t, err)
	defer driver.Close()

	ctx := context.Background()
	now := time.Now()
	oldTime := now.Add(-2 * time.Hour)
	youngTime := now.Add(-30 * time.Minute)

	// Standalone completed job older than TTL
	oldJob := &queue.Job{
		ID:          "job-completed-old",
		Type:        "test-task",
		Status:      queue.StatusCompleted,
		CreatedAt:   oldTime,
		UpdatedAt:   oldTime,
		CompletedAt: &oldTime,
	}
	require.NoError(t, driver.CreateJob(ctx, oldJob))

	// Standalone completed job younger than TTL
	youngJob := &queue.Job{
		ID:          "job-completed-young",
		Type:        "test-task",
		Status:      queue.StatusCompleted,
		CreatedAt:   youngTime,
		UpdatedAt:   youngTime,
		CompletedAt: &youngTime,
	}
	require.NoError(t, driver.CreateJob(ctx, youngJob))

	pruned, err := driver.CleanupJobs(ctx, queue.CleanupOptions{
		CompletedTTL: 1 * time.Hour,
		FailedTTL:    24 * time.Hour,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, pruned)

	_, err = driver.GetJob(ctx, "job-completed-old")
	require.ErrorIs(t, err, ErrJobNotFound)

	gotYoung, err := driver.GetJob(ctx, "job-completed-young")
	require.NoError(t, err)
	assert.Equal(t, "job-completed-young", gotYoung.ID)
}

func TestSQLiteCleanupJobs_StandalonePendingAndRunning(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "jobs_cleanup_active.db")

	driver, err := NewDriver(dbPath, 0)
	require.NoError(t, err)
	defer driver.Close()

	ctx := context.Background()
	now := time.Now()
	oldTime := now.Add(-2 * time.Hour)

	pendingJob := &queue.Job{
		ID:        "job-pending-old",
		Type:      "test-task",
		Status:    queue.StatusPending,
		CreatedAt: oldTime,
		UpdatedAt: oldTime,
	}
	require.NoError(t, driver.CreateJob(ctx, pendingJob))

	runningJob := &queue.Job{
		ID:        "job-running-old",
		Type:      "test-task",
		Status:    queue.StatusRunning,
		CreatedAt: oldTime,
		UpdatedAt: oldTime,
		StartedAt: &oldTime,
	}
	require.NoError(t, driver.CreateJob(ctx, runningJob))

	pruned, err := driver.CleanupJobs(ctx, queue.CleanupOptions{
		CompletedTTL: 1 * time.Hour,
		FailedTTL:    1 * time.Hour,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, pruned)

	_, err = driver.GetJob(ctx, "job-pending-old")
	require.NoError(t, err)

	_, err = driver.GetJob(ctx, "job-running-old")
	require.NoError(t, err)
}

func TestSQLiteCleanupJobs_StandaloneFailed(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "jobs_cleanup_failed.db")

	driver, err := NewDriver(dbPath, 0)
	require.NoError(t, err)
	defer driver.Close()

	ctx := context.Background()
	now := time.Now()
	failedYoungTime := now.Add(-5 * time.Hour)
	failedOldTime := now.Add(-25 * time.Hour)

	// Failed job younger than FailedTTL (5h < 24h), even though older than CompletedTTL (5h > 1h)
	youngFailedJob := &queue.Job{
		ID:          "job-failed-young",
		Type:        "test-task",
		Status:      queue.StatusFailed,
		CreatedAt:   failedYoungTime,
		UpdatedAt:   failedYoungTime,
		CompletedAt: &failedYoungTime,
	}
	require.NoError(t, driver.CreateJob(ctx, youngFailedJob))

	// Failed job older than FailedTTL (25h > 24h)
	oldFailedJob := &queue.Job{
		ID:          "job-failed-old",
		Type:        "test-task",
		Status:      queue.StatusFailed,
		CreatedAt:   failedOldTime,
		UpdatedAt:   failedOldTime,
		CompletedAt: &failedOldTime,
	}
	require.NoError(t, driver.CreateJob(ctx, oldFailedJob))

	pruned, err := driver.CleanupJobs(ctx, queue.CleanupOptions{
		CompletedTTL: 1 * time.Hour,
		FailedTTL:    24 * time.Hour,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, pruned)

	_, err = driver.GetJob(ctx, "job-failed-old")
	require.ErrorIs(t, err, ErrJobNotFound)

	gotYoung, err := driver.GetJob(ctx, "job-failed-young")
	require.NoError(t, err)
	assert.Equal(t, "job-failed-young", gotYoung.ID)
}

func TestSQLiteCleanupJobs_Hierarchy(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "jobs_cleanup_hierarchy.db")

	driver, err := NewDriver(dbPath, 0)
	require.NoError(t, err)
	defer driver.Close()

	ctx := context.Background()
	now := time.Now()
	oldTime := now.Add(-2 * time.Hour)
	failedYoungTime := now.Add(-5 * time.Hour)

	// Tree 1: Parent completed, but child pending -> whole tree preserved
	p1ID := "tree1-parent"
	require.NoError(t, driver.CreateJob(ctx, &queue.Job{
		ID:          p1ID,
		Type:        "pull_manga",
		Status:      queue.StatusCompleted,
		CreatedAt:   oldTime,
		UpdatedAt:   oldTime,
		CompletedAt: &oldTime,
	}))
	c1ID := "tree1-child"
	require.NoError(t, driver.CreateJob(ctx, &queue.Job{
		ID:        c1ID,
		ParentID:  &p1ID,
		Type:      "pull_chapter",
		Status:    queue.StatusPending,
		CreatedAt: oldTime,
		UpdatedAt: oldTime,
	}))

	// Tree 2: Parent and all children completed, older than CompletedTTL -> whole tree purged
	p2ID := "tree2-parent"
	require.NoError(t, driver.CreateJob(ctx, &queue.Job{
		ID:          p2ID,
		Type:        "pull_manga",
		Status:      queue.StatusCompleted,
		CreatedAt:   oldTime,
		UpdatedAt:   oldTime,
		CompletedAt: &oldTime,
	}))
	c2ID := "tree2-child"
	require.NoError(t, driver.CreateJob(ctx, &queue.Job{
		ID:          c2ID,
		ParentID:  &p2ID,
		Type:        "pull_chapter",
		Status:      queue.StatusCompleted,
		CreatedAt:   oldTime,
		UpdatedAt:   oldTime,
		CompletedAt: &oldTime,
	}))
	g2ID := "tree2-grandchild"
	require.NoError(t, driver.CreateJob(ctx, &queue.Job{
		ID:          g2ID,
		ParentID:  &c2ID,
		Type:        "pull_page",
		Status:      queue.StatusCompleted,
		CreatedAt:   oldTime,
		UpdatedAt:   oldTime,
		CompletedAt: &oldTime,
	}))

	// Tree 3: Parent completed, 1 child failed -> whole tree inherits FailedTTL
	p3ID := "tree3-parent"
	require.NoError(t, driver.CreateJob(ctx, &queue.Job{
		ID:          p3ID,
		Type:        "pull_manga",
		Status:      queue.StatusCompleted,
		CreatedAt:   failedYoungTime,
		UpdatedAt:   failedYoungTime,
		CompletedAt: &failedYoungTime,
	}))
	c3aID := "tree3-child-ok"
	require.NoError(t, driver.CreateJob(ctx, &queue.Job{
		ID:          c3aID,
		ParentID:  &p3ID,
		Type:        "pull_chapter",
		Status:      queue.StatusCompleted,
		CreatedAt:   failedYoungTime,
		UpdatedAt:   failedYoungTime,
		CompletedAt: &failedYoungTime,
	}))
	c3bID := "tree3-child-fail"
	require.NoError(t, driver.CreateJob(ctx, &queue.Job{
		ID:          c3bID,
		ParentID:  &p3ID,
		Type:        "pull_chapter",
		Status:      queue.StatusFailed,
		CreatedAt:   failedYoungTime,
		UpdatedAt:   failedYoungTime,
		CompletedAt: &failedYoungTime,
	}))

	// Pass 1: CompletedTTL = 1h, FailedTTL = 24h.
	// Only Tree 2 should be purged (Tree 1 has pending child, Tree 3 has failed child with 5h < 24h).
	pruned, err := driver.CleanupJobs(ctx, queue.CleanupOptions{
		CompletedTTL: 1 * time.Hour,
		FailedTTL:    24 * time.Hour,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, pruned)

	// Tree 1 preserved
	_, err = driver.GetJob(ctx, p1ID)
	require.NoError(t, err)
	_, err = driver.GetJob(ctx, c1ID)
	require.NoError(t, err)

	// Tree 2 purged
	_, err = driver.GetJob(ctx, p2ID)
	require.ErrorIs(t, err, ErrJobNotFound)
	_, err = driver.GetJob(ctx, c2ID)
	require.ErrorIs(t, err, ErrJobNotFound)
	_, err = driver.GetJob(ctx, g2ID)
	require.ErrorIs(t, err, ErrJobNotFound)

	// Tree 3 preserved
	_, err = driver.GetJob(ctx, p3ID)
	require.NoError(t, err)
	_, err = driver.GetJob(ctx, c3aID)
	require.NoError(t, err)
	_, err = driver.GetJob(ctx, c3bID)
	require.NoError(t, err)

	// Pass 2: FailedTTL = 4h. Tree 3 (5h old) is now older than FailedTTL and should be purged.
	pruned, err = driver.CleanupJobs(ctx, queue.CleanupOptions{
		CompletedTTL: 1 * time.Hour,
		FailedTTL:    4 * time.Hour,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, pruned)

	// Tree 3 purged
	_, err = driver.GetJob(ctx, p3ID)
	require.ErrorIs(t, err, ErrJobNotFound)
	_, err = driver.GetJob(ctx, c3aID)
	require.ErrorIs(t, err, ErrJobNotFound)
	_, err = driver.GetJob(ctx, c3bID)
	require.ErrorIs(t, err, ErrJobNotFound)

	// Tree 1 STILL preserved
	_, err = driver.GetJob(ctx, p1ID)
	require.NoError(t, err)
	_, err = driver.GetJob(ctx, c1ID)
	require.NoError(t, err)
}

func TestSQLiteCleanupJobs_BatchSize(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "jobs_cleanup_batch.db")

	driver, err := NewDriver(dbPath, 0)
	require.NoError(t, err)
	defer driver.Close()

	ctx := context.Background()
	now := time.Now()
	oldTime := now.Add(-2 * time.Hour)

	for i := 0; i < 10; i++ {
		job := &queue.Job{
			ID:          filepath.Join("batch-job-", string(rune('a'+i))),
			Type:        "test-task",
			Status:      queue.StatusCompleted,
			CreatedAt:   oldTime,
			UpdatedAt:   oldTime,
			CompletedAt: &oldTime,
		}
		require.NoError(t, driver.CreateJob(ctx, job))
	}

	pruned, err := driver.CleanupJobs(ctx, queue.CleanupOptions{
		CompletedTTL: 1 * time.Hour,
		FailedTTL:    24 * time.Hour,
		BatchSize:    3,
	})
	require.NoError(t, err)
	assert.Equal(t, 3, pruned)

	remaining, err := driver.ListJobs(ctx, queue.JobFilter{All: true})
	require.NoError(t, err)
	assert.Len(t, remaining, 7)
}

func TestSQLiteAutoCleanupTicker(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "jobs_autocleanup.db")

	driver, err := NewDriver(dbPath, 0)
	require.NoError(t, err)
	defer driver.Close()

	ctx := context.Background()
	oldTime := time.Now().Add(-2 * time.Hour)

	driver.SetCleanupConfig(20*time.Millisecond, queue.CleanupOptions{
		CompletedTTL: 1 * time.Hour,
		FailedTTL:    24 * time.Hour,
	})

	job := &queue.Job{
		ID:          "job-autoclean",
		Type:        "test-task",
		Status:      queue.StatusCompleted,
		CreatedAt:   oldTime,
		UpdatedAt:   oldTime,
		CompletedAt: &oldTime,
	}
	require.NoError(t, driver.CreateJob(ctx, job))

	err = driver.Start(ctx)
	require.NoError(t, err)

	assert.Eventually(t, func() bool {
		_, err := driver.GetJob(ctx, "job-autoclean")
		return errors.Is(err, ErrJobNotFound)
	}, 2*time.Second, 20*time.Millisecond)

	err = driver.Stop(ctx)
	require.NoError(t, err)
}
