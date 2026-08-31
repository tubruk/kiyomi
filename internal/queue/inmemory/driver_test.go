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

func TestInMemoryParentChildHierarchyAndFiltering(t *testing.T) {
	driver := NewDriver(10)
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

func TestInMemoryRecursiveCancelJob(t *testing.T) {
	driver := NewDriver(10)
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
	err := driver.CancelJob(ctx, "non-existent")
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

func TestInMemoryHandlerPanicRecovery(t *testing.T) {
	driver := NewDriver(10)
	ctx := context.Background()

	var mu sync.Mutex
	processed := make(map[string]bool)

	err := driver.Register("panic-task", queue.JobHandlerFunc(func(ctx context.Context, job *queue.Job) error {
		mu.Lock()
		processed[job.ID] = true
		mu.Unlock()
		panic("inmemory panic")
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
	}
	err = driver.Enqueue(ctx, panicJob)
	require.NoError(t, err)

	assert.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return processed["job-panic-1"]
	}, 3*time.Second, 20*time.Millisecond)

	assert.Eventually(t, func() bool {
		j, err := driver.GetJob(ctx, "job-panic-1")
		if err != nil {
			return false
		}
		return j.Status == queue.StatusFailed && j.Error == "panic in job handler: inmemory panic"
	}, 3*time.Second, 20*time.Millisecond)

	// Worker loop should still be alive to process normal jobs
	normalJob := &queue.Job{
		ID:         "job-normal-1",
		Type:       "normal-task",
		Status:     queue.StatusPending,
		MaxRetries: 1,
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

func TestInMemoryCancelJobWhileRunningNotOverwritten(t *testing.T) {
	driver := NewDriver(10)
	ctx := context.Background()

	runningChan := make(chan struct{})
	releaseChan := make(chan struct{})

	err := driver.Register("slow-cancel", queue.JobHandlerFunc(func(ctx context.Context, job *queue.Job) error {
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

	// Wait a bit for processJob to finish
	time.Sleep(50 * time.Millisecond)

	// Verify job is STILL failed and error was NOT overwritten
	jAfter, err := driver.GetJob(ctx, job.ID)
	require.NoError(t, err)
	assert.Equal(t, queue.StatusFailed, jAfter.Status)
	assert.Equal(t, "job cancelled by user", jAfter.Error)

	err = driver.Stop(ctx)
	require.NoError(t, err)
}

func TestInMemoryDeleteJobCascadeAndCycles(t *testing.T) {
	driver := NewDriver(10)
	ctx := context.Background()

	rootID := "job-root-del"
	childID := "job-child-del"
	grandchildID := "job-grandchild-del"

	require.NoError(t, driver.CreateJob(ctx, &queue.Job{
		ID:     rootID,
		Type:   "pull_manga",
		Status: queue.StatusPending,
	}))

	require.NoError(t, driver.CreateJob(ctx, &queue.Job{
		ID:       childID,
		ParentID: &rootID,
		Type:     "pull_chapter",
		Status:   queue.StatusPending,
	}))

	require.NoError(t, driver.CreateJob(ctx, &queue.Job{
		ID:       grandchildID,
		ParentID: &childID,
		Type:     "pull_page",
		Status:   queue.StatusPending,
	}))

	// Delete root job
	err := driver.DeleteJob(ctx, rootID)
	require.NoError(t, err)

	// Verify all 3 are deleted
	_, err = driver.GetJob(ctx, rootID)
	require.ErrorIs(t, err, ErrJobNotFound)
	_, err = driver.GetJob(ctx, childID)
	require.ErrorIs(t, err, ErrJobNotFound)
	_, err = driver.GetJob(ctx, grandchildID)
	require.ErrorIs(t, err, ErrJobNotFound)
}

func TestInMemoryMultipleStopAndEnqueueAfterStop(t *testing.T) {
	driver := NewDriver(10)
	ctx := context.Background()

	err := driver.Start(ctx)
	require.NoError(t, err)

	// Multiple concurrent stops
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = driver.Stop(ctx)
		}()
	}
	wg.Wait()

	// Calling Stop again should not panic
	err = driver.Stop(ctx)
	require.NoError(t, err)

	// Calling Enqueue after Stop returns an error
	err = driver.Enqueue(ctx, &queue.Job{
		ID:     "job-after-stop",
		Type:   "test",
		Status: queue.StatusPending,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "queue driver is stopped")
}

func TestInMemoryCleanupJobs_StandaloneCompleted(t *testing.T) {
	driver := NewDriver(10)
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

func TestInMemoryCleanupJobs_StandalonePendingAndRunning(t *testing.T) {
	driver := NewDriver(10)
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

func TestInMemoryCleanupJobs_StandaloneFailed(t *testing.T) {
	driver := NewDriver(10)
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

func TestInMemoryCleanupJobs_Hierarchy(t *testing.T) {
	driver := NewDriver(10)
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

func TestInMemoryCleanupJobs_BatchSize(t *testing.T) {
	driver := NewDriver(10)
	ctx := context.Background()
	now := time.Now()
	oldTime := now.Add(-2 * time.Hour)

	for i := 0; i < 10; i++ {
		job := &queue.Job{
			ID:          string(rune('a' + i)),
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
