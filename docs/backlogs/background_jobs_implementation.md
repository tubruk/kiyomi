# Implementation Plan: Background Job Queue

> **Status**: Proposed / Backlog  
> **Target Components**: `internal/queue`, `internal/api`, `web/src`  
> **Reference Design**: [background_jobs.md](../design/background_jobs.md)

---

## Overview

This document outlines the concrete implementation plan for the generic Background Job Queue system in Kiyomi. It specifies how we leverage the `Enqueuer` and `Worker` abstractions to support multiple queue drivers via configuration:
1. **SQLite (Default)**: A lightweight, persistent, zero-dependency driver utilizing the application's existing SQLite database. Recommended for local desktop/standalone usage.
2. **Machinery (Redis)**: A robust distributed task queue driver using `github.com/RichardKnop/machinery`. Recommended for production or containerized environments.
3. **InMemory**: A Go channel-based non-persistent driver ideal for testing and development.

---

## 1. Directory Structure

We will isolate the background jobs logic inside a new package under `internal/queue/`.

```
internal/queue/
├── queue.go              # Shared interfaces (Enqueuer, Worker, JobHandler) and Job model
├── registry.go           # Handler registry and routing logic
├── inmemory/             # Lightweight fallback driver (for testing/development)
│   └── driver.go
├── sqlite/               # Native SQLite-backed persistent driver (Default)
│   └── driver.go
└── machinery/            # Machinery-based driver (Redis)
    ├── driver.go         # Implementation of Enqueuer/Worker wrapping Machinery
    └── config.go         # Broker and backend configuration (Redis)
```

---

## 2. Interface Definitions & Abstraction

To ensure decoupling, the codebase will only interact with the `Enqueuer` and `Worker` interfaces.

### `internal/queue/queue.go`
```go
package queue

import (
	"context"
	"time"
)

type JobStatus string

const (
	StatusPending   JobStatus = "pending"
	StatusRunning   JobStatus = "running"
	StatusCompleted JobStatus = "completed"
	StatusFailed    JobStatus = "failed"
)

type Job struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`
	Payload     []byte            `json:"payload"`
	Status      JobStatus         `json:"status"`
	MaxRetries  int               `json:"max_retries"`
	RetryCount  int               `json:"retry_count"`
	ErrorLog    string            `json:"error_log,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	SubmittedAt time.Time         `json:"submitted_at"`
	StartedAt   *time.Time        `json:"started_at,omitempty"`
	FinishedAt  *time.Time        `json:"finished_at,omitempty"`
}

type JobHandler interface {
	Handle(ctx context.Context, payload []byte) error
}

// JobStore defines the repository interface for persisting and querying job states.
type JobStore interface {
	Create(ctx context.Context, job *Job) error
	Get(ctx context.Context, id string) (*Job, error)
	Update(ctx context.Context, job *Job) error
	List(ctx context.Context, filter map[string]string) ([]*Job, error)
}

type Enqueuer interface {
	Enqueue(ctx context.Context, job *Job) error
}

type Worker interface {
	Register(jobType string, handler JobHandler) error
	Start(ctx context.Context) error
	Stop() error
}
```

---

## 3. SQLite Driver Implementation (Pure Go)

To keep the application zero-dependency and easy to cross-compile, the default SQLite queue driver must use a **pure Go SQLite library** (such as `modernc.org/sqlite` or `github.com/ncruces/go-sqlite3`) instead of CGO-based options like `go-sqlite3`.

### Dedicated SQLite Database File
To prevent database locking contention and keep different system boundaries clean, the background worker infrastructure will run on its own **dedicated SQLite database file** (e.g., `jobs.db`) rather than sharing the database with Kiyomi's other features (such as library indexing and metadata storage).

### Schema Design
A `jobs` database table will be initialized:
```sql
CREATE TABLE IF NOT EXISTS jobs (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    payload BLOB NOT NULL,
    status TEXT NOT NULL CHECK(status IN ('pending', 'running', 'completed', 'failed')),
    max_retries INTEGER NOT NULL DEFAULT 0,
    retry_count INTEGER NOT NULL DEFAULT 0,
    error_log TEXT,
    submitted_at DATETIME NOT NULL,
    started_at DATETIME,
    finished_at DATETIME,
    next_run_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_jobs_polling ON jobs (status, next_run_at);
CREATE INDEX IF NOT EXISTS idx_jobs_submitted_at ON jobs (submitted_at);

CREATE TABLE IF NOT EXISTS job_metadata (
    job_id TEXT NOT NULL,
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    PRIMARY KEY (job_id, key),
    FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_job_metadata_lookup ON job_metadata (key, value);
```

### Database Initialization
The `jobs.db` database schema is initialized on application startup by executing a hardcoded SQL creation script. Schema migrations or updates are handled via verification queries checks (e.g. checking column existence) and executing incremental SQL statements.

### Worker Concurrency & Safety
1. **WAL Mode**: Ensure database connections enable WAL (Write-Ahead Logging) and set a busy timeout (e.g. `_busy_timeout=5000`) so concurrent readers/writers do not deadlock.
2. **Centralized Coordinator**: The worker implementation utilizes a single polling ticker in a select loop that selects the next pending jobs, updates their state to `running` using a transaction (`BEGIN IMMEDIATE`), and distributes them to Go worker channels. This avoids multiple worker threads competing for database locks.
3. **Concurrency Group Keys**: While Machinery enforces task rate-limiting and routing natively, the local SQLite worker enforces concurrency group keys using a simple in-memory map inside the coordinator. Before claiming a job, the coordinator checks if the job's group key has reached its concurrent limit.
4. **Graceful Shutdown**: The worker pool implements a default 30-second graceful shutdown timeout. Active worker contexts are canceled if tasks exceed this window.

---

## 4. Machinery Driver Implementation

The Machinery driver will wrap the Machinery server, map jobs, and run workers.

### Configuration
For a self-contained SQLite-based application like Kiyomi, the Machinery driver will support two modes depending on configuration:
1. **SQLite / SQL Broker & Backend**: Uses the relational database to store tasks and states.
2. **Redis Broker & Backend**: For higher throughput or production environments where Redis is available.

### Wrapper Adapter
```go
package machinery

import (
	"context"
	"encoding/json"
	
	"github.com/RichardKnop/machinery/v1"
	machineryConfig "github.com/RichardKnop/machinery/v1/config"
	"github.com/RichardKnop/machinery/v1/tasks"
	
	"kiyomi/internal/queue"
)

type Driver struct {
	server *machinery.Server
}

func NewDriver(cfg *machineryConfig.Config) (*Driver, error) {
	server, err := machinery.NewServer(cfg)
	if err != nil {
		return nil, err
	}
	return &Driver{server: server}, nil
}

func (d *Driver) Enqueue(ctx context.Context, job *queue.Job) error {
	signature := &tasks.Signature{
		UUID: job.ID,
		Name: job.Type,
		Args: []tasks.Arg{
			{
				Type:  "string",
				Value: string(job.Payload),
			},
		},
		RetryCount: job.MaxRetries,
	}
	_, err := d.server.SendTaskWithContext(ctx, signature)
	return err
}

func (d *Driver) Register(jobType string, handler queue.JobHandler) error {
	// Adapter function matching Machinery's expectation: func(string) error
	machineryFunc := func(payloadStr string) error {
		return handler.Handle(context.Background(), []byte(payloadStr))
	}
	return d.server.RegisterTask(jobType, machineryFunc)
}

func (d *Driver) Start(ctx context.Context) error {
	worker := d.server.NewWorker("kiyomi-worker-pool", 0)
	return worker.Launch()
}

func (d *Driver) Stop() error {
	// Triggers graceful shutdown of the worker pool
	return nil
}
```

---

## 5. Implementation Roadmap

### Core Queue & Drivers (`internal/queue`)
- Setup package directory and define shared interfaces.
- Implement the native, persistent `sqlite` driver to serve as the default zero-dependency local queue.
- Implement the `machinery` driver wrapper (Redis broker & backend) for production deployments.
- Integrate a simple `inmemory` driver to facilitate unit testing without external dependencies.
- Write robust unit tests verifying behavior across all three drivers.

### Dependency Injection & Configuration Setup
- Add configuration settings for background jobs (e.g. choice of driver, DB connections, concurrent worker limits).
- Wire the queue driver into the dependency injection lifecycle.
- Implement the registration system to bind job types (like `download_page`) to their handlers.

### REST API Handlers (`internal/api`)
- Implement `GET /api/v1/jobs` to fetch job states. Support filtering jobs by status, type, and arbitrary metadata (e.g., manga ID).
- Implement `POST /api/v1/jobs` to dynamically submit jobs.
- Implement `DELETE /api/v1/jobs/:id` to cancel pending/running jobs.
- Write integration tests for all background job API endpoints.

### Frontend Job Monitor Component (`web/src`)
- Create a user interface dashboard using shadcn/ui showing:
  - Total active, pending, and failed jobs.
  - Progress bar/status for active jobs.
  - Job history with failure logs.
  - Buttons to cancel pending jobs or retry failed ones.
