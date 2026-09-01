package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"

	sdk "github.com/tubruk/kiyomi/plugin-sdk"
)

// Search queries Atsumaru for manga matching query or browsing mode.
func (p *AtsumaruPlugin) Search(ctx context.Context, query string, opts sdk.SearchOptions) ([]sdk.SearchResult, error) {
	query = strings.TrimSpace(query)

	// If query is a full URL (e.g. https://atsu.moe/manga/2VgNt or atsu.moe/manga/2VgNt)
	if strings.Contains(query, "atsu.moe/manga/") {
		parts := strings.Split(query, "atsu.moe/manga/")
		if len(parts) > 1 {
			subParts := strings.Split(strings.Trim(parts[1], "/"), "/")
			if len(subParts) > 0 && subParts[0] != "" {
				mangaID := subParts[0]
				meta, err := p.Details(ctx, mangaID)
				if err == nil && meta.Title != "" {
					return []sdk.SearchResult{
						{
							RemoteID:     mangaID,
							Title:        meta.Title,
							CoverURL:     meta.CoverURL,
							URL:          fmt.Sprintf("%s/manga/%s", p.getBaseURL(), mangaID),
							Availability: sdk.AvailabilityAvailable,
						},
					}, nil
				}
			}
		}
	}

	p.mu.RLock()
	adult := p.adultMode
	p.mu.RUnlock()

	// If text query is provided, use Typesense search documents endpoint
	if query != "" {
		return p.searchByKeyword(ctx, query, opts, adult)
	}

	// Otherwise, use popular or recently updated feed
	return p.browseFeed(ctx, opts, adult)
}

func (p *AtsumaruPlugin) searchByKeyword(ctx context.Context, query string, opts sdk.SearchOptions, adult bool) ([]sdk.SearchResult, error) {
	page := 1
	limit := opts.Limit
	if limit <= 0 {
		limit = 20
	}
	if opts.Offset > 0 {
		page = (opts.Offset / limit) + 1
	}

	filterBy := "hidden:!=true && medium:!=[`Novel`] && views:>0"
	if !adult {
		filterBy += " && isAdult:!=true"
	}

	params := url.Values{}
	params.Set("q", query)
	params.Set("query_by", "title,englishTitle,otherNames,authors")
	params.Set("query_by_weights", "4,3,2,1")
	params.Set("num_typos", "4,3,2,1")
	params.Set("filter_by", filterBy)
	params.Set("page", strconv.Itoa(page))
	params.Set("per_page", strconv.Itoa(limit))

	endpoint := fmt.Sprintf("%s/collections/manga/documents/search?%s", p.getBaseURL(), params.Encode())

	req, err := p.newRequest(ctx, http.MethodGet, endpoint)
	if err != nil {
		return nil, fmt.Errorf("atsumaru search: %w", err)
	}

	resp, err := p.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("atsumaru search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("atsumaru search status %d", resp.StatusCode)
	}

	var apiResp atsumaruSearchResultsDto
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("atsumaru search decode: %w", err)
	}

	var results []sdk.SearchResult
	for _, hit := range apiResp.Hits {
		item := hit.Document
		coverURL := p.getCoverURL(item)
		results = append(results, sdk.SearchResult{
			RemoteID:     item.ID,
			Title:        item.Title,
			CoverURL:     coverURL,
			URL:          fmt.Sprintf("%s/manga/%s", p.getBaseURL(), item.ID),
			Availability: sdk.AvailabilityAvailable,
		})
	}

	return results, nil
}

func (p *AtsumaruPlugin) browseFeed(ctx context.Context, opts sdk.SearchOptions, adult bool) ([]sdk.SearchResult, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := opts.Offset

	var endpoint string
	if opts.Mode == "latest" {
		endpoint = fmt.Sprintf("%s/api/home2/recentlyUpdated?offset=%d&limit=%d&types=Manga,Manwha,Manhua,OEL&mediums=Comic", p.getBaseURL(), offset, limit)
	} else {
		endpoint = fmt.Sprintf("%s/api/home2/popular?offset=%d&limit=%d&types=Manga,Manwha,Manhua,OEL&mediums=Comic&timeframe=daily", p.getBaseURL(), offset, limit)
	}

	if adult {
		endpoint += "&adult=1"
	}

	req, err := p.newRequest(ctx, http.MethodGet, endpoint)
	if err != nil {
		return nil, fmt.Errorf("atsumaru browse: %w", err)
	}

	resp, err := p.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("atsumaru browse request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("atsumaru browse status %d", resp.StatusCode)
	}

	var apiResp atsumaruBrowseDto
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("atsumaru browse decode: %w", err)
	}

	var results []sdk.SearchResult
	for _, item := range apiResp.Items {
		coverURL := p.getCoverURL(item)
		results = append(results, sdk.SearchResult{
			RemoteID:     item.ID,
			Title:        item.Title,
			CoverURL:     coverURL,
			URL:          fmt.Sprintf("%s/manga/%s", p.getBaseURL(), item.ID),
			Availability: sdk.AvailabilityAvailable,
		})
	}

	return results, nil
}

func (p *AtsumaruPlugin) getCoverURL(item atsumaruMangaDto) string {
    if item.LargeImage != "" {
        return p.resolveImageURL(item.LargeImage)
    }
    // Flexible Poster handling
    if item.Poster != nil {
        if url, ok := item.Poster.GetStringField("largeImage"); ok && url != "" {
            return p.resolveImageURL(url)
        }
        if url, ok := item.Poster.GetStringField("image"); ok && url != "" {
            return p.resolveImageURL(url)
        }
        if url, ok := item.Poster.GetString(); ok && url != "" {
            return p.resolveImageURL(url)
        }
    }
    if item.MediumImage != "" {
        return p.resolveImageURL(item.MediumImage)
    }
    if item.Image != "" {
        return p.resolveImageURL(item.Image)
    }
    return ""
}

// Details fetches manga metadata by remote ID from Atsumaru.
func (p *AtsumaruPlugin) Details(ctx context.Context, remoteID string) (sdk.MangaMetadata, error) {
	endpoint := fmt.Sprintf("%s/api/manga/page?id=%s", p.getBaseURL(), remoteID)

	req, err := p.newRequest(ctx, http.MethodGet, endpoint)
	if err != nil {
		return sdk.MangaMetadata{}, fmt.Errorf("atsumaru details: %w", err)
	}

	resp, err := p.doRequest(req)
	if err != nil {
		return sdk.MangaMetadata{}, fmt.Errorf("atsumaru details request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return sdk.MangaMetadata{}, fmt.Errorf("atsumaru details status %d", resp.StatusCode)
	}

	var apiResp atsumaruMangaObjectDto
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return sdk.MangaMetadata{}, fmt.Errorf("atsumaru details decode: %w", err)
	}

	item := apiResp.MangaPage

	// Parse authors and artists
	var authors []string
	var artists []string
	seenAuthors := make(map[string]bool)
	seenArtists := make(map[string]bool)

	for _, a := range item.Authors {
		name := strings.TrimSpace(a.Name)
		if name == "" {
			continue
		}
		if strings.EqualFold(a.Type, "Artist") {
			if !seenArtists[name] {
				seenArtists[name] = true
				artists = append(artists, name)
			}
		} else {
			if !seenAuthors[name] {
				seenAuthors[name] = true
				authors = append(authors, name)
			}
		}
	}

	// Parse tags and genres
	var tags []string
	seenTags := make(map[string]bool)
	addTag := func(t string) {
		trimmed := strings.TrimSpace(t)
		if trimmed == "" {
			return
		}
		lower := strings.ToLower(trimmed)
		if !seenTags[lower] {
			seenTags[lower] = true
			tags = append(tags, trimmed)
		}
	}

	if item.Type != "" {
		addTag(item.Type)
	}
	for _, g := range item.Genres {
		addTag(g.Name)
	}
	for _, t := range item.Tags {
		addTag(t.Name)
	}

	// Parse aliases – keep only CJK (Japanese/Korean/Chinese) alternatives.
	// The primary title is stored in Metadata.Title, so we only collect
	// alternate titles that contain Japanese Hiragana/Katakana, Kanji (Han),
	// or Korean Hangul characters. Romaji and pure‑ASCII titles are omitted.

	var aliases []string
	seenAliases := make(map[string]bool)

	// Helper to detect any CJK script in a string.
	containsCJK := func(s string) bool {
		for _, r := range s {
			if unicode.Is(unicode.Hiragana, r) ||
				unicode.Is(unicode.Katakana, r) ||
				unicode.Is(unicode.Han, r) || // Kanji used by Japanese/Chinese/Korean
				unicode.Is(unicode.Hangul, r) {
				return true
			}
		}
		return false
	}

	// Mark the primary title as seen to avoid duplication (lower‑cased).
	mainTitleLower := strings.ToLower(strings.TrimSpace(item.Title))
	if mainTitleLower != "" {
		seenAliases[mainTitleLower] = true
	}

	// Iterate otherNames and keep only those containing CJK characters.
	for _, name := range item.OtherNames {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}
		lower := strings.ToLower(trimmed)
		if seenAliases[lower] {
			continue // dedup
		}
		if containsCJK(trimmed) {
			seenAliases[lower] = true
			aliases = append(aliases, trimmed)
		}
	}

	// Reading mode & country
	readingMode := sdk.ReadingModeUnspecified
	var country string
	typeLower := strings.ToLower(strings.TrimSpace(item.Type))
	switch typeLower {
	case "manga":
		readingMode = sdk.ReadingModeRTL
		country = "JP"
	case "manhwa", "manwha":
		readingMode = sdk.ReadingModeLongstrip
		country = "KR"
	case "manhua":
		readingMode = sdk.ReadingModeLongstrip
		country = "CN"
	case "oel", "comic":
		readingMode = sdk.ReadingModeLTR
		country = "US"
	}

	// Release Year & Start Date
	var releaseYear int
	var startDate string
	if item.Released > 0 {
		releaseTime := time.UnixMilli(item.Released).UTC()
		releaseYear = releaseTime.Year()
		startDate = releaseTime.Format("2006-01-02")
	}

	// Score
	score := item.AvgRating
	if score <= 0 {
		score = item.MBRating
	}

	// Status
	status := "unknown"
	switch strings.ToLower(strings.TrimSpace(item.Status)) {
	case "ongoing":
		status = "ongoing"
	case "completed":
		status = "completed"
	case "hiatus", "on hiatus", "on_hiatus":
		status = "hiatus"
	case "canceled", "cancelled":
		status = "cancelled"
	}

	coverURL := p.getCoverURL(item)

	return sdk.MangaMetadata{
		RemoteID:      item.ID,
		Title:         item.Title,
		Aliases:       aliases,
		Synopsis:      item.Synopsis,
		CoverURL:      coverURL,
		Status:        status,
		Authors:       authors,
		Artists:       artists,
		Tags:          tags,
		TotalChapters: int(item.TotalChapters),
		ReadingMode:   readingMode,
		Score:         score,
		ReleaseYear:   releaseYear,
		StartDate:     startDate,
		Country:       country,
		URL:           fmt.Sprintf("%s/manga/%s", p.getBaseURL(), item.ID),
		Availability:  sdk.AvailabilityAvailable,
	}, nil
}

// Cover returns cover image reference for the specified manga.
func (p *AtsumaruPlugin) Cover(ctx context.Context, remoteID string, size sdk.ImageSize) (sdk.ImageRef, error) {
	details, err := p.Details(ctx, remoteID)
	if err != nil {
		return sdk.ImageRef{}, err
	}
	return sdk.ImageRef{URL: details.CoverURL}, nil
}

// Aliases returns alternative titles for the specified manga.
func (p *AtsumaruPlugin) Aliases(ctx context.Context, remoteID string) ([]string, error) {
	details, err := p.Details(ctx, remoteID)
	if err != nil {
		return nil, err
	}
	return details.Aliases, nil
}
