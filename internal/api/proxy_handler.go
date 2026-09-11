package api

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/tubruk/kiyomi/internal/cache"
	"github.com/tubruk/kiyomi/internal/library"
	"github.com/tubruk/kiyomi/internal/security/ssrfguard"
)

func (h *Handler) streamDataURI(c echo.Context, urlStr string) error {
	// data:[<mediatype>][;base64],<data>
	// Strip "data:" prefix.
	dataStart := strings.Index(urlStr, ",")
	if dataStart < 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid data URI"})
	}

	header := urlStr[5:dataStart]
	data := urlStr[dataStart+1:]

	contentType := "text/plain"
	if idx := strings.Index(header, ";"); idx >= 0 {
		contentType = header[:idx]
		if strings.HasSuffix(header, ";base64") {
			contentType = strings.TrimSuffix(contentType, ";base64")
		}
	} else if header != "" {
		contentType = header
	}
	if contentType == "" {
		contentType = "text/plain"
	}

	var body []byte
	var err error
	if strings.HasSuffix(header, ";base64") {
		body, err = base64.StdEncoding.DecodeString(data)
	} else {
		body = []byte(data)
	}
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "failed to decode data URI"})
	}

	c.Response().Header().Set("Content-Type", contentType)
	c.Response().Header().Set("Content-Length", strconv.Itoa(len(body)))
	c.Response().Header().Set("Cache-Control", "public, max-age=86400")

	c.Response().WriteHeader(http.StatusOK)
	_, err = c.Response().Writer.Write(body)
	return err
}

func (h *Handler) getChapterPages(c echo.Context) error {
	chapterRef := c.Param("chapterId")
	providerID := c.Param("providerId")
	if providerID == "" {
		providerID = c.QueryParam("provider_id")
	}
	if providerID == "" {
		providerID = c.QueryParam("providerId")
	}

	mangaRef := c.Param("mangaId")
	if mangaRef == "" {
		mangaRef = c.QueryParam("mangaId")
	}
	if mangaRef == "" {
		mangaRef = c.QueryParam("manga_id")
	}
	if mangaRef == "" {
		mangaRef = c.Param("remoteId")
	}

	if providerID == "" {
		if mangaRef != "" {
			if chMeta, foundProviderID, err := h.lib.FindChapter(mangaRef, chapterRef); err == nil {
				if chMeta.Content != nil && chMeta.Content.ProviderID != "" {
					providerID = chMeta.Content.ProviderID
				} else {
					providerID = foundProviderID
				}
			}
		}
		if providerID == "" {
			if mangas, err := h.lib.ListManga(); err == nil && len(mangas) > 0 {
				searchCtx, cancel := context.WithCancel(c.Request().Context())
				defer cancel()

				const maxWorkers = 16
				numWorkers := len(mangas)
				if numWorkers > maxWorkers {
					numWorkers = maxWorkers
				}
				sem := make(chan struct{}, numWorkers)
				var (
					wg            sync.WaitGroup
					foundOnce     sync.Once
					foundProvider string
					foundManga    string
				)

			searchLoop:
				for _, m := range mangas {
					select {
					case <-searchCtx.Done():
						break searchLoop
					case sem <- struct{}{}:
					}

					wg.Add(1)
					go func(m library.Manga) {
						defer wg.Done()
						defer func() { <-sem }()

						select {
						case <-searchCtx.Done():
							return
						default:
						}

						if chMeta, _, err := h.lib.FindChapter(m.ID, chapterRef); err == nil && chMeta.Content != nil && chMeta.Content.ProviderID != "" {
							foundOnce.Do(func() {
								foundProvider = chMeta.Content.ProviderID
								foundManga = m.ID
								cancel()
							})
						}
					}(m)
				}
				wg.Wait()

				if foundProvider != "" {
					providerID = foundProvider
					if mangaRef == "" {
						mangaRef = foundManga
					}
				}
			}
		}
	}

	if providerID == "" {
		c.Set("handler_error", "provider ID could not be determined for chapter")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "provider ID could not be determined for chapter"})
	}

	refresh := c.QueryParam("refresh") == "true"
	if !refresh {
		savedPages, err := h.lib.GetChapterPages(mangaRef, providerID, chapterRef)
		if err == nil && len(savedPages) > 0 {
			resPages := make([]echo.Map, 0, len(savedPages))
			for _, p := range savedPages {
				resPages = append(resPages, echo.Map{
					"index":  p.Index,
					"url":    p.URL,
					"source": "library",
				})
			}
			return c.JSON(http.StatusOK, echo.Map{
				"pages": resPages,
			})
		}
	}

	_, contentProvider, err := h.getProvider(providerID)
	if err != nil || contentProvider == nil {
		return handleProviderError(c, providerID, fmt.Errorf("provider not available: %s", providerID))
	}

	remoteMangaID := mangaRef
	remoteChapterRef := chapterRef
	if mangaRef != "" {
		if info, err := h.lib.GetManga(mangaRef); err == nil {
			if info.Bindings.Content != nil && info.Bindings.Content.ProviderID == providerID && info.Bindings.Content.ProviderMangaID != "" {
				remoteMangaID = info.Bindings.Content.ProviderMangaID
			} else {
				for _, p := range info.Bindings.Providers {
					if p.ProviderID == providerID && p.ProviderMangaID != "" {
						remoteMangaID = p.ProviderMangaID
						break
					}
				}
			}
			if chMeta, err := h.lib.GetChapter(mangaRef, providerID, chapterRef); err == nil {
				if chMeta.Content != nil && chMeta.Content.ChapterRef != "" {
					remoteChapterRef = chMeta.Content.ChapterRef
				}
			}
		}
	}

	pages, err := contentProvider.FetchPages(c.Request().Context(), remoteMangaID, remoteChapterRef)
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

	if saveErr := h.lib.SaveChapterPages(mangaRef, providerID, chapterRef, pageItems); saveErr != nil {
		slog.Warn("failed to save chapter pages to library",
			slog.String("error", saveErr.Error()),
			slog.String("manga_ref", mangaRef),
			slog.String("chapter_ref", chapterRef),
		)
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
		"pages": resPages,
	})
}

func (h *Handler) proxyPageImage(c echo.Context) error {
	mangaID := c.Param("mangaId")
	chapterID := c.Param("chapterId")
	pageStr := c.Param("pageIndex")

	pageIndex, err := strconv.Atoi(pageStr)
	if err != nil || pageIndex < 1 {
		c.Set("handler_error", "invalid page number")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid page number"})
	}

	providerID := c.QueryParam("provider_id")
	if providerID == "" {
		providerID = c.QueryParam("providerId")
	}
	var chMeta *library.ChapterMeta
	if providerID == "" {
		// No provider specified — search all provider subdirs
		var foundProviderID string
		chMeta, foundProviderID, err = h.lib.FindChapter(mangaID, chapterID)
		if err != nil {
			c.Set("handler_error", "chapter meta not found")
			return c.JSON(http.StatusNotFound, echo.Map{"error": "chapter meta not found"})
		}
		providerID = foundProviderID
		if chMeta != nil && chMeta.Content != nil && chMeta.Content.ProviderID != "" {
			providerID = chMeta.Content.ProviderID
		}
	} else {
		chMeta, err = h.lib.GetChapter(mangaID, providerID, chapterID)
		if err != nil {
			c.Set("handler_error", "chapter meta not found")
			return c.JSON(http.StatusNotFound, echo.Map{"error": "chapter meta not found"})
		}
	}

	// Check if page file already exists on disk
	chapterDir := h.lib.ProviderChapterDir(mangaID, providerID, chapterID)
	for _, ext := range []string{".jpg", ".jpeg", ".png", ".webp", ".gif", ".avif"} {
		localPath := filepath.Join(chapterDir, fmt.Sprintf("%d%s", pageIndex, ext))
		if info, statErr := os.Stat(localPath); statErr == nil && !info.IsDir() {
			c.Response().Header().Set("Cache-Control", "public, max-age=86400")
			return c.File(localPath)
		}
	}

	imageURL := c.QueryParam("url")
	if imageURL == "" && providerID != "" {
		if savedPages, err := h.lib.GetChapterPages(mangaID, providerID, chapterID); err == nil {
			for _, p := range savedPages {
				if p.Index == pageIndex {
					imageURL = p.URL
					break
				}
			}
		}
	}

	if imageURL == "" {
		c.Set("handler_error", "url parameter is required")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "url parameter is required"})
	}

	var content *library.ContentSource
	if chMeta != nil && chMeta.Content != nil {
		content = chMeta.Content
	} else if providerID != "" {
		content = &library.ContentSource{ProviderID: providerID}
	}

	return h.streamRemoteImage(c, imageURL, content)
}

func (h *Handler) proxyImageDirect(c echo.Context) error {
	imageURL := c.QueryParam("url")
	if imageURL == "" {
		c.Set("handler_error", "url parameter is required")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "url parameter is required"})
	}
	return h.streamRemoteImage(c, imageURL, nil)
}

func (h *Handler) streamRemoteImage(c echo.Context, urlStr string, content *library.ContentSource) error {
	// Handle data: URIs directly — no HTTP fetch needed.
	if len(urlStr) >= 5 && urlStr[:5] == "data:" {
		return h.streamDataURI(c, urlStr)
	}

	allowPrivate := h.cfg != nil && h.cfg.AllowPrivateNetworks
	if _, err := ssrfguard.ValidateURL(urlStr, allowPrivate); err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	queryReferer := c.QueryParam("referer")
	providerID := ""
	if content != nil {
		providerID = content.ProviderID
	}

	if h.imageCache == nil {
		req, err := h.reqBuilder.BuildRequest(c.Request().Context(), urlStr, providerID, queryReferer)
		if err != nil {
			c.Set("handler_error", err.Error())
			return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
		}

		resp, err := h.httpClient.Do(req)
		if err != nil {
			c.Set("handler_error", err.Error())
			return c.JSON(http.StatusBadGateway, echo.Map{"error": err.Error()})
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			c.Set("handler_error", "upstream provider error")
			return c.JSON(resp.StatusCode, echo.Map{"error": "upstream provider error", "status": resp.StatusCode})
		}

		if contentType := resp.Header.Get("Content-Type"); contentType != "" {
			c.Response().Header().Set("Content-Type", contentType)
		}
		if contentLength := resp.Header.Get("Content-Length"); contentLength != "" {
			c.Response().Header().Set("Content-Length", contentLength)
		}
		c.Response().Header().Set("Cache-Control", "public, max-age=86400")

		c.Response().WriteHeader(resp.StatusCode)
		_, err = io.Copy(c.Response().Writer, resp.Body)
		return err
	}

	rc, meta, err := h.imageCache.GetOrFetch(c.Request().Context(), urlStr, func(ctx context.Context) (io.ReadCloser, cache.Meta, error) {
		req, err := h.reqBuilder.BuildRequest(ctx, urlStr, providerID, queryReferer)
		if err != nil {
			return nil, cache.Meta{}, err
		}

		resp, err := h.httpClient.Do(req)
		if err != nil {
			return nil, cache.Meta{}, err
		}

		if resp.StatusCode >= 400 {
			_ = resp.Body.Close()
			return nil, cache.Meta{}, fmt.Errorf("upstream provider error: status %d", resp.StatusCode)
		}

		contentType := resp.Header.Get("Content-Type")
		var contentLength int64
		if clStr := resp.Header.Get("Content-Length"); clStr != "" {
			contentLength, _ = strconv.ParseInt(clStr, 10, 64)
		}

		return resp.Body, cache.Meta{
			ContentType:   contentType,
			ContentLength: contentLength,
		}, nil
	})
	if err != nil {
		c.Set("handler_error", err.Error())
		return c.JSON(http.StatusBadGateway, echo.Map{"error": err.Error()})
	}
	defer rc.Close()

	if meta.ContentType != "" {
		c.Response().Header().Set("Content-Type", meta.ContentType)
	}
	if meta.ContentLength > 0 {
		c.Response().Header().Set("Content-Length", strconv.FormatInt(meta.ContentLength, 10))
	}
	c.Response().Header().Set("Cache-Control", "public, max-age=86400")

	c.Response().WriteHeader(http.StatusOK)
	_, err = io.Copy(c.Response().Writer, rc)
	return err
}
