package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/tubruk/kiyomi/internal/library"
	"github.com/tubruk/kiyomi/internal/queue"
	"github.com/tubruk/kiyomi/pkg/provider/sdk"
)

func (h *Handler) listLibraryManga(c echo.Context) error {
	mangas, err := h.lib.ListManga()
	if err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	res := make([]echo.Map, 0, len(mangas))
	for _, m := range mangas {
		var providerID string
		var readingMode string
		if m.Meta.Content != nil {
			providerID = m.Meta.Content.ProviderID
			readingMode = m.Meta.Content.ReadingMode
		}
		item := echo.Map{
			"id":                  m.ID,
			"title":               m.Meta.Title,
			"cover":               m.Meta.CoverURL,
			"content_provider_id": providerID,
			"sourceId":            providerID,
			"external_links":      m.Meta.ExternalLinks,
			"externalLinks":       m.Meta.ExternalLinks,
			"meta":                m.Meta,
		}
		if readingMode != "" {
			item["reading_mode"] = readingMode
		}
		res = append(res, item)
	}
	return c.JSON(http.StatusOK, res)
}

func (h *Handler) getLibraryManga(c echo.Context) error {
	id := c.Param("mangaId")
	meta, err := h.lib.GetManga(id)
	if err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
	}
	var providerID string
	var readingMode string
	if meta.Content != nil {
		providerID = meta.Content.ProviderID
		readingMode = meta.Content.ReadingMode
	}
	resp := echo.Map{
		"id":                  id,
		"title":               meta.Title,
		"aliases":             meta.Aliases,
		"cover":               meta.CoverURL,
		"description":         meta.Description,
		"authors":             meta.Authors,
		"artists":             meta.Artists,
		"tags":                meta.Tags,
		"content_provider_id": providerID,
		"sourceId":            providerID,
		"external_links":      meta.ExternalLinks,
		"externalLinks":       meta.ExternalLinks,
		"meta":                meta,
	}
	if readingMode != "" {
		resp["reading_mode"] = readingMode
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) createLibraryManga(c echo.Context) error {
	var body struct {
		ID      string                 `json:"id"`
		Meta    library.MangaMeta      `json:"meta"`
		Content *library.ContentSource `json:"content"`
	}
	if err := c.Bind(&body); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	if body.ID == "" {
		c.Set("handler_error", "id is required")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "id is required"})
	}

	if body.Meta.UserStatus != "" && !library.IsValidUserStatus(body.Meta.UserStatus) {
		c.Set("handler_error", "invalid user_status")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid user_status"})
	}

	if body.Meta.Content == nil && body.Content != nil {
		body.Meta.Content = body.Content
	}

	if err := h.lib.SaveManga(body.ID, &body.Meta); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, echo.Map{
		"id":   body.ID,
		"meta": body.Meta,
	})
}

func (h *Handler) updateLibraryManga(c echo.Context) error {
	id := c.Param("mangaId")
	var meta library.MangaMeta
	if err := c.Bind(&meta); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	if meta.UserStatus != "" && !library.IsValidUserStatus(meta.UserStatus) {
		c.Set("handler_error", "invalid user_status")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid user_status"})
	}

	if err := h.lib.SaveManga(id, &meta); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"id":   id,
		"meta": meta,
	})
}

func (h *Handler) patchLibraryManga(c echo.Context) error {
	id := c.Param("mangaId")
	existing, err := h.lib.GetManga(id)
	if err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
	}

	if err := c.Bind(existing); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	if existing.UserStatus != "" && !library.IsValidUserStatus(existing.UserStatus) {
		c.Set("handler_error", "invalid user_status")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid user_status"})
	}

	if err := h.lib.SaveManga(id, existing); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"id":   id,
		"meta": existing,
	})
}

func (h *Handler) deleteLibraryManga(c echo.Context) error {
	id := c.Param("mangaId")
	if err := h.lib.DeleteManga(id); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) refreshChaptersFromContent(ctx context.Context, providerID string, mangaID string) (int, int, error) {
	meta, err := h.lib.GetManga(mangaID)
	if err != nil {
		return 0, 0, err
	}

	if meta.Content == nil || meta.Content.ProviderID == "" {
		return 0, 0, fmt.Errorf("manga has no connected content provider")
	}

	providerID = meta.Content.ProviderID
	_, contentProvider, err := h.getProvider(providerID)
	if err != nil {
		return 0, 0, err
	}

	remoteID := meta.Content.ProviderMangaID
	if remoteID == "" {
		remoteID = mangaID
	}

	chapters, err := contentProvider.FetchChapters(ctx, remoteID)
	if err != nil {
		return 0, 0, err
	}

	existingChapters, err := h.lib.ListChapters(mangaID, providerID)
	if err != nil {
		return 0, 0, err
	}

	existingMap := make(map[string]bool, len(existingChapters))
	for _, ch := range existingChapters {
		existingMap[ch.ID] = true
	}

	var toAdd []sdk.Chapter
	for _, ch := range chapters {
		if !existingMap[ch.ID] {
			toAdd = append(toAdd, ch)
		}
	}

	now := time.Now()
	var added int32
	var orphaned int32

	if len(toAdd) > 0 {
		const maxWorkers = 16
		numWorkers := min(len(toAdd), maxWorkers)
		sem := make(chan struct{}, numWorkers)
		var (
			wg      sync.WaitGroup
			errOnce sync.Once
			saveErr error
		)

	loop:
		for _, ch := range toAdd {
			if ctx.Err() != nil {
				errOnce.Do(func() {
					saveErr = ctx.Err()
				})
				break loop
			}

			select {
			case <-ctx.Done():
				errOnce.Do(func() {
					saveErr = ctx.Err()
				})
				break loop
			case sem <- struct{}{}:
			}

			wg.Add(1)
			go func(ch sdk.Chapter) {
				defer wg.Done()
				defer func() { <-sem }()

				chMeta := &library.ChapterMeta{
					Title:       ch.Name,
					Number:      ch.Number,
					UploadDate:  ch.UploadDate,
					SourceOrder: ch.SourceOrder,
					Content: &library.ContentSource{
						ProviderID:   providerID,
						ChapterRef:   ch.ID,
						LastSyncedAt: now,
					},
				}
				if err := h.lib.SaveChapter(mangaID, providerID, ch.ID, chMeta); err != nil {
					errOnce.Do(func() {
						saveErr = err
					})
					return
				}
				atomic.AddInt32(&added, 1)
			}(ch)
		}
		wg.Wait()

		if saveErr != nil {
			return int(added), int(orphaned), saveErr
		}
	}

	// Orphan detection: chapters on disk not in API result
	if meta.Content.ProviderID != library.LocalProviderID {
		for _, ch := range existingChapters {
			found := false
			for _, apiCh := range chapters {
				if ch.ID == apiCh.ID {
					found = true
					break
				}
			}
			if !found {
				orphaned++
			}
		}
	}

	if ctx.Err() != nil {
		return int(added), int(orphaned), ctx.Err()
	}

	meta.Content.LastSyncedAt = now
	if err := h.lib.SaveManga(mangaID, meta); err != nil {
		return int(added), int(orphaned), err
	}

	return int(added), int(orphaned), nil
}

func (h *Handler) refreshLibraryManga(c echo.Context) error {
	mangaID := c.Param("mangaId")

	meta, err := h.lib.GetManga(mangaID)
	if err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
	}

	if meta.Content == nil || meta.Content.ProviderID == "" {
		c.Set("handler_error", "manga has no connected content provider")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "manga has no connected content provider"})
	}

	providerID := meta.Content.ProviderID
	added, orphaned, err := h.refreshChaptersFromContent(c.Request().Context(), providerID, mangaID)
	if err != nil {
		return handleProviderError(c, providerID, err)
	}

	return c.JSON(http.StatusOK, echo.Map{
		"added":       added,
		"orphaned":    orphaned,
		"updated":     0,
		"provider_id": providerID,
		"manga_id":    mangaID,
	})
}

func (h *Handler) listChapters(c echo.Context) error {
	id := c.Param("mangaId")
	providerID := c.QueryParam("provider_id")
	var chapters []library.ChapterInfo
	var err error
	if providerID != "" {
		chapters, err = h.lib.ListChapters(id, providerID)
	} else {
		chapters, err = h.lib.ListChapters(id)
	}
	if err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	// Determine active provider namespace; chapters from other namespaces are orphaned.
	activeProviderID := providerID
	if activeProviderID == "" {
		if meta, metaErr := h.lib.GetManga(id); metaErr == nil && meta.Content != nil {
			activeProviderID = meta.Content.ProviderID
		}
	}

	var res []echo.Map
	for _, ch := range chapters {
		var uploadDateStr string
		if !ch.Meta.UploadDate.IsZero() {
			uploadDateStr = ch.Meta.UploadDate.UTC().Format(time.RFC3339)
		}
		var downloadedAtStr string
		if !ch.Meta.DownloadedAt.IsZero() {
			downloadedAtStr = ch.Meta.DownloadedAt.UTC().Format(time.RFC3339)
		}
		chProviderID := ""
		if ch.Meta.Content != nil {
			chProviderID = ch.Meta.Content.ProviderID
		}
		if chProviderID == "" {
			chProviderID = library.LocalProviderID
		}
		orphaned := activeProviderID != "" && chProviderID != activeProviderID
		metaCopy := ch.Meta
		metaCopy.Orphaned = orphaned
		res = append(res, echo.Map{
			"id":               ch.ID,
			"manga_id":         ch.MangaID,
			"title":            ch.Meta.Title,
			"number":           ch.Meta.Number,
			"volume":           ch.Meta.Volume,
			"uploadDate":       uploadDateStr,
			"sourceOrder":      ch.Meta.SourceOrder,
			"provider_id":      chProviderID,
			"is_downloaded":    ch.Meta.IsDownloaded,
			"downloaded_pages": ch.Meta.DownloadedPages,
			"page_count":       ch.Meta.PageCount,
			"downloaded_at":    downloadedAtStr,
			"meta":             metaCopy,
		})
	}
	return c.JSON(http.StatusOK, echo.Map{
		"chapters": res,
	})
}

func (h *Handler) getChapter(c echo.Context) error {
	mangaID := c.Param("mangaId")
	providerID := c.Param("providerId")
	chapterID := c.Param("chapterId")

	meta, err := h.lib.GetChapter(mangaID, providerID, chapterID)
	if err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"id":          chapterID,
		"manga_id":    mangaID,
		"provider_id": providerID,
		"meta":        meta,
	})
}

func (h *Handler) saveChapter(c echo.Context) error {
	mangaID := c.Param("mangaId")
	providerID := c.Param("providerId")
	chapterID := c.Param("chapterId")
	var meta library.ChapterMeta
	if err := c.Bind(&meta); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	if err := h.lib.SaveChapter(mangaID, providerID, chapterID, &meta); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"id":          chapterID,
		"manga_id":    mangaID,
		"provider_id": providerID,
		"meta":        meta,
	})
}

func (h *Handler) deleteChapter(c echo.Context) error {
	mangaID := c.Param("mangaId")
	providerID := c.Param("providerId")
	chapterID := c.Param("chapterId")

	if err := h.lib.DeleteChapter(mangaID, providerID, chapterID); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) patchChapterProgress(c echo.Context) error {
	mangaID := c.Param("mangaId")
	providerID := c.Param("providerId")
	chapterID := c.Param("chapterId")

	var req struct {
		IsRead       *bool `json:"is_read"`
		LastReadPage *int  `json:"last_read_page"`
	}
	if err := c.Bind(&req); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	existing, err := h.lib.GetChapter(mangaID, providerID, chapterID)
	if err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
	}

	isRead := existing.IsRead
	if req.IsRead != nil {
		isRead = *req.IsRead
	}

	lastReadPage := existing.LastReadPage
	if req.LastReadPage != nil {
		lastReadPage = *req.LastReadPage
	}

	info, err := h.lib.UpdateChapterProgress(mangaID, providerID, chapterID, isRead, lastReadPage)
	if err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, info)
}

func (h *Handler) patchChaptersProgressBatch(c echo.Context) error {
	mangaID := c.Param("mangaId")
	providerID := c.Param("providerId")

	var req struct {
		ChapterIDs   []string `json:"chapter_ids"`
		IsRead       *bool    `json:"is_read"`
		LastReadPage *int     `json:"last_read_page"`
	}
	if err := c.Bind(&req); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	if mangaID == "" || providerID == "" || len(req.ChapterIDs) == 0 {
		c.Set("handler_error", "mangaId, providerId, and chapter_ids are required")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "mangaId, providerId, and chapter_ids are required"})
	}

	isRead := false
	if req.IsRead != nil {
		isRead = *req.IsRead
	}

	lastReadPage := 0
	if req.LastReadPage != nil {
		lastReadPage = *req.LastReadPage
	}

	updated, err := h.lib.BatchUpdateChapterProgress(mangaID, providerID, req.ChapterIDs, isRead, lastReadPage)
	if err != nil {
		if strings.Contains(err.Error(), "no such file or directory") || strings.Contains(err.Error(), "not found") {
			c.Set("handler_error", err.Error())
			return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
		}
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	chapterIDs := make([]string, 0, len(updated))
	for _, u := range updated {
		chapterIDs = append(chapterIDs, u.ID)
	}

	return c.JSON(http.StatusOK, echo.Map{
		"updated":     len(updated),
		"chapter_ids": chapterIDs,
	})
}

// pullChapter handles POST /library/manga/:mangaId/providers/:providerId/chapters/:chapterId/pull.
// Creates and enqueues a pull_chapter job for the specified chapter.
func (h *Handler) pullChapter(c echo.Context) error {
	mangaID := c.Param("mangaId")
	providerID := c.Param("providerId")
	chapterID := c.Param("chapterId")

	if mangaID == "" || providerID == "" || chapterID == "" {
		c.Set("handler_error", "mangaId, providerId, and chapterId are required")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "mangaId, providerId, and chapterId are required"})
	}

	if _, err := h.lib.GetManga(mangaID); err != nil {
		c.Set("handler_error", "manga not found")
		return c.JSON(http.StatusNotFound, echo.Map{"error": "manga not found"})
	}

	if err := h.requireContentCapability(providerID); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	if h.jobStore == nil || h.enqueuer == nil {
		return c.JSON(http.StatusServiceUnavailable, echo.Map{"error": "job queue not configured"})
	}

	now := time.Now()
	payloadBytes, err := json.Marshal(queue.PullChapterPayload{
		MangaID:    mangaID,
		ProviderID: providerID,
		ChapterID:  chapterID,
	})
	if err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": fmt.Sprintf("marshal pull_chapter payload: %v", err)})
	}

	job := &queue.Job{
		ID:               uuid.New().String(),
		Type:             queue.JobTypePullChapter,
		Payload:          string(payloadBytes),
		Status:           queue.StatusPending,
		MaxRetries:       3,
		ConcurrencyGroup: "pull:" + providerID,
		CreatedAt:        now,
		UpdatedAt:        now,
		Metadata: map[string]string{
			"manga_id":    mangaID,
			"provider_id": providerID,
			"chapter_id":  chapterID,
		},
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

	return c.JSON(http.StatusAccepted, echo.Map{
		"job_id":     job.ID,
		"chapter_id": chapterID,
	})
}

// pullChaptersBatch handles POST /library/manga/:mangaId/providers/:providerId/chapters/pull.
// Creates and enqueues a pull_chapter job for each requested chapter ID.
func (h *Handler) pullChaptersBatch(c echo.Context) error {
	mangaID := c.Param("mangaId")
	providerID := c.Param("providerId")

	var req struct {
		ChapterIDs []string `json:"chapter_ids"`
	}
	if err := c.Bind(&req); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	if mangaID == "" || providerID == "" || len(req.ChapterIDs) == 0 {
		c.Set("handler_error", "mangaId, providerId, and chapter_ids are required")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "mangaId, providerId, and chapter_ids are required"})
	}

	if _, err := h.lib.GetManga(mangaID); err != nil {
		c.Set("handler_error", "manga not found")
		return c.JSON(http.StatusNotFound, echo.Map{"error": "manga not found"})
	}

	if err := h.requireContentCapability(providerID); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	if h.jobStore == nil || h.enqueuer == nil {
		return c.JSON(http.StatusServiceUnavailable, echo.Map{"error": "job queue not configured"})
	}

	now := time.Now()
	jobIDs := make([]string, 0, len(req.ChapterIDs))
	for _, chapterID := range req.ChapterIDs {
		payloadBytes, err := json.Marshal(queue.PullChapterPayload{
			MangaID:    mangaID,
			ProviderID: providerID,
			ChapterID:  chapterID,
		})
		if err != nil {
			c.Set("handler_error", err.Error())
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": fmt.Sprintf("marshal pull_chapter payload: %v", err)})
		}

		job := &queue.Job{
			ID:               uuid.New().String(),
			Type:             queue.JobTypePullChapter,
			Payload:          string(payloadBytes),
			Status:           queue.StatusPending,
			MaxRetries:       3,
			ConcurrencyGroup: "pull:" + providerID,
			CreatedAt:        now,
			UpdatedAt:        now,
			Metadata: map[string]string{
				"manga_id":    mangaID,
				"provider_id": providerID,
				"chapter_id":  chapterID,
			},
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

		jobIDs = append(jobIDs, job.ID)
	}

	return c.JSON(http.StatusOK, echo.Map{
		"chapter_count": len(req.ChapterIDs),
		"job_ids":       jobIDs,
		"job_count":     len(jobIDs),
	})
}

func (h *Handler) refreshChaptersBatch(c echo.Context) error {
	mangaID := c.Param("mangaId")
	providerID := c.Param("providerId")

	var req struct {
		ChapterIDs []string `json:"chapter_ids"`
	}
	if err := c.Bind(&req); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	if mangaID == "" || providerID == "" || len(req.ChapterIDs) == 0 {
		c.Set("handler_error", "mangaId, providerId, and chapter_ids are required")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "mangaId, providerId, and chapter_ids are required"})
	}

	if err := h.requireContentCapability(providerID); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	_, contentProvider, err := h.getProvider(providerID)
	if err != nil || contentProvider == nil {
		return handleProviderError(c, providerID, fmt.Errorf("provider not available: %s", providerID))
	}

	remoteMangaID := mangaID
	if mangaMeta, err := h.lib.GetManga(mangaID); err == nil {
		if mangaMeta.Content != nil && mangaMeta.Content.ProviderID == providerID && mangaMeta.Content.ProviderMangaID != "" {
			remoteMangaID = mangaMeta.Content.ProviderMangaID
		} else {
			for _, p := range mangaMeta.Providers {
				if p.ProviderID == providerID && p.ProviderMangaID != "" {
					remoteMangaID = p.ProviderMangaID
					break
				}
			}
			if remoteMangaID == mangaID && mangaMeta.Content != nil && mangaMeta.Content.ProviderMangaID != "" {
				remoteMangaID = mangaMeta.Content.ProviderMangaID
			}
		}
	}

	chapters, err := contentProvider.FetchChapters(c.Request().Context(), remoteMangaID)
	if err != nil {
		return handleProviderError(c, providerID, err)
	}

	upstreamMap := make(map[string]sdk.Chapter, len(chapters))
	for _, ch := range chapters {
		upstreamMap[ch.ID] = ch
	}

	now := time.Now()
	refreshedIDs := make([]string, 0)
	for _, chID := range req.ChapterIDs {
		upCh, found := upstreamMap[chID]
		if !found {
			continue
		}

		chMeta, err := h.lib.GetChapter(mangaID, providerID, chID)
		if err != nil || chMeta == nil {
			chMeta = &library.ChapterMeta{
				Content: &library.ContentSource{
					ProviderID: providerID,
					ChapterRef: chID,
				},
			}
		}

		chMeta.Title = upCh.Name
		chMeta.Number = upCh.Number
		chMeta.UploadDate = upCh.UploadDate
		chMeta.SourceOrder = upCh.SourceOrder
		if chMeta.Content == nil {
			chMeta.Content = &library.ContentSource{
				ProviderID: providerID,
				ChapterRef: chID,
			}
		}
		chMeta.Content.LastSyncedAt = now

		if err := h.lib.SaveChapter(mangaID, providerID, chID, chMeta); err != nil {
			c.Set("handler_error", err.Error())
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
		}
		refreshedIDs = append(refreshedIDs, chID)
	}

	return c.JSON(http.StatusOK, echo.Map{
		"refreshed":   len(refreshedIDs),
		"chapter_ids": refreshedIDs,
	})
}

// getProviderCapabilities returns the capabilities for a provider ID.
func (h *Handler) getProviderCapabilities(providerID string) []string {
	if prov, ok := h.registry.Get(providerID); ok {
		return prov.Capabilities()
	}
	return nil
}

// requireContentCapability returns nil if provider has Content capability, error otherwise.
func (h *Handler) requireContentCapability(providerID string) error {
	caps := h.getProviderCapabilities(providerID)
	if slices.Contains(caps, "content") {
		return nil
	}
	return fmt.Errorf("provider %q lacks content capability", providerID)
}

func (h *Handler) listProviders(c echo.Context) error {
	mangaID := c.Param("mangaId")
	meta, err := h.lib.GetManga(mangaID)
	if err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
	}
	providers := meta.Providers
	if providers == nil {
		providers = []library.ProviderRef{}
	}
	return c.JSON(http.StatusOK, echo.Map{
		"providers": providers,
	})
}

func (h *Handler) addProvider(c echo.Context) error {
	mangaID := c.Param("mangaId")
	// Check manga exists
	meta, err := h.lib.GetManga(mangaID)
	if err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
	}

	var body struct {
		ProviderID      string `json:"provider_id"`
		ProviderMangaID string `json:"provider_manga_id"`
		MangaTitle      string `json:"manga_title"`
		SetAsContent    bool   `json:"set_as_content"`
	}
	if err := c.Bind(&body); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}
	if body.ProviderID == "" || body.ProviderMangaID == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "provider_id and provider_manga_id are required"})
	}

	if body.SetAsContent {
		if err := h.requireContentCapability(body.ProviderID); err != nil {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
		}
	}

	ref := library.ProviderRef{
		ProviderID:      body.ProviderID,
		ProviderMangaID: body.ProviderMangaID,
		MangaTitle:      body.MangaTitle,
	}
	if err := h.lib.AddProvider(mangaID, ref); err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			return c.JSON(http.StatusConflict, echo.Map{"error": err.Error()})
		}
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	if body.SetAsContent {
		title := body.MangaTitle
		if title == "" {
			title = meta.Title
		}
		if err := h.lib.SwitchContentProvider(mangaID, body.ProviderID, body.ProviderMangaID, title); err != nil {
			c.Set("handler_error", err.Error())
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
		}
	}

	added := 0
	if body.SetAsContent {
		var refreshErr error
		added, _, refreshErr = h.refreshChaptersFromContent(c.Request().Context(), body.ProviderID, mangaID)
		if refreshErr != nil {
			c.Set("handler_error", refreshErr.Error())
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": refreshErr.Error()})
		}
	}

	updated, _ := h.lib.GetManga(mangaID)
	return c.JSON(http.StatusCreated, echo.Map{
		"providers": updated.Providers,
		"added":     added,
	})
}

func (h *Handler) removeProvider(c echo.Context) error {
	mangaID := c.Param("mangaId")
	providerID := c.Param("providerId")
	providerMangaID := c.Param("providerMangaId")

	// Check manga exists
	if _, err := h.lib.GetManga(mangaID); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
	}

	capLookup := h.getProviderCapabilities
	if err := h.lib.RemoveProvider(mangaID, providerID, providerMangaID, capLookup); err != nil {
		if strings.Contains(err.Error(), "cannot remove last content-capable") {
			return c.JSON(http.StatusConflict, echo.Map{"error": err.Error()})
		}
		if strings.Contains(err.Error(), "not found") {
			return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
		}
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) switchContentProvider(c echo.Context) error {
	mangaID := c.Param("mangaId")
	// Check manga exists
	meta, err := h.lib.GetManga(mangaID)
	if err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
	}

	var body struct {
		ProviderID      string `json:"provider_id"`
		ProviderMangaID string `json:"provider_manga_id"`
	}
	if err := c.Bind(&body); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}
	if body.ProviderID == "" || body.ProviderMangaID == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "provider_id and provider_manga_id are required"})
	}

	if err := h.requireContentCapability(body.ProviderID); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	// Verify provider exists in providers list or let it be added
	if err := h.lib.SwitchContentProvider(mangaID, body.ProviderID, body.ProviderMangaID, meta.Title); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	updated, _ := h.lib.GetManga(mangaID)
	return c.JSON(http.StatusOK, echo.Map{
		"id":    mangaID,
		"meta":  updated,
		"added": 0,
	})
}

func (h *Handler) deleteChapterFiles(c echo.Context) error {
	mangaID := c.Param("mangaId")
	providerID := c.Param("providerId")
	chapterID := c.Param("chapterId")
	if err := h.lib.DeleteChapterFiles(mangaID, providerID, chapterID); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) deleteChapterFilesBatch(c echo.Context) error {
	mangaID := c.Param("mangaId")
	providerID := c.Param("providerId")

	var req struct {
		ChapterIDs []string `json:"chapter_ids"`
	}
	if err := c.Bind(&req); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	if mangaID == "" || providerID == "" || len(req.ChapterIDs) == 0 {
		c.Set("handler_error", "mangaId, providerId, and chapter_ids are required")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "mangaId, providerId, and chapter_ids are required"})
	}

	if err := h.lib.BatchDeleteChapterFiles(mangaID, providerID, req.ChapterIDs); err != nil {
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "no such file or directory") {
			c.Set("handler_error", err.Error())
			return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
		}
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"deleted": len(req.ChapterIDs),
	})
}

func (h *Handler) deleteChaptersBatch(c echo.Context) error {
	mangaID := c.Param("mangaId")
	providerID := c.Param("providerId")

	var req struct {
		ChapterIDs []string `json:"chapter_ids"`
	}
	if err := c.Bind(&req); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	if mangaID == "" || providerID == "" || len(req.ChapterIDs) == 0 {
		c.Set("handler_error", "mangaId, providerId, and chapter_ids are required")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "mangaId, providerId, and chapter_ids are required"})
	}

	if err := h.lib.BatchDeleteChapters(mangaID, providerID, req.ChapterIDs); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"removed": len(req.ChapterIDs),
	})
}

// pullManga handles POST /api/v1/library/manga/:mangaId/pull.
// Enqueues a pull_manga job. If providerID query param is missing, falls back
// to the manga's active content provider.
func (h *Handler) pullManga(c echo.Context) error {
	if h.jobStore == nil || h.enqueuer == nil {
		c.Set("handler_error", "job queue not configured")
		return c.JSON(http.StatusServiceUnavailable, echo.Map{"error": "job queue not configured"})
	}

	mangaID := c.Param("mangaId")
	if mangaID == "" {
		c.Set("handler_error", "mangaId is required")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "mangaId is required"})
	}

	meta, err := h.lib.GetManga(mangaID)
	if err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
	}

	providerID := c.QueryParam("provider_id")
	if providerID == "" {
		if meta.Content == nil || meta.Content.ProviderID == "" {
			c.Set("handler_error", "manga has no connected content provider and provider_id query param missing")
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "manga has no connected content provider and provider_id query param missing"})
		}
		providerID = meta.Content.ProviderID
	}

	providerMangaID := c.QueryParam("provider_manga_id")
	if providerMangaID == "" && meta.Content != nil {
		providerMangaID = meta.Content.ProviderMangaID
	}
	if providerMangaID == "" {
		c.Set("handler_error", "provider_manga_id required when manga has no content binding")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "provider_manga_id required when manga has no content binding"})
	}

	if err := h.requireContentCapability(providerID); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	payloadBytes, err := json.Marshal(queue.PullMangaPayload{
		MangaID:         mangaID,
		ProviderID:      providerID,
		ProviderMangaID: providerMangaID,
		CoverURL:        meta.CoverURL,
	})
	if err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	now := time.Now()
	job := &queue.Job{
		ID:               uuid.New().String(),
		Type:             queue.JobTypePullManga,
		Payload:          string(payloadBytes),
		MaxRetries:       3,
		ConcurrencyGroup: "pull:" + providerID,
		CreatedAt:        now,
		UpdatedAt:        now,
		Metadata: map[string]string{
			"manga_id":    mangaID,
			"provider_id": providerID,
		},
	}

	if err := h.jobStore.CreateJob(c.Request().Context(), job); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	if err := h.enqueuer.Enqueue(c.Request().Context(), job); err != nil {
		_ = h.jobStore.DeleteJob(c.Request().Context(), job.ID)
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusAccepted, echo.Map{
		"job_id":      job.ID,
		"provider_id": providerID,
	})
}

