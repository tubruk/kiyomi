package sdk

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var (
	volRegex     = regexp.MustCompile(`(?i)\bvol(?:ume)?\.?\s*\d+`)
	chapterRegex = regexp.MustCompile(`(?i)(?:ch(?:apter)?\.?|c)?\s*(\d+(?:\.\d+)?)`)
)

// ExtractText extracts trimmed text content matching the CSS selector within a selection.
func ExtractText(s *goquery.Selection, selector string) string {
	if selector == "" {
		return strings.TrimSpace(s.Text())
	}
	return strings.TrimSpace(s.Find(selector).First().Text())
}

// ExtractAttr extracts an attribute value matching the CSS selector within a selection.
func ExtractAttr(s *goquery.Selection, selector, attr string) string {
	target := s
	if selector != "" {
		target = s.Find(selector).First()
	}
	val, _ := target.Attr(attr)
	return strings.TrimSpace(val)
}

// ExtractImageURL extracts image URL trying common lazy-load attributes (src, data-src, data-original, data-lazy-src).
func ExtractImageURL(s *goquery.Selection, selector string) string {
	target := s
	if selector != "" {
		target = s.Find(selector).First()
	}

	attrs := []string{"src", "data-src", "data-original", "data-lazy-src"}
	for _, attr := range attrs {
		if val, ok := target.Attr(attr); ok && strings.TrimSpace(val) != "" {
			return strings.TrimSpace(val)
		}
	}
	return ""
}

// ParseChapterNumber strips volume prefixes and parses floating-point chapter numbers.
func ParseChapterNumber(title string) float32 {
	cleanName := volRegex.ReplaceAllString(title, "")
	matches := chapterRegex.FindStringSubmatch(cleanName)
	if len(matches) >= 2 && matches[1] != "" {
		if num, err := strconv.ParseFloat(matches[1], 32); err == nil {
			return float32(num)
		}
	}
	return -1.0
}

