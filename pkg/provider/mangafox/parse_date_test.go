package mangafox

import (
	"testing"
	"time"
)

func TestParseDate(t *testing.T) {
	expectedTime := time.Date(2020, time.December, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		input    string
		expected time.Time
	}{
		{
			name:     "empty string",
			input:    "",
			expected: time.Time{},
		},
		{
			name:     "whitespace string",
			input:    "   \t\n  ",
			expected: time.Time{},
		},
		{
			name:     "layout Jan 02, 2006",
			input:    "Dec 01, 2020",
			expected: expectedTime,
		},
		{
			name:     "layout Jan 2, 2006",
			input:    "Dec 1, 2020",
			expected: expectedTime,
		},
		{
			name:     "layout Jan 02,2006",
			input:    "Dec 01,2020",
			expected: expectedTime,
		},
		{
			name:     "layout 02 Jan 2006",
			input:    "01 Dec 2020",
			expected: expectedTime,
		},
		{
			name:     "layout 2006-01-02",
			input:    "2020-12-01",
			expected: expectedTime,
		},
		{
			name:     "layout 2006/01/02",
			input:    "2020/12/01",
			expected: expectedTime,
		},
		{
			name:     "layout January 2, 2006",
			input:    "December 1, 2020",
			expected: expectedTime,
		},
		{
			name:     "layout 01/02/2006",
			input:    "12/01/2020",
			expected: expectedTime,
		},
		{
			name:     "unrecognized layout",
			input:    "not a valid date",
			expected: time.Time{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseDate(tt.input)
			if !got.Equal(tt.expected) {
				t.Errorf("parseDate(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParseDateRelative(t *testing.T) {
	now := time.Now().UTC()

	today := parseDate("Today")
	if today.Year() != now.Year() || today.Month() != now.Month() || today.Day() != now.Day() {
		t.Errorf("expected today's date, got %v", today)
	}

	yesterday := parseDate("Yesterday")
	expectedYesterday := now.AddDate(0, 0, -1)
	if yesterday.Year() != expectedYesterday.Year() || yesterday.Month() != expectedYesterday.Month() || yesterday.Day() != expectedYesterday.Day() {
		t.Errorf("expected yesterday's date, got %v", yesterday)
	}

	hoursAgo := parseDate("2 hours ago")
	if hoursAgo.IsZero() || hoursAgo.After(now) || now.Sub(hoursAgo) > 3*time.Hour {
		t.Errorf("unexpected hours ago: %v", hoursAgo)
	}

	daysAgo := parseDate("3 days ago")
	if daysAgo.IsZero() || daysAgo.After(now) {
		t.Errorf("unexpected days ago: %v", daysAgo)
	}

	weeksAgo := parseDate("2 weeks ago")
	if weeksAgo.IsZero() || weeksAgo.After(now) {
		t.Errorf("unexpected weeks ago: %v", weeksAgo)
	}

	monthsAgo := parseDate("1 month ago")
	if monthsAgo.IsZero() || monthsAgo.After(now) {
		t.Errorf("unexpected months ago: %v", monthsAgo)
	}
}
