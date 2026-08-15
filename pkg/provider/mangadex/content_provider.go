package mangadex

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/tubruk/kiyomi/pkg/provider/sdk"
)

var _ sdk.Content = (*Provider)(nil)

func (p *Provider) HasStableChapterID() bool {
	return true
}

func (p *Provider) RateLimit() sdk.RateLimitHint {
	return sdk.RateLimitHint{
		RequestsPerSecond: 5.0,
	}
}

// FetchChapters fetches chapter feed for a manga.
func (p *Provider) FetchChapters(ctx context.Context, mangaRef string) ([]sdk.Chapter, error) {
	endpoint := fmt.Sprintf("%s/manga/%s/feed?translatedLanguage[]=en&order[chapter]=asc&limit=100", p.baseURL(), mangaRef)

	req, err := p.newRequest(ctx, http.MethodGet, endpoint)
	if err != nil {
		return nil, fmt.Errorf("mangadex chapters: %w", err)
	}

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mangadex chapters request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mangadex chapters status %d", resp.StatusCode)
	}

	var apiResp struct {
		Data []struct {
			ID         string `json:"id"`
			Attributes struct {
				Volume    string `json:"volume"`
				Chapter   string `json:"chapter"`
				Title     string `json:"title"`
				PublishAt string `json:"publishAt"`
			} `json:"attributes"`
		} `json:"data"`
	}

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

// FetchPages fetches page image URLs for a chapter using MangaDex@Home API.
func (p *Provider) FetchPages(ctx context.Context, mangaRef, chapterRef string) ([]sdk.Page, error) {
	endpoint := fmt.Sprintf("%s/at-home/server/%s", p.baseURL(), chapterRef)

	req, err := p.newRequest(ctx, http.MethodGet, endpoint)
	if err != nil {
		return nil, fmt.Errorf("mangadex pages: %w", err)
	}

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mangadex pages request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mangadex pages status %d", resp.StatusCode)
	}

	var apiResp struct {
		BaseURL string `json:"baseUrl"`
		Chapter struct {
			Hash string   `json:"hash"`
			Data []string `json:"data"`
		} `json:"chapter"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("mangadex pages decode: %w", err)
	}

	var pages []sdk.Page
	for idx, fileName := range apiResp.Chapter.Data {
		pageURL := fmt.Sprintf("%s/data/%s/%s", apiResp.BaseURL, apiResp.Chapter.Hash, fileName)
		pages = append(pages, sdk.Page{
			Index: idx + 1,
			URL:   pageURL,
		})
	}

	return pages, nil
}

// FetchPageStream streams an image from a page URL.
func (p *Provider) FetchPageStream(ctx context.Context, page sdk.Page) (io.ReadCloser, error) {
	req, err := p.newRequest(ctx, http.MethodGet, page.URL)
	if err != nil {
		return nil, err
	}

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("mangadex fetch page stream status %d", resp.StatusCode)
	}

	return resp.Body, nil
}
