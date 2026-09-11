package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
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
	for _, info := range mangas {
		var providerID string
		var readingMode string
		if info.Bindings.Content != nil {
			providerID = info.Bindings.Content.ProviderID
			readingMode = info.Bindings.Content.ReadingMode
		}
		item := echo.Map{
			"id":                  info.ID,
			"title":               info.Metadata.Title,
			"cover":               info.Metadata.CoverURL,
			"content_provider_id": providerID,
			"sourceId":            providerID,
			"external_links":      info.Metadata.ExternalLinks,
			"externalLinks":       info.Metadata.ExternalLinks,
			"metadata":            info.Metadata,
			"user_state":          info.UserState,
			"bindings":            info.Bindings,
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
	info, err := h.lib.GetManga(id)
	if err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
	}
	var providerID string
	var readingMode string
	if info.Bindings.Content != nil {
		providerID = info.Bindings.Content.ProviderID
		readingMode = info.Bindings.Content.ReadingMode
	}
	resp := echo.Map{
		"id":                  id,
		"title":               info.Metadata.Title,
		"aliases":             info.Metadata.Aliases,
		"cover":               info.Metadata.CoverURL,
		"description":         info.Metadata.Description,
		"authors":             info.Metadata.Authors,
		"artists":             info.Metadata.Artists,
		"tags":                info.Metadata.Tags,
		"content_provider_id": providerID,
		"sourceId":            providerID,
		"external_links":      info.Metadata.ExternalLinks,
		"externalLinks":       info.Metadata.ExternalLinks,
		"metadata":            info.Metadata,
		"user_state":          info.UserState,
		"bindings":            info.Bindings,
	}
	if readingMode != "" {
		resp["reading_mode"] = readingMode
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) createLibraryManga(c echo.Context) error {
	var body struct {
		ID        string                 `json:"id"`
		Metadata  library.MangaMetadata  `json:"metadata"`
		UserState library.UserState      `json:"user_state"`
		Bindings  library.MangaBindings  `json:"bindings"`
		Content   *library.ContentSource `json:"content"`
	}
	if err := c.Bind(&body); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	if body.ID == "" {
		c.Set("handler_error", "id is required")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "id is required"})
	}

	if body.UserState.Status != "" && !library.IsValidUserStatus(body.UserState.Status) {
		c.Set("handler_error", "invalid user_status")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid user_status"})
	}

	if body.Bindings.Content == nil && body.Content != nil {
		body.Bindings.Content = body.Content
	}

	if err := h.lib.SaveManga(body.ID, library.Manga{
		Metadata:  body.Metadata,
		UserState: body.UserState,
		Bindings:  body.Bindings,
	}); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, echo.Map{
		"id":         body.ID,
		"metadata":   body.Metadata,
		"user_state": body.UserState,
		"bindings":   body.Bindings,
	})
}

func (h *Handler) updateLibraryManga(c echo.Context) error {
	id := c.Param("mangaId")
	var body library.Manga
	if err := c.Bind(&body); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	if body.UserState.Status != "" && !library.IsValidUserStatus(body.UserState.Status) {
		c.Set("handler_error", "invalid user_status")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid user_status"})
	}

	if err := h.lib.SaveManga(id, body); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"id":         id,
		"metadata":   body.Metadata,
		"user_state": body.UserState,
		"bindings":   body.Bindings,
	})
}

func (h *Handler) patchLibraryManga(c echo.Context) error {
	id := c.Param("mangaId")
	var body struct {
		Metadata   *library.MangaMetadata `json:"metadata,omitempty"`
		UserState  *library.UserState     `json:"user_state,omitempty"`
		Bindings   *library.MangaBindings `json:"bindings,omitempty"`
		UserStatus string                 `json:"user_status,omitempty"`
		UserRating *float64               `json:"user_rating,omitempty"`
		UserFav    *bool                  `json:"user_favorite,omitempty"`
		UserNotes  string                 `json:"user_notes,omitempty"`
	}
	if err := c.Bind(&body); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	// Legacy callers (api.patchLibraryManga) still send flat user_* fields.
	// Merge them into UserState so the deprecated endpoint keeps working
	// until every consumer migrates to /library/manga/:id/user_state. Always
	// start from the stored state so unrelated fields (status, rating,
	// favorite, notes) survive a single-field PATCH.
	if body.UserState == nil &&
		(body.UserStatus != "" || body.UserRating != nil || body.UserFav != nil || body.UserNotes != "") {
		existing, err := h.lib.GetUserState(id)
		if err != nil {
			c.Set("handler_error", err.Error())
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
		}
		body.UserState = &existing
	}
	if body.UserState != nil {
		if body.UserStatus != "" {
			body.UserState.Status = body.UserStatus
		}
		if body.UserRating != nil {
			body.UserState.Rating = *body.UserRating
		}
		if body.UserFav != nil {
			body.UserState.Favorite = *body.UserFav
		}
		if body.UserNotes != "" {
			body.UserState.Notes = body.UserNotes
		}
	}

	if body.UserState != nil && body.UserState.Status != "" && !library.IsValidUserStatus(body.UserState.Status) {
		c.Set("handler_error", "invalid user_status")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid user_status"})
	}

	info := library.Manga{ID: id}
	if body.Metadata != nil {
		if err := h.lib.SaveMetadata(id, *body.Metadata); err != nil {
			c.Set("handler_error", err.Error())
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
		}
		info.Metadata = *body.Metadata
	} else {
		m, err := h.lib.GetMetadata(id)
		if err != nil {
			c.Set("handler_error", err.Error())
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
		}
		info.Metadata = m
	}

	if body.UserState != nil {
		if err := h.lib.SaveUserState(id, *body.UserState); err != nil {
			c.Set("handler_error", err.Error())
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
		}
		info.UserState = *body.UserState
	} else {
		u, err := h.lib.GetUserState(id)
		if err != nil {
			c.Set("handler_error", err.Error())
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
		}
		info.UserState = u
	}

	if body.Bindings != nil {
		if err := h.lib.SaveBindings(id, *body.Bindings); err != nil {
			c.Set("handler_error", err.Error())
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
		}
		info.Bindings = *body.Bindings
	} else {
		b, err := h.lib.GetBindings(id)
		if err != nil {
			c.Set("handler_error", err.Error())
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
		}
		info.Bindings = b
	}

	return c.JSON(http.StatusOK, echo.Map{
		"id":         id,
		"metadata":   info.Metadata,
		"user_state": info.UserState,
		"bindings":   info.Bindings,
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
	info, err := h.lib.GetManga(mangaID)
	if err != nil {
		return 0, 0, err
	}

	if info.Bindings.Content == nil || info.Bindings.Content.ProviderID == "" {
		return 0, 0, fmt.Errorf("manga has no connected content provider")
	}

	providerID = info.Bindings.Content.ProviderID
	_, contentProvider, err := h.getProvider(providerID)
	if err != nil {
		return 0, 0, err
	}

	remoteID := info.Bindings.Content.ProviderMangaID
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
	if info.Bindings.Content.ProviderID != library.LocalProviderID {
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

	if info.Bindings.Content != nil {
		info.Bindings.Content.LastSyncedAt = now
		if err := h.lib.SaveBindings(mangaID, info.Bindings); err != nil {
			return int(added), int(orphaned), err
		}
	}

	return int(added), int(orphaned), nil
}

func (h *Handler) refreshLibraryManga(c echo.Context) error {
	mangaID := c.Param("mangaId")

	info, err := h.lib.GetManga(mangaID)
	if err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
	}

	if info.Bindings.Content == nil || info.Bindings.Content.ProviderID == "" {
		c.Set("handler_error", "manga has no connected content provider")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "manga has no connected content provider"})
	}

	providerID := info.Bindings.Content.ProviderID
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
		if info, infoErr := h.lib.GetManga(id); infoErr == nil && info.Bindings.Content != nil {
			activeProviderID = info.Bindings.Content.ProviderID
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

// findChapter resolves a chapter's metadata and provider ID by searching the manga's
// provider directories. Returns the chapter metadata, provider ID, or an error.
func (h *Handler) findChapter(mangaID, chapterID string) (*library.ChapterMeta, string, error) {
	meta, foundProviderID, err := h.lib.FindChapter(mangaID, chapterID)
	if err != nil {
		return nil, "", err
	}
	providerID := foundProviderID
	if meta != nil && meta.Content != nil && meta.Content.ProviderID != "" {
		providerID = meta.Content.ProviderID
	}
	return meta, providerID, nil
}

func (h *Handler) getChapter(c echo.Context) error {
	mangaID := c.Param("mangaId")
	chapterID := c.Param("chapterId")

	meta, providerID, err := h.findChapter(mangaID, chapterID)
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
	chapterID := c.Param("chapterId")

	var body struct {
		ID         string `json:"id"`
		ChapterID  string `json:"chapter_id"`
		ProviderID string `json:"provider_id"`
		library.ChapterMeta
	}
	if err := c.Bind(&body); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	meta := body.ChapterMeta
	if chapterID == "" {
		chapterID = body.ID
		if chapterID == "" {
			chapterID = body.ChapterID
		}
		if chapterID == "" && meta.Content != nil {
			chapterID = meta.Content.ChapterRef
		}
	}

	if chapterID == "" {
		c.Set("handler_error", "chapterId is required")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "chapterId is required"})
	}

	// Resolve provider ID
	providerID := ""
	if existingMeta, existingProvID, err := h.findChapter(mangaID, chapterID); err == nil {
		providerID = existingProvID
		if existingMeta != nil && existingMeta.Content != nil && existingMeta.Content.ProviderID != "" {
			providerID = existingMeta.Content.ProviderID
		}
	}
	if providerID == "" && meta.Content != nil && meta.Content.ProviderID != "" {
		providerID = meta.Content.ProviderID
	}
	if providerID == "" && body.ProviderID != "" {
		providerID = body.ProviderID
	}
	if providerID == "" {
		providerID = c.QueryParam("provider_id")
	}
	if providerID == "" {
		providerID = c.QueryParam("providerId")
	}
	if providerID == "" {
		if manga, err := h.lib.GetManga(mangaID); err == nil && manga.Bindings.Content != nil {
			providerID = manga.Bindings.Content.ProviderID
		}
	}
	if providerID == "" {
		providerID = library.LocalProviderID
	}

	// If chapterID is not empty, set it on the chapter before saving
	if meta.Content == nil {
		meta.Content = &library.ContentSource{
			ProviderID: providerID,
			ChapterRef: chapterID,
		}
	} else {
		if meta.Content.ProviderID == "" {
			meta.Content.ProviderID = providerID
		}
		if meta.Content.ChapterRef == "" {
			meta.Content.ChapterRef = chapterID
		}
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
	chapterID := c.Param("chapterId")

	_, providerID, err := h.findChapter(mangaID, chapterID)
	if err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
	}

	if err := h.lib.DeleteChapter(mangaID, providerID, chapterID); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) patchChapterProgress(c echo.Context) error {
	mangaID := c.Param("mangaId")
	chapterID := c.Param("chapterId")

	var req struct {
		IsRead       *bool `json:"is_read"`
		LastReadPage *int  `json:"last_read_page"`
	}
	if err := c.Bind(&req); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	existing, providerID, err := h.findChapter(mangaID, chapterID)
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

	var req struct {
		ChapterIDs   []string `json:"chapter_ids"`
		IsRead       *bool    `json:"is_read"`
		LastReadPage *int     `json:"last_read_page"`
	}
	if err := c.Bind(&req); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	if mangaID == "" || len(req.ChapterIDs) == 0 {
		c.Set("handler_error", "mangaId and chapter_ids are required")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "mangaId and chapter_ids are required"})
	}

	isRead := false
	if req.IsRead != nil {
		isRead = *req.IsRead
	}

	lastReadPage := 0
	if req.LastReadPage != nil {
		lastReadPage = *req.LastReadPage
	}

	type chapterEntry struct {
		id         string
		providerID string
	}
	var entries []chapterEntry
	for _, chID := range req.ChapterIDs {
		_, provID, err := h.findChapter(mangaID, chID)
		if err != nil {
			c.Set("handler_error", fmt.Sprintf("chapter %s not found", chID))
			return c.JSON(http.StatusNotFound, echo.Map{"error": fmt.Sprintf("chapter %s not found", chID)})
		}
		entries = append(entries, chapterEntry{id: chID, providerID: provID})
	}

	byProvider := make(map[string][]string)
	for _, e := range entries {
		byProvider[e.providerID] = append(byProvider[e.providerID], e.id)
	}

	totalUpdated := 0
	updatedMap := make(map[string]bool)
	for provID, chIDs := range byProvider {
		updated, err := h.lib.BatchUpdateChapterProgress(mangaID, provID, chIDs, isRead, lastReadPage)
		if err != nil {
			c.Set("handler_error", err.Error())
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
		}
		totalUpdated += len(updated)
		for _, u := range updated {
			updatedMap[u.ID] = true
		}
	}

	var resultIDs []string
	for _, id := range req.ChapterIDs {
		if updatedMap[id] {
			resultIDs = append(resultIDs, id)
		}
	}

	return c.JSON(http.StatusOK, echo.Map{
		"updated":     totalUpdated,
		"chapter_ids": resultIDs,
	})
}

// pullChapter handles POST /library/manga/:mangaId/chapters/:chapterId/pull.
// Creates and enqueues a pull_chapter job for the specified chapter.
func (h *Handler) pullChapter(c echo.Context) error {
	mangaID := c.Param("mangaId")
	chapterID := c.Param("chapterId")

	if mangaID == "" || chapterID == "" {
		c.Set("handler_error", "mangaId and chapterId are required")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "mangaId and chapterId are required"})
	}

	if _, err := h.lib.GetManga(mangaID); err != nil {
		c.Set("handler_error", "manga not found")
		return c.JSON(http.StatusNotFound, echo.Map{"error": "manga not found"})
	}

	_, providerID, err := h.findChapter(mangaID, chapterID)
	if err != nil {
		c.Set("handler_error", "chapter not found")
		return c.JSON(http.StatusNotFound, echo.Map{"error": "chapter not found"})
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

// pullChaptersBatch handles POST /library/manga/:mangaId/batch/chapters/pull.
// Creates and enqueues a pull_chapter job for each requested chapter ID.
func (h *Handler) pullChaptersBatch(c echo.Context) error {
	mangaID := c.Param("mangaId")

	var req struct {
		ChapterIDs []string `json:"chapter_ids"`
	}
	if err := c.Bind(&req); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	if mangaID == "" || len(req.ChapterIDs) == 0 {
		c.Set("handler_error", "mangaId and chapter_ids are required")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "mangaId and chapter_ids are required"})
	}

	if _, err := h.lib.GetManga(mangaID); err != nil {
		c.Set("handler_error", "manga not found")
		return c.JSON(http.StatusNotFound, echo.Map{"error": "manga not found"})
	}

	if h.jobStore == nil || h.enqueuer == nil {
		return c.JSON(http.StatusServiceUnavailable, echo.Map{"error": "job queue not configured"})
	}

	type chapterJobInfo struct {
		chapterID  string
		providerID string
	}
	jobsToCreate := make([]chapterJobInfo, 0, len(req.ChapterIDs))
	for _, chapterID := range req.ChapterIDs {
		_, providerID, err := h.findChapter(mangaID, chapterID)
		if err != nil {
			c.Set("handler_error", fmt.Sprintf("chapter %s not found", chapterID))
			return c.JSON(http.StatusNotFound, echo.Map{"error": fmt.Sprintf("chapter %s not found", chapterID)})
		}
		if err := h.requireContentCapability(providerID); err != nil {
			c.Set("handler_error", err.Error())
			return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
		}
		jobsToCreate = append(jobsToCreate, chapterJobInfo{chapterID: chapterID, providerID: providerID})
	}

	now := time.Now()
	jobIDs := make([]string, 0, len(jobsToCreate))
	for _, item := range jobsToCreate {
		payloadBytes, err := json.Marshal(queue.PullChapterPayload{
			MangaID:    mangaID,
			ProviderID: item.providerID,
			ChapterID:  item.chapterID,
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
			ConcurrencyGroup: "pull:" + item.providerID,
			CreatedAt:        now,
			UpdatedAt:        now,
			Metadata: map[string]string{
				"manga_id":    mangaID,
				"provider_id": item.providerID,
				"chapter_id":  item.chapterID,
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

func (h *Handler) refreshChapter(c echo.Context) error {
	mangaID := c.Param("mangaId")
	chapterID := c.Param("chapterId")

	if mangaID == "" || chapterID == "" {
		c.Set("handler_error", "mangaId and chapterId are required")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "mangaId and chapterId are required"})
	}

	chMeta, providerID, err := h.findChapter(mangaID, chapterID)
	if err != nil {
		c.Set("handler_error", "chapter not found")
		return c.JSON(http.StatusNotFound, echo.Map{"error": "chapter not found"})
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
	if info, err := h.lib.GetManga(mangaID); err == nil {
		if info.Bindings.Content != nil && info.Bindings.Content.ProviderID == providerID && info.Bindings.Content.ProviderMangaID != "" {
			remoteMangaID = info.Bindings.Content.ProviderMangaID
		} else {
			for _, p := range info.Bindings.Providers {
				if p.ProviderID == providerID && p.ProviderMangaID != "" {
					remoteMangaID = p.ProviderMangaID
					break
				}
			}
			if remoteMangaID == mangaID && info.Bindings.Content != nil && info.Bindings.Content.ProviderMangaID != "" {
				remoteMangaID = info.Bindings.Content.ProviderMangaID
			}
		}
	}

	chapters, err := contentProvider.FetchChapters(c.Request().Context(), remoteMangaID)
	if err != nil {
		return handleProviderError(c, providerID, err)
	}

	var upCh *sdk.Chapter
	for _, ch := range chapters {
		if ch.ID == chapterID || (chMeta.Content != nil && ch.ID == chMeta.Content.ChapterRef) {
			chCopy := ch
			upCh = &chCopy
			break
		}
	}

	if upCh == nil {
		c.Set("handler_error", "chapter not found in provider")
		return c.JSON(http.StatusNotFound, echo.Map{"error": "chapter not found in provider"})
	}

	now := time.Now()
	chMeta.Title = upCh.Name
	chMeta.Number = upCh.Number
	chMeta.UploadDate = upCh.UploadDate
	chMeta.SourceOrder = upCh.SourceOrder
	if chMeta.Content == nil {
		chMeta.Content = &library.ContentSource{
			ProviderID: providerID,
			ChapterRef: upCh.ID,
		}
	}
	chMeta.Content.LastSyncedAt = now

	if err := h.lib.SaveChapter(mangaID, providerID, chapterID, chMeta); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"id":          chapterID,
		"manga_id":    mangaID,
		"provider_id": providerID,
		"meta":        chMeta,
	})
}

func (h *Handler) refreshChaptersBatch(c echo.Context) error {
	mangaID := c.Param("mangaId")

	var req struct {
		ChapterIDs []string `json:"chapter_ids"`
	}
	if err := c.Bind(&req); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	if mangaID == "" || len(req.ChapterIDs) == 0 {
		c.Set("handler_error", "mangaId and chapter_ids are required")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "mangaId and chapter_ids are required"})
	}

	mangaInfo, err := h.lib.GetManga(mangaID)
	defaultProviderID := ""
	if err == nil && mangaInfo.Bindings.Content != nil {
		defaultProviderID = mangaInfo.Bindings.Content.ProviderID
	}

	type chapterTarget struct {
		id         string
		providerID string
	}
	var targets []chapterTarget
	for _, chID := range req.ChapterIDs {
		provID := defaultProviderID
		if _, foundProvID, findErr := h.findChapter(mangaID, chID); findErr == nil && foundProvID != "" {
			provID = foundProvID
		}
		if provID == "" {
			provID = library.LocalProviderID
		}
		if err := h.requireContentCapability(provID); err != nil {
			c.Set("handler_error", err.Error())
			return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
		}
		targets = append(targets, chapterTarget{id: chID, providerID: provID})
	}

	byProvider := make(map[string][]string)
	for _, t := range targets {
		byProvider[t.providerID] = append(byProvider[t.providerID], t.id)
	}

	now := time.Now()
	refreshedSet := make(map[string]bool)
	for providerID, chIDs := range byProvider {
		_, contentProvider, err := h.getProvider(providerID)
		if err != nil || contentProvider == nil {
			return handleProviderError(c, providerID, fmt.Errorf("provider not available: %s", providerID))
		}

		remoteMangaID := mangaID
		if mangaInfo.Bindings.Content != nil && mangaInfo.Bindings.Content.ProviderID == providerID && mangaInfo.Bindings.Content.ProviderMangaID != "" {
			remoteMangaID = mangaInfo.Bindings.Content.ProviderMangaID
		} else {
			for _, p := range mangaInfo.Bindings.Providers {
				if p.ProviderID == providerID && p.ProviderMangaID != "" {
					remoteMangaID = p.ProviderMangaID
					break
				}
			}
			if remoteMangaID == mangaID && mangaInfo.Bindings.Content != nil && mangaInfo.Bindings.Content.ProviderMangaID != "" {
				remoteMangaID = mangaInfo.Bindings.Content.ProviderMangaID
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

		for _, chID := range chIDs {
			upCh, found := upstreamMap[chID]
			if !found {
				// Also try matching by chapter ref if chMeta already exists
				if chMeta, err := h.lib.GetChapter(mangaID, providerID, chID); err == nil && chMeta.Content != nil && chMeta.Content.ChapterRef != "" {
					upCh, found = upstreamMap[chMeta.Content.ChapterRef]
				}
			}
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
			refreshedSet[chID] = true
		}
	}

	var refreshedIDs []string
	for _, id := range req.ChapterIDs {
		if refreshedSet[id] {
			refreshedIDs = append(refreshedIDs, id)
		}
	}

	return c.JSON(http.StatusOK, echo.Map{
		"refreshed":   len(refreshedIDs),
		"chapter_ids": refreshedIDs,
	})
}

func (h *Handler) getChapterFiles(c echo.Context) error {
	mangaID := c.Param("mangaId")
	chapterID := c.Param("chapterId")

	_, providerID, err := h.findChapter(mangaID, chapterID)
	if err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
	}

	chapterDir := h.lib.ProviderChapterDir(mangaID, providerID, chapterID)
	entries, err := os.ReadDir(chapterDir)
	if err != nil {
		if os.IsNotExist(err) {
			return c.JSON(http.StatusOK, echo.Map{"files": []echo.Map{}})
		}
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	exts := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".gif": true, ".avif": true,
	}
	var files []echo.Map
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if !exts[ext] {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, echo.Map{
			"name": entry.Name(),
			"size": info.Size(),
		})
	}
	if files == nil {
		files = []echo.Map{}
	}

	return c.JSON(http.StatusOK, echo.Map{
		"files": files,
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
	bindings, err := h.lib.GetBindings(mangaID)
	if err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
	}
	providers := bindings.Providers
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
	info, err := h.lib.GetManga(mangaID)
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
			title = info.Metadata.Title
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

	updated, _ := h.lib.GetBindings(mangaID)
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
	info, err := h.lib.GetManga(mangaID)
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
	if err := h.lib.SwitchContentProvider(mangaID, body.ProviderID, body.ProviderMangaID, info.Metadata.Title); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	updated, _ := h.lib.GetBindings(mangaID)
	return c.JSON(http.StatusOK, echo.Map{
		"id":       mangaID,
		"bindings": updated,
		"added":    0,
	})
}

// mergeLibraryManga handles POST /api/v1/library/manga/merge.
// Merges one or more source manga into a single keep manga, consolidating
// providers, metadata, chapters, and cover files. Source manga directories
// are removed after their content is moved into keep.
//
// content_provider is optional: when omitted, the handler inherits keep's
// current Content binding if it survives the merged Providers list.
// Chapter collisions always resolve keep-wins (no strategy exposed).
//
// Best-effort filesystem operations: if a chapter move fails the handler logs
// and continues — sources are still deleted, the merged meta is still saved.
// SaveManga is the point of no return; failures abort the handler.
func (h *Handler) mergeLibraryManga(c echo.Context) error {
	var body struct {
		KeepMangaID     string   `json:"keep_manga_id"`
		SourceMangaIDs  []string `json:"source_manga_ids"`
		ContentProvider *struct {
			ProviderID      string `json:"provider_id"`
			ProviderMangaID string `json:"provider_manga_id"`
		} `json:"content_provider"`
		Metadata map[string]string `json:"metadata"`
	}
	if err := c.Bind(&body); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	if body.KeepMangaID == "" {
		c.Set("handler_error", "keep_manga_id is required")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "keep_manga_id is required"})
	}
	if len(body.SourceMangaIDs) == 0 {
		c.Set("handler_error", "source_manga_ids is required")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "source_manga_ids is required"})
	}
	for _, srcID := range body.SourceMangaIDs {
		if srcID == body.KeepMangaID {
			c.Set("handler_error", "keep_manga_id cannot appear in source_manga_ids")
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "keep_manga_id cannot appear in source_manga_ids"})
		}
	}
	if body.ContentProvider != nil && (body.ContentProvider.ProviderID == "" || body.ContentProvider.ProviderMangaID == "") {
		c.Set("handler_error", "content_provider.provider_id and content_provider.provider_manga_id are required")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "content_provider.provider_id and content_provider.provider_manga_id are required"})
	}

	// 1. Load keep.
	keepInfo, err := h.lib.GetManga(body.KeepMangaID)
	if err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
	}

	// 2. Load sources.
	sources := make(map[string]library.Manga, len(body.SourceMangaIDs))
	for _, srcID := range body.SourceMangaIDs {
		srcInfo, err := h.lib.GetManga(srcID)
		if err != nil {
			c.Set("handler_error", err.Error())
			return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
		}
		sources[srcID] = srcInfo
	}

	// 3. Build merged metadata in a working copy of keep's metadata.
	mergedMeta := keepInfo.Metadata
	for _, srcInfo := range sources {
		mergedMeta.Aliases = append(mergedMeta.Aliases, srcInfo.Metadata.Aliases...)
		mergedMeta.Tags = append(mergedMeta.Tags, srcInfo.Metadata.Tags...)
		mergedMeta.Authors = append(mergedMeta.Authors, srcInfo.Metadata.Authors...)
		mergedMeta.Artists = append(mergedMeta.Artists, srcInfo.Metadata.Artists...)
		mergedMeta.Publishers = append(mergedMeta.Publishers, srcInfo.Metadata.Publishers...)
	}

	// Scalar fields: per-field selection. Only `source:<id>` overrides; "keep"
	// and "merge" both leave keep's value untouched. `metadata.providers` is
	// honored by step 4 (always union) and otherwise ignored here.
	if sel, ok := body.Metadata["title"]; ok {
		if srcID, found := strings.CutPrefix(sel, "source:"); found {
			if srcInfo, ok := sources[srcID]; ok {
				mergedMeta.Title = srcInfo.Metadata.Title
			}
		}
	}
	if sel, ok := body.Metadata["description"]; ok {
		if srcID, found := strings.CutPrefix(sel, "source:"); found {
			if srcInfo, ok := sources[srcID]; ok {
				mergedMeta.Description = srcInfo.Metadata.Description
			}
		}
	}
	if sel, ok := body.Metadata["release_year"]; ok {
		if srcID, found := strings.CutPrefix(sel, "source:"); found {
			if srcInfo, ok := sources[srcID]; ok {
				mergedMeta.ReleaseYear = srcInfo.Metadata.ReleaseYear
			}
		}
	}
	coverSel, _ := body.Metadata["cover_url"]
	if srcID, found := strings.CutPrefix(coverSel, "source:"); found {
		if srcInfo, ok := sources[srcID]; ok {
			mergedMeta.CoverURL = srcInfo.Metadata.CoverURL
		}
	}

	// 4. Bindings: providers union, content provider resolution.
	sourceProviders := make(map[string][]library.ProviderRef, len(sources))
	for srcID, srcInfo := range sources {
		sourceProviders[srcID] = srcInfo.Bindings.Providers
	}
	mergedProviders := mergeProviderRefs(keepInfo.Bindings.Providers, sourceProviders)

	cpProviderID, cpProviderMangaID := "", ""
	if body.ContentProvider != nil {
		cpProviderID = body.ContentProvider.ProviderID
		cpProviderMangaID = body.ContentProvider.ProviderMangaID
	} else if keepInfo.Bindings.Content != nil {
		cpProviderID = keepInfo.Bindings.Content.ProviderID
		cpProviderMangaID = keepInfo.Bindings.Content.ProviderMangaID
	} else {
		c.Set("handler_error", "cannot infer content_provider: keep manga has no active content provider that survives merge; specify content_provider explicitly")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "cannot infer content_provider: keep manga has no active content provider that survives merge; specify content_provider explicitly"})
	}

	mergedBindings := library.MangaBindings{
		Providers: mergedProviders,
		Content: &library.ContentSource{
			ProviderID:      cpProviderID,
			ProviderMangaID: cpProviderMangaID,
			ReadingMode:     keepReadingMode(keepInfo.Bindings.Content),
		},
	}

	cpValid := false
	for _, p := range mergedProviders {
		if p.ProviderID == cpProviderID && p.ProviderMangaID == cpProviderMangaID {
			cpValid = true
			break
		}
	}
	if !cpValid {
		c.Set("handler_error", "content_provider not present in merged providers list")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "content_provider not present in merged providers list"})
	}

	// 5. Move chapter dirs (best-effort, keep wins on collision).
	for _, srcID := range body.SourceMangaIDs {
		if err := h.lib.MergeChapters(body.KeepMangaID, srcID); err != nil {
			slog.Warn("merge: chapters move failed",
				slog.String("keep", body.KeepMangaID),
				slog.String("source", srcID),
				slog.String("error", err.Error()))
		}
	}

	// 6. Cover replacement (best-effort).
	if srcID, found := strings.CutPrefix(coverSel, "source:"); found {
		if _, ok := sources[srcID]; ok {
			if _, err := h.lib.ReplaceCoverFromSource(body.KeepMangaID, srcID); err != nil {
				slog.Warn("merge: cover replacement failed",
					slog.String("keep", body.KeepMangaID),
					slog.String("source", srcID),
					slog.String("error", err.Error()))
			}
		}
	}

	// 7. Delete source manga dirs (best-effort).
	for _, srcID := range body.SourceMangaIDs {
		if err := h.lib.DeleteManga(srcID); err != nil {
			slog.Warn("merge: source delete failed",
				slog.String("source", srcID),
				slog.String("error", err.Error()))
		}
	}

	// 8. Save merged keep per concern — point of no return.
	if err := h.lib.SaveMetadata(body.KeepMangaID, mergedMeta); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	if err := h.lib.SaveBindings(body.KeepMangaID, mergedBindings); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	// 9. Return updated Manga.
	saved, _ := h.lib.GetManga(body.KeepMangaID)
	var providerID, readingMode string
	if saved.Bindings.Content != nil {
		providerID = saved.Bindings.Content.ProviderID
		readingMode = saved.Bindings.Content.ReadingMode
	}
	resp := echo.Map{
		"id":                  body.KeepMangaID,
		"title":               saved.Metadata.Title,
		"aliases":             saved.Metadata.Aliases,
		"cover":               saved.Metadata.CoverURL,
		"description":         saved.Metadata.Description,
		"authors":             saved.Metadata.Authors,
		"artists":             saved.Metadata.Artists,
		"tags":                saved.Metadata.Tags,
		"content_provider_id": providerID,
		"sourceId":            providerID,
		"external_links":      saved.Metadata.ExternalLinks,
		"externalLinks":       saved.Metadata.ExternalLinks,
		"metadata":            saved.Metadata,
		"user_state":          saved.UserState,
		"bindings":            saved.Bindings,
	}
	if readingMode != "" {
		resp["reading_mode"] = readingMode
	}
	return c.JSON(http.StatusOK, resp)
}

// keepReadingMode returns the ReadingMode from the existing Content if any.
func keepReadingMode(c *library.ContentSource) string {
	if c == nil {
		return ""
	}
	return c.ReadingMode
}

// mergeProviderRefs returns the union of keep and source provider refs, deduped
// on (ProviderID, ProviderMangaID) and preserving first-seen order.
func mergeProviderRefs(keep []library.ProviderRef, sources map[string][]library.ProviderRef) []library.ProviderRef {
	seen := make(map[string]bool)
	var out []library.ProviderRef
	add := func(refs []library.ProviderRef) {
		for _, p := range refs {
			key := p.ProviderID + "\x00" + p.ProviderMangaID
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, p)
		}
	}
	add(keep)
	for _, srcRefs := range sources {
		add(srcRefs)
	}
	return out
}

func (h *Handler) deleteChapterFiles(c echo.Context) error {
	mangaID := c.Param("mangaId")
	chapterID := c.Param("chapterId")

	_, providerID, err := h.findChapter(mangaID, chapterID)
	if err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
	}

	if err := h.lib.DeleteChapterFiles(mangaID, providerID, chapterID); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) deleteChapterFilesBatch(c echo.Context) error {
	mangaID := c.Param("mangaId")

	var req struct {
		ChapterIDs []string `json:"chapter_ids"`
	}
	if err := c.Bind(&req); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	if mangaID == "" || len(req.ChapterIDs) == 0 {
		c.Set("handler_error", "mangaId and chapter_ids are required")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "mangaId and chapter_ids are required"})
	}

	byProvider := make(map[string][]string)
	for _, chID := range req.ChapterIDs {
		_, provID, err := h.findChapter(mangaID, chID)
		if err != nil {
			c.Set("handler_error", fmt.Sprintf("chapter %s not found", chID))
			return c.JSON(http.StatusNotFound, echo.Map{"error": fmt.Sprintf("chapter %s not found", chID)})
		}
		byProvider[provID] = append(byProvider[provID], chID)
	}

	for provID, chIDs := range byProvider {
		if err := h.lib.BatchDeleteChapterFiles(mangaID, provID, chIDs); err != nil {
			if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "no such file or directory") {
				c.Set("handler_error", err.Error())
				return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
			}
			c.Set("handler_error", err.Error())
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
		}
	}

	return c.JSON(http.StatusOK, echo.Map{
		"deleted": len(req.ChapterIDs),
	})
}

func (h *Handler) deleteChaptersBatch(c echo.Context) error {
	mangaID := c.Param("mangaId")

	var req struct {
		ChapterIDs []string `json:"chapter_ids"`
	}
	if err := c.Bind(&req); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	if mangaID == "" || len(req.ChapterIDs) == 0 {
		c.Set("handler_error", "mangaId and chapter_ids are required")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "mangaId and chapter_ids are required"})
	}

	byProvider := make(map[string][]string)
	for _, chID := range req.ChapterIDs {
		_, provID, err := h.findChapter(mangaID, chID)
		if err == nil && provID != "" {
			byProvider[provID] = append(byProvider[provID], chID)
		}
	}

	for provID, chIDs := range byProvider {
		if err := h.lib.BatchDeleteChapters(mangaID, provID, chIDs); err != nil {
			c.Set("handler_error", err.Error())
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
		}
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

	info, err := h.lib.GetManga(mangaID)
	if err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
	}

	providerID := c.QueryParam("provider_id")
	if providerID == "" {
		if info.Bindings.Content == nil || info.Bindings.Content.ProviderID == "" {
			c.Set("handler_error", "manga has no connected content provider and provider_id query param missing")
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "manga has no connected content provider and provider_id query param missing"})
		}
		providerID = info.Bindings.Content.ProviderID
	}

	providerMangaID := c.QueryParam("provider_manga_id")
	if providerMangaID == "" && info.Bindings.Content != nil {
		providerMangaID = info.Bindings.Content.ProviderMangaID
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
		CoverURL:        info.Metadata.CoverURL,
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

// getLibraryMetadata handles GET /library/manga/:mangaId/metadata.
func (h *Handler) getLibraryMetadata(c echo.Context) error {
	id := c.Param("mangaId")
	meta, err := h.lib.GetMetadata(id)
	if err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, meta)
}

// patchLibraryMetadata handles PATCH /library/manga/:mangaId/metadata.
func (h *Handler) patchLibraryMetadata(c echo.Context) error {
	id := c.Param("mangaId")
	var body library.MangaMetadata
	if err := c.Bind(&body); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}
	if err := h.lib.SaveMetadata(id, body); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, body)
}

// getLibraryUserState handles GET /library/manga/:mangaId/user_state.
func (h *Handler) getLibraryUserState(c echo.Context) error {
	id := c.Param("mangaId")
	us, err := h.lib.GetUserState(id)
	if err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, us)
}

// patchLibraryUserState handles PATCH /library/manga/:mangaId/user_state.
func (h *Handler) patchLibraryUserState(c echo.Context) error {
	id := c.Param("mangaId")
	var body library.UserState
	if err := c.Bind(&body); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}
	if body.Status != "" && !library.IsValidUserStatus(body.Status) {
		c.Set("handler_error", "invalid user_status")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid user_status"})
	}
	if err := h.lib.SaveUserState(id, body); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, body)
}

// getLibraryBindings handles GET /library/manga/:mangaId/bindings.
func (h *Handler) getLibraryBindings(c echo.Context) error {
	id := c.Param("mangaId")
	b, err := h.lib.GetBindings(id)
	if err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, b)
}

// refreshMetadata handles POST /library/manga/:mangaId/metadata/refresh.
// Enqueues a refresh_metadata job that pulls fresh details from the manga's
// active content provider's metadata capability and writes to metadata.json.
func (h *Handler) refreshMetadata(c echo.Context) error {
	if h.jobStore == nil || h.enqueuer == nil {
		c.Set("handler_error", "job queue not configured")
		return c.JSON(http.StatusServiceUnavailable, echo.Map{"error": "job queue not configured"})
	}

	mangaID := c.Param("mangaId")
	if mangaID == "" {
		c.Set("handler_error", "mangaId is required")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "mangaId is required"})
	}

	info, err := h.lib.GetManga(mangaID)
	if err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
	}

	// Pick the provider to refresh from. Default to the active content
	// provider; fallback to the first entry in bindings.Providers.
	providerID := ""
	providerMangaID := ""
	if info.Bindings.Content != nil && info.Bindings.Content.ProviderID != "" {
		providerID = info.Bindings.Content.ProviderID
		providerMangaID = info.Bindings.Content.ProviderMangaID
	} else if len(info.Bindings.Providers) > 0 {
		providerID = info.Bindings.Providers[0].ProviderID
		providerMangaID = info.Bindings.Providers[0].ProviderMangaID
	}
	if providerID == "" || providerMangaID == "" {
		c.Set("handler_error", "manga has no provider binding to refresh metadata from")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "manga has no provider binding to refresh metadata from"})
	}

	payloadBytes, err := json.Marshal(queue.RefreshMetadataPayload{
		MangaID:         mangaID,
		ProviderID:      providerID,
		ProviderMangaID: providerMangaID,
	})
	if err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": fmt.Sprintf("marshal refresh_metadata payload: %v", err)})
	}

	now := time.Now()
	job := &queue.Job{
		ID:         uuid.New().String(),
		Type:       queue.JobTypeRefreshMetadata,
		Payload:    string(payloadBytes),
		Status:     queue.StatusPending,
		MaxRetries: 3,
		// Metadata refreshes are not provider-fan-out hot — share a single bucket.
		ConcurrencyGroup: "refresh_metadata",
		CreatedAt:        now,
		UpdatedAt:        now,
		Metadata: map[string]string{
			"manga_id":    mangaID,
			"provider_id": providerID,
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
		"job_id":      job.ID,
		"provider_id": providerID,
	})
}
