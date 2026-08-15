package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/tubruk/kiyomi/internal/library"
	"github.com/tubruk/kiyomi/pkg/provider/mangadex"
	"github.com/tubruk/kiyomi/pkg/provider/sdk"
)

func (h *Handler) listLibraryManga(c echo.Context) error {
	mangas, err := h.lib.ListManga()
	if err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	var res []echo.Map
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
	if id == "" {
		id = c.Param("id")
	}
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
		"meta":                meta,
	}
	if readingMode != "" {
		resp["reading_mode"] = readingMode
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) createLibraryManga(c echo.Context) error {
	var body struct {
		ID   string            `json:"id"`
		Meta library.MangaMeta `json:"meta"`
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
	if id == "" {
		id = c.Param("id")
	}
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
	if id == "" {
		id = c.Param("id")
	}
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
	if id == "" {
		id = c.Param("id")
	}
	if err := h.lib.DeleteManga(id); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) refreshLibraryManga(c echo.Context) error {
	mangaID := c.Param("mangaId")
	if mangaID == "" {
		mangaID = c.Param("id")
	}

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
	_, contentProvider, err := h.getProvider(providerID)
	if err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	remoteID := meta.Content.MangaID
	if remoteID == "" {
		remoteID = mangaID
	}

	chapters, err := contentProvider.FetchChapters(c.Request().Context(), remoteID)
	if err != nil {
		return handleProviderError(c, providerID, err)
	}

	existingChapters, err := h.lib.ListChapters(mangaID)
	if err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
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
	updated := 0

	if len(toAdd) > 0 {
		const maxWorkers = 16
		numWorkers := len(toAdd)
		if numWorkers > maxWorkers {
			numWorkers = maxWorkers
		}
		sem := make(chan struct{}, numWorkers)
		var (
			wg      sync.WaitGroup
			errOnce sync.Once
			saveErr error
		)

		for _, ch := range toAdd {
			wg.Add(1)
			sem <- struct{}{}
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
				if err := h.lib.SaveChapter(mangaID, ch.ID, chMeta); err != nil {
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
			c.Set("handler_error", saveErr.Error())
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": saveErr.Error()})
		}
	}

	meta.Content.LastSyncedAt = now
	if err := h.lib.SaveManga(mangaID, meta); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"added":      int(added),
		"updated":    updated,
		"providerId": providerID,
		"mangaId":    mangaID,
	})
}

func (h *Handler) listChapters(c echo.Context) error {
	id := c.Param("mangaId")
	if id == "" {
		id = c.Param("id")
	}
	chapters, err := h.lib.ListChapters(id)
	if err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	var res []echo.Map
	for _, ch := range chapters {
		var uploadDateStr string
		if !ch.Meta.UploadDate.IsZero() {
			uploadDateStr = ch.Meta.UploadDate.UTC().Format(time.RFC3339)
		}
		res = append(res, echo.Map{
			"id":          ch.ID,
			"manga_id":    ch.MangaID,
			"title":       ch.Meta.Title,
			"number":      ch.Meta.Number,
			"volume":      ch.Meta.Volume,
			"uploadDate":  uploadDateStr,
			"sourceOrder": ch.Meta.SourceOrder,
			"meta":        ch.Meta,
		})
	}
	return c.JSON(http.StatusOK, echo.Map{
		"chapters": res,
	})
}

func (h *Handler) getChapter(c echo.Context) error {
	mangaID := c.Param("mangaId")
	if mangaID == "" {
		mangaID = c.Param("id")
	}
	chapterID := c.Param("chapterId")
	if chapterID == "" {
		chapterID = c.Param("ch")
	}
	meta, err := h.lib.GetChapter(mangaID, chapterID)
	if err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, echo.Map{
		"id":       chapterID,
		"manga_id": mangaID,
		"meta":     meta,
	})
}

func (h *Handler) saveChapter(c echo.Context) error {
	mangaID := c.Param("mangaId")
	if mangaID == "" {
		mangaID = c.Param("id")
	}
	chapterID := c.Param("chapterId")
	if chapterID == "" {
		chapterID = c.Param("ch")
	}
	var meta library.ChapterMeta
	if err := c.Bind(&meta); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	if err := h.lib.SaveChapter(mangaID, chapterID, &meta); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"id":       chapterID,
		"manga_id": mangaID,
		"meta":     meta,
	})
}

func (h *Handler) deleteChapter(c echo.Context) error {
	mangaID := c.Param("mangaId")
	if mangaID == "" {
		mangaID = c.Param("id")
	}
	chapterID := c.Param("chapterId")
	if chapterID == "" {
		chapterID = c.Param("ch")
	}
	if err := h.lib.DeleteChapter(mangaID, chapterID); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) patchChapterProgress(c echo.Context) error {
	mangaID := c.Param("id")
	if mangaID == "" {
		mangaID = c.Param("mangaId")
	}
	chapterID := c.Param("ch")
	if chapterID == "" {
		chapterID = c.Param("chapterId")
	}

	var req struct {
		IsRead       *bool `json:"is_read"`
		LastReadPage *int  `json:"last_read_page"`
	}
	if err := c.Bind(&req); err != nil {
		slog.Error("failed to bind chapter progress request",
			slog.String("error", err.Error()),
			slog.String("manga_id", mangaID),
			slog.String("chapter_id", chapterID),
		)
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	existing, err := h.lib.GetChapter(mangaID, chapterID)
	if err != nil {
		slog.Error("chapter not found for progress update",
			slog.String("error", err.Error()),
			slog.String("manga_id", mangaID),
			slog.String("chapter_id", chapterID),
		)
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

	info, err := h.lib.UpdateChapterProgress(mangaID, chapterID, isRead, lastReadPage)
	if err != nil {
		slog.Error("failed to update chapter progress",
			slog.String("error", err.Error()),
			slog.String("manga_id", mangaID),
			slog.String("chapter_id", chapterID),
		)
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, info)
}

func (h *Handler) refreshChapterPages(c echo.Context) error {
	mangaID := c.Param("mangaId")
	if mangaID == "" {
		mangaID = c.Param("id")
	}
	if mangaID == "" {
		mangaID = c.QueryParam("mangaId")
	}
	if mangaID == "" {
		mangaID = c.QueryParam("mangaRef")
	}
	if mangaID == "" {
		mangaID = c.Param("remoteId")
	}

	chapterID := c.Param("chapterId")
	if chapterID == "" {
		chapterID = c.Param("ch")
	}
	if chapterID == "" {
		chapterID = c.QueryParam("chapterId")
	}
	if chapterID == "" {
		chapterID = c.QueryParam("ch")
	}

	if chapterID == "" {
		c.Set("handler_error", "chapterId is required")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "chapterId is required"})
	}

	providerID := c.QueryParam("providerId")
	if providerID == "" {
		providerID = c.QueryParam("provider")
	}

	remoteMangaID := mangaID
	remoteChapterID := chapterID

	if mangaID != "" {
		if chMeta, err := h.lib.GetChapter(mangaID, chapterID); err == nil && chMeta.Content != nil {
			if providerID == "" && chMeta.Content.ProviderID != "" {
				providerID = chMeta.Content.ProviderID
			}
			if chMeta.Content.ChapterRef != "" {
				remoteChapterID = chMeta.Content.ChapterRef
			}
		}
		if providerID == "" {
			if mangaMeta, err := h.lib.GetManga(mangaID); err == nil && mangaMeta.Content != nil {
				if mangaMeta.Content.ProviderID != "" {
					providerID = mangaMeta.Content.ProviderID
				}
				if mangaMeta.Content.MangaID != "" {
					remoteMangaID = mangaMeta.Content.MangaID
				}
			}
		}
	}

	if providerID == "" {
		if mangas, err := h.lib.ListManga(); err == nil && len(mangas) > 0 {
			for _, m := range mangas {
				if chMeta, err := h.lib.GetChapter(m.ID, chapterID); err == nil && chMeta.Content != nil && chMeta.Content.ProviderID != "" {
					providerID = chMeta.Content.ProviderID
					if mangaID == "" {
						mangaID = m.ID
						remoteMangaID = m.ID
					}
					if chMeta.Content.ChapterRef != "" {
						remoteChapterID = chMeta.Content.ChapterRef
					}
					break
				}
			}
		}
	}

	if providerID == "" {
		providerID = mangadex.ProviderID
	}

	_, contentProvider, err := h.getProvider(providerID)
	if err != nil {
		contentProvider, _ = h.registry.GetContent(mangadex.ProviderID)
	}
	if contentProvider == nil {
		return handleProviderError(c, providerID, fmt.Errorf("provider not available: %s", providerID))
	}

	pages, err := contentProvider.FetchPages(c.Request().Context(), remoteMangaID, remoteChapterID)
	if err != nil {
		return handleProviderError(c, providerID, err)
	}

	pageItems := make([]library.PageItem, 0, len(pages))
	for _, p := range pages {
		pageItems = append(pageItems, library.PageItem{
			Index:  p.Index,
			URL:    p.URL,
			Source: "provider",
		})
	}

	if err := h.lib.SaveChapterPages(mangaID, chapterID, pageItems); err != nil {
		slog.Error("failed to save refreshed chapter pages",
			slog.String("error", err.Error()),
			slog.String("manga_id", mangaID),
			slog.String("chapter_id", chapterID),
		)
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	resPages := make([]echo.Map, 0, len(pages))
	for _, p := range pages {
		resPages = append(resPages, echo.Map{
			"index":  p.Index,
			"url":    p.URL,
			"source": "provider",
		})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"pages":   resPages,
		"message": "chapter pages refreshed successfully",
	})
}

