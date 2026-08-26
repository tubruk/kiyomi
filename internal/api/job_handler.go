package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/tubruk/kiyomi/internal/queue"
)

// listJobs handles GET /api/v1/jobs.
// Supports filtering by status, type, and arbitrary metadata keys (e.g. ?metadata.manga_id=123).
func (h *Handler) listJobs(c echo.Context) error {
	if h.jobStore == nil {
		return c.JSON(http.StatusServiceUnavailable, echo.Map{"error": "job store not configured"})
	}

	filter := queue.JobFilter{
		Status:   queue.JobStatus(c.QueryParam("status")),
		Type:     c.QueryParam("type"),
		Metadata: make(map[string]string),
	}

	for k, vals := range c.QueryParams() {
		if strings.HasPrefix(k, "metadata.") && len(vals) > 0 {
			key := strings.TrimPrefix(k, "metadata.")
			filter.Metadata[key] = vals[0]
		}
	}

	jobs, err := h.jobStore.ListJobs(c.Request().Context(), filter)
	if err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to list jobs"})
	}

	if jobs == nil {
		jobs = []*queue.Job{}
	}

	return c.JSON(http.StatusOK, jobs)
}

// enqueueJob handles POST /api/v1/jobs.
// Parses JSON payload for a job and schedules it.
func (h *Handler) enqueueJob(c echo.Context) error {
	if h.jobStore == nil || h.enqueuer == nil {
		return c.JSON(http.StatusServiceUnavailable, echo.Map{"error": "job queue not configured"})
	}

	var req struct {
		ID               string            `json:"id"`
		Type             string            `json:"type"`
		Payload          string            `json:"payload"`
		MaxRetries       int               `json:"max_retries"`
		ConcurrencyGroup string            `json:"concurrency_group"`
		Metadata         map[string]string `json:"metadata"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request body"})
	}

	if req.Type == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "missing job type"})
	}

	jobID := req.ID
	if jobID == "" {
		jobID = uuid.New().String()
	}

	now := time.Now()
	job := &queue.Job{
		ID:               jobID,
		Type:             req.Type,
		Payload:          req.Payload,
		Status:           queue.StatusPending,
		MaxRetries:       req.MaxRetries,
		ConcurrencyGroup: req.ConcurrencyGroup,
		CreatedAt:        now,
		UpdatedAt:        now,
		Metadata:         req.Metadata,
	}

	if job.Metadata == nil {
		job.Metadata = make(map[string]string)
	}

	if err := h.jobStore.CreateJob(c.Request().Context(), job); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to persist job"})
	}

	if err := h.enqueuer.Enqueue(c.Request().Context(), job); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to enqueue job"})
	}

	return c.JSON(http.StatusCreated, job)
}

// cancelJob handles DELETE /api/v1/jobs/:id.
// Cancels/aborts a pending or running job by marking it as failed.
func (h *Handler) cancelJob(c echo.Context) error {
	if h.jobStore == nil {
		return c.JSON(http.StatusServiceUnavailable, echo.Map{"error": "job store not configured"})
	}

	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "missing job ID"})
	}

	job, err := h.jobStore.GetJob(c.Request().Context(), id)
	if err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusNotFound, echo.Map{"error": "job not found"})
	}

	if job.Status == queue.StatusPending || job.Status == queue.StatusRunning {
		job.Status = queue.StatusFailed
		job.Error = "job cancelled by user"
		now := time.Now()
		job.CompletedAt = &now
		job.UpdatedAt = now
		if err := h.jobStore.UpdateJob(c.Request().Context(), job); err != nil {
			c.Set("handler_error", err.Error())
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to update job status"})
		}
	}

	return c.JSON(http.StatusOK, job)
}
