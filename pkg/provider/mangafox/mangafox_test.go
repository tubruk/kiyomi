package mangafox_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tubruk/kiyomi/pkg/fingerprint"
	"github.com/tubruk/kiyomi/pkg/provider/mangafox"
	"github.com/tubruk/kiyomi/pkg/provider/sdk"
)

func TestMangaFoxProviderInterface(t *testing.T) {
	p, err := mangafox.NewProvider(nil)
	if err != nil {
		t.Fatalf("failed to create mangafox provider: %v", err)
	}

	if p.ID() != "mangafox" {
		t.Errorf("expected ID mangafox, got %s", p.ID())
	}
	if p.Name() != "MangaFox" {
		t.Errorf("expected Name MangaFox, got %s", p.Name())
	}
	if p.ConcurrencyLimit() != 1 {
		t.Errorf("expected concurrency limit 1, got %d", p.ConcurrencyLimit())
	}

	var _ sdk.Metadata = p
	var _ sdk.Content = p
}

func TestMangaFoxSearchParsing(t *testing.T) {
	mockHTML := `
	<ul class="manga-list-4-list">
		<li>
			<p class="manga-list-4-item-title"><a href="/manga/one_piece/">One Piece</a></p>
			<img class="manga-list-4-cover" src="https://fmcdn.mfcdn.net/store/manga/106/cover.jpg" />
		</li>
	</ul>
	`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockHTML))
	}))
	defer ts.Close()

	client := ts.Client()
	p, err := mangafox.NewProvider(client)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	p.HttpSource.Config.BaseURL = ts.URL

	results, err := p.Search(context.Background(), "One Piece", sdk.SearchOptions{})
	if err != nil {
		t.Fatalf("search error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Title != "One Piece" {
		t.Errorf("expected title One Piece, got %s", results[0].Title)
	}
	if results[0].RemoteID != "one_piece" {
		t.Errorf("expected remote ID one_piece, got %s", results[0].RemoteID)
	}
}

func TestMangaFoxDetailsNormalization(t *testing.T) {
	mockHTML := `
	<div class="detail-info-right-title-font">One Piece</div>
	<img class="detail-info-cover-img" src="https://fmcdn.mfcdn.net/store/manga/106/cover.jpg" />
	<p class="detail-info-right-say"><a title="Author">Eiichiro Oda</a></p>
	<span class="detail-info-right-title-tip">Ongoing</span>
	`

	var requestedURL string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedURL = r.URL.String()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockHTML))
	}))
	defer ts.Close()

	p, err := mangafox.NewProvider(ts.Client())
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	p.HttpSource.Config.BaseURL = ts.URL

	tests := []string{
		"one_piece",
		"/manga/one_piece/",
		"manga/one_piece",
		"https://fanfox.net/manga/one_piece/",
	}

	for _, inputID := range tests {
		meta, err := p.Details(context.Background(), inputID)
		if err != nil {
			t.Errorf("Details(%q) error: %v", inputID, err)
			continue
		}
		if meta.RemoteID != "one_piece" {
			t.Errorf("Details(%q) RemoteID = %q, want %q", inputID, meta.RemoteID, "one_piece")
		}
		if requestedURL != "/manga/one_piece/" {
			t.Errorf("Details(%q) requested path = %q, want %q", inputID, requestedURL, "/manga/one_piece/")
		}
	}
}

func TestMangaFoxFingerprintStore(t *testing.T) {
	fpStore := fingerprint.NewMemoryStore()
	_ = fpStore.Set("mangafox", fingerprint.Profile{
		UserAgent: "MangaFoxUA/1.0",
	})

	p, err := mangafox.NewProvider(nil, fpStore)
	if err != nil {
		t.Fatalf("failed to create mangafox provider: %v", err)
	}
	if p.Fingerprint != fpStore {
		t.Errorf("expected fingerprint store to be set on provider")
	}
}

func TestMangaFoxFetchChaptersCleanIDs(t *testing.T) {
	mockHTML := `
	<ul class="detail-main-list">
		<li>
			<a href="/manga/one_piece/v01/c001/1.html">
				<span class="title3">Vol.01 Ch.001 Romance Dawn</span>
				<span class="title2">Dec 1, 2020</span>
			</a>
		</li>
		<li>
			<a href="/manga/one_piece/c002/1.html">
				<span class="title3">Ch.002 They Call Him Straw Hat Luffy</span>
				<span class="title2">Dec 2, 2020</span>
			</a>
		</li>
	</ul>
	`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockHTML))
	}))
	defer ts.Close()

	p, err := mangafox.NewProvider(ts.Client())
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	p.HttpSource.Config.BaseURL = ts.URL

	chapters, err := p.FetchChapters(context.Background(), "one_piece")
	if err != nil {
		t.Fatalf("FetchChapters error: %v", err)
	}

	if len(chapters) != 2 {
		t.Fatalf("expected 2 chapters, got %d", len(chapters))
	}

	expectedIDs := []string{
		"v01-c001",
		"c002",
	}

	for i, expectedID := range expectedIDs {
		if chapters[i].ID != expectedID {
			t.Errorf("chapter[%d].ID = %q, want %q", i, chapters[i].ID, expectedID)
		}
	}

	if chapters[0].UploadDate.IsZero() {
		t.Errorf("expected chapter[0].UploadDate to not be zero")
	}
}

func TestMangaFoxFetchPagesCleanIDs(t *testing.T) {
	var requestedPaths []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPaths = append(requestedPaths, r.URL.Path)
		w.WriteHeader(http.StatusOK)
		mockMobileHTML := `<html><body><img data-original="https://fmcdn.mfcdn.net/store/manga/106/001.jpg" /></body></html>`
		_, _ = w.Write([]byte(mockMobileHTML))
	}))
	defer ts.Close()

	p, err := mangafox.NewProvider(ts.Client())
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	p.HttpSource.Config.BaseURL = ts.URL

	testRefs := []struct {
		mangaRef     string
		chapterRef   string
		expectedPath string
	}{
		{
			mangaRef:     "one_piece",
			chapterRef:   "v01-c001",
			expectedPath: "/manga/one_piece/v01/c001/1.html",
		},
		{
			mangaRef:     "one_piece",
			chapterRef:   "c002",
			expectedPath: "/manga/one_piece/c002/1.html",
		},
		{
			mangaRef:     "one_piece",
			chapterRef:   "one_piece~v01~c001~1.html",
			expectedPath: "/manga/one_piece/v01/c001/1.html",
		},
		{
			mangaRef:     "one_piece",
			chapterRef:   "/manga/one_piece/v01/c001/1.html",
			expectedPath: "/manga/one_piece/v01/c001/1.html",
		},
		{
			mangaRef:     "one_piece",
			chapterRef:   sdk.EncodeID("mangafox", "/manga/one_piece/v01/c001/1.html"),
			expectedPath: "/manga/one_piece/v01/c001/1.html",
		},
	}

	for _, tt := range testRefs {
		requestedPaths = nil
		pages, err := p.FetchPages(context.Background(), tt.mangaRef, tt.chapterRef)
		if err != nil {
			t.Errorf("FetchPages(%q, %q) unexpected error: %v", tt.mangaRef, tt.chapterRef, err)
			continue
		}
		if len(pages) != 1 {
			t.Errorf("FetchPages(%q, %q) expected 1 page, got %d", tt.mangaRef, tt.chapterRef, len(pages))
		}
		if len(requestedPaths) == 0 || requestedPaths[0] != tt.expectedPath {
			t.Errorf("FetchPages(%q, %q) requested path %v, want %q", tt.mangaRef, tt.chapterRef, requestedPaths, tt.expectedPath)
		}
	}
}

func TestMangaFoxReadingMode(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		switch r.URL.Path {
		case "/manga/standard_manga/":
			_, _ = w.Write([]byte(`
				<div class="detail-info-right-title-font">Standard Manga</div>
				<p class="detail-info-right-tag"><a>Action</a><a>Adventure</a></p>
			`))
		case "/manga/webtoon_manga/":
			_, _ = w.Write([]byte(`
				<div class="detail-info-right-title-font">Webtoon Manga</div>
				<p class="detail-info-right-tag"><a>Action</a><a>Webtoons</a></p>
			`))
		case "/manga/manhwa_manga/":
			_, _ = w.Write([]byte(`
				<div class="detail-info-right-title-font">Manhwa Manga</div>
				<p class="detail-info-right-tag"><a>Fantasy</a><a>Manhwa</a></p>
			`))
		case "/manga/longstrip_manga/":
			_, _ = w.Write([]byte(`
				<div class="detail-info-right-title-font">Long Strip Manga</div>
				<p class="detail-info-right-tag"><a>Drama</a><a>Long Strip</a></p>
			`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	p, err := mangafox.NewProvider(ts.Client())
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	p.HttpSource.Config.BaseURL = ts.URL

	tests := []struct {
		slug     string
		expected sdk.ReadingMode
	}{
		{"standard_manga", sdk.ReadingModeRTL},
		{"webtoon_manga", sdk.ReadingModeLongstrip},
		{"manhwa_manga", sdk.ReadingModeLongstrip},
		{"longstrip_manga", sdk.ReadingModeLongstrip},
	}

	for _, tt := range tests {
		meta, err := p.Details(context.Background(), tt.slug)
		if err != nil {
			t.Fatalf("Details(%q) failed: %v", tt.slug, err)
		}
		if meta.ReadingMode != tt.expected {
			t.Errorf("Details(%q).ReadingMode = %q, want %q", tt.slug, meta.ReadingMode, tt.expected)
		}
	}
}


