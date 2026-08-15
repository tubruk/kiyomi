package scraper

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	sdk "github.com/tubruk/kiyomi/plugin-sdk"
	"github.com/tubruk/kiyomi/plugin-sdk/utils"
)

var (
	whitespaceRegex = regexp.MustCompile(`\s+`)
	scoreRegex      = regexp.MustCompile(`(?i)(?:score:?\s*)?(\d+(?:\.\d+)?)\s*(?:/\s*(\d+(?:\.\d+)?)|%)?`)
)

// HTTPDoer is an interface for sending HTTP requests, satisfied by *http.Client and sdkhttp.Client.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// HTMLSource provides selector-based web scraping helpers with BaseURL context and HTTP fetching.
type HTMLSource struct {
	BaseURL string
	Client  HTTPDoer
}

// NewHTMLSource creates a new HTMLSource. If client is nil, http.DefaultClient is used.
func NewHTMLSource(baseURL string, client HTTPDoer) *HTMLSource {
	if client == nil {
		client = http.DefaultClient
	}
	return &HTMLSource{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Client:  client,
	}
}

// ResolveURL resolves a relative or absolute URL against the source's BaseURL.
func (h *HTMLSource) ResolveURL(targetURL string) string {
	resolved, err := utils.ResolveURL(h.BaseURL, targetURL)
	if err != nil {
		return targetURL
	}
	return resolved
}

// FetchDocument issues a GET request to pathOrURL and parses the response into an HTMLDocument.
func (h *HTMLSource) FetchDocument(ctx context.Context, pathOrURL string) (*HTMLDocument, error) {
	return h.FetchDocumentWithHeaders(ctx, pathOrURL, nil)
}

// FetchDocumentWithHeaders issues a GET request with custom headers and returns an HTMLDocument.
func (h *HTMLSource) FetchDocumentWithHeaders(ctx context.Context, pathOrURL string, headers map[string]string) (*HTMLDocument, error) {
	fullURL := h.ResolveURL(pathOrURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %s: %w", fullURL, err)
	}

	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := h.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed for %s: %w", fullURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http %d for %s", resp.StatusCode, fullURL)
	}

	return NewHTMLDocument(resp.Body, h.BaseURL)
}

// NewDocument parses an HTML reader into an HTMLDocument bound to the source's BaseURL.
func (h *HTMLSource) NewDocument(r io.Reader) (*HTMLDocument, error) {
	return NewHTMLDocument(r, h.BaseURL)
}

// NewDocumentFromString parses an HTML string into an HTMLDocument bound to the source's BaseURL.
func (h *HTMLSource) NewDocumentFromString(htmlContent string) (*HTMLDocument, error) {
	return NewHTMLDocumentFromString(htmlContent, h.BaseURL)
}

// HTMLDocument wraps a goquery.Document with BaseURL-aware selector helpers.
type HTMLDocument struct {
	Doc     *goquery.Document
	BaseURL string
}

// NewHTMLDocument creates an HTMLDocument from an io.Reader.
func NewHTMLDocument(r io.Reader, baseURL string) (*HTMLDocument, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML document: %w", err)
	}
	return &HTMLDocument{
		Doc:     doc,
		BaseURL: strings.TrimRight(baseURL, "/"),
	}, nil
}

// NewHTMLDocumentFromString creates an HTMLDocument from an HTML string.
func NewHTMLDocumentFromString(htmlContent, baseURL string) (*HTMLDocument, error) {
	return NewHTMLDocument(strings.NewReader(htmlContent), baseURL)
}

// Find selects elements matching the CSS selector and returns a Selection.
func (d *HTMLDocument) Find(selector string) *Selection {
	return &Selection{
		Sel:     d.Doc.Find(selector),
		BaseURL: d.BaseURL,
	}
}

// FindEach iterates over all elements matching the selector.
func (d *HTMLDocument) FindEach(selector string, fn func(i int, s *Selection)) {
	d.Doc.Find(selector).Each(func(i int, s *goquery.Selection) {
		fn(i, &Selection{Sel: s, BaseURL: d.BaseURL})
	})
}

// Text extracts trimmed text from the first element matching selector.
func (d *HTMLDocument) Text(selector string) string {
	return strings.TrimSpace(d.Doc.Find(selector).First().Text())
}

// CleanText extracts unescaped, normalized text with compressed whitespace.
func (d *HTMLDocument) CleanText(selector string) string {
	return CleanString(d.Doc.Find(selector).First().Text())
}

// Attr returns the attribute value of the first matching element.
func (d *HTMLDocument) Attr(selector, attrName string) string {
	val, _ := d.Doc.Find(selector).First().Attr(attrName)
	return strings.TrimSpace(val)
}

// AttrOr returns the attribute value of the first matching element, or defaultValue if absent/empty.
func (d *HTMLDocument) AttrOr(selector, attrName, defaultValue string) string {
	val := d.Attr(selector, attrName)
	if val == "" {
		return defaultValue
	}
	return val
}

// Href extracts the href attribute and resolves it against BaseURL.
func (d *HTMLDocument) Href(selector string) string {
	return d.Find(selector).First().Href()
}

// Src extracts image src (checking lazy load attributes) and resolves it against BaseURL.
func (d *HTMLDocument) Src(selector string) string {
	return d.Find(selector).First().Src()
}

// Images extracts and resolves all image URLs under the specified selector.
func (d *HTMLDocument) Images(selector string) []string {
	return d.Find(selector).Images()
}

// ResolveURL resolves a URL against the document's BaseURL.
func (d *HTMLDocument) ResolveURL(targetURL string) string {
	resolved, err := utils.ResolveURL(d.BaseURL, targetURL)
	if err != nil {
		return targetURL
	}
	return resolved
}

// ParseDate parses date strings using standard layouts or relative date phrases.
func (d *HTMLDocument) ParseDate(dateStr string) (time.Time, error) {
	return utils.ParseDate(dateStr)
}

// ParseChapterNumber extracts chapter float number from strings.
func (d *HTMLDocument) ParseChapterNumber(numStr string) (float32, bool) {
	return utils.ParseChapterNumber(numStr)
}

// ParseScore extracts a normalized rating / score from text.
func (d *HTMLDocument) ParseScore(scoreStr string) (float32, bool) {
	return ParseScore(scoreStr)
}

// ExtractSearchResults iterates over matching items and maps them to sdk.SearchResult.
func (d *HTMLDocument) ExtractSearchResults(itemSelector string, fn func(s *Selection) (*sdk.SearchResult, error)) ([]sdk.SearchResult, error) {
	var results []sdk.SearchResult
	var err error

	d.Doc.Find(itemSelector).EachWithBreak(func(i int, s *goquery.Selection) bool {
		sel := &Selection{Sel: s, BaseURL: d.BaseURL}
		res, itemErr := fn(sel)
		if itemErr != nil {
			err = itemErr
			return false
		}
		if res != nil {
			results = append(results, *res)
		}
		return true
	})

	if err != nil {
		return nil, err
	}
	return results, nil
}

// ExtractChapters iterates over matching items and maps them to sdk.Chapter.
func (d *HTMLDocument) ExtractChapters(itemSelector string, fn func(s *Selection) (*sdk.Chapter, error)) ([]sdk.Chapter, error) {
	var chapters []sdk.Chapter
	var err error

	d.Doc.Find(itemSelector).EachWithBreak(func(i int, s *goquery.Selection) bool {
		sel := &Selection{Sel: s, BaseURL: d.BaseURL}
		ch, itemErr := fn(sel)
		if itemErr != nil {
			err = itemErr
			return false
		}
		if ch != nil {
			chapters = append(chapters, *ch)
		}
		return true
	})

	if err != nil {
		return nil, err
	}
	return chapters, nil
}

// ExtractPages iterates over matching items and maps them to sdk.Page.
func (d *HTMLDocument) ExtractPages(itemSelector string, fn func(s *Selection) (*sdk.Page, error)) ([]sdk.Page, error) {
	var pages []sdk.Page
	var err error

	d.Doc.Find(itemSelector).EachWithBreak(func(i int, s *goquery.Selection) bool {
		sel := &Selection{Sel: s, BaseURL: d.BaseURL}
		page, itemErr := fn(sel)
		if itemErr != nil {
			err = itemErr
			return false
		}
		if page != nil {
			pages = append(pages, *page)
		}
		return true
	})

	if err != nil {
		return nil, err
	}
	return pages, nil
}

// Selection wraps a goquery.Selection with BaseURL-aware extraction helpers.
type Selection struct {
	Sel     *goquery.Selection
	BaseURL string
}

// Find selects elements inside this selection matching the CSS selector.
func (s *Selection) Find(selector string) *Selection {
	return &Selection{
		Sel:     s.Sel.Find(selector),
		BaseURL: s.BaseURL,
	}
}

// FindEach iterates over elements matching the selector inside this selection.
func (s *Selection) FindEach(selector string, fn func(i int, sub *Selection)) {
	s.Sel.Find(selector).Each(func(i int, sel *goquery.Selection) {
		fn(i, &Selection{Sel: sel, BaseURL: s.BaseURL})
	})
}

// First returns the first element in the selection.
func (s *Selection) First() *Selection {
	return &Selection{Sel: s.Sel.First(), BaseURL: s.BaseURL}
}

// Last returns the last element in the selection.
func (s *Selection) Last() *Selection {
	return &Selection{Sel: s.Sel.Last(), BaseURL: s.BaseURL}
}

// Eq returns the element at index i.
func (s *Selection) Eq(i int) *Selection {
	return &Selection{Sel: s.Sel.Eq(i), BaseURL: s.BaseURL}
}

// Parent returns the direct parent element.
func (s *Selection) Parent() *Selection {
	return &Selection{Sel: s.Sel.Parent(), BaseURL: s.BaseURL}
}

// Children returns direct children.
func (s *Selection) Children() *Selection {
	return &Selection{Sel: s.Sel.Children(), BaseURL: s.BaseURL}
}

// Siblings returns sibling elements.
func (s *Selection) Siblings() *Selection {
	return &Selection{Sel: s.Sel.Siblings(), BaseURL: s.BaseURL}
}

// Length returns the number of matched DOM nodes.
func (s *Selection) Length() int {
	if s == nil || s.Sel == nil {
		return 0
	}
	return s.Sel.Length()
}

// Exists reports whether any element was matched.
func (s *Selection) Exists() bool {
	return s.Length() > 0
}

// Text returns the trimmed raw text of the selection.
func (s *Selection) Text() string {
	if s == nil || s.Sel == nil {
		return ""
	}
	return strings.TrimSpace(s.Sel.Text())
}

// CleanText returns unescaped text with normalized whitespace.
func (s *Selection) CleanText() string {
	if s == nil || s.Sel == nil {
		return ""
	}
	return CleanString(s.Sel.Text())
}

// Attr returns the specified attribute value of the first element.
func (s *Selection) Attr(name string) string {
	if s == nil || s.Sel == nil {
		return ""
	}
	val, _ := s.Sel.Attr(name)
	return strings.TrimSpace(val)
}

// AttrOr returns the attribute value, or defaultValue if empty.
func (s *Selection) AttrOr(name, defaultValue string) string {
	val := s.Attr(name)
	if val == "" {
		return defaultValue
	}
	return val
}

// HasAttr checks if the attribute exists.
func (s *Selection) HasAttr(name string) bool {
	if s == nil || s.Sel == nil {
		return false
	}
	_, exists := s.Sel.Attr(name)
	return exists
}

// Href extracts the href attribute and resolves it against BaseURL.
func (s *Selection) Href() string {
	raw := s.Attr("href")
	if raw == "" {
		return ""
	}
	return s.ResolveURL(raw)
}

// Src extracts image URL, inspecting common lazy load attributes (data-src, data-original, etc.) and resolving against BaseURL.
func (s *Selection) Src() string {
	if s == nil || s.Sel == nil {
		return ""
	}

	attrs := []string{
		"data-original",
		"data-src",
		"data-lazy-src",
		"data-src-img",
		"data-cfsrc",
		"data-url",
		"src",
	}

	for _, attr := range attrs {
		if val := s.Attr(attr); val != "" && !isPlaceholder(val) {
			return s.ResolveURL(val)
		}
	}

	// Fallback to srcset
	if srcset := s.Attr("srcset"); srcset != "" {
		first := strings.Split(srcset, ",")[0]
		firstURL := strings.Fields(strings.TrimSpace(first))[0]
		if firstURL != "" && !isPlaceholder(firstURL) {
			return s.ResolveURL(firstURL)
		}
	}

	return ""
}

// Images collects all image URLs inside this selection (from img tags or child elements).
func (s *Selection) Images(subSelector ...string) []string {
	var urls []string
	sel := s
	if len(subSelector) > 0 && subSelector[0] != "" {
		sel = s.Find(subSelector[0])
	} else if !sel.Sel.Is("img") {
		sel = s.Find("img")
	}

	sel.Sel.Each(func(i int, img *goquery.Selection) {
		item := &Selection{Sel: img, BaseURL: s.BaseURL}
		if u := item.Src(); u != "" {
			urls = append(urls, u)
		}
	})

	return urls
}

// ResolveURL resolves targetURL against the selection's BaseURL.
func (s *Selection) ResolveURL(targetURL string) string {
	resolved, err := utils.ResolveURL(s.BaseURL, targetURL)
	if err != nil {
		return targetURL
	}
	return resolved
}

// ParseDate parses date from selection text or passed string.
func (s *Selection) ParseDate(dateStr ...string) (time.Time, error) {
	input := s.CleanText()
	if len(dateStr) > 0 && dateStr[0] != "" {
		input = dateStr[0]
	}
	return utils.ParseDate(input)
}

// ParseChapterNumber parses chapter number from selection text or passed string.
func (s *Selection) ParseChapterNumber(numStr ...string) (float32, bool) {
	input := s.CleanText()
	if len(numStr) > 0 && numStr[0] != "" {
		input = numStr[0]
	}
	return utils.ParseChapterNumber(input)
}

// ParseScore parses score from selection text or passed string.
func (s *Selection) ParseScore(scoreStr ...string) (float32, bool) {
	input := s.CleanText()
	if len(scoreStr) > 0 && scoreStr[0] != "" {
		input = scoreStr[0]
	}
	return ParseScore(input)
}

// GoQuery returns the underlying *goquery.Selection.
func (s *Selection) GoQuery() *goquery.Selection {
	if s == nil {
		return nil
	}
	return s.Sel
}

// CleanString unescapes HTML entities, normalizes line breaks and whitespace.
func CleanString(s string) string {
	unescaped := html.UnescapeString(s)
	normalized := whitespaceRegex.ReplaceAllString(unescaped, " ")
	return strings.TrimSpace(normalized)
}

// ParseScore extracts a normalized rating / score from text strings (e.g. "8.5/10", "4.2/5", "85%").
func ParseScore(input string) (float32, bool) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return 0, false
	}

	matches := scoreRegex.FindStringSubmatch(trimmed)
	if len(matches) >= 2 && matches[1] != "" {
		scoreVal, err := strconv.ParseFloat(matches[1], 32)
		if err != nil {
			return 0, false
		}

		// Check if out of 10 or 5 or %
		if len(matches) >= 3 && matches[2] != "" {
			maxVal, err := strconv.ParseFloat(matches[2], 32)
			if err == nil && maxVal > 0 {
				if maxVal == 5 {
					// Normalize 5-star to 10-scale
					return float32(scoreVal * 2), true
				}
				if maxVal == 100 {
					// Normalize 100-scale to 10-scale
					return float32(scoreVal / 10), true
				}
				return float32(scoreVal), true
			}
		}

		if strings.HasSuffix(trimmed, "%") {
			return float32(scoreVal / 10), true
		}

		return float32(scoreVal), true
	}

	return 0, false
}

func isPlaceholder(url string) bool {
	lower := strings.ToLower(url)
	return strings.HasPrefix(lower, "data:image") ||
		strings.Contains(lower, "blank.gif") ||
		strings.Contains(lower, "placeholder") ||
		strings.Contains(lower, "lazyload.png") ||
		strings.Contains(lower, "transparent.png")
}
