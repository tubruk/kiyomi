package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
	sdk "github.com/tubruk/kiyomi/plugin-sdk"
)

// HasStableChapterID indicates that chapter identifiers parsed from MangaFox are stable.
func (p *MangaFoxPlugin) HasStableChapterID() bool {
	return true
}

// RateLimit returns recommended rate limiting parameters for MangaFox.
func (p *MangaFoxPlugin) RateLimit() sdk.RateLimitHint {
	return sdk.RateLimitHint{
		RequestsPerSecond: 1.0,
		RequestsPerMinute: 60.0,
	}
}

// FetchChapters scrapes the chapter list for the specified manga.
func (p *MangaFoxPlugin) FetchChapters(ctx context.Context, mangaRef string) ([]sdk.Chapter, error) {
	slug := extractSlug(mangaRef)
	doc, err := p.getDocument(ctx, p.mangaURL(slug))
	if err != nil {
		return nil, fmt.Errorf("mangafox chapters failed: %w", err)
	}

	var chapters []sdk.Chapter
	doc.Find("#chapterlist ul.detail-main-list li a, ul.detail-main-list li a").Each(func(idx int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists || href == "" {
			return
		}

		name := strings.TrimSpace(s.Find(".title3").First().Text())
		if name == "" {
			name = strings.TrimSpace(s.Text())
		}

		dateStr := strings.TrimSpace(s.Find(".title2").First().Text())
		uploadDate := parseDate(dateStr)

		chapters = append(chapters, sdk.Chapter{
			ID:          cleanChapterID(href, slug),
			Name:        name,
			URL:         p.resolveURL(href),
			Number:      parseChapterNumber(name),
			UploadDate:  uploadDate,
			SourceOrder: idx,
		})
	})

	return chapters, nil
}

// FetchPages retrieves chapter image page links, trying chapterfun unpacking first, then mobile fallback.
func (p *MangaFoxPlugin) FetchPages(ctx context.Context, mangaRef, chapterRef string) ([]sdk.Page, error) {
	fullURL := p.resolveChapterURL(mangaRef, chapterRef)

	doc, err := p.getDocument(ctx, fullURL)
	if err == nil {
		if pages := p.fetchChapterfunPages(ctx, fullURL, doc); len(pages) > 0 {
			return pages, nil
		}
	}

	// Fallback to mobile page scraping
	mobileURL := strings.Replace(fullURL, "https://fanfox.net", "https://m.fanfox.net", 1)
	mobileURL = strings.Replace(mobileURL, "http://fanfox.net", "https://m.fanfox.net", 1)

	req, err := p.newRequest(ctx, mobileURL)
	if err == nil {
		req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X) AppleWebKit/605.1.15")
		req.Header.Set("Cookie", "readway=2; isAdult=1")

		p.mu.RLock()
		client := p.client
		p.mu.RUnlock()

		if client == nil {
			client = http.DefaultClient
		}

		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == 200 {
			docMobile, err := goquery.NewDocumentFromReader(resp.Body)
			_ = resp.Body.Close()
			if err == nil {
				var pages []sdk.Page
				docMobile.Find("img").Each(func(_ int, s *goquery.Selection) {
					src := s.AttrOr("data-original", s.AttrOr("src", ""))
					if src != "" && (strings.Contains(src, "store/manga") || strings.Contains(src, "zjcdn") || strings.Contains(src, "mfcdn")) {
						if strings.HasPrefix(src, "//") {
							src = "https:" + src
						}
						pages = append(pages, sdk.Page{
							Index: len(pages),
							URL:   src,
						})
					}
				})
				if len(pages) > 0 {
					return pages, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("no pages found for chapter %q in manga %q", chapterRef, mangaRef)
}

// FetchPageStream retrieves an image stream for a page.
func (p *MangaFoxPlugin) FetchPageStream(ctx context.Context, page sdk.Page) (io.ReadCloser, error) {
	req, err := p.newRequest(ctx, page.URL)
	if err != nil {
		return nil, err
	}

	p.mu.RLock()
	client := p.client
	p.mu.RUnlock()

	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("fetch page: http status %d", resp.StatusCode)
	}

	return resp.Body, nil
}
