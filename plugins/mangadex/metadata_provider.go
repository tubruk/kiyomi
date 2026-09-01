package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	sdk "github.com/tubruk/kiyomi/plugin-sdk"
)

// Search queries MangaDex for manga matching query or browsing mode.
func (p *MangaDexPlugin) Search(ctx context.Context, query string, opts sdk.SearchOptions) ([]sdk.SearchResult, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := opts.Offset

	query = strings.TrimSpace(query)

	// If query is a URL (e.g. https://mangadex.org/title/f9c33ab9-c603-4f0e-8470-42171f11c769/...) or UUID,
	// resolve directly via Details.
	if strings.Contains(query, "mangadex.org/title/") {
		parts := strings.Split(query, "mangadex.org/title/")
		if len(parts) > 1 {
			subParts := strings.Split(strings.Trim(parts[1], "/"), "/")
			if len(subParts) > 0 && subParts[0] != "" {
				uuid := subParts[0]
				meta, err := p.Details(ctx, uuid)
				if err == nil && meta.Title != "" {
					return []sdk.SearchResult{
						{
							RemoteID:     uuid,
							Title:        meta.Title,
							CoverURL:     meta.CoverURL,
							URL:          fmt.Sprintf("https://mangadex.org/title/%s", uuid),
							Availability: sdk.AvailabilityAvailable,
						},
					}, nil
				}
			}
		}
	}

	endpoint := fmt.Sprintf("%s/manga?limit=%d&offset=%d&includes[]=cover_art&contentRating[]=safe&contentRating[]=suggestive", p.getBaseURL(), limit, offset)
	if query != "" {
		endpoint += "&title=" + url.QueryEscape(query)
	} else if opts.Mode == "latest" {
		endpoint += "&order[latestUploadedChapter]=desc"
	} else {
		endpoint += "&order[followedCount]=desc"
	}

	req, err := p.newRequest(ctx, http.MethodGet, endpoint)
	if err != nil {
		return nil, fmt.Errorf("mangadex search: %w", err)
	}

	resp, err := p.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("mangadex search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mangadex search status %d", resp.StatusCode)
	}

	var apiResp mangaDexSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("mangadex search decode: %w", err)
	}

	var results []sdk.SearchResult
	for _, item := range apiResp.Data {
		title := item.Attributes.Title["en"]
		if title == "" {
			for _, v := range item.Attributes.Title {
				title = v
				break
			}
		}

		var coverFileName string
		for _, rel := range item.Relationships {
			if rel.Type == "cover_art" {
				coverFileName = rel.Attributes.FileName
				break
			}
		}

		var coverURL string
		if coverFileName != "" {
			coverURL = fmt.Sprintf("https://uploads.mangadex.org/covers/%s/%s", item.ID, coverFileName)
		}

		hasEnglish := false
		for _, lang := range item.Attributes.AvailableTranslatedLanguages {
			if lang == "en" {
				hasEnglish = true
				break
			}
		}

		availability := sdk.AvailabilityAvailable
		if !hasEnglish || item.Attributes.LatestUploadedChapter == nil || *item.Attributes.LatestUploadedChapter == "" {
			availability = sdk.AvailabilityUnavailable
		}

		results = append(results, sdk.SearchResult{
			RemoteID:     item.ID,
			Title:        title,
			CoverURL:     coverURL,
			URL:          fmt.Sprintf("https://mangadex.org/title/%s", item.ID),
			Availability: availability,
		})
	}

	return results, nil
}

func parseAltTitles(mainTitle string, altTitles []map[string]string) []string {
	var aliases []string
	seen := make(map[string]bool)
	if mainTitle != "" {
		seen[strings.ToLower(strings.TrimSpace(mainTitle))] = true
	}
	for _, alt := range altTitles {
		for _, v := range alt {
			trimmed := strings.TrimSpace(v)
			if trimmed == "" {
				continue
			}
			lower := strings.ToLower(trimmed)
			if !seen[lower] {
				seen[lower] = true
				aliases = append(aliases, trimmed)
			}
		}
	}
	return aliases
}

// Details fetches manga metadata by remote ID from MangaDex.
func (p *MangaDexPlugin) Details(ctx context.Context, remoteID string) (sdk.MangaMetadata, error) {
	endpoint := fmt.Sprintf("%s/manga/%s?includes[]=cover_art&includes[]=author&includes[]=artist", p.getBaseURL(), remoteID)

	req, err := p.newRequest(ctx, http.MethodGet, endpoint)
	if err != nil {
		return sdk.MangaMetadata{}, fmt.Errorf("mangadex details: %w", err)
	}

	resp, err := p.doRequest(req)
	if err != nil {
		return sdk.MangaMetadata{}, fmt.Errorf("mangadex details request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return sdk.MangaMetadata{}, fmt.Errorf("mangadex details status %d", resp.StatusCode)
	}

	var apiResp mangaDexDetailsResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return sdk.MangaMetadata{}, fmt.Errorf("mangadex details decode: %w", err)
	}

	title := apiResp.Data.Attributes.Title["en"]
	if title == "" {
		for _, v := range apiResp.Data.Attributes.Title {
			title = v
			break
		}
	}

	desc := apiResp.Data.Attributes.Description["en"]

	var authors []string
	var artists []string
	var coverFileName string
	seenAuthor := make(map[string]bool)
	seenArtist := make(map[string]bool)

	for _, rel := range apiResp.Data.Relationships {
		name := strings.TrimSpace(rel.Attributes.Name)
		switch rel.Type {
		case "author":
			if name != "" && !seenAuthor[name] {
				seenAuthor[name] = true
				authors = append(authors, name)
			}
		case "artist":
			if name != "" && !seenArtist[name] {
				seenArtist[name] = true
				artists = append(artists, name)
			}
		case "cover_art":
			coverFileName = rel.Attributes.FileName
		}
	}

	var tags []string
	seenTags := make(map[string]bool)
	hasLongStrip := false
	for _, tag := range apiResp.Data.Attributes.Tags {
		if tagName, ok := tag.Attributes.Name["en"]; ok && tagName != "" {
			tagName = strings.TrimSpace(tagName)
			lower := strings.ToLower(tagName)
			if !seenTags[lower] {
				seenTags[lower] = true
				tags = append(tags, tagName)
			}
			if strings.EqualFold(tagName, "long strip") || strings.EqualFold(tagName, "longstrip") || strings.EqualFold(tagName, "webtoon") {
				hasLongStrip = true
			}
		}
	}
	if apiResp.Data.Attributes.PublicationDemographic != nil && *apiResp.Data.Attributes.PublicationDemographic != "" {
		demo := strings.TrimSpace(*apiResp.Data.Attributes.PublicationDemographic)
		lower := strings.ToLower(demo)
		if !seenTags[lower] {
			seenTags[lower] = true
			formattedDemo := strings.ToUpper(demo[:1]) + strings.ToLower(demo[1:])
			tags = append(tags, formattedDemo)
		}
	}

	readingMode := sdk.ReadingModeUnspecified
	origLang := strings.ToLower(strings.TrimSpace(apiResp.Data.Attributes.OriginalLanguage))
	if hasLongStrip || origLang == "ko" || origLang == "zh" || strings.HasPrefix(origLang, "zh-") || strings.HasPrefix(origLang, "zh_") {
		readingMode = sdk.ReadingModeLongstrip
	} else if origLang == "ja" {
		readingMode = sdk.ReadingModeRTL
	}

	var country string
	if origLang == "ja" {
		country = "JP"
	} else if origLang == "ko" {
		country = "KR"
	} else if origLang == "zh" || strings.HasPrefix(origLang, "zh-") || strings.HasPrefix(origLang, "zh_") {
		country = "CN"
	} else if origLang == "en" {
		country = "US"
	}

	var releaseYear int
	var startDate string
	if apiResp.Data.Attributes.Year != nil && *apiResp.Data.Attributes.Year > 0 {
		releaseYear = *apiResp.Data.Attributes.Year
		startDate = fmt.Sprintf("%d", releaseYear)
	}

	var coverURL string
	if coverFileName != "" {
		coverURL = fmt.Sprintf("https://uploads.mangadex.org/covers/%s/%s", apiResp.Data.ID, coverFileName)
	}

	hasEnglish := false
	for _, lang := range apiResp.Data.Attributes.AvailableTranslatedLanguages {
		if lang == "en" {
			hasEnglish = true
			break
		}
	}

	availability := sdk.AvailabilityAvailable
	if !hasEnglish || apiResp.Data.Attributes.LatestUploadedChapter == nil || *apiResp.Data.Attributes.LatestUploadedChapter == "" {
		availability = sdk.AvailabilityUnavailable
	}

	return sdk.MangaMetadata{
		RemoteID:     apiResp.Data.ID,
		Title:        title,
		Aliases:      parseAltTitles(title, apiResp.Data.Attributes.AltTitles),
		Synopsis:     desc,
		CoverURL:     coverURL,
		Status:       apiResp.Data.Attributes.Status,
		Authors:      authors,
		Artists:      artists,
		Tags:         tags,
		ReadingMode:  readingMode,
		ReleaseYear:  releaseYear,
		StartDate:    startDate,
		Country:      country,
		URL:          fmt.Sprintf("https://mangadex.org/title/%s", apiResp.Data.ID),
		Availability: availability,
	}, nil
}

// Cover returns cover image reference for the specified manga.
func (p *MangaDexPlugin) Cover(ctx context.Context, remoteID string, size sdk.ImageSize) (sdk.ImageRef, error) {
	details, err := p.Details(ctx, remoteID)
	if err != nil {
		return sdk.ImageRef{}, err
	}
	return sdk.ImageRef{URL: details.CoverURL}, nil
}

// Aliases returns alternative titles for the specified manga.
func (p *MangaDexPlugin) Aliases(ctx context.Context, remoteID string) ([]string, error) {
	details, err := p.Details(ctx, remoteID)
	if err != nil {
		return nil, err
	}
	return details.Aliases, nil
}
