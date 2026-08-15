package mangadex

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/tubruk/kiyomi/pkg/provider/sdk"
)

var _ sdk.Metadata = (*Provider)(nil)

// Search searches for manga on MangaDex.
func (p *Provider) Search(ctx context.Context, query string, opts sdk.SearchOptions) ([]sdk.SearchResult, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := opts.Offset

	endpoint := fmt.Sprintf("%s/manga?limit=%d&offset=%d&includes[]=cover_art&contentRating[]=safe&contentRating[]=suggestive", p.baseURL(), limit, offset)
	if query != "" {
		endpoint += "&title=" + url.QueryEscape(query)
	} else if opts.Mode == "latest" {
		endpoint += "&order[latestUploadedChapter]=desc"
	} else {
		// Popular / Top manga by follower count
		endpoint += "&order[followedCount]=desc"
	}

	req, err := p.newRequest(ctx, http.MethodGet, endpoint)
	if err != nil {
		return nil, fmt.Errorf("mangadex search: %w", err)
	}

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mangadex search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mangadex search status %d", resp.StatusCode)
	}

	var apiResp struct {
		Data []struct {
			ID         string `json:"id"`
			Attributes struct {
				Title                        map[string]string `json:"title"`
				AvailableTranslatedLanguages []string          `json:"availableTranslatedLanguages"`
				LatestUploadedChapter        *string           `json:"latestUploadedChapter"`
			} `json:"attributes"`
			Relationships []struct {
				Type       string `json:"type"`
				Attributes struct {
					FileName string `json:"fileName"`
				} `json:"attributes"`
			} `json:"relationships"`
		} `json:"data"`
	}

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

// Details fetches full metadata for a manga.
func (p *Provider) Details(ctx context.Context, remoteID string) (sdk.MangaMetadata, error) {
	endpoint := fmt.Sprintf("%s/manga/%s?includes[]=cover_art&includes[]=author&includes[]=artist", p.baseURL(), remoteID)

	req, err := p.newRequest(ctx, http.MethodGet, endpoint)
	if err != nil {
		return sdk.MangaMetadata{}, fmt.Errorf("mangadex details: %w", err)
	}

	resp, err := p.Client.Do(req)
	if err != nil {
		return sdk.MangaMetadata{}, fmt.Errorf("mangadex details request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return sdk.MangaMetadata{}, fmt.Errorf("mangadex details status %d", resp.StatusCode)
	}

	var apiResp struct {
		Data struct {
			ID         string `json:"id"`
			Attributes struct {
				Title                        map[string]string `json:"title"`
				Description                  map[string]string `json:"description"`
				Status                       string            `json:"status"`
				OriginalLanguage             string            `json:"originalLanguage"`
				AvailableTranslatedLanguages []string          `json:"availableTranslatedLanguages"`
				LatestUploadedChapter        *string           `json:"latestUploadedChapter"`
				Tags                         []struct {
					Attributes struct {
						Name map[string]string `json:"name"`
					} `json:"attributes"`
				} `json:"tags"`
			} `json:"attributes"`
			Relationships []struct {
				Type       string `json:"type"`
				Attributes struct {
					FileName string `json:"fileName"`
					Name     string `json:"name"`
				} `json:"attributes"`
			} `json:"relationships"`
		} `json:"data"`
	}

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

	var author, artist, coverFileName string
	for _, rel := range apiResp.Data.Relationships {
		switch rel.Type {
		case "author":
			author = rel.Attributes.Name
		case "artist":
			artist = rel.Attributes.Name
		case "cover_art":
			coverFileName = rel.Attributes.FileName
		}
	}

	var genres []string
	hasLongStrip := false
	for _, tag := range apiResp.Data.Attributes.Tags {
		if tagName, ok := tag.Attributes.Name["en"]; ok && tagName != "" {
			genres = append(genres, tagName)
			if strings.EqualFold(tagName, "long strip") || strings.EqualFold(tagName, "longstrip") || strings.EqualFold(tagName, "webtoon") {
				hasLongStrip = true
			}
		}
	}

	readingMode := sdk.ReadingModeUnspecified
	origLang := strings.ToLower(strings.TrimSpace(apiResp.Data.Attributes.OriginalLanguage))
	if hasLongStrip || origLang == "ko" || origLang == "zh" || strings.HasPrefix(origLang, "zh-") || strings.HasPrefix(origLang, "zh_") {
		readingMode = sdk.ReadingModeLongstrip
	} else if origLang == "ja" {
		readingMode = sdk.ReadingModeRTL
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
		Synopsis:     desc,
		CoverURL:     coverURL,
		Status:       apiResp.Data.Attributes.Status,
		Author:       author,
		Artist:       artist,
		Genres:       genres,
		ReadingMode:  readingMode,
		URL:          fmt.Sprintf("https://mangadex.org/title/%s", apiResp.Data.ID),
		Availability: availability,
	}, nil
}

// Cover returns cover image URL.
func (p *Provider) Cover(ctx context.Context, remoteID string, size sdk.ImageSize) (sdk.ImageRef, error) {
	details, err := p.Details(ctx, remoteID)
	if err != nil {
		return sdk.ImageRef{}, err
	}
	return sdk.ImageRef{URL: details.CoverURL}, nil
}

// Aliases returns title aliases.
func (p *Provider) Aliases(ctx context.Context, remoteID string) ([]string, error) {
	return nil, nil
}
