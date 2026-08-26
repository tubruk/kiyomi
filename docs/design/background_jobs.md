# Generic Background Job Queue System

## Overview
Kiyomi requires a generic, extensible **Background Job Queue** capable of running tasks asynchronously. While its initial driver is the granular content pull feature, the runner is decoupled from pull-specific logic to support general background maintenance tasks (such as library scans, metadata enrichment, cache eviction, and tracking synchronization).

---

## 1. Core Architecture

The background runner consists of a central scheduler, a task queue, and registered handlers.

```
                  ┌──────────────────────┐
                  │   Client API / UI    │
                  └──────────┬───────────┘
                             │ (submit / status)
                             ▼
                  ┌──────────────────────┐
                  │    Job Scheduler     │
                  └──────────┬───────────┘
                             │
                             ▼
                  ┌──────────────────────┐
                  │   Generic Queue      │◄─── (Restores from jobs.db on startup)
                  └──────────┬───────────┘
                             │
            ┌────────────────┴────────────────┐
            ▼ (matches Type to Handler)       ▼
     [ JobHandler A ]                  [ JobHandler B ]
  (e.g. PullPageHandler)             (e.g. LibraryScanHandler)
```

### Generic Job Schema
All tasks submitted to the runner are defined by a generic struct:

```go
type JobStatus string

const (
    StatusPending   JobStatus = "pending"
    StatusRunning   JobStatus = "running"
    StatusCompleted JobStatus = "completed"
    StatusFailed    JobStatus = "failed"
)

type Job struct {
    ID          string          `json:"id"`
    Type        string          `json:"type"`       // e.g., "pull_page", "library_scan"
    Payload     json.RawMessage `json:"payload"`    // Type-specific arguments
    Status      JobStatus       `json:"status"`
    MaxRetries  int             `json:"max_retries"`
    RetryCount  int             `json:"retry_count"`
    ErrorLog    string            `json:"error_log,omitempty"`
    Metadata    map[string]string `json:"metadata,omitempty"` // Queryable metadata tags
    SubmittedAt time.Time         `json:"submitted_at"`
    StartedAt   *time.Time        `json:"started_at,omitempty"`
    FinishedAt  *time.Time        `json:"finished_at,omitempty"`
}
```

### Pluggable Handler Interface
Each task type registers a `JobHandler` with the scheduler:

```go
type JobHandler interface {
    Handle(ctx context.Context, payload []byte) error
}
```

### Queue Abstractions

To decouple Kiyomi from specific queue libraries or storage backends, the system interacts with background processing via abstract interfaces:

```go
// Enqueuer defines the interface for submitting background jobs to the queue.
type Enqueuer interface {
    Enqueue(ctx context.Context, job *Job) error
}

// Worker defines the interface for managing the consumer worker pool.
type Worker interface {
    // Register registers a handler for a specific job type.
    Register(jobType string, handler JobHandler) error
    // Start begins processing jobs from the queue.
    Start(ctx context.Context) error
    // Stop gracefully stops the worker pool.
    Stop() error
}
```

This abstraction allows switching the underlying queue driver (e.g., in-memory channel, SQLite/file-based queue, or distributed library such as machinery) without modifying the core business logic that schedules or executes jobs. When using the SQLite driver, it utilizes a dedicated database file (`jobs.db`) to isolate background job states from the main application's library indexing and metadata storage.

---

## 2. Supported Job Implementations

The generic queue executes various background operations by mapping job types to specific handlers:

### A. Content Acquisition
* **`pull_page`**: Fetches a single page image file from the upstream provider and writes it to `<library_root>/<manga_id>/<provider_id>/<chapter_id>/`.
* **`pull_chapter`**: Resolves page lists from a content provider and enqueues individual `pull_page` jobs.
* **`pull_manga`**: Fetches the content provider's chapter list and schedules pulls for all missing chapters.

> **Naming rationale**: "Pull" is used instead of "download" to distinguish server-to-upstream acquisition from client-initiated downloads. "Download" implies the client is pulling a file to their device. "Pull" conveys the server fetching content from an external source into the local library, avoiding user confusion about the direction of data flow.

### B. Library Maintenance
* **`library_scan`**: Scans `<library_root>` folders, verifies manifest integrity, checks for numbering gaps, and indexes metadata.
* **`prune_orphans`**: Cleans up directories on disk that are empty or no longer referenced in the library metadata.
* **`library_export`**: Packs a downloaded chapter's folder into a standard `.cbz` archive for local export.

### C. Metadata & Media Optimization
* **`metadata_refresh`**: Pulls updated ratings, synopses, genres, and aliases from metadata providers (MAL, Kitsu, AniList).
* **`media_optimization`**: Post-processes downloaded pages (e.g., converting heavy PNGs to WebP or resizing images) to save disk space.

### D. External Tracking Integration
* **`tracker_sync_push`**: Asynchronously pushes read progress updates to external tracking platforms. If the external tracker's API is down, the job retries with backoff.
* **`tracker_sync_pull`**: Periodically imports list updates (e.g. "Plan to read" shifts) from MyAnimeList/AniList into the library.

### E. Cache & Prefetching
* **`cache_maintenance`**: Cleans up expired covers, thumbnails, and temporary page streams from the cache folder.
* **`chapter_prefetch`**: Predictively fetches page links for the next chapter when a user is reading near the end of their current chapter.

---

## 3. Concurrency & Rate Limiting (Key Locks)

To support rate-limiting (essential for content provider pulls) without hardcoding it into the generic queue:
* Handlers can specify a **Concurrency Group Key** (e.g. `pull:mangadex`).
* The scheduler guarantees that only `N` jobs belonging to the same Concurrency Group Key will execute at once across the worker threads.
* Throttling delays and backoff strategies are managed directly by the respective `JobHandler` implementation.

---

## 4. REST API
* `GET /api/v1/jobs`: List active and queued tasks. Supports filtering by status, type, and arbitrary metadata (e.g. `?metadata.manga_id=123` or `?type=pull_page&metadata.provider_id=mangadex`).
* `POST /api/v1/jobs`: Submit a generic job.
* `DELETE /api/v1/jobs/:id`: Cancel/Abort a pending or running job.
