package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/tubruk/kiyomi/internal/library"
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

func (h *Handler) refreshChaptersFromContent(ctx context.Context, mangaID string) (int, error) {
	meta, err := h.lib.GetManga(mangaID)
	if err != nil {
		return 0, err
	}

	if meta.Content == nil || meta.Content.ProviderID == "" {
		return 0, fmt.Errorf("manga has no connected content provider")
	}

	providerID := meta.Content.ProviderID
	_, contentProvider, err := h.getProvider(providerID)
	if err != nil {
		return 0, err
	}

	remoteID := meta.Content.ProviderMangaID
	if remoteID == "" {
		remoteID = mangaID
	}

	chapters, err := contentProvider.FetchChapters(ctx, remoteID)
	if err != nil {
		return 0, err
	}

	existingChapters, err := h.lib.ListChapters(mangaID)
	if err != nil {
		return 0, err
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

	if len(toAdd) > 0 {
		const maxWorkers = 16
		numWorkers := min(len(toAdd), maxWorkers)
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
			return int(added), saveErr
		}
	}

	meta.Content.LastSyncedAt = now
	if err := h.lib.SaveManga(mangaID, meta); err != nil {
		return int(added), err
	}

	return int(added), nil
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
	added, err := h.refreshChaptersFromContent(c.Request().Context(), mangaID)
	if err != nil {
		return handleProviderError(c, providerID, err)
	}

	return c.JSON(http.StatusOK, echo.Map{
		"added":       added,
		"updated":     0,
		"provider_id": providerID,
		"manga_id":    mangaID,
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
				if mangaMeta.Content.ProviderMangaID != "" {
					remoteMangaID = mangaMeta.Content.ProviderMangaID
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
		c.Set("handler_error", "provider ID could not be determined for chapter")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "provider ID could not be determined for chapter"})
	}

	_, contentProvider, err := h.getProvider(providerID)
	if err != nil || contentProvider == nil {
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
	if mangaID == "" {
		mangaID = c.Param("id")
	}
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
	if mangaID == "" {
		mangaID = c.Param("id")
	}
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
		if err := h.lib.DeleteAllChapters(mangaID); err != nil {
			slog.Warn("delete all chapters before refresh", "manga_id", mangaID, "error", err)
		}
		var refreshErr error
		added, refreshErr = h.refreshChaptersFromContent(c.Request().Context(), mangaID)
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
	if mangaID == "" {
		mangaID = c.Param("id")
	}
	providerID := c.Param("providerId")
	providerMangaID := c.Param("providerMangaID")

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
	if mangaID == "" {
		mangaID = c.Param("id")
	}
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

	added := 0
	var refreshErr error
	if err := h.lib.DeleteAllChapters(mangaID); err != nil {
		slog.Warn("delete all chapters before refresh", "manga_id", mangaID, "error", err)
	}
	added, refreshErr = h.refreshChaptersFromContent(c.Request().Context(), mangaID)
	if refreshErr != nil {
		c.Set("handler_error", refreshErr.Error())
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": refreshErr.Error()})
	}

	updated, _ := h.lib.GetManga(mangaID)
	return c.JSON(http.StatusOK, echo.Map{
		"id":    mangaID,
		"meta":  updated,
		"added": added,
	})
}

