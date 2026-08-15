package main

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
	sdk "github.com/tubruk/kiyomi/plugin-sdk"
)

func (p *MangaFoxPlugin) getBaseURL() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.baseURL != "" {
		return p.baseURL
	}
	return DefaultBaseURL
}

// SetBaseURL overrides the base URL used by the plugin (useful for tests).
func (p *MangaFoxPlugin) SetBaseURL(u string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.baseURL = strings.TrimRight(u, "/")
}

func (p *MangaFoxPlugin) newRequest(ctx context.Context, targetURL string) (*http.Request, error) {
	p.mu.RLock()
	ua := p.userAgent
	p.mu.RUnlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	return req, nil
}

func (p *MangaFoxPlugin) getDocument(ctx context.Context, targetURL string) (*goquery.Document, error) {
	req, err := p.newRequest(ctx, targetURL)
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
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http status %d for %s", resp.StatusCode, targetURL)
	}

	return goquery.NewDocumentFromReader(resp.Body)
}

func (p *MangaFoxPlugin) resolveURL(relativePath string) string {
	baseURL := p.getBaseURL()
	if strings.HasPrefix(relativePath, "http://") || strings.HasPrefix(relativePath, "https://") {
		return relativePath
	}
	if strings.HasPrefix(relativePath, "//") {
		return "https:" + relativePath
	}
	return strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(relativePath, "/")
}

// extractSlug strips leading /manga/, manga/, scheme/host, or trailing slashes,
// returning clean slug (e.g. "one_piece", "onepunch_man").
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

func (p *MangaFoxPlugin) mangaURL(remoteID string) string {
	slug := extractSlug(remoteID)
	return fmt.Sprintf("%s/manga/%s/", p.getBaseURL(), slug)
}

func (p *MangaFoxPlugin) resolveChapterURL(mangaRef, chapterRef string) string {
	chapterRef = strings.TrimSpace(chapterRef)

	if strings.HasPrefix(chapterRef, "http://") || strings.HasPrefix(chapterRef, "https://") {
		return chapterRef
	}

	mangaSlug := extractSlug(mangaRef)

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
		return p.resolveURL(fmt.Sprintf("/manga/%s/%s/1.html", mangaSlug, relPath))
	}
	return p.resolveURL(fmt.Sprintf("/manga/%s/1.html", relPath))
}

func (p *MangaFoxPlugin) fetchChapterfunPages(ctx context.Context, fullURL string, doc *goquery.Document) []sdk.Page {
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

	p.mu.RLock()
	client := p.client
	p.mu.RUnlock()

	if client == nil {
		client = http.DefaultClient
	}

	maxPage := (imageCount + 1) / 2
	var pages []sdk.Page
	seen := make(map[string]bool)

	for pageNum := 1; pageNum <= maxPage; pageNum++ {
		cfURL := fmt.Sprintf("%s/chapterfun.ashx?cid=%s&page=%d", dirURL, chapterID, pageNum)

		req, err := p.newRequest(ctx, cfURL)
		if err != nil {
			continue
		}
		req.Header.Set("Referer", fullURL)

		resp, err := client.Do(req)
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

func extractImageAttr(s *goquery.Selection) string {
	if s == nil || s.Length() == 0 {
		return ""
	}
	attrs := []string{"data-original", "data-src", "data-lazy-src", "src"}
	for _, attr := range attrs {
		if val, exists := s.Attr(attr); exists && val != "" && !strings.HasPrefix(val, "data:image") {
			if strings.HasPrefix(val, "//") {
				return "https:" + val
			}
			return val
		}
	}
	return ""
}

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

func parseChapterNumber(title string) float32 {
	if matches := reChapterNum.FindStringSubmatch(title); len(matches) > 1 {
		if num, err := strconv.ParseFloat(matches[1], 32); err == nil {
			return float32(num)
		}
	}
	if matches := reFallbackNum.FindStringSubmatch(title); len(matches) > 1 {
		if num, err := strconv.ParseFloat(matches[1], 32); err == nil {
			return float32(num)
		}
	}
	return 0
}

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

	if matches := reRelative.FindStringSubmatch(dateStr); len(matches) == 3 {
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
