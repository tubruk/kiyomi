package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/tubruk/kiyomi/internal/library"
	"github.com/tubruk/kiyomi/pkg/provider/sdk"
)

func (h *Handler) getProvider(providerID string) (sdk.Metadata, sdk.Content, error) {
	meta, okMeta := h.registry.GetMetadata(providerID)
	content, okContent := h.registry.GetContent(providerID)
	if !okMeta || !okContent {
		return nil, nil, fmt.Errorf("provider not found: %s", providerID)
	}
	return meta, content, nil
}

func (h *Handler) listContentProviders(c echo.Context) error {
	contentProviders := h.registry.ListContent()
	providers := make([]echo.Map, 0, len(contentProviders))
	for _, cp := range contentProviders {
		baseURL := ""
		lang := "en"
		if hsGetter, ok := cp.(interface{ GetConfig() sdk.ProviderConfig }); ok {
			cfg := hsGetter.GetConfig()
			baseURL = cfg.BaseURL
			if cfg.Language != "" {
				lang = cfg.Language
			}
		}
		caps := []string{"content"}
		if prov, ok := h.registry.Get(cp.ID()); ok {
			caps = prov.Capabilities()
		}

		providers = append(providers, echo.Map{
			"id":           cp.ID(),
			"name":         cp.Name(),
			"icon":         cp.Icon(),
			"baseUrl":      baseURL,
			"lang":         lang,
			"hasLatest":    true,
			"capabilities": caps,
		})
	}
	sort.Slice(providers, func(i, j int) bool {
		return providers[i]["id"].(string) < providers[j]["id"].(string)
	})
	return c.JSON(http.StatusOK, providers)
}

func (h *Handler) getProviderMangaCatalog(c echo.Context) error {
	providerID := c.Param("providerId")
	if providerID == "" {
		providerID = c.Param("id")
	}
	metaProvider, _, err := h.getProvider(providerID)
	if err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
	}

	q := c.QueryParam("q")
	mode := c.QueryParam("mode")
	if mode == "" {
		mode = "popular"
	}
	pageStr := c.QueryParam("page")
	page, _ := strconv.Atoi(pageStr)
	if page <= 0 {
		page = 1
	}

	var opts sdk.SearchOptions
	var results []sdk.SearchResult
	ctx := c.Request().Context()

	if q != "" {
		opts = sdk.SearchOptions{
			Limit:  20,
			Offset: (page - 1) * 20,
		}
		results, err = metaProvider.Search(ctx, q, opts)
	} else {
		opts = sdk.SearchOptions{
			Limit:  20,
			Offset: (page - 1) * 20,
			Mode:   mode,
		}
		results, err = metaProvider.Search(ctx, "", opts)
	}

	if err != nil {
		return handleProviderError(c, providerID, err)
	}

	mangas := make([]echo.Map, 0, len(results))
	for _, res := range results {
		manga := echo.Map{
			"id":       res.RemoteID,
			"title":    res.Title,
			"cover":    res.CoverURL,
			"provider": providerID,
		}
		if res.Availability != "" {
			manga["availability"] = res.Availability
		}
		mangas = append(mangas, manga)
	}

	return c.JSON(http.StatusOK, echo.Map{
		"mangas":  mangas,
		"page":    page,
		"hasNext": len(results) >= 20,
	})
}

func (h *Handler) getPopularManga(c echo.Context) error {
	providerID := c.Param("providerId")
	if providerID == "" {
		providerID = c.Param("id")
	}
	metaProvider, _, err := h.getProvider(providerID)
	if err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
	}

	pageStr := c.QueryParam("page")
	page, _ := strconv.Atoi(pageStr)
	if page <= 0 {
		page = 1
	}

	results, err := metaProvider.Search(c.Request().Context(), "", sdk.SearchOptions{
		Limit:  20,
		Offset: (page - 1) * 20,
		Mode:   "popular",
	})
	if err != nil {
		return handleProviderError(c, providerID, err)
	}

	var mangas []echo.Map
	for _, res := range results {
		manga := echo.Map{
			"id":       res.RemoteID,
			"title":    res.Title,
			"cover":    res.CoverURL,
			"provider": providerID,
		}
		if res.Availability != "" {
			manga["availability"] = res.Availability
		}
		mangas = append(mangas, manga)
	}

	return c.JSON(http.StatusOK, echo.Map{
		"mangas":  mangas,
		"page":    page,
		"hasNext": len(results) >= 20,
	})
}

func (h *Handler) getLatestManga(c echo.Context) error {
	providerID := c.Param("providerId")
	if providerID == "" {
		providerID = c.Param("id")
	}
	metaProvider, _, err := h.getProvider(providerID)
	if err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
	}

	pageStr := c.QueryParam("page")
	page, _ := strconv.Atoi(pageStr)
	if page <= 0 {
		page = 1
	}

	results, err := metaProvider.Search(c.Request().Context(), "", sdk.SearchOptions{
		Limit:  20,
		Offset: (page - 1) * 20,
		Mode:   "latest",
	})
	if err != nil {
		return handleProviderError(c, providerID, err)
	}

	var mangas []echo.Map
	for _, res := range results {
		manga := echo.Map{
			"id":       res.RemoteID,
			"title":    res.Title,
			"cover":    res.CoverURL,
			"provider": providerID,
		}
		if res.Availability != "" {
			manga["availability"] = res.Availability
		}
		mangas = append(mangas, manga)
	}

	return c.JSON(http.StatusOK, echo.Map{
		"mangas":  mangas,
		"page":    page,
		"hasNext": len(results) >= 20,
	})
}

func (h *Handler) searchManga(c echo.Context) error {
	providerID := c.Param("providerId")
	if providerID == "" {
		providerID = c.Param("id")
	}
	query := c.QueryParam("q")
	metaProvider, _, err := h.getProvider(providerID)
	if err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
	}

	results, err := metaProvider.Search(c.Request().Context(), query, sdk.SearchOptions{Limit: 20})
	if err != nil {
		return handleProviderError(c, providerID, err)
	}

	var mangas []echo.Map
	for _, res := range results {
		manga := echo.Map{
			"id":       res.RemoteID,
			"title":    res.Title,
			"cover":    res.CoverURL,
			"provider": providerID,
		}
		if res.Availability != "" {
			manga["availability"] = res.Availability
		}
		mangas = append(mangas, manga)
	}

	return c.JSON(http.StatusOK, echo.Map{
		"mangas":  mangas,
		"page":    1,
		"hasNext": false,
	})
}

func (h *Handler) getProviderMangaDetails(c echo.Context) error {
	providerID := c.Param("providerId")
	if providerID == "" {
		providerID = c.Param("id")
	}
	remoteID := c.Param("remoteId")

	metaProvider, _, err := h.getProvider(providerID)
	if err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
	}

	meta, err := metaProvider.Details(c.Request().Context(), remoteID)
	if err != nil {
		return handleProviderError(c, providerID, err)
	}

	details := echo.Map{
		"id":            meta.RemoteID,
		"title":         meta.Title,
		"description":   meta.Synopsis,
		"cover":         meta.CoverURL,
		"status":        meta.Status,
		"authors":       []string{meta.Author},
		"artists":       []string{meta.Artist},
		"tags":          meta.Genres,
		"totalChapters": meta.TotalChapters,
	}
	if meta.ReadingMode != "" {
		details["reading_mode"] = meta.ReadingMode
	}
	if meta.Availability != "" {
		details["availability"] = meta.Availability
	}

	return c.JSON(http.StatusOK, details)
}

func (h *Handler) getProviderMangaChapters(c echo.Context) error {
	providerID := c.Param("providerId")
	if providerID == "" {
		providerID = c.Param("id")
	}
	remoteID := c.Param("remoteId")
	if remoteID == "" {
		remoteID = c.Param("mangaId")
	}

	_, contentProvider, err := h.getProvider(providerID)
	if err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
	}

	chapters, err := contentProvider.FetchChapters(c.Request().Context(), remoteID)
	if err != nil {
		return handleProviderError(c, providerID, err)
	}

	formattedChapters := make([]echo.Map, 0, len(chapters))
	for _, ch := range chapters {
		var uploadDateStr string
		if !ch.UploadDate.IsZero() {
			uploadDateStr = ch.UploadDate.UTC().Format(time.RFC3339)
		}
		formattedChapters = append(formattedChapters, echo.Map{
			"id":          ch.ID,
			"title":       ch.Name,
			"number":      ch.Number,
			"uploadDate":  uploadDateStr,
			"url":         ch.URL,
			"sourceOrder": ch.SourceOrder,
		})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"chapters": formattedChapters,
	})
}

func (h *Handler) importProviderManga(c echo.Context) error {
	var body struct {
		ProviderID string `json:"provider_id"`
		RemoteID   string `json:"remote_id"`
		UserStatus string `json:"user_status"`
	}
	if err := c.Bind(&body); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	if body.UserStatus != "" && !library.IsValidUserStatus(body.UserStatus) {
		c.Set("handler_error", "invalid user_status")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid user_status"})
	}

	metaProvider, contentProvider, err := h.getProvider(body.ProviderID)
	if err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	ctx := c.Request().Context()
	meta, err := metaProvider.Details(ctx, body.RemoteID)
	if err != nil {
		return handleProviderError(c, body.ProviderID, fmt.Errorf("fetch details: %w", err))
	}

	localID := body.RemoteID
	var authors []string
	if meta.Author != "" {
		authors = append(authors, meta.Author)
	}
	var artists []string
	if meta.Artist != "" {
		artists = append(artists, meta.Artist)
	}

	mangaMeta := &library.MangaMeta{
		Title:       meta.Title,
		Aliases:     meta.Aliases,
		Description: meta.Synopsis,
		Authors:     authors,
		Artists:     artists,
		Tags:        meta.Genres,
		CoverURL:    meta.CoverURL,
		UserStatus:  body.UserStatus,
		Content: &library.ContentSource{
			ProviderID:      body.ProviderID,
			ProviderMangaID: body.RemoteID,
			ReadingMode:     string(meta.ReadingMode),
			LastSyncedAt:    time.Now(),
		},
		Providers: []library.ProviderRef{
			{
				ProviderID:      body.ProviderID,
				ProviderMangaID: body.RemoteID,
				MangaTitle:      meta.Title,
			},
		},
	}

	if err := h.lib.SaveManga(localID, mangaMeta); err != nil {
		errMsg := fmt.Sprintf("save manga: %v", err)
		c.Set("handler_error", errMsg)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": errMsg})
	}

	// Fetch chapter list from provider and save manifests
	chapters, err := contentProvider.FetchChapters(ctx, body.RemoteID)
	if err == nil && len(chapters) > 0 {
		now := time.Now()
		const maxWorkers = 16
		numWorkers := len(chapters)
		if numWorkers > maxWorkers {
			numWorkers = maxWorkers
		}
		sem := make(chan struct{}, numWorkers)
		var wg sync.WaitGroup

		for _, ch := range chapters {
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
						ProviderID:   body.ProviderID,
						ChapterRef:   ch.ID,
						LastSyncedAt: now,
					},
				}
				_ = h.lib.SaveChapter(localID, ch.ID, chMeta)
			}(ch)
		}
		wg.Wait()
	}

	return c.JSON(http.StatusCreated, echo.Map{
		"id":   localID,
		"meta": mangaMeta,
	})
}

func handleProviderError(c echo.Context, providerID string, err error) error {
	pe := sdk.ClassifyError(providerID, err)
	status := pe.HTTPStatus()
	uri := c.Request().RequestURI
	if uri == "" && c.Request().URL != nil {
		uri = c.Request().URL.String()
	}

	slog.Error("provider request failed",
		slog.String("provider_id", providerID),
		slog.String("uri", uri),
		slog.Int("status", status),
		slog.String("error", err.Error()),
	)

	c.Set("handler_error", pe.Message)

	return c.JSON(status, echo.Map{
		"error":       pe.Message,
		"provider_id": pe.ProviderID,
		"kind":        pe.Kind.String(),
	})
}
