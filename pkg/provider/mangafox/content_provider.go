package mangafox

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/tubruk/kiyomi/pkg/provider/sdk"
)

var _ sdk.Content = (*Provider)(nil)

// HasStableChapterID returns true since MangaFox uses URL paths that don't change.
func (p *Provider) HasStableChapterID() bool { return true }

// RateLimit returns the rate limit hint for MangaFox.
func (p *Provider) RateLimit() sdk.RateLimitHint {
	return sdk.RateLimitHint{RequestsPerSecond: 1}
}

// FetchChapters implements sdk.Content for MangaFox.
func (p *Provider) FetchChapters(ctx context.Context, mangaRef string) ([]sdk.Chapter, error) {
	slug := extractSlug(mangaRef)
	doc, err := p.GetDocument(ctx, p.mangaURL(slug))
	if err != nil {
		return nil, sdk.NewProviderError(sdk.KindTransient, ProviderID, "failed to fetch chapter list page", err)
	}

	var chapters []sdk.Chapter
	doc.Find("#chapterlist ul.detail-main-list li a, ul.detail-main-list li a").Each(func(idx int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists || href == "" {
			return
		}

		name := sdk.ExtractText(s, ".title3")
		if name == "" {
			name = sdk.ExtractText(s, "")
		}

		dateStr := sdk.ExtractText(s, ".title2")
		uploadDate := parseDate(dateStr)

		chapters = append(chapters, sdk.Chapter{
			ID:          cleanChapterID(href, slug),
			Name:        name,
			URL:         p.ResolveURL(href),
			Number:      sdk.ParseChapterNumber(name),
			UploadDate:  uploadDate,
			SourceOrder: idx,
		})
	})

	return chapters, nil
}

var reRelativeDate = regexp.MustCompile(`(?i)^(\d+)\s+(hour|day|week|month)s?\s+ago`)

// parseDate parses date strings commonly found on MangaFox into a time.Time.
func parseDate(dateStr string) time.Time {
	dateStr = strings.TrimSpace(dateStr)
	if dateStr == "" {
		return time.Time{}
	}

	now := time.Now().UTC()
	if strings.EqualFold(dateStr, "today") {
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	}
	if strings.EqualFold(dateStr, "yesterday") {
		return time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, time.UTC)
	}

	if matches := reRelativeDate.FindStringSubmatch(dateStr); len(matches) == 3 {
		n, err := strconv.Atoi(matches[1])
		if err == nil {
			switch strings.ToLower(matches[2]) {
			case "hour":
				return now.Add(-time.Duration(n) * time.Hour)
			case "day":
				return now.AddDate(0, 0, -n)
			case "week":
				return now.AddDate(0, 0, -n*7)
			case "month":
				return now.AddDate(0, -n, 0)
			}
		}
	}

	layouts := []string{
		"Jan 02, 2006",
		"Jan 2, 2006",
		"Jan 02,2006",
		"Jan 2,2006",
		"02 Jan 2006",
		"2 Jan 2006",
		"2006-01-02",
		"2006/01/02",
		"January 2, 2006",
		"January 02, 2006",
		"01/02/2006",
		"1/2/2006",
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, dateStr); err == nil {
			return t.UTC()
		}
	}

	return time.Time{}
}

var (
	reChapterID  = regexp.MustCompile(`chapterid\s*=\s*(\d+)`)
	reImageCount = regexp.MustCompile(`imagecount\s*=\s*(\d+)`)
)

// cleanChapterID converts a MangaFox chapter href or path into a clean, short chapter ID without repeating manga slug or .html extension.
// For example, "/manga/one_piece/v01/c001/1.html" becomes "v01-c001", and "/manga/one_piece/c002/1.html" becomes "c002".
func cleanChapterID(href, slug string) string {
	href = strings.TrimSpace(href)
	if u, err := url.Parse(href); err == nil && (u.Scheme != "" || u.Host != "") {
		href = u.Path
	} else if idx := strings.Index(href, "://"); idx != -1 {
		rest := href[idx+3:]
		if slashIdx := strings.Index(rest, "/"); slashIdx != -1 {
			href = rest[slashIdx:]
		}
	}
	href = strings.Trim(href, "/")
	if strings.HasPrefix(href, "manga/") {
		href = strings.TrimPrefix(href, "manga/")
		href = strings.Trim(href, "/")
	}

	if slug != "" && strings.HasPrefix(href, slug+"/") {
		href = strings.TrimPrefix(href, slug+"/")
		href = strings.Trim(href, "/")
	}

	parts := strings.Split(href, "/")
	var cleanParts []string
	for _, p := range parts {
		if p == "" || p == "1.html" || strings.HasSuffix(p, ".html") {
			continue
		}
		cleanParts = append(cleanParts, p)
	}

	if len(cleanParts) == 0 {
		return href
	}

	return strings.Join(cleanParts, "-")
}

// resolveChapterURL converts mangaRef and chapterRef into a full upstream URL.
func (p *Provider) resolveChapterURL(mangaRef, chapterRef string) string {
	chapterRef = strings.TrimSpace(chapterRef)

	// Fallback for legacy Base64-encoded IDs.
	if _, path, err := sdk.DecodeID(chapterRef); err == nil && path != "" {
		chapterRef = path
	}

	// If it's already a full HTTP/HTTPS URL, return it directly.
	if strings.HasPrefix(chapterRef, "http://") || strings.HasPrefix(chapterRef, "https://") {
		return chapterRef
	}

	mangaSlug := extractSlug(mangaRef)

	// Replace '~' separators back to '/' for legacy IDs like "one_piece~v01~c001~1.html"
	chapterRef = strings.ReplaceAll(chapterRef, "~", "/")
	chapterRef = strings.Trim(chapterRef, "/")

	if strings.HasPrefix(chapterRef, "manga/") {
		chapterRef = strings.TrimPrefix(chapterRef, "manga/")
		chapterRef = strings.Trim(chapterRef, "/")
	}

	if mangaSlug != "" && strings.HasPrefix(chapterRef, mangaSlug+"/") {
		chapterRef = strings.TrimPrefix(chapterRef, mangaSlug+"/")
		chapterRef = strings.Trim(chapterRef, "/")
	}

	if strings.HasSuffix(chapterRef, "/1.html") {
		chapterRef = strings.TrimSuffix(chapterRef, "/1.html")
	} else if strings.HasSuffix(chapterRef, ".html") {
		chapterRef = strings.TrimSuffix(chapterRef, ".html")
	}

	relPath := strings.ReplaceAll(chapterRef, "-", "/")

	if mangaSlug != "" {
		return p.ResolveURL(fmt.Sprintf("/manga/%s/%s/1.html", mangaSlug, relPath))
	}
	return p.ResolveURL(fmt.Sprintf("/manga/%s/1.html", relPath))
}

// FetchPages implements sdk.Content for MangaFox.
func (p *Provider) FetchPages(ctx context.Context, mangaRef, chapterRef string) ([]sdk.Page, error) {
	fullURL := p.resolveChapterURL(mangaRef, chapterRef)

	doc, err := p.GetDocument(ctx, fullURL)
	if err == nil {
		if pages := p.fetchChapterfunPages(ctx, fullURL, doc); len(pages) > 0 {
			return pages, nil
		}
	}

	mobileURL := strings.Replace(fullURL, "https://fanfox.net", "https://m.fanfox.net", 1)
	mobileURL = strings.Replace(mobileURL, "http://fanfox.net", "https://m.fanfox.net", 1)

	req, err := p.NewRequest(ctx, mobileURL)
	if err == nil {
		req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X) AppleWebKit/605.1.15")
		req.Header.Set("Cookie", "readway=2; isAdult=1")
		resp, err := p.Client.Do(req)
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

	return nil, sdk.NewProviderError(sdk.KindPermanent, ProviderID, "no pages found for chapter", nil)
}

// FetchPageStream implements sdk.Content for MangaFox.
func (p *Provider) FetchPageStream(ctx context.Context, page sdk.Page) (io.ReadCloser, error) {
	req, err := p.NewRequest(ctx, page.URL)
	if err != nil {
		return nil, err
	}
	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("fetch page: http status %d", resp.StatusCode)
	}
	return resp.Body, nil
}

func (p *Provider) fetchChapterfunPages(ctx context.Context, fullURL string, doc *goquery.Document) []sdk.Page {
	var chapterID string
	var imageCount int

	doc.Find("script").Each(func(_ int, s *goquery.Selection) {
		text := s.Text()
		if chapterID == "" {
			if matches := reChapterID.FindStringSubmatch(text); len(matches) > 1 {
				chapterID = matches[1]
			}
		}
		if imageCount == 0 {
			if matches := reImageCount.FindStringSubmatch(text); len(matches) > 1 {
				imageCount, _ = strconv.Atoi(matches[1])
			}
		}
	})

	if chapterID == "" || imageCount <= 0 {
		return nil
	}

	dirURL := fullURL
	if idx := strings.LastIndex(fullURL, "/"); idx != -1 {
		dirURL = fullURL[:idx]
	}

	maxPage := (imageCount + 1) / 2
	var pages []sdk.Page
	seen := make(map[string]bool)

	for pageNum := 1; pageNum <= maxPage; pageNum++ {
		cfURL := fmt.Sprintf("%s/chapterfun.ashx?cid=%s&page=%d", dirURL, chapterID, pageNum)

		req, err := p.NewRequest(ctx, cfURL)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)")
		req.Header.Set("Referer", fullURL)

		resp, err := p.Client.Do(req)
		if err != nil {
			continue
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			continue
		}

		unpacked := unpackDeanEdwardsPacker(string(bodyBytes))
		extracted := extractMangaFoxChapterfunURLs(unpacked)
		if len(extracted) == 0 {
			extracted = extractImageURLs(unpacked)
		}

		for _, imgURL := range extracted {
			if !seen[imgURL] {
				seen[imgURL] = true
				pages = append(pages, sdk.Page{
					Index: len(pages),
					URL:   imgURL,
				})
			}
		}
	}

	return pages
}

var (
	rePix  = regexp.MustCompile(`pix\s*=\s*["']([^"']+)["']`)
	rePath = regexp.MustCompile(`["'](/[^"']+\.(?:jpg|jpeg|png|webp|gif)(?:\?[^"']*)?)["']`)
)

func extractMangaFoxChapterfunURLs(unpacked string) []string {
	var urls []string
	pix := ""
	if m := rePix.FindStringSubmatch(unpacked); len(m) > 1 {
		pix = m[1]
		if strings.HasPrefix(pix, "//") {
			pix = "https:" + pix
		}
	}

	matches := rePath.FindAllStringSubmatch(unpacked, -1)
	for _, m := range matches {
		if len(m) > 1 {
			relPath := m[1]
			relPath = strings.ReplaceAll(relPath, `\/`, `/`)
			if pix != "" {
				urls = append(urls, pix+relPath)
			} else if strings.HasPrefix(relPath, "//") {
				urls = append(urls, "https:"+relPath)
			}
		}
	}
	return urls
}

func unpackDeanEdwardsPacker(js string) string {
	re := regexp.MustCompile(`'(.*?)',\s*(\d+),\s*(\d+),\s*'(.*?)'\.split\('\|'\)`)
	matches := re.FindStringSubmatch(js)
	if len(matches) < 5 {
		return js
	}

	p := matches[1]
	radix, _ := strconv.Atoi(matches[2])
	keywords := strings.Split(matches[4], "|")

	tokenRe := regexp.MustCompile(`\b[0-9a-zA-Z]+\b`)
	return tokenRe.ReplaceAllStringFunc(p, func(token string) string {
		val, err := strconv.ParseInt(token, radix, 64)
		if err != nil || val < 0 || val >= int64(len(keywords)) {
			return token
		}
		kw := keywords[val]
		if kw == "" {
			return token
		}
		return kw
	})
}

func extractImageURLs(text string) []string {
	re := regexp.MustCompile(`(?:https?:)?//[^\s"'<>\\]+?\.(?:jpg|jpeg|png|webp|gif)(?:\?[^\s"'<>\\]*)?`)
	matches := re.FindAllString(text, -1)
	var result []string
	seen := make(map[string]bool)

	for _, m := range matches {
		if strings.Contains(m, "logo") || strings.Contains(m, "avatar") || strings.Contains(m, "icon") {
			continue
		}
		if strings.HasPrefix(m, "//") {
			m = "https:" + m
		}
		m = strings.ReplaceAll(m, `\/`, `/`)

		if !seen[m] {
			seen[m] = true
			result = append(result, m)
		}
	}
	return result
}
