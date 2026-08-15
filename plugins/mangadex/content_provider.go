package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	sdk "github.com/tubruk/kiyomi/plugin-sdk"
)

// HasStableChapterID indicates that MangaDex chapter UUIDs do not change over time.
func (p *MangaDexPlugin) HasStableChapterID() bool {
	return true
}

// RateLimit returns the recommended rate limit hint for MangaDex.
func (p *MangaDexPlugin) RateLimit() sdk.RateLimitHint {
	return sdk.RateLimitHint{
		RequestsPerSecond: 5.0,
		RequestsPerMinute: 300.0,
	}
}

// FetchChapters returns all English chapters for the specified manga UUID.
func (p *MangaDexPlugin) FetchChapters(ctx context.Context, mangaRef string) ([]sdk.Chapter, error) {
	endpoint := fmt.Sprintf("%s/manga/%s/feed?translatedLanguage[]=en&order[chapter]=asc&limit=100", p.getBaseURL(), mangaRef)

	req, err := p.newRequest(ctx, http.MethodGet, endpoint)
	if err != nil {
		return nil, fmt.Errorf("mangadex chapters: %w", err)
	}

	resp, err := p.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("mangadex chapters request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mangadex chapters status %d", resp.StatusCode)
	}

	var apiResp mangaFeedResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("mangadex chapters decode: %w", err)
	}

	var chapters []sdk.Chapter
	for idx, ch := range apiResp.Data {
		num, _ := strconv.ParseFloat(ch.Attributes.Chapter, 32)
		name := ch.Attributes.Title
		if name == "" {
			if ch.Attributes.Chapter != "" {
				name = fmt.Sprintf("Chapter %s", ch.Attributes.Chapter)
			} else {
				name = "Oneshot"
			}
		}

		var uploadDate time.Time
		if ch.Attributes.PublishAt != "" {
			if t, err := time.Parse(time.RFC3339, ch.Attributes.PublishAt); err == nil {
				uploadDate = t
			}
		}

		chapters = append(chapters, sdk.Chapter{
			ID:          ch.ID,
			Name:        name,
			Number:      float32(num),
			URL:         fmt.Sprintf("https://mangadex.org/chapter/%s", ch.ID),
			UploadDate:  uploadDate,
			SourceOrder: idx + 1,
		})
	}

	return chapters, nil
}

// FetchPages retrieves image URLs for a chapter using MangaDex@Home API.
func (p *MangaDexPlugin) FetchPages(ctx context.Context, mangaRef, chapterRef string) ([]sdk.Page, error) {
	endpoint := fmt.Sprintf("%s/at-home/server/%s", p.getBaseURL(), chapterRef)

	req, err := p.newRequest(ctx, http.MethodGet, endpoint)
	if err != nil {
		return nil, fmt.Errorf("mangadex pages: %w", err)
	}

	resp, err := p.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("mangadex pages request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mangadex pages status %d", resp.StatusCode)
	}

	var apiResp mangaDexAtHomeResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("mangadex pages decode: %w", err)
	}

	p.mu.RLock()
	useDataSaver := p.dataSaver
	p.mu.RUnlock()

	var files []string
	subPath := "data"
	if useDataSaver && len(apiResp.Chapter.DataSaver) > 0 {
		files = apiResp.Chapter.DataSaver
		subPath = "data-saver"
	} else {
		files = apiResp.Chapter.Data
	}

	var pages []sdk.Page
	for idx, fileName := range files {
		pageURL := fmt.Sprintf("%s/%s/%s/%s", apiResp.BaseURL, subPath, apiResp.Chapter.Hash, fileName)
		pages = append(pages, sdk.Page{
			Index: idx + 1,
			URL:   pageURL,
		})
	}

	return pages, nil
}

// FetchPageStream retrieves a streaming io.ReadCloser for a page image.
func (p *MangaDexPlugin) FetchPageStream(ctx context.Context, page sdk.Page) (io.ReadCloser, error) {
	req, err := p.newRequest(ctx, http.MethodGet, page.URL)
	if err != nil {
		return nil, err
	}

	resp, err := p.doRequest(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("mangadex fetch page stream status %d", resp.StatusCode)
	}

	return resp.Body, nil
}
