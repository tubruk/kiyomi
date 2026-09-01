package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	sdk "github.com/tubruk/kiyomi/plugin-sdk"
)

// HasStableChapterID indicates that Atsumaru chapter IDs are stable identifiers.
func (p *AtsumaruPlugin) HasStableChapterID() bool {
	return true
}

// RateLimit returns the recommended rate limit hint for Atsumaru (2 req/sec to avoid Cloudflare challenges).
func (p *AtsumaruPlugin) RateLimit() sdk.RateLimitHint {
	return sdk.RateLimitHint{
		RequestsPerSecond: 2.0,
		RequestsPerMinute: 120.0,
	}
}

// FetchChapters returns all chapters for the specified manga ID.
func (p *AtsumaruPlugin) FetchChapters(ctx context.Context, mangaRef string) ([]sdk.Chapter, error) {
	// First fetch manga details to resolve scanlator map
	scanlatorMap := make(map[string]string)
	detailsEndpoint := fmt.Sprintf("%s/api/manga/page?id=%s", p.getBaseURL(), mangaRef)
	if dReq, err := p.newRequest(ctx, http.MethodGet, detailsEndpoint); err == nil {
		if dResp, err := p.doRequest(dReq); err == nil {
			if dResp.StatusCode == http.StatusOK {
				var dObj atsumaruMangaObjectDto
				if json.NewDecoder(dResp.Body).Decode(&dObj) == nil {
					for _, sc := range dObj.MangaPage.Scanlators {
						if sc.ID != "" && sc.Name != "" {
							scanlatorMap[sc.ID] = sc.Name
						}
					}
				}
			}
			dResp.Body.Close()
		}
	}

	endpoint := fmt.Sprintf("%s/api/manga/allChapters?mangaId=%s", p.getBaseURL(), mangaRef)

	req, err := p.newRequest(ctx, http.MethodGet, endpoint)
	if err != nil {
		return nil, fmt.Errorf("atsumaru chapters: %w", err)
	}

	resp, err := p.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("atsumaru chapters request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("atsumaru chapters status %d", resp.StatusCode)
	}

	var apiResp atsumaruAllChaptersDto
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("atsumaru chapters decode: %w", err)
	}

	// Sort chapters by number ascending, then createdAt ascending
	rawChapters := apiResp.Chapters
	sort.Slice(rawChapters, func(i, j int) bool {
		if rawChapters[i].Number != rawChapters[j].Number {
			return rawChapters[i].Number < rawChapters[j].Number
		}
		return rawChapters[i].CreatedAt < rawChapters[j].CreatedAt
	})

	var chapters []sdk.Chapter
	for idx, ch := range rawChapters {
		name := strings.TrimSpace(ch.Title)
		if name == "" {
			if ch.Number > 0 {
				name = fmt.Sprintf("Chapter %g", ch.Number)
			} else {
				name = "Chapter"
			}
		}

		var uploadDate time.Time
		if ch.CreatedAt > 0 {
			uploadDate = time.UnixMilli(ch.CreatedAt).UTC()
		}

		chapters = append(chapters, sdk.Chapter{
			ID:          ch.ID,
			Name:        name,
			Number:      ch.Number,
			URL:         fmt.Sprintf("%s/read/%s/%s", p.getBaseURL(), mangaRef, ch.ID),
			UploadDate:  uploadDate,
			SourceOrder: idx + 1,
		})
	}

	return chapters, nil
}

// FetchPages retrieves page image URLs for a given manga and chapter.
func (p *AtsumaruPlugin) FetchPages(ctx context.Context, mangaRef, chapterRef string) ([]sdk.Page, error) {
	endpoint := fmt.Sprintf("%s/api/read/chapter?mangaId=%s&chapterId=%s", p.getBaseURL(), mangaRef, chapterRef)

	req, err := p.newRequest(ctx, http.MethodGet, endpoint)
	if err != nil {
		return nil, fmt.Errorf("atsumaru pages: %w", err)
	}

	resp, err := p.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("atsumaru pages request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("atsumaru pages status %d", resp.StatusCode)
	}

	var apiResp atsumaruPageObjectDto
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("atsumaru pages decode: %w", err)
	}

	var pages []sdk.Page
	for idx, img := range apiResp.ReadChapter.Pages {
		imageURL := p.resolveImageURL(img.Image)
		pages = append(pages, sdk.Page{
			Index: idx + 1,
			URL:   imageURL,
			Headers: map[string]string{
				"Accept":  "image/avif,image/webp,*/*",
				"Referer": p.getBaseURL() + "/",
			},
		})
	}

	return pages, nil
}

// FetchPageStream retrieves a streaming io.ReadCloser for a page image.
func (p *AtsumaruPlugin) FetchPageStream(ctx context.Context, page sdk.Page) (io.ReadCloser, error) {
	req, err := p.newRequest(ctx, http.MethodGet, page.URL)
	if err != nil {
		return nil, err
	}

	for k, v := range page.Headers {
		req.Header.Set(k, v)
	}

	resp, err := p.doRequest(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("atsumaru fetch page stream status %d", resp.StatusCode)
	}

	return resp.Body, nil
}
