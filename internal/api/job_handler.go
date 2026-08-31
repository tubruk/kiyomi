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
// Supports filtering by status, type, parent_id, all, and arbitrary metadata keys (e.g. ?metadata.manga_id=123).
func (h *Handler) listJobs(c echo.Context) error {
	if h.jobStore == nil {
		return c.JSON(http.StatusServiceUnavailable, echo.Map{"error": "job store not configured"})
	}

	filter := queue.JobFilter{
		Status:   queue.JobStatus(c.QueryParam("status")),
		Type:     c.QueryParam("type"),
		Metadata: make(map[string]string),
	}

	parentIDParam := c.QueryParam("parent_id")
	allParam := c.QueryParam("all")

	if allParam == "true" || parentIDParam == "all" {
		filter.All = true
	} else if parentIDParam != "" {
		filter.ParentID = &parentIDParam
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
		ParentID         *string           `json:"parent_id"`
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
		ParentID:         req.ParentID,
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
		_ = h.jobStore.DeleteJob(c.Request().Context(), job.ID)
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to enqueue job"})
	}

	return c.JSON(http.StatusCreated, job)
}

// cancelJob handles POST /api/v1/jobs/:id/cancel and DELETE /api/v1/jobs/:id?action=cancel.
// Cancels/aborts a pending or running job and its child jobs recursively.
func (h *Handler) cancelJob(c echo.Context) error {
	if h.jobStore == nil {
		return c.JSON(http.StatusServiceUnavailable, echo.Map{"error": "job store not configured"})
	}

	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "missing job ID"})
	}

	if err := h.jobStore.CancelJob(c.Request().Context(), id); err != nil {
		c.Set("handler_error", err.Error())
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			return c.JSON(http.StatusNotFound, echo.Map{"error": "job not found"})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to cancel job"})
	}

	job, err := h.jobStore.GetJob(c.Request().Context(), id)
	if err != nil {
		c.Set("handler_error", err.Error())
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			return c.JSON(http.StatusNotFound, echo.Map{"error": "job not found"})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to get updated job"})
	}

	return c.JSON(http.StatusOK, job)
}

// deleteJob handles DELETE /api/v1/jobs/:id.
// Permanently removes a job and its child jobs from the database.
func (h *Handler) deleteJob(c echo.Context) error {
	if h.jobStore == nil {
		return c.JSON(http.StatusServiceUnavailable, echo.Map{"error": "job store not configured"})
	}

	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "missing job ID"})
	}

	// If action=cancel is explicitly requested, delegate to cancelJob
	if c.QueryParam("action") == "cancel" {
		return h.cancelJob(c)
	}

	if err := h.jobStore.DeleteJob(c.Request().Context(), id); err != nil {
		c.Set("handler_error", err.Error())
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			return c.JSON(http.StatusNotFound, echo.Map{"error": "job not found"})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to delete job"})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "job removed", "id": id})
}

// cleanupJobs handles DELETE /api/v1/jobs.
// Cleans up completed/failed jobs matching the filter.
func (h *Handler) cleanupJobs(c echo.Context) error {
	if h.jobStore == nil {
		return c.JSON(http.StatusServiceUnavailable, echo.Map{"error": "job store not configured"})
	}

	statusParam := c.QueryParam("status")
	if statusParam == "" {
		statusParam = "completed"
	}

	var statuses []queue.JobStatus
	if statusParam == "finished" || statusParam == "all" {
		statuses = []queue.JobStatus{queue.StatusCompleted, queue.StatusFailed}
	} else {
		for _, s := range strings.Split(statusParam, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				statuses = append(statuses, queue.JobStatus(s))
			}
		}
	}

	deletedCount := 0
	ctx := c.Request().Context()

	for _, s := range statuses {
		jobs, err := h.jobStore.ListJobs(ctx, queue.JobFilter{
			Status: s,
			All:    true,
		})
		if err != nil {
			c.Set("handler_error", err.Error())
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to list jobs for cleanup"})
		}

		for _, j := range jobs {
			// Delete root jobs or jobs directly; cascading takes care of children
			if err := h.jobStore.DeleteJob(ctx, j.ID); err == nil {
				deletedCount++
			}
		}
	}

	return c.JSON(http.StatusOK, echo.Map{
		"deleted": deletedCount,
		"status":  statusParam,
	})
}
