package mangadex

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tubruk/kiyomi/pkg/fingerprint"
	"github.com/tubruk/kiyomi/pkg/provider/sdk"
)

func TestMangaDexProvider(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/manga":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"data": [
					{
						"id": "manga-uuid-1",
						"attributes": {
							"title": {"en": "Sample Manga"}
						},
						"relationships": [
							{"type": "cover_art", "attributes": {"fileName": "cover.jpg"}}
						]
					}
				]
			}`))
		case "/manga/manga-uuid-1/feed":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"data": [
					{
						"id": "ch-uuid-1",
						"attributes": {
							"chapter": "1",
							"title": "First Chapter"
						}
					}
				]
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	p := NewProvider(server.Client())
	if p.ID() != "mangadex" {
		t.Errorf("expected ID mangadex, got %s", p.ID())
	}
	if !p.HasStableChapterID() {
		t.Errorf("expected HasStableChapterID to be true")
	}

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/manga", nil)
	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("failed test request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}
}

func TestMangaDexSearchAndDetailsAvailability(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/manga":
			_, _ = w.Write([]byte(`{
				"data": [
					{
						"id": "available-manga",
						"attributes": {
							"title": {"en": "Available Manga"},
							"availableTranslatedLanguages": ["en"],
							"latestUploadedChapter": "ch-123"
						},
						"relationships": []
					},
					{
						"id": "no-lang-manga",
						"attributes": {
							"title": {"en": "No Lang Manga"},
							"availableTranslatedLanguages": [],
							"latestUploadedChapter": "ch-123"
						},
						"relationships": []
					},
					{
						"id": "no-chapter-manga",
						"attributes": {
							"title": {"en": "No Chapter Manga"},
							"availableTranslatedLanguages": ["en"],
							"latestUploadedChapter": null
						},
						"relationships": []
					}
				]
			}`))
		case "/manga/available-manga":
			_, _ = w.Write([]byte(`{
				"data": {
					"id": "available-manga",
					"attributes": {
						"title": {"en": "Available Manga"},
						"availableTranslatedLanguages": ["en"],
						"latestUploadedChapter": "ch-123"
					},
					"relationships": []
				}
			}`))
		case "/manga/unavailable-manga":
			_, _ = w.Write([]byte(`{
				"data": {
					"id": "unavailable-manga",
					"attributes": {
						"title": {"en": "Unavailable Manga"},
						"availableTranslatedLanguages": [],
						"latestUploadedChapter": null
					},
					"relationships": []
				}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	p := NewProvider(server.Client())
	p.Config.BaseURL = server.URL

	// Test Search availability
	results, err := p.Search(context.Background(), "", sdk.SearchOptions{})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[0].Availability != sdk.AvailabilityAvailable {
		t.Errorf("expected result 0 availability to be %s, got %s", sdk.AvailabilityAvailable, results[0].Availability)
	}
	if results[1].Availability != sdk.AvailabilityUnavailable {
		t.Errorf("expected result 1 availability to be %s, got %s", sdk.AvailabilityUnavailable, results[1].Availability)
	}
	if results[2].Availability != sdk.AvailabilityUnavailable {
		t.Errorf("expected result 2 availability to be %s, got %s", sdk.AvailabilityUnavailable, results[2].Availability)
	}

	// Test Details available
	detailsAvail, err := p.Details(context.Background(), "available-manga")
	if err != nil {
		t.Fatalf("details available failed: %v", err)
	}
	if detailsAvail.Availability != sdk.AvailabilityAvailable {
		t.Errorf("expected details availability %s, got %s", sdk.AvailabilityAvailable, detailsAvail.Availability)
	}

	// Test Details unavailable
	detailsUnavail, err := p.Details(context.Background(), "unavailable-manga")
	if err != nil {
		t.Fatalf("details unavailable failed: %v", err)
	}
	if detailsUnavail.Availability != sdk.AvailabilityUnavailable {
		t.Errorf("expected details availability %s, got %s", sdk.AvailabilityUnavailable, detailsUnavail.Availability)
	}
}

func TestMangaDexInterfaceCompliance(t *testing.T) {
	p := NewProvider(nil)
	var _ sdk.Content = p
	var _ sdk.Metadata = p
}

func TestMangaDexFingerprintStore(t *testing.T) {
	fpStore := fingerprint.NewMemoryStore()
	_ = fpStore.Set("mangadex", fingerprint.Profile{
		UserAgent: "MangaDexUA/1.0",
	})

	p := NewProvider(nil, fpStore)
	if p.Fingerprint != fpStore {
		t.Errorf("expected fingerprint store to be set on provider")
	}
}

func TestMangaDexFetchChapters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/manga/manga-uuid-1/feed" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"data": [
					{
						"id": "ch-uuid-1",
						"attributes": {
							"chapter": "1",
							"title": "First Chapter",
							"publishAt": "2023-11-14T22:13:20Z"
						}
					}
				]
			}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	p := NewProvider(server.Client())
	p.Config.BaseURL = server.URL

	chapters, err := p.FetchChapters(context.Background(), "manga-uuid-1")
	if err != nil {
		t.Fatalf("FetchChapters failed: %v", err)
	}
	if len(chapters) != 1 {
		t.Fatalf("expected 1 chapter, got %d", len(chapters))
	}
	expectedUploadDate, _ := time.Parse(time.RFC3339, "2023-11-14T22:13:20Z")
	if !chapters[0].UploadDate.Equal(expectedUploadDate) {
		t.Errorf("expected UploadDate %v, got %v", expectedUploadDate, chapters[0].UploadDate)
	}
}

func TestMangaDexReadingMode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/manga/ja-manga":
			_, _ = w.Write([]byte(`{
				"data": {
					"id": "ja-manga",
					"attributes": {
						"title": {"en": "Japanese Manga"},
						"originalLanguage": "ja",
						"tags": [{"attributes": {"name": {"en": "Action"}}}]
					}
				}
			}`))
		case "/manga/ko-manhwa":
			_, _ = w.Write([]byte(`{
				"data": {
					"id": "ko-manhwa",
					"attributes": {
						"title": {"en": "Korean Manhwa"},
						"originalLanguage": "ko",
						"tags": []
					}
				}
			}`))
		case "/manga/zh-manhua":
			_, _ = w.Write([]byte(`{
				"data": {
					"id": "zh-manhua",
					"attributes": {
						"title": {"en": "Chinese Manhua"},
						"originalLanguage": "zh",
						"tags": []
					}
				}
			}`))
		case "/manga/longstrip-tag":
			_, _ = w.Write([]byte(`{
				"data": {
					"id": "longstrip-tag",
					"attributes": {
						"title": {"en": "Longstrip Manga"},
						"originalLanguage": "ja",
						"tags": [{"attributes": {"name": {"en": "Long Strip"}}}]
					}
				}
			}`))
		case "/manga/en-comic":
			_, _ = w.Write([]byte(`{
				"data": {
					"id": "en-comic",
					"attributes": {
						"title": {"en": "English Comic"},
						"originalLanguage": "en",
						"tags": [{"attributes": {"name": {"en": "Action"}}}]
					}
				}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	p := NewProvider(server.Client())
	p.Config.BaseURL = server.URL

	tests := []struct {
		remoteID string
		expected sdk.ReadingMode
	}{
		{"ja-manga", sdk.ReadingModeRTL},
		{"ko-manhwa", sdk.ReadingModeLongstrip},
		{"zh-manhua", sdk.ReadingModeLongstrip},
		{"longstrip-tag", sdk.ReadingModeLongstrip},
		{"en-comic", sdk.ReadingModeUnspecified},
	}

	for _, tt := range tests {
		details, err := p.Details(context.Background(), tt.remoteID)
		if err != nil {
			t.Fatalf("Details(%q) failed: %v", tt.remoteID, err)
		}
		if details.ReadingMode != tt.expected {
			t.Errorf("Details(%q).ReadingMode = %q, want %q", tt.remoteID, details.ReadingMode, tt.expected)
		}
	}
}


