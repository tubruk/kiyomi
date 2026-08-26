package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"

	"github.com/tubruk/kiyomi/internal/queue"
)

var (
	ErrJobNotFound     = errors.New("job not found")
	ErrHandlerConflict = errors.New("handler already registered for this job type")
)

// Driver implements queue.JobStore, queue.Enqueuer, and queue.Worker interfaces using SQLite.
type Driver struct {
	db             *sql.DB
	handlers       map[string]queue.JobHandler
	handlersMu     sync.RWMutex
	activeGroups   map[string]int
	groupLimits    map[string]int
	activeMu       sync.Mutex
	wg             sync.WaitGroup
	stopChan       chan struct{}
	pollTrigger    chan struct{}
	pollInterval   time.Duration
	maxConcurrency int
	runningCount   int
}

// NewDriver initializes a new SQLite queue driver connected to a dedicated jobs.db file.
func NewDriver(dbPath string, maxConcurrency int) (*Driver, error) {
	// Ensure parent directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	// DS with WAL mode and busy timeout
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys=ON", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}
	db.SetMaxOpenConns(1)

	// Double check pragmas
	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable WAL: %w", err)
	}
	if _, err := db.Exec("PRAGMA busy_timeout=5000;"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("set busy timeout: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON;"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	// Create tables
	schema := `
	CREATE TABLE IF NOT EXISTS jobs (
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
	CREATE TABLE IF NOT EXISTS job_metadata (
		job_id TEXT NOT NULL,
		key TEXT NOT NULL,
		value TEXT NOT NULL,
		PRIMARY KEY (job_id, key),
		FOREIGN KEY (job_id) REFERENCES jobs (id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
	`
	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("create tables: %w", err)
	}

	return &Driver{
		db:             db,
		handlers:       make(map[string]queue.JobHandler),
		activeGroups:   make(map[string]int),
		groupLimits:    make(map[string]int),
		stopChan:       make(chan struct{}),
		pollTrigger:    make(chan struct{}, 64),
		pollInterval:   100 * time.Millisecond,
		maxConcurrency: maxConcurrency,
	}, nil
}

// Close closes the underlying SQLite database connection.
func (d *Driver) Close() error {
	return d.db.Close()
}

// SetGroupLimit sets the concurrency limit for a group.
func (d *Driver) SetGroupLimit(group string, limit int) {
	d.activeMu.Lock()
	defer d.activeMu.Unlock()
	d.groupLimits[group] = limit
}

// CreateJob persists a new job to the SQLite database.
func (d *Driver) CreateJob(ctx context.Context, job *queue.Job) error {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(ctx,
		`INSERT INTO jobs (id, type, payload, status, retries, max_retries, concurrency_group, error, created_at, updated_at, started_at, completed_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		job.ID, job.Type, job.Payload, string(job.Status), job.Retries, job.MaxRetries, job.ConcurrencyGroup,
		job.Error, formatTime(&job.CreatedAt), formatTime(&job.UpdatedAt), formatTime(job.StartedAt), formatTime(job.CompletedAt),
	)
	if err != nil {
		return fmt.Errorf("insert job: %w", err)
	}

	for k, v := range job.Metadata {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO job_metadata (job_id, key, value) VALUES (?, ?, ?)`,
			job.ID, k, v,
		)
		if err != nil {
			return fmt.Errorf("insert metadata: %w", err)
		}
	}

	return tx.Commit()
}

// GetJob retrieves a job from the database.
func (d *Driver) GetJob(ctx context.Context, id string) (*queue.Job, error) {
	row := d.db.QueryRowContext(ctx,
		`SELECT id, type, payload, status, retries, max_retries, concurrency_group, error, created_at, updated_at, started_at, completed_at 
		 FROM jobs WHERE id = ?`, id)

	var j queue.Job
	var statusStr, createdAtStr, updatedAtStr string
	var startedAtStr, completedAtStr, errorStr sql.NullString

	err := row.Scan(&j.ID, &j.Type, &j.Payload, &statusStr, &j.Retries, &j.MaxRetries, &j.ConcurrencyGroup,
		&errorStr, &createdAtStr, &updatedAtStr, &startedAtStr, &completedAtStr)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrJobNotFound
	} else if err != nil {
		return nil, err
	}

	j.Status = queue.JobStatus(statusStr)
	if errorStr.Valid {
		j.Error = errorStr.String
	}
	j.CreatedAt = *parseTime(&createdAtStr)
	j.UpdatedAt = *parseTime(&updatedAtStr)
	if startedAtStr.Valid {
		j.StartedAt = parseTime(&startedAtStr.String)
	}
	if completedAtStr.Valid {
		j.CompletedAt = parseTime(&completedAtStr.String)
	}

	// Fetch metadata
	rows, err := d.db.QueryContext(ctx, `SELECT key, value FROM job_metadata WHERE job_id = ?`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	j.Metadata = make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		j.Metadata[k] = v
	}

	return &j, nil
}

// UpdateJob updates the job in the database.
func (d *Driver) UpdateJob(ctx context.Context, job *queue.Job) error {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx,
		`UPDATE jobs SET type = ?, payload = ?, status = ?, retries = ?, max_retries = ?, concurrency_group = ?, error = ?, 
		                created_at = ?, updated_at = ?, started_at = ?, completed_at = ?
		 WHERE id = ?`,
		job.Type, job.Payload, string(job.Status), job.Retries, job.MaxRetries, job.ConcurrencyGroup, job.Error,
		formatTime(&job.CreatedAt), formatTime(&job.UpdatedAt), formatTime(job.StartedAt), formatTime(job.CompletedAt), job.ID,
	)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrJobNotFound
	}

	// Update metadata
	_, err = tx.ExecContext(ctx, `DELETE FROM job_metadata WHERE job_id = ?`, job.ID)
	if err != nil {
		return err
	}

	for k, v := range job.Metadata {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO job_metadata (job_id, key, value) VALUES (?, ?, ?)`,
			job.ID, k, v,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// DeleteJob deletes a job and its metadata from the database.
func (d *Driver) DeleteJob(ctx context.Context, id string) error {
	res, err := d.db.ExecContext(ctx, `DELETE FROM jobs WHERE id = ?`, id)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrJobNotFound
	}
	return nil
}

// ListJobs returns jobs matching the filter criteria.
func (d *Driver) ListJobs(ctx context.Context, filter queue.JobFilter) ([]*queue.Job, error) {
	query := `SELECT id, type, payload, status, retries, max_retries, concurrency_group, error, created_at, updated_at, started_at, completed_at FROM jobs j WHERE 1=1`
	var args []interface{}

	if filter.Status != "" {
		query += ` AND j.status = ?`
		args = append(args, string(filter.Status))
	}
	if filter.Type != "" {
		query += ` AND j.type = ?`
		args = append(args, filter.Type)
	}

	for k, v := range filter.Metadata {
		query += ` AND EXISTS (SELECT 1 FROM job_metadata m WHERE m.job_id = j.id AND m.key = ? AND m.value = ?)`
		args = append(args, k, v)
	}

	query += ` ORDER BY created_at DESC`

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*queue.Job
	for rows.Next() {
		var j queue.Job
		var statusStr, createdAtStr, updatedAtStr string
		var startedAtStr, completedAtStr, errorStr sql.NullString

		err := rows.Scan(&j.ID, &j.Type, &j.Payload, &statusStr, &j.Retries, &j.MaxRetries, &j.ConcurrencyGroup,
			&errorStr, &createdAtStr, &updatedAtStr, &startedAtStr, &completedAtStr)
		if err != nil {
			return nil, err
		}

		j.Status = queue.JobStatus(statusStr)
		if errorStr.Valid {
			j.Error = errorStr.String
		}
		j.CreatedAt = *parseTime(&createdAtStr)
		j.UpdatedAt = *parseTime(&updatedAtStr)
		if startedAtStr.Valid {
			j.StartedAt = parseTime(&startedAtStr.String)
		}
		if completedAtStr.Valid {
			j.CompletedAt = parseTime(&completedAtStr.String)
		}
		jobs = append(jobs, &j)
	}

	// Populate metadata for all returned jobs
	for _, j := range jobs {
		mRows, err := d.db.QueryContext(ctx, `SELECT key, value FROM job_metadata WHERE job_id = ?`, j.ID)
		if err != nil {
			return nil, err
		}
		j.Metadata = make(map[string]string)
		for mRows.Next() {
			var k, v string
			if err := mRows.Scan(&k, &v); err != nil {
				mRows.Close()
				return nil, err
			}
			j.Metadata[k] = v
		}
		mRows.Close()
	}

	return jobs, nil
}

// Enqueue marks the job as pending and schedules it for processing.
func (d *Driver) Enqueue(ctx context.Context, job *queue.Job) error {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var exists bool
	err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM jobs WHERE id = ?)`, job.ID).Scan(&exists)
	if err != nil {
		return err
	}

	now := time.Now()
	job.Status = queue.StatusPending
	job.UpdatedAt = now

	if !exists {
		job.CreatedAt = now
		_, err = tx.ExecContext(ctx,
			`INSERT INTO jobs (id, type, payload, status, retries, max_retries, concurrency_group, error, created_at, updated_at, started_at, completed_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			job.ID, job.Type, job.Payload, string(job.Status), job.Retries, job.MaxRetries, job.ConcurrencyGroup,
			job.Error, formatTime(&job.CreatedAt), formatTime(&job.UpdatedAt), formatTime(job.StartedAt), formatTime(job.CompletedAt),
		)
		if err != nil {
			return err
		}
		for k, v := range job.Metadata {
			_, err = tx.ExecContext(ctx,
				`INSERT INTO job_metadata (job_id, key, value) VALUES (?, ?, ?)`,
				job.ID, k, v,
			)
			if err != nil {
				return err
			}
		}
	} else {
		_, err = tx.ExecContext(ctx,
			`UPDATE jobs SET status = ?, updated_at = ?, error = ? WHERE id = ?`,
			string(queue.StatusPending), formatTime(&now), nil, job.ID,
		)
		if err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// Trigger poller
	select {
	case d.pollTrigger <- struct{}{}:
	default:
	}

	return nil
}

// Register maps a handler to a job type.
func (d *Driver) Register(jobType string, handler queue.JobHandler) error {
	d.handlersMu.Lock()
	defer d.handlersMu.Unlock()

	if _, exists := d.handlers[jobType]; exists {
		return ErrHandlerConflict
	}
	d.handlers[jobType] = handler
	return nil
}

// Start runs the single-poller coordinator loop.
func (d *Driver) Start(ctx context.Context) error {
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		ticker := time.NewTicker(d.pollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-d.stopChan:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				d.pollAndDispatch(ctx)
			case <-d.pollTrigger:
				d.pollAndDispatch(ctx)
			}
		}
	}()
	return nil
}

// Stop handles graceful shutdown with a 30-second timeout.
func (d *Driver) Stop(ctx context.Context) error {
	close(d.stopChan)

	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		d.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-shutdownCtx.Done():
		return shutdownCtx.Err()
	}
}

func (d *Driver) pollAndDispatch(ctx context.Context) {
	d.activeMu.Lock()
	defer d.activeMu.Unlock()

	if d.maxConcurrency > 0 && d.runningCount >= d.maxConcurrency {
		return
	}

	conn, err := d.db.Conn(ctx)
	if err != nil {
		return
	}
	defer conn.Close()

	_, err = conn.ExecContext(ctx, "BEGIN IMMEDIATE;")
	if err != nil {
		return
	}

	committed := false
	defer func() {
		if !committed {
			_, _ = conn.ExecContext(ctx, "ROLLBACK;")
		}
	}()

	// Fetch pending jobs
	rows, err := conn.QueryContext(ctx,
		`SELECT id, type, payload, status, retries, max_retries, concurrency_group FROM jobs WHERE status = 'pending' ORDER BY created_at ASC`)
	if err != nil {
		return
	}
	defer rows.Close()

	var jobsToRun []*queue.Job
	for rows.Next() {
		var j queue.Job
		var statusStr string
		if err := rows.Scan(&j.ID, &j.Type, &j.Payload, &statusStr, &j.Retries, &j.MaxRetries, &j.ConcurrencyGroup); err != nil {
			return
		}
		j.Status = queue.JobStatus(statusStr)

		// Check handler registration
		d.handlersMu.RLock()
		_, hasHandler := d.handlers[j.Type]
		d.handlersMu.RUnlock()

		if !hasHandler {
			// Job has no registered handler, claim it and fail it immediately
			now := time.Now()
			_, _ = conn.ExecContext(ctx,
				`UPDATE jobs SET status = ?, error = ?, updated_at = ?, completed_at = ? WHERE id = ?`,
				string(queue.StatusFailed), fmt.Sprintf("no handler registered for job type %s", j.Type),
				formatTime(&now), formatTime(&now), j.ID,
			)
			continue
		}

		if d.maxConcurrency > 0 && d.runningCount >= d.maxConcurrency {
			break
		}

		// Check concurrency group limit
		if j.ConcurrencyGroup != "" {
			if limit, exists := d.groupLimits[j.ConcurrencyGroup]; exists && limit > 0 {
				if d.activeGroups[j.ConcurrencyGroup] >= limit {
					// Skip this job for now as the concurrency limit is reached
					continue
				}
			}
		}

		// Claim the job
		now := time.Now()
		_, err = conn.ExecContext(ctx,
			`UPDATE jobs SET status = ?, started_at = ?, updated_at = ? WHERE id = ?`,
			string(queue.StatusRunning), formatTime(&now), formatTime(&now), j.ID,
		)
		if err != nil {
			return
		}

		if j.ConcurrencyGroup != "" {
			d.activeGroups[j.ConcurrencyGroup]++
		}
		d.runningCount++
		jobsToRun = append(jobsToRun, &j)
	}
	rows.Close()

	_, err = conn.ExecContext(ctx, "COMMIT;")
	if err != nil {
		// Roll back our in-memory updates since tx commit failed
		for _, j := range jobsToRun {
			if j.ConcurrencyGroup != "" {
				d.activeGroups[j.ConcurrencyGroup]--
			}
			d.runningCount--
		}
		return
	}
	committed = true

	// Dispatch enqueued jobs to goroutines
	for _, j := range jobsToRun {
		d.wg.Add(1)
		go d.runJob(j)
	}
}

func (d *Driver) runJob(j *queue.Job) {
	defer d.wg.Done()

	d.handlersMu.RLock()
	handler := d.handlers[j.Type]
	d.handlersMu.RUnlock()

	// Fetch full job detail (with metadata)
	fullJob, err := d.GetJob(context.Background(), j.ID)
	if err != nil {
		d.finalizeJob(j, err)
		return
	}

	execErr := handler.Handle(context.Background(), fullJob)
	d.finalizeJob(j, execErr)
}

func (d *Driver) finalizeJob(j *queue.Job, execErr error) {
	d.activeMu.Lock()
	defer d.activeMu.Unlock()

	now := time.Now()
	ctx := context.Background()

	if execErr != nil {
		j.Retries++
		permanent := queue.IsPermanent(execErr)
		if permanent || j.Retries >= j.MaxRetries {
			_, _ = d.db.ExecContext(ctx,
				`UPDATE jobs SET status = ?, retries = ?, error = ?, updated_at = ?, completed_at = ? WHERE id = ?`,
				string(queue.StatusFailed), j.Retries, execErr.Error(), formatTime(&now), formatTime(&now), j.ID,
			)
		} else {
			_, _ = d.db.ExecContext(ctx,
				`UPDATE jobs SET status = ?, retries = ?, updated_at = ? WHERE id = ?`,
				string(queue.StatusPending), j.Retries, formatTime(&now), j.ID,
			)
		}
	} else {
		_, _ = d.db.ExecContext(ctx,
			`UPDATE jobs SET status = ?, updated_at = ?, completed_at = ? WHERE id = ?`,
			string(queue.StatusCompleted), formatTime(&now), formatTime(&now), j.ID,
		)
	}

	if j.ConcurrencyGroup != "" {
		d.activeGroups[j.ConcurrencyGroup]--
	}
	d.runningCount--

	// Trigger poller since a slot/job became free
	select {
	case d.pollTrigger <- struct{}{}:
	default:
	}
}

func formatTime(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.Format(time.RFC3339Nano)
}

func parseTime(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339Nano, *s)
	if err != nil {
		t, err = time.Parse(time.RFC3339, *s)
		if err != nil {
			return nil
		}
	}
	return &t
}
