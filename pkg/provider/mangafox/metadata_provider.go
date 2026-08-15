package mangafox

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/tubruk/kiyomi/pkg/provider/sdk"
)

var _ sdk.Metadata = (*Provider)(nil)

// extractSlug strips any leading /manga/, manga/, scheme/host, or trailing slashes,
// returning only the clean slug (e.g. "onepunch_man", "one_piece").
func extractSlug(rawID string) string {
	rawID = strings.TrimSpace(rawID)
	if u, err := url.Parse(rawID); err == nil && (u.Scheme != "" || u.Host != "") {
		rawID = u.Path
	} else if idx := strings.Index(rawID, "://"); idx != -1 {
		rest := rawID[idx+3:]
		if slashIdx := strings.Index(rest, "/"); slashIdx != -1 {
			rawID = rest[slashIdx:]
		}
	}
	rawID = strings.Trim(rawID, "/")
	if strings.HasPrefix(rawID, "manga/") {
		rawID = strings.TrimPrefix(rawID, "manga/")
		rawID = strings.Trim(rawID, "/")
	}
	return rawID
}

// mangaURL formats canonical https://fanfox.net/manga/<slug>/.
func (p *Provider) mangaURL(remoteID string) string {
	baseURL := p.Config.BaseURL
	if baseURL == "" {
		baseURL = BaseURL
	}
	slug := extractSlug(remoteID)
	return fmt.Sprintf("%s/manga/%s/", strings.TrimRight(baseURL, "/"), slug)
}

// Search implements sdk.Metadata for MangaFox.
func (p *Provider) Search(ctx context.Context, query string, opts sdk.SearchOptions) ([]sdk.SearchResult, error) {
	query = strings.TrimSpace(query)
	baseURL := p.Config.BaseURL
	if baseURL == "" {
		baseURL = BaseURL
	}

	var searchURL string
	if query == "" {
		if opts.Mode == "latest" {
			searchURL = fmt.Sprintf("%s/releases/", baseURL)
		} else {
			// Popular / Top
			searchURL = fmt.Sprintf("%s/directory/", baseURL)
		}
	} else {
		searchURL = fmt.Sprintf("%s/search?title=%s&page=1", baseURL, url.QueryEscape(query))
	}

	doc, err := p.GetDocument(ctx, searchURL)
	if err != nil {
		return nil, sdk.NewProviderError(sdk.KindTransient, ProviderID, "failed to fetch search page", err)
	}

	var results []sdk.SearchResult
	doc.Find(".manga-list-4-list li, .manga-list-1-list li, .manga-list-1 li").Each(func(_ int, s *goquery.Selection) {
		href := sdk.ExtractAttr(s, "p.manga-list-4-item-title a, p.manga-list-1-item-title a, h3.manga-list-1-item-title a, p.title a", "href")
		title := sdk.ExtractText(s, "p.manga-list-4-item-title a, p.manga-list-1-item-title a, h3.manga-list-1-item-title a, p.title a")
		if href == "" || title == "" {
			return
		}

		coverURL := sdk.ExtractImageURL(s, "img.manga-list-4-cover, img.manga-list-1-cover, img")
		slug := extractSlug(href)

		results = append(results, sdk.SearchResult{
			RemoteID: slug,
			Title:    title,
			CoverURL: coverURL,
			URL:      p.mangaURL(slug),
		})
	})

	return results, nil
}

// Details implements sdk.Metadata for MangaFox.
func (p *Provider) Details(ctx context.Context, remoteID string) (sdk.MangaMetadata, error) {
	slug := extractSlug(remoteID)
	doc, err := p.GetDocument(ctx, p.mangaURL(slug))
	if err != nil {
		return sdk.MangaMetadata{}, sdk.NewProviderError(sdk.KindTransient, ProviderID, "failed to fetch manga details page", err)
	}

	title := sdk.ExtractText(doc.Selection, ".detail-info-right-title-font")
	coverURL := sdk.ExtractImageURL(doc.Selection, ".detail-info-cover-img")
	author := sdk.ExtractText(doc.Selection, ".detail-info-right-say a[title='Author'], .detail-info-right-say a")

	statusText := sdk.ExtractText(doc.Selection, ".detail-info-right-title-tip")
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

	synopsis := sdk.ExtractText(doc.Selection, ".fullcontent, .detail-info-right-content")

	return sdk.MangaMetadata{
		RemoteID:    slug,
		Title:       title,
		CoverURL:    coverURL,
		Synopsis:    synopsis,
		Author:      author,
		Artist:      author,
		Status:      status,
		Genres:      genres,
		ReadingMode: readingMode,
		URL:         p.mangaURL(slug),
	}, nil
}

// Cover implements sdk.Metadata for MangaFox.
func (p *Provider) Cover(ctx context.Context, remoteID string, _ sdk.ImageSize) (sdk.ImageRef, error) {
	details, err := p.Details(ctx, remoteID)
	if err != nil {
		return sdk.ImageRef{}, err
	}
	return sdk.ImageRef{URL: details.CoverURL}, nil
}

// Aliases implements sdk.Metadata for MangaFox.
func (p *Provider) Aliases(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}
