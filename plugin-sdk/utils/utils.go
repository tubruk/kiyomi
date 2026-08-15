package utils

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	relativeTimeRegex = regexp.MustCompile(`(?i)^\s*(\d+)\s+(sec|second|min|minute|hour|hr|day|week|month|year)s?\s+ago\s*$`)
	chapterRegex      = regexp.MustCompile(`(?i)(?:ch(?:apter)?\.?\s*|#\s*|^|v(?:ol(?:ume)?)?\.?\s*\d+\s+ch(?:apter)?\.?\s*)(\d+(?:\.\d+)?)\b`)
)

var standardDateLayouts = []string{
	time.RFC3339,
	time.RFC3339Nano,
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02",
	"2006/01/02",
	"Jan 02, 2006",
	"Jan 2, 2006",
	"January 2, 2006",
	"02 Jan 2006",
	"02-Jan-2006",
	"01/02/2006",
	"02/01/2006",
	"Jan 02,2006",
}

// ParseRelativeDate attempts to parse relative time phrases such as "2 hours ago", "yesterday", or "just now".
func ParseRelativeDate(input string, now time.Time) (time.Time, bool) {
	trimmed := strings.ToLower(strings.TrimSpace(input))
	if trimmed == "" {
		return time.Time{}, false
	}

	if trimmed == "today" || trimmed == "just now" || trimmed == "moments ago" {
		return now, true
	}
	if trimmed == "yesterday" {
		return now.AddDate(0, 0, -1), true
	}

	matches := relativeTimeRegex.FindStringSubmatch(trimmed)
	if len(matches) == 3 {
		amount, err := strconv.Atoi(matches[1])
		if err != nil {
			return time.Time{}, false
		}
		unit := matches[2]
		switch {
		case strings.HasPrefix(unit, "sec"):
			return now.Add(-time.Duration(amount) * time.Second), true
		case strings.HasPrefix(unit, "min"):
			return now.Add(-time.Duration(amount) * time.Minute), true
		case strings.HasPrefix(unit, "hour") || unit == "hr":
			return now.Add(-time.Duration(amount) * time.Hour), true
		case strings.HasPrefix(unit, "day"):
			return now.AddDate(0, 0, -amount), true
		case strings.HasPrefix(unit, "week"):
			return now.AddDate(0, 0, -amount*7), true
		case strings.HasPrefix(unit, "month"):
			return now.AddDate(0, -amount, 0), true
		case strings.HasPrefix(unit, "year"):
			return now.AddDate(-amount, 0, 0), true
		}
	}

	return time.Time{}, false
}

// ParseDate attempts to parse a date string using standard layouts or relative date parsing.
func ParseDate(input string) (time.Time, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return time.Time{}, fmt.Errorf("empty date string")
	}

	if t, ok := ParseRelativeDate(trimmed, time.Now()); ok {
		return t, nil
	}

	for _, layout := range standardDateLayouts {
		if t, err := time.Parse(layout, trimmed); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unrecognized date format: %q", input)
}

// ResolveURL resolves a target URL against a base URL, correctly handling relative paths,
// leading slashes, and protocol-relative (//example.com/...) URLs.
func ResolveURL(baseURL, targetURL string) (string, error) {
	target := strings.TrimSpace(targetURL)
	if target == "" {
		return "", fmt.Errorf("empty target URL")
	}

	// Protocol-relative URL
	if strings.HasPrefix(target, "//") {
		base, err := url.Parse(baseURL)
		if err != nil || base.Scheme == "" {
			return "https:" + target, nil
		}
		return base.Scheme + ":" + target, nil
	}

	targetParsed, err := url.Parse(target)
	if err != nil {
		return "", fmt.Errorf("failed to parse target URL: %w", err)
	}

	if targetParsed.IsAbs() {
		return targetParsed.String(), nil
	}

	baseParsed, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse base URL: %w", err)
	}

	resolved := baseParsed.ResolveReference(targetParsed)
	return resolved.String(), nil
}

// ParseChapterNumber extracts floating-point chapter number from chapter titles/labels.
func ParseChapterNumber(input string) (float32, bool) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return 0, false
	}

	matches := chapterRegex.FindStringSubmatch(trimmed)
	if len(matches) >= 2 {
		num, err := strconv.ParseFloat(matches[1], 32)
		if err == nil {
			return float32(num), true
		}
	}

	num, err := strconv.ParseFloat(trimmed, 32)
	if err == nil {
		return float32(num), true
	}

	return 0, false
}
