package api

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/tubruk/kiyomi/internal/cache"
	"github.com/tubruk/kiyomi/internal/library"
	"github.com/tubruk/kiyomi/pkg/provider/sdk"
)

func (h *Handler) getChapterPages(c echo.Context) error {
	chapterRef := c.Param("chapterId")
	if chapterRef == "" {
		chapterRef = c.Param("ch")
	}
	providerID := c.Param("providerId")
	if providerID == "" {
		providerID = c.QueryParam("providerId")
	}
	if providerID == "" {
		providerID = c.QueryParam("provider")
	}

	mangaRef := c.QueryParam("mangaId")
	if mangaRef == "" {
		mangaRef = c.QueryParam("mangaRef")
	}
	if mangaRef == "" {
		mangaRef = c.Param("mangaId")
	}
	if mangaRef == "" {
		mangaRef = c.Param("remoteId")
	}

	if providerID == "" {
		if mangaRef != "" {
			if chMeta, err := h.lib.GetChapter(mangaRef, chapterRef); err == nil && chMeta.Content != nil && chMeta.Content.ProviderID != "" {
				providerID = chMeta.Content.ProviderID
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
					go func(m library.MangaInfo) {
						defer wg.Done()
						defer func() { <-sem }()

						select {
						case <-searchCtx.Done():
							return
						default:
						}

						if chMeta, err := h.lib.GetChapter(m.ID, chapterRef); err == nil && chMeta.Content != nil && chMeta.Content.ProviderID != "" {
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
		savedPages, err := h.lib.GetChapterPages(mangaRef, chapterRef)
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

	pages, err := contentProvider.FetchPages(c.Request().Context(), mangaRef, chapterRef)
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

	if saveErr := h.lib.SaveChapterPages(mangaRef, chapterRef, pageItems); saveErr != nil {
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
	if mangaID == "" {
		mangaID = c.Param("id")
	}
	chapterID := c.Param("chapterId")
	if chapterID == "" {
		chapterID = c.Param("ch")
	}
	pageStr := c.Param("pageIndex")
	if pageStr == "" {
		pageStr = c.Param("n")
	}

	pageIndex, err := strconv.Atoi(pageStr)
	if err != nil || pageIndex < 1 {
		c.Set("handler_error", "invalid page number")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid page number"})
	}

	chMeta, err := h.lib.GetChapter(mangaID, chapterID)
	if err != nil {
		c.Set("handler_error", "chapter meta not found")
		return c.JSON(http.StatusNotFound, echo.Map{"error": "chapter meta not found"})
	}

	imageURL := c.QueryParam("url")
	if imageURL == "" {
		c.Set("handler_error", "url parameter is required")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "url parameter is required"})
	}

	return h.streamRemoteImage(c, imageURL, chMeta.Content)
}

func (h *Handler) proxyImageDirect(c echo.Context) error {
	imageURL := c.QueryParam("url")
	if imageURL == "" {
		c.Set("handler_error", "url parameter is required")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "url parameter is required"})
	}
	return h.streamRemoteImage(c, imageURL, nil)
}

func (h *Handler) buildProxyRequest(ctx context.Context, urlStr string, refererParam string, content *library.ContentSource) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, err
	}

	userAgent := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

	if content != nil && content.ProviderID != "" && h.fpStore != nil {
		prof, err := h.fpStore.Get(content.ProviderID)
		if err == nil {
			if prof.UserAgent != "" {
				userAgent = prof.UserAgent
			}
			if len(prof.Cookies) > 0 {
				if h.httpClient.Jar == nil {
					jar, _ := cookiejar.New(nil)
					h.httpClient.Jar = jar
				}
				for domainURL, rawHeader := range prof.Cookies {
					u, parseErr := url.Parse(domainURL)
					if parseErr != nil || u.Host == "" {
						continue
					}
					parts := strings.Split(rawHeader, ";")
					var jarCookies []*http.Cookie
					for _, part := range parts {
						part = strings.TrimSpace(part)
						if part == "" {
							continue
						}
						kv := strings.SplitN(part, "=", 2)
						if len(kv) == 2 {
							jarCookies = append(jarCookies, &http.Cookie{
								Name:  strings.TrimSpace(kv[0]),
								Value: strings.TrimSpace(kv[1]),
								Path:  "/",
							})
						}
					}
					if len(jarCookies) > 0 {
						h.httpClient.Jar.SetCookies(u, jarCookies)
					}
				}
			}
		}
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8")

	referer := refererParam
	if referer == "" && content != nil && content.ProviderID != "" {
		if cp, ok := h.registry.GetContent(content.ProviderID); ok {
			if hsGetter, ok := cp.(interface{ GetConfig() sdk.ProviderConfig }); ok {
				cfg := hsGetter.GetConfig()
				if cfg.BaseURL != "" {
					referer = cfg.BaseURL
					if !strings.HasSuffix(referer, "/") {
						referer += "/"
					}
				}
			}
		}
	}
	if referer == "" {
		if strings.Contains(urlStr, "fanfox.net") || strings.Contains(urlStr, "mfcdn.net") || strings.Contains(urlStr, "mangafox.me") || strings.Contains(urlStr, "zjcdn") {
			referer = "https://fanfox.net/"
		} else if strings.Contains(urlStr, "mangadex.org") || strings.Contains(urlStr, "mangadex.network") {
			referer = "https://mangadex.org/"
		}
	}
	if referer != "" {
		req.Header.Set("Referer", referer)
	}

	return req, nil
}

func (h *Handler) streamRemoteImage(c echo.Context, urlStr string, content *library.ContentSource) error {
	queryReferer := c.QueryParam("referer")

	if h.imageCache == nil {
		req, err := h.buildProxyRequest(c.Request().Context(), urlStr, queryReferer, content)
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
		req, err := h.buildProxyRequest(ctx, urlStr, queryReferer, content)
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
