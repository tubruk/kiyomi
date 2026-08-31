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

	t.Run("POST /api/v1/jobs/:id/cancel - cancel pending job", func(t *testing.T) {
		// Retrieve the enqueued job ID from the store
		jobs, err := store.ListJobs(context.Background(), queue.JobFilter{})
		require.NoError(t, err)
		require.NotEmpty(t, jobs)
		jobID := jobs[0].ID

		req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/"+jobID+"/cancel", nil)
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

	t.Run("POST /api/v1/jobs - with parent_id", func(t *testing.T) {
		parentID := "root-job-001"
		body := map[string]interface{}{
			"parent_id": parentID,
			"type":      "child-type",
			"payload":   "child-payload",
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
		assert.NotNil(t, respJob.ParentID)
		assert.Equal(t, parentID, *respJob.ParentID)
		assert.Equal(t, "child-type", respJob.Type)
	})

	t.Run("Job Hierarchy - root listing, parent_id filter, all=true, and recursive cancellation", func(t *testing.T) {
		_, eH, storeH := setupJobTestHandler(t)

		// Helper to create a job via API
		createJob := func(jobType string, parentID *string) queue.Job {
			bodyMap := map[string]interface{}{
				"type":    jobType,
				"payload": "payload-" + jobType,
			}
			if parentID != nil {
				bodyMap["parent_id"] = *parentID
			}
			bBytes, _ := json.Marshal(bodyMap)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewReader(bBytes))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			eH.ServeHTTP(rec, req)
			require.Equal(t, http.StatusCreated, rec.Code)

			var created queue.Job
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
			return created
		}

		// Create P1 (root), P2 (root)
		p1 := createJob("pull_manga", nil)
		p2 := createJob("pull_manga", nil)

		// Create C1, C2 under P1
		c1 := createJob("pull_chapter", &p1.ID)
		c2 := createJob("pull_chapter", &p1.ID)

		// Create G1 under C1
		g1 := createJob("pull_page", &c1.ID)

		// 1. Default GET /api/v1/jobs returns only top-level root jobs (P1 and P2)
		{
			req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs", nil)
			rec := httptest.NewRecorder()
			eH.ServeHTTP(rec, req)
			require.Equal(t, http.StatusOK, rec.Code)

			var jobs []queue.Job
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &jobs))
			require.Len(t, jobs, 2)

			jobMap := make(map[string]queue.Job)
			for _, j := range jobs {
				jobMap[j.ID] = j
			}
			require.Contains(t, jobMap, p1.ID)
			require.Contains(t, jobMap, p2.ID)
			assert.Equal(t, 2, jobMap[p1.ID].ChildCount)
			assert.Equal(t, 0, jobMap[p2.ID].ChildCount)
		}

		// 2. GET /api/v1/jobs?parent_id=<p1.ID> returns C1 and C2
		{
			req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs?parent_id="+p1.ID, nil)
			rec := httptest.NewRecorder()
			eH.ServeHTTP(rec, req)
			require.Equal(t, http.StatusOK, rec.Code)

			var jobs []queue.Job
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &jobs))
			require.Len(t, jobs, 2)

			jobMap := make(map[string]queue.Job)
			for _, j := range jobs {
				jobMap[j.ID] = j
			}
			require.Contains(t, jobMap, c1.ID)
			require.Contains(t, jobMap, c2.ID)
			assert.Equal(t, 1, jobMap[c1.ID].ChildCount)
			assert.Equal(t, 0, jobMap[c2.ID].ChildCount)
		}

		// 3. GET /api/v1/jobs?parent_id=<c1.ID> returns G1
		{
			req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs?parent_id="+c1.ID, nil)
			rec := httptest.NewRecorder()
			eH.ServeHTTP(rec, req)
			require.Equal(t, http.StatusOK, rec.Code)

			var jobs []queue.Job
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &jobs))
			require.Len(t, jobs, 1)
			assert.Equal(t, g1.ID, jobs[0].ID)
		}

		// 4. GET /api/v1/jobs?all=true returns all 5 jobs
		{
			req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs?all=true", nil)
			rec := httptest.NewRecorder()
			eH.ServeHTTP(rec, req)
			require.Equal(t, http.StatusOK, rec.Code)

			var jobs []queue.Job
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &jobs))
			assert.Len(t, jobs, 5)
		}

		// 5. GET /api/v1/jobs?parent_id=all returns all 5 jobs
		{
			req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs?parent_id=all", nil)
			rec := httptest.NewRecorder()
			eH.ServeHTTP(rec, req)
			require.Equal(t, http.StatusOK, rec.Code)

			var jobs []queue.Job
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &jobs))
			assert.Len(t, jobs, 5)
		}

		// 6. POST /api/v1/jobs/:id/cancel cancels P1 and recursively cancels C1, C2, and G1
		{
			req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/"+p1.ID+"/cancel", nil)
			rec := httptest.NewRecorder()
			eH.ServeHTTP(rec, req)
			require.Equal(t, http.StatusOK, rec.Code)

			var respJob queue.Job
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &respJob))
			assert.Equal(t, queue.StatusFailed, respJob.Status)
			assert.Contains(t, respJob.Error, "cancelled")

			// Check all descendants are cancelled in store
			checkStatus := func(id string, expected queue.JobStatus) {
				j, err := storeH.GetJob(context.Background(), id)
				require.NoError(t, err)
				assert.Equal(t, expected, j.Status, "job %s status mismatch", id)
			}
			checkStatus(p1.ID, queue.StatusFailed)
			checkStatus(c1.ID, queue.StatusFailed)
			checkStatus(c2.ID, queue.StatusFailed)
			checkStatus(g1.ID, queue.StatusFailed)

			// P2 remains pending
			checkStatus(p2.ID, queue.StatusPending)
		}

		// 7. DELETE /api/v1/jobs/nonexistent-id returns 404
		{
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/jobs/nonexistent-job-id", nil)
			rec := httptest.NewRecorder()
			eH.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusNotFound, rec.Code)
		}

		// 8. DELETE /api/v1/jobs/:id permanently removes a job and its descendants
		{
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/jobs/"+p1.ID, nil)
			rec := httptest.NewRecorder()
			eH.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusOK, rec.Code)

			// P1, C1, C2, G1 are now gone from store
			_, err := storeH.GetJob(context.Background(), p1.ID)
			assert.Error(t, err)
			_, err = storeH.GetJob(context.Background(), c1.ID)
			assert.Error(t, err)
		}

		// 9. DELETE /api/v1/jobs?status=all bulk cleans up matching jobs
		{
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/jobs?status=all", nil)
			rec := httptest.NewRecorder()
			eH.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusOK, rec.Code)

			var cleanupResp map[string]interface{}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &cleanupResp))
			assert.Contains(t, cleanupResp, "deleted")
		}
	})
}
