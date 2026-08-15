package scraper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdk "github.com/tubruk/kiyomi/plugin-sdk"
)

const sampleHTML = `
<!DOCTYPE html>
<html>
<head><title>Manga Test Site</title></head>
<body>
	<div class="manga-detail">
		<h1 class="title">  One Piece &amp; More  </h1>
		<span class="score">Score: 9.2 / 10</span>
		<img class="cover" data-src="/images/cover.jpg" src="data:image/gif;base64,R0lGODlh" alt="Cover" />
		<div class="synopsis">
			<p>A story about pirates.</p>
			<p>Searching for the ultimate treasure.</p>
		</div>
	</div>

	<div class="chapter-list">
		<div class="chapter-item">
			<a class="chapter-link" href="/manga/one-piece/ch-1000">Chapter 1000: Straw Hat Luffy</a>
			<span class="date">2 days ago</span>
		</div>
		<div class="chapter-item">
			<a class="chapter-link" href="/manga/one-piece/ch-999.5">Chapter 999.5: Special Extra</a>
			<span class="date">2023-01-15</span>
		</div>
	</div>

	<div class="gallery">
		<img data-original="/pages/01.jpg" />
		<img data-src="/pages/02.jpg" />
		<img src="/pages/03.jpg" />
		<img srcset="/pages/04.jpg 1x, /pages/04-2x.jpg 2x" />
	</div>
</body>
</html>
`

func TestHTMLDocument_BasicSelections(t *testing.T) {
	doc, err := NewHTMLDocumentFromString(sampleHTML, "https://manga.example.com")
	require.NoError(t, err)

	// Clean text and entity unescaping
	assert.Equal(t, "One Piece & More", doc.CleanText("h1.title"))

	// Score parsing
	score, ok := doc.ParseScore(doc.Text(".score"))
	assert.True(t, ok)
	assert.InDelta(t, 9.2, score, 0.01)

	// Lazy image src resolution
	coverSrc := doc.Src("img.cover")
	assert.Equal(t, "https://manga.example.com/images/cover.jpg", coverSrc)

	// Attribute reading
	assert.Equal(t, "Cover", doc.Attr("img.cover", "alt"))
	assert.Equal(t, "default-val", doc.AttrOr("img.cover", "non-existent", "default-val"))

	// Gallery images
	images := doc.Images(".gallery")
	require.Len(t, images, 4)
	assert.Equal(t, "https://manga.example.com/pages/01.jpg", images[0])
	assert.Equal(t, "https://manga.example.com/pages/02.jpg", images[1])
	assert.Equal(t, "https://manga.example.com/pages/03.jpg", images[2])
	assert.Equal(t, "https://manga.example.com/pages/04.jpg", images[3])
}

func TestHTMLDocument_ExtractChapters(t *testing.T) {
	doc, err := NewHTMLDocumentFromString(sampleHTML, "https://manga.example.com")
	require.NoError(t, err)

	chapters, err := doc.ExtractChapters(".chapter-item", func(s *Selection) (*sdk.Chapter, error) {
		link := s.Find("a.chapter-link")
		title := link.CleanText()
		href := link.Href()
		chNum, _ := s.ParseChapterNumber(title)
		date, _ := s.ParseDate(s.Find(".date").CleanText())

		return &sdk.Chapter{
			ID:         href,
			Name:       title,
			URL:        href,
			Number:     chNum,
			UploadDate: date,
		}, nil
	})

	require.NoError(t, err)
	require.Len(t, chapters, 2)

	assert.Equal(t, "Chapter 1000: Straw Hat Luffy", chapters[0].Name)
	assert.Equal(t, float32(1000), chapters[0].Number)
	assert.Equal(t, "https://manga.example.com/manga/one-piece/ch-1000", chapters[0].URL)
	assert.False(t, chapters[0].UploadDate.IsZero())

	assert.Equal(t, "Chapter 999.5: Special Extra", chapters[1].Name)
	assert.Equal(t, float32(999.5), chapters[1].Number)
}

func TestHTMLDocument_ExtractSearchResultsAndPages(t *testing.T) {
	searchHTML := `
	<div class="search-results">
		<div class="manga-card">
			<a class="card-link" href="/manga/solo-leveling">Solo Leveling</a>
			<img class="thumb" data-src="/covers/solo.jpg" />
		</div>
	</div>
	`
	doc, err := NewHTMLDocumentFromString(searchHTML, "https://manga.example.com")
	require.NoError(t, err)

	results, err := doc.ExtractSearchResults(".manga-card", func(s *Selection) (*sdk.SearchResult, error) {
		link := s.Find("a.card-link")
		return &sdk.SearchResult{
			RemoteID:     link.Attr("href"),
			Title:        link.CleanText(),
			URL:          link.Href(),
			CoverURL:     s.Find("img.thumb").Src(),
			Availability: sdk.AvailabilityAvailable,
		}, nil
	})

	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "Solo Leveling", results[0].Title)
	assert.Equal(t, "https://manga.example.com/manga/solo-leveling", results[0].URL)
	assert.Equal(t, "https://manga.example.com/covers/solo.jpg", results[0].CoverURL)

	// Test ExtractPages
	pages, err := doc.ExtractPages(".gallery img", func(s *Selection) (*sdk.Page, error) {
		return &sdk.Page{
			Index: 0,
			URL:   s.Src(),
		}, nil
	})
	require.NoError(t, err)
	assert.Empty(t, pages) // No gallery in searchHTML
}

func TestHTMLSource_FetchDocument(t *testing.T) {
	var receivedHeader string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeader = r.Header.Get("X-Test-Auth")
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(sampleHTML))
	}))
	defer ts.Close()

	source := NewHTMLSource(ts.URL, http.DefaultClient)
	require.NotNil(t, source)

	ctx := context.Background()
	doc, err := source.FetchDocumentWithHeaders(ctx, "/manga/one-piece", map[string]string{
		"X-Test-Auth": "secret123",
	})
	require.NoError(t, err)
	assert.Equal(t, "secret123", receivedHeader)
	assert.Equal(t, "One Piece & More", doc.CleanText("h1.title"))

	// Test NewDocument from reader
	doc2, err := source.NewDocumentFromString("<div><span>Hello</span></div>")
	require.NoError(t, err)
	assert.Equal(t, "Hello", doc2.Text("span"))
}

func TestParseScore_Variants(t *testing.T) {
	tests := []struct {
		input    string
		expected float32
		valid    bool
	}{
		{"9.5/10", 9.5, true},
		{"Score: 8.0 / 10", 8.0, true},
		{"4.5 / 5", 9.0, true}, // 4.5/5 normalized to 9.0/10
		{"88%", 8.8, true},     // 88% normalized to 8.8/10
		{"7.4", 7.4, true},
		{"invalid score", 0, false},
		{"", 0, false},
	}

	for _, tc := range tests {
		score, ok := ParseScore(tc.input)
		assert.Equal(t, tc.valid, ok, "testing: %s", tc.input)
		if tc.valid {
			assert.InDelta(t, tc.expected, score, 0.01, "testing: %s", tc.input)
		}
	}
}

func TestSelection_Navigation(t *testing.T) {
	html := `
	<ul class="parent-list">
		<li class="item" id="item1">Item 1</li>
		<li class="item" id="item2">Item 2</li>
		<li class="item" id="item3">Item 3</li>
	</ul>
	`
	doc, err := NewHTMLDocumentFromString(html, "https://example.com")
	require.NoError(t, err)

	list := doc.Find("ul.parent-list")
	assert.Equal(t, 3, list.Children().Length())
	assert.Equal(t, "Item 1", list.Find("li").First().Text())
	assert.Equal(t, "Item 3", list.Find("li").Last().Text())
	assert.Equal(t, "Item 2", list.Find("li").Eq(1).Text())
	assert.True(t, list.Find("li#item1").HasAttr("id"))
	assert.Equal(t, "item1", list.Find("li#item1").Attr("id"))
	assert.Equal(t, "parent-list", list.Find("li#item1").Parent().Attr("class"))
	assert.Equal(t, 2, list.Find("li#item1").Siblings().Length())
}
