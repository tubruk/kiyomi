package scraper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdk "github.com/tubruk/kiyomi/plugin-sdk"
)

const sampleJSON = `
{
	"code": 200,
	"message": "success",
	"data": {
		"series": {
			"id": "manga-123",
			"title": "Chainsaw Man",
			"score": 8.75,
			"is_completed": false,
			"created_at": "2023-05-10T12:00:00Z",
			"updated_timestamp": 1683720000,
			"tags": ["Action", "Supernatural", "Shounen"]
		},
		"chapters": [
			{
				"id": "ch-1",
				"chapter_number": 1.0,
				"name": "Dog & Chainsaw",
				"published_at": 1683720000000
			},
			{
				"id": "ch-2",
				"chapter_number": 2.0,
				"name": "The Place Where Pochita Is",
				"published_at": 1683806400000
			}
		],
		"pages": [
			"https://cdn.example.com/ch1/01.png",
			"https://cdn.example.com/ch1/02.png"
		]
	}
}
`

func TestJSONExtractor_PathTraversalsAndTypes(t *testing.T) {
	ext, err := NewJSONExtractor([]byte(sampleJSON))
	require.NoError(t, err)

	// Top level
	assert.Equal(t, 200, ext.Int("code"))
	assert.Equal(t, "success", ext.String("message"))
	assert.True(t, ext.Exists("data.series"))
	assert.False(t, ext.Exists("non_existent_key"))
	assert.False(t, ext.IsEmpty("data.series.title"))
	assert.True(t, ext.IsEmpty("non_existent"))

	// Nested object with dot notation
	assert.Equal(t, "manga-123", ext.String("data.series.id"))
	assert.Equal(t, "Chainsaw Man", ext.String("data.series.title"))
	assert.InDelta(t, 8.75, ext.Float("data.series.score"), 0.01)
	assert.InDelta(t, float32(8.75), ext.Float32("data.series.score"), 0.01)
	assert.False(t, ext.Bool("data.series.is_completed"))

	// Fallback defaults
	assert.Equal(t, "default-title", ext.StringOr("default-title", "data.series.unknown"))
	assert.Equal(t, 999, ext.IntOr(999, "data.series.unknown"))
	assert.True(t, ext.BoolOr(true, "data.series.unknown"))

	// Tags array
	tags := ext.StringArray("data.series.tags")
	require.Len(t, tags, 3)
	assert.Equal(t, "Action", tags[0])
	assert.Equal(t, "Supernatural", tags[1])
	assert.Equal(t, "Shounen", tags[2])

	// Array indexing with brackets and dots
	assert.Equal(t, "ch-1", ext.String("data.chapters[0].id"))
	assert.Equal(t, "ch-2", ext.String("data.chapters.1.id"))
	assert.Equal(t, "Dog & Chainsaw", ext.String("data.chapters[0].name"))

	// Time parsing
	createdAt, err := ext.Time(time.RFC3339, "data.series.created_at")
	require.NoError(t, err)
	assert.Equal(t, 2023, createdAt.Year())

	// Unix timestamp (seconds)
	updatedAt := ext.UnixTime("data.series.updated_timestamp")
	assert.False(t, updatedAt.IsZero())

	// Unix timestamp (milliseconds)
	ch1Date := ext.UnixTime("data.chapters[0].published_at")
	assert.False(t, ch1Date.IsZero())
}

func TestJSONExtractor_DomainMappers(t *testing.T) {
	ext, err := NewJSONExtractor([]byte(sampleJSON))
	require.NoError(t, err)

	// Chapters extraction
	chapters, err := ext.ExtractChapters("data.chapters", func(item *JSONExtractor) (*sdk.Chapter, error) {
		return &sdk.Chapter{
			ID:          item.String("id"),
			Name:        item.String("name"),
			Number:      item.Float32("chapter_number"),
			UploadDate:  item.UnixTime("published_at"),
			SourceOrder: item.Int("chapter_number"),
		}, nil
	})
	require.NoError(t, err)
	require.Len(t, chapters, 2)
	assert.Equal(t, "ch-1", chapters[0].ID)
	assert.Equal(t, "Dog & Chainsaw", chapters[0].Name)
	assert.Equal(t, float32(1.0), chapters[0].Number)
	assert.Equal(t, "ch-2", chapters[1].ID)

	// Pages extraction
	pages, err := ext.ExtractPages("data.pages", func(item *JSONExtractor) (*sdk.Page, error) {
		return &sdk.Page{
			Index: 0,
			URL:   item.String(),
		}, nil
	})
	require.NoError(t, err)
	require.Len(t, pages, 2)
	assert.Equal(t, "https://cdn.example.com/ch1/01.png", pages[0].URL)
	assert.Equal(t, "https://cdn.example.com/ch1/02.png", pages[1].URL)

	// Search results extraction
	results, err := ext.ExtractSearchResults("data.chapters", func(item *JSONExtractor) (*sdk.SearchResult, error) {
		return &sdk.SearchResult{
			RemoteID: item.String("id"),
			Title:    item.String("name"),
		}, nil
	})
	require.NoError(t, err)
	require.Len(t, results, 2)

	// Map extraction
	seriesMap := ext.Map("data.series")
	require.NotNil(t, seriesMap)
	assert.Equal(t, "Chainsaw Man", seriesMap["title"].String())

	// Unmarshal
	var series struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}
	err = ext.Get("data.series").Unmarshal(&series)
	require.NoError(t, err)
	assert.Equal(t, "manga-123", series.ID)
	assert.Equal(t, "Chainsaw Man", series.Title)
}

func TestJSONSource_HTTPMethods(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/series" && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(sampleJSON))
			return
		}
		if r.URL.Path == "/api/search" && r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"results":[{"id":"s-1","title":"Search 1"}]}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	src := NewJSONSource(ts.URL, http.DefaultClient)

	// Test FetchExtractor
	ext, err := src.FetchExtractor(context.Background(), "/api/series")
	require.NoError(t, err)
	assert.Equal(t, "Chainsaw Man", ext.String("data.series.title"))

	// Test FetchJSON target
	var resp struct {
		Code int `json:"code"`
	}
	err = src.FetchJSON(context.Background(), "/api/series", &resp)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.Code)

	// Test PostExtractor
	postExt, err := src.PostExtractor(context.Background(), "/api/search", map[string]string{"q": "hero"})
	require.NoError(t, err)
	assert.Equal(t, "Search 1", postExt.String("results[0].title"))
}
