package main

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	sdk "github.com/tubruk/kiyomi/plugin-sdk"
)

// Search searches MangaFox catalog by keyword or browsing mode.
func (p *MangaFoxPlugin) Search(ctx context.Context, query string, opts sdk.SearchOptions) ([]sdk.SearchResult, error) {
	query = strings.TrimSpace(query)
	baseURL := p.getBaseURL()

	var searchURL string
	if query == "" {
		if opts.Mode == "latest" {
			searchURL = fmt.Sprintf("%s/releases/", baseURL)
		} else {
			searchURL = fmt.Sprintf("%s/directory/", baseURL)
		}
	} else {
		searchURL = fmt.Sprintf("%s/search?title=%s&page=1", baseURL, url.QueryEscape(query))
	}

	doc, err := p.getDocument(ctx, searchURL)
	if err != nil {
		return nil, fmt.Errorf("mangafox search failed: %w", err)
	}

	var results []sdk.SearchResult
	doc.Find(".manga-list-4-list li, .manga-list-1-list li, .manga-list-1 li").Each(func(_ int, s *goquery.Selection) {
		linkSel := s.Find("p.manga-list-4-item-title a, p.manga-list-1-item-title a, h3.manga-list-1-item-title a, p.title a").First()
		href, _ := linkSel.Attr("href")
		title := strings.TrimSpace(linkSel.Text())
		if href == "" || title == "" {
			return
		}

		imgSel := s.Find("img.manga-list-4-cover, img.manga-list-1-cover, img").First()
		coverURL := extractImageAttr(imgSel)
		slug := extractSlug(href)

		results = append(results, sdk.SearchResult{
			RemoteID:     slug,
			Title:        title,
			CoverURL:     coverURL,
			URL:          p.mangaURL(slug),
			Availability: sdk.AvailabilityAvailable,
		})
	})

	return results, nil
}

// Details retrieves detailed manga metadata from MangaFox.
func (p *MangaFoxPlugin) Details(ctx context.Context, remoteID string) (sdk.MangaMetadata, error) {
	slug := extractSlug(remoteID)
	targetURL := p.mangaURL(slug)
	doc, err := p.getDocument(ctx, targetURL)
	if err != nil {
		return sdk.MangaMetadata{}, fmt.Errorf("mangafox details failed: %w", err)
	}

	title := strings.TrimSpace(doc.Find(".detail-info-right-title-font").First().Text())
	coverURL := extractImageAttr(doc.Find(".detail-info-cover-img").First())

	authorSel := doc.Find(".detail-info-right-say a[title='Author'], .detail-info-right-say a").First()
	author := strings.TrimSpace(authorSel.Text())

	statusText := strings.TrimSpace(doc.Find(".detail-info-right-title-tip").First().Text())
	status := "Unknown"
	if strings.Contains(statusText, "Ongoing") {
		status = "Ongoing"
	} else if strings.Contains(statusText, "Completed") {
		status = "Completed"
	}

	var genres []string
	doc.Find(".detail-info-right-tag a").Each(func(_ int, s *goquery.Selection) {
		if genre := strings.TrimSpace(s.Text()); genre != "" {
			genres = append(genres, genre)
		}
	})

	readingMode := sdk.ReadingModeRTL
	for _, genre := range genres {
		g := strings.ToLower(strings.TrimSpace(genre))
		if g == "webtoon" || g == "webtoons" || g == "long strip" || g == "longstrip" || g == "manhwa" || g == "manhua" {
			readingMode = sdk.ReadingModeLongstrip
			break
		}
	}

	synopsis := strings.TrimSpace(doc.Find(".fullcontent, .detail-info-right-content").First().Text())

	return sdk.MangaMetadata{
		RemoteID:     slug,
		Title:        title,
		CoverURL:     coverURL,
		Synopsis:     synopsis,
		Author:       author,
		Artist:       author,
		Status:       status,
		Genres:       genres,
		ReadingMode:  readingMode,
		URL:          targetURL,
		Availability: sdk.AvailabilityAvailable,
	}, nil
}

// Cover returns cover image reference for the specified manga.
func (p *MangaFoxPlugin) Cover(ctx context.Context, remoteID string, _ sdk.ImageSize) (sdk.ImageRef, error) {
	details, err := p.Details(ctx, remoteID)
	if err != nil {
		return sdk.ImageRef{}, err
	}
	return sdk.ImageRef{URL: details.CoverURL}, nil
}

// Aliases returns alternative titles for the specified manga.
func (p *MangaFoxPlugin) Aliases(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}
