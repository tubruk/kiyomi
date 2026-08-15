package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRelativeDate(t *testing.T) {
	now := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)

	t1, ok := ParseRelativeDate("2 hours ago", now)
	assert.True(t, ok)
	assert.Equal(t, now.Add(-2*time.Hour), t1)

	t2, ok := ParseRelativeDate("3 days ago", now)
	assert.True(t, ok)
	assert.Equal(t, now.AddDate(0, 0, -3), t2)

	t3, ok := ParseRelativeDate("yesterday", now)
	assert.True(t, ok)
	assert.Equal(t, now.AddDate(0, 0, -1), t3)

	t4, ok := ParseRelativeDate("just now", now)
	assert.True(t, ok)
	assert.Equal(t, now, t4)

	_, ok = ParseRelativeDate("invalid string", now)
	assert.False(t, ok)
}

func TestParseDate(t *testing.T) {
	t1, err := ParseDate("2026-08-16")
	require.NoError(t, err)
	assert.Equal(t, 2026, t1.Year())
	assert.Equal(t, time.Month(8), t1.Month())
	assert.Equal(t, 16, t1.Day())

	t2, err := ParseDate("Aug 16, 2026")
	require.NoError(t, err)
	assert.Equal(t, 2026, t2.Year())

	_, err = ParseDate("not a date")
	assert.Error(t, err)
}

func TestResolveURL(t *testing.T) {
	base := "https://example.com/manga/123"

	// Relative path
	u1, err := ResolveURL(base, "chapter-1")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/manga/chapter-1", u1)

	// Absolute path
	u2, err := ResolveURL(base, "/images/cover.jpg")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/images/cover.jpg", u2)

	// Protocol relative
	u3, err := ResolveURL(base, "//cdn.example.com/img.png")
	require.NoError(t, err)
	assert.Equal(t, "https://cdn.example.com/img.png", u3)

	// Absolute URL
	u4, err := ResolveURL(base, "https://other.com/pic.jpg")
	require.NoError(t, err)
	assert.Equal(t, "https://other.com/pic.jpg", u4)
}

func TestParseChapterNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected float32
		ok       bool
	}{
		{"Chapter 12", 12.0, true},
		{"Ch. 12.5", 12.5, true},
		{"Vol. 2 Ch. 45 - The Battle", 45.0, true},
		{"#99", 99.0, true},
		{"3.14", 3.14, true},
		{"Prologue", 0, false},
	}

	for _, tt := range tests {
		num, ok := ParseChapterNumber(tt.input)
		assert.Equal(t, tt.ok, ok, "Input: %s", tt.input)
		if ok {
			assert.InDelta(t, tt.expected, num, 0.001, "Input: %s", tt.input)
		}
	}
}
