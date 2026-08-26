package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tubruk/kiyomi/internal/config"
	"github.com/tubruk/kiyomi/internal/library"
	"github.com/tubruk/kiyomi/internal/queue"
	"github.com/tubruk/kiyomi/internal/queue/inmemory"
)

func setupJobTestHandler(t *testing.T) (*Handler, *echo.Echo, *inmemory.Driver) {
	t.Helper()
	tmpDir := t.TempDir()
	cfg := &config.Config{
		LibraryDir:       tmpDir,
		CacheDir:         filepath.Join(tmpDir, "cache"),
		QueueDriver:      "inmemory",
		QueueConcurrency: 4,
	}
	lib := library.NewLibrary(tmpDir)

	inmemDriver := inmemory.NewDriver(10)
	h := NewHandler(cfg, lib, inmemDriver, inmemDriver)
	e := echo.New()
	h.RegisterRoutes(e)

	return h, e, inmemDriver
}

func TestJobHandlers(t *testing.T) {
	_, e, store := setupJobTestHandler(t)

	t.Run("POST /api/v1/jobs - success", func(t *testing.T) {
		body := map[string]interface{}{
			"type":              "test-type",
			"payload":           "test-payload",
			"max_retries":       5,
			"concurrency_group": "group-a",
			"metadata": map[string]string{
				"manga_id": "123",
			},
		}
		bodyBytes, err := json.Marshal(body)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewReader(bodyBytes))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)

		var respJob queue.Job
		err = json.Unmarshal(rec.Body.Bytes(), &respJob)
		require.NoError(t, err)

		assert.NotEmpty(t, respJob.ID)
		assert.Equal(t, "test-type", respJob.Type)
		assert.Equal(t, "test-payload", respJob.Payload)
		assert.Equal(t, queue.StatusPending, respJob.Status)
		assert.Equal(t, 5, respJob.MaxRetries)
		assert.Equal(t, "group-a", respJob.ConcurrencyGroup)
		assert.Equal(t, "123", respJob.Metadata["manga_id"])
	})

	t.Run("POST /api/v1/jobs - validation missing type", func(t *testing.T) {
		body := map[string]interface{}{
			"payload": "test-payload",
		}
		bodyBytes, err := json.Marshal(body)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewReader(bodyBytes))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("GET /api/v1/jobs - list and filter", func(t *testing.T) {
		// Clear existing jobs by instantiating new store / handlers if needed,
		// or just query the existing job from the previous subtest.
		req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs?type=test-type&metadata.manga_id=123", nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var jobs []queue.Job
		err := json.Unmarshal(rec.Body.Bytes(), &jobs)
		require.NoError(t, err)

		assert.Len(t, jobs, 1)
		assert.Equal(t, "test-type", jobs[0].Type)
		assert.Equal(t, "123", jobs[0].Metadata["manga_id"])

		// Filter matching nothing
		req2 := httptest.NewRequest(http.MethodGet, "/api/v1/jobs?type=other-type", nil)
		rec2 := httptest.NewRecorder()
		e.ServeHTTP(rec2, req2)
		assert.Equal(t, http.StatusOK, rec2.Code)

		var jobs2 []queue.Job
		err = json.Unmarshal(rec2.Body.Bytes(), &jobs2)
		require.NoError(t, err)
		assert.Len(t, jobs2, 0)
	})

	t.Run("DELETE /api/v1/jobs/:id - cancel pending job", func(t *testing.T) {
		// Retrieve the enqueued job ID from the store
		jobs, err := store.ListJobs(context.Background(), queue.JobFilter{})
		require.NoError(t, err)
		require.NotEmpty(t, jobs)
		jobID := jobs[0].ID

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/jobs/"+jobID, nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var respJob queue.Job
		err = json.Unmarshal(rec.Body.Bytes(), &respJob)
		require.NoError(t, err)
		assert.Equal(t, queue.StatusFailed, respJob.Status)
		assert.Contains(t, respJob.Error, "cancelled")

		// Verify in store
		storedJob, err := store.GetJob(context.Background(), jobID)
		require.NoError(t, err)
		assert.Equal(t, queue.StatusFailed, storedJob.Status)
	})
}
