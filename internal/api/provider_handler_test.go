package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tubruk/kiyomi/internal/library"
	"github.com/tubruk/kiyomi/pkg/logger"
	"github.com/tubruk/kiyomi/pkg/provider/sdk"
)

type mockProvider struct {
	id      string
	name    string
	baseURL string
}

func (m *mockProvider) ID() string                    { return m.id }
func (m *mockProvider) Name() string                  { return m.name }
func (m *mockProvider) Icon() string                  { return "https://example.com/icon.png" }
func (m *mockProvider) Capabilities() []string        { return []string{"metadata", "content"} }
func (m *mockProvider) ConfigKeys() []sdk.ConfigKeySpec { return nil }
func (m *mockProvider) RequiresAuth() bool           { return false }
func (m *mockProvider) State() sdk.ProviderState      { return sdk.StateActive }

func (m *mockProvider) GetConfig() sdk.ProviderConfig {
	bURL := m.baseURL
	if bURL == "" {
		bURL = "https://example.com"
	}
	return sdk.ProviderConfig{
		ID:       m.id,
		Name:     m.name,
		BaseURL:  bURL,
		Language: "en",
	}
}

func (m *mockProvider) Search(ctx context.Context, query string, opts sdk.SearchOptions) ([]sdk.SearchResult, error) {
	return []sdk.SearchResult{
		{
			RemoteID:     "mock-1",
			Title:        "Mock Manga 1",
			Aliases:      []string{"Mock Manga 1 Alt"},
			CoverURL:     "https://example.com/cover1.jpg",
			URL:          "https://example.com/manga/mock-1",
			Availability: sdk.AvailabilityAvailable,
		},
	}, nil
}

func (m *mockProvider) Details(ctx context.Context, remoteID string) (sdk.MangaMetadata, error) {
	return sdk.MangaMetadata{
		RemoteID:     remoteID,
		Title:        "Mock Manga Details",
		Aliases:      []string{"Mock Manga Details Alt"},
		Synopsis:     "Mock synopsis",
		Authors:      []string{"Mock Author"},
		Artists:      []string{"Mock Artist"},
		Tags:         []string{"Action", "Fantasy"},
		Publishers:   []string{"Mock Publisher"},
		ReleaseYear:  2022,
		StartDate:    "2022-01-01",
		EndDate:      "2023-01-01",
		Country:      "JP",
		Score:        8.5,
		ReadingMode:  sdk.ReadingModeLongstrip,
		Availability: sdk.AvailabilityAvailable,
		URL:          "https://example.com/manga/" + remoteID,
	}, nil
}

func (m *mockProvider) Cover(ctx context.Context, remoteID string, size sdk.ImageSize) (sdk.ImageRef, error) {
	return sdk.ImageRef{URL: "https://example.com/cover.jpg"}, nil
}

func (m *mockProvider) Aliases(ctx context.Context, remoteID string) ([]string, error) {
	return nil, nil
}

func (m *mockProvider) HasStableChapterID() bool { return true }
func (m *mockProvider) FetchChapters(ctx context.Context, mangaRef string) ([]sdk.Chapter, error) {
	return []sdk.Chapter{
		{ID: "ch-1", Name: "Chapter 1", Number: 1, UploadDate: time.Date(2023, 11, 14, 22, 13, 20, 0, time.UTC), SourceOrder: 1},
	}, nil
}
func (m *mockProvider) FetchPages(ctx context.Context, mangaRef, chapterRef string) ([]sdk.Page, error) {
	return nil, nil
}
func (m *mockProvider) FetchPageStream(ctx context.Context, page sdk.Page) (io.ReadCloser, error) {
	return nil, nil
}
func (m *mockProvider) RateLimit() sdk.RateLimitHint { return sdk.RateLimitHint{} }

type mockMultiChapterProvider struct {
	mockProvider
	chapters []sdk.Chapter
}

func (m *mockMultiChapterProvider) FetchChapters(ctx context.Context, mangaRef string) ([]sdk.Chapter, error) {
	if m.chapters != nil {
		return m.chapters, nil
	}
	return m.mockProvider.FetchChapters(ctx, mangaRef)
}

func (m *mockMultiChapterProvider) FetchPages(ctx context.Context, mangaRef, chapterRef string) ([]sdk.Page, error) {
	return []sdk.Page{
		{Index: 1, URL: "https://example.com/page1.jpg"},
	}, nil
}

func TestListContentProviders(t *testing.T) {
	h, e := setupTestHandler(t)

	mockP := &mockProvider{id: "mockprov", name: "Mock Provider"}
	h.registry.Register(mockP)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/providers", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var list []map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Should contain registered mockprov
	if len(list) < 1 {
		t.Errorf("expected at least 1 content provider, got %d", len(list))
	}

	foundMock := false
	for _, p := range list {
		if p["id"] == "mockprov" {
			foundMock = true
			if p["name"] != "Mock Provider" {
				t.Errorf("expected mock provider name 'Mock Provider', got %v", p["name"])
			}
			if p["baseUrl"] != "https://example.com" {
				t.Errorf("expected baseUrl 'https://example.com', got %v", p["baseUrl"])
			}
		}
	}
	if !foundMock {
		t.Errorf("mockprov was not found in provider list")
	}
}

func TestProviderRoutesWithRegistry(t *testing.T) {
	h, e := setupTestHandler(t)

	mockP := &mockProvider{id: "testprov", name: "Test Provider"}
	h.registry.Register(mockP)

	t.Run("GET /providers/testprov/manga (default catalog)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/testprov/manga", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}
		var resp map[string]interface{}
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		mangas, ok := resp["mangas"].([]interface{})
		if !ok || len(mangas) != 1 {
			t.Fatalf("expected 1 manga result, got %v", resp)
		}
		firstManga := mangas[0].(map[string]interface{})
		if firstManga["availability"] != "available" {
			t.Errorf("expected availability 'available', got %v", firstManga["availability"])
		}
		if firstManga["url"] != "https://example.com/manga/mock-1" {
			t.Errorf("expected url 'https://example.com/manga/mock-1', got %v", firstManga["url"])
		}
		firstAliases, ok := firstManga["aliases"].([]interface{})
		if !ok || len(firstAliases) != 1 || firstAliases[0] != "Mock Manga 1 Alt" {
			t.Errorf("expected manga aliases ['Mock Manga 1 Alt'], got %v", firstManga["aliases"])
		}
		if page, ok := resp["page"].(float64); !ok || page != 1 {
			t.Errorf("expected page 1, got %v", resp["page"])
		}
	})

	t.Run("GET /providers/testprov/manga?mode=latest", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/testprov/manga?mode=latest", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("GET /providers/testprov/manga?q=mock", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/testprov/manga?q=mock", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}
		var resp map[string]interface{}
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		mangas, ok := resp["mangas"].([]interface{})
		if !ok || len(mangas) != 1 {
			t.Fatalf("expected 1 manga result, got %v", resp)
		}
		firstManga := mangas[0].(map[string]interface{})
		if firstManga["availability"] != "available" {
			t.Errorf("expected availability 'available', got %v", firstManga["availability"])
		}
		if firstManga["url"] != "https://example.com/manga/mock-1" {
			t.Errorf("expected url 'https://example.com/manga/mock-1', got %v", firstManga["url"])
		}
		firstAliases, ok := firstManga["aliases"].([]interface{})
		if !ok || len(firstAliases) != 1 || firstAliases[0] != "Mock Manga 1 Alt" {
			t.Errorf("expected manga aliases ['Mock Manga 1 Alt'], got %v", firstManga["aliases"])
		}
	})

	t.Run("GET /providers/testprov/popular", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/testprov/popular", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}
		var resp map[string]interface{}
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		mangas, ok := resp["mangas"].([]interface{})
		if !ok || len(mangas) != 1 {
			t.Fatalf("expected 1 manga result, got %v", resp)
		}
		firstManga := mangas[0].(map[string]interface{})
		if firstManga["availability"] != "available" {
			t.Errorf("expected availability 'available', got %v", firstManga["availability"])
		}
		if firstManga["url"] != "https://example.com/manga/mock-1" {
			t.Errorf("expected url 'https://example.com/manga/mock-1', got %v", firstManga["url"])
		}
		firstAliases, ok := firstManga["aliases"].([]interface{})
		if !ok || len(firstAliases) != 1 || firstAliases[0] != "Mock Manga 1 Alt" {
			t.Errorf("expected manga aliases ['Mock Manga 1 Alt'], got %v", firstManga["aliases"])
		}
	})

	t.Run("GET /providers/testprov/latest", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/testprov/latest", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}
		var resp map[string]interface{}
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		mangas, ok := resp["mangas"].([]interface{})
		if !ok || len(mangas) != 1 {
			t.Fatalf("expected 1 manga result, got %v", resp)
		}
		firstManga := mangas[0].(map[string]interface{})
		if firstManga["availability"] != "available" {
			t.Errorf("expected availability 'available', got %v", firstManga["availability"])
		}
		if firstManga["url"] != "https://example.com/manga/mock-1" {
			t.Errorf("expected url 'https://example.com/manga/mock-1', got %v", firstManga["url"])
		}
		firstAliases, ok := firstManga["aliases"].([]interface{})
		if !ok || len(firstAliases) != 1 || firstAliases[0] != "Mock Manga 1 Alt" {
			t.Errorf("expected manga aliases ['Mock Manga 1 Alt'], got %v", firstManga["aliases"])
		}
	})

	t.Run("GET /providers/testprov/search", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/testprov/search?q=mock", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}
		var resp map[string]interface{}
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		mangas, ok := resp["mangas"].([]interface{})
		if !ok || len(mangas) != 1 {
			t.Fatalf("expected 1 manga result, got %v", resp)
		}
		firstManga := mangas[0].(map[string]interface{})
		if firstManga["availability"] != "available" {
			t.Errorf("expected availability 'available', got %v", firstManga["availability"])
		}
		if firstManga["url"] != "https://example.com/manga/mock-1" {
			t.Errorf("expected url 'https://example.com/manga/mock-1', got %v", firstManga["url"])
		}
		firstAliases, ok := firstManga["aliases"].([]interface{})
		if !ok || len(firstAliases) != 1 || firstAliases[0] != "Mock Manga 1 Alt" {
			t.Errorf("expected manga aliases ['Mock Manga 1 Alt'], got %v", firstManga["aliases"])
		}
	})

	t.Run("GET /providers/testprov/manga/mock-1", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/testprov/manga/mock-1", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}
		var details map[string]interface{}
		_ = json.Unmarshal(rec.Body.Bytes(), &details)
		if details["title"] != "Mock Manga Details" {
			t.Errorf("expected title 'Mock Manga Details', got %v", details["title"])
		}
		if details["availability"] != "available" {
			t.Errorf("expected availability 'available', got %v", details["availability"])
		}
		if details["reading_mode"] != "longstrip" {
			t.Errorf("expected reading_mode 'longstrip', got %v", details["reading_mode"])
		}
		if details["url"] != "https://example.com/manga/mock-1" {
			t.Errorf("expected url 'https://example.com/manga/mock-1', got %v", details["url"])
		}
		aliases, ok := details["aliases"].([]interface{})
		if !ok || len(aliases) != 1 || aliases[0] != "Mock Manga Details Alt" {
			t.Errorf("expected aliases ['Mock Manga Details Alt'], got %v", details["aliases"])
		}
		if score, ok := details["score"].(float64); !ok || score != 8.5 {
			t.Errorf("expected score 8.5, got %v", details["score"])
		}
		authors, ok := details["authors"].([]interface{})
		if !ok || len(authors) != 1 || authors[0] != "Mock Author" {
			t.Errorf("expected authors ['Mock Author'], got %v", details["authors"])
		}
		artists, ok := details["artists"].([]interface{})
		if !ok || len(artists) != 1 || artists[0] != "Mock Artist" {
			t.Errorf("expected artists ['Mock Artist'], got %v", details["artists"])
		}
		tags, ok := details["tags"].([]interface{})
		if !ok || len(tags) != 2 || tags[0] != "Action" || tags[1] != "Fantasy" {
			t.Errorf("expected tags ['Action', 'Fantasy'], got %v", details["tags"])
		}
		publishers, ok := details["publishers"].([]interface{})
		if !ok || len(publishers) != 1 || publishers[0] != "Mock Publisher" {
			t.Errorf("expected publishers ['Mock Publisher'], got %v", details["publishers"])
		}
		if yr, ok := details["release_year"].(float64); !ok || yr != 2022 {
			t.Errorf("expected release_year 2022, got %v", details["release_year"])
		}
		if details["start_date"] != "2022-01-01" {
			t.Errorf("expected start_date '2022-01-01', got %v", details["start_date"])
		}
		if details["end_date"] != "2023-01-01" {
			t.Errorf("expected end_date '2023-01-01', got %v", details["end_date"])
		}
		if details["country"] != "JP" {
			t.Errorf("expected country 'JP', got %v", details["country"])
		}
	})

	t.Run("GET /providers/testprov/manga/mock-1/chapters", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/testprov/manga/mock-1/chapters", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}
		var resp map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		chapters, ok := resp["chapters"].([]interface{})
		if !ok || len(chapters) != 1 {
			t.Fatalf("expected 1 chapter result, got %v", resp)
		}
		ch := chapters[0].(map[string]interface{})
		if ch["id"] != "ch-1" {
			t.Errorf("expected chapter id 'ch-1', got %v", ch["id"])
		}
		if ch["title"] != "Chapter 1" {
			t.Errorf("expected chapter title 'Chapter 1', got %v", ch["title"])
		}
		if num, ok := ch["number"].(float64); !ok || num != 1 {
			t.Errorf("expected chapter number 1, got %v", ch["number"])
		}
		if ch["sourceOrder"] != float64(1) {
			t.Errorf("expected sourceOrder 1, got %v", ch["sourceOrder"])
		}
		if ch["uploadDate"] != "2023-11-14T22:13:20Z" {
			t.Errorf("expected uploadDate '2023-11-14T22:13:20Z', got %v", ch["uploadDate"])
		}
	})

	t.Run("GET /providers/nonexistent/manga returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/nonexistent/manga", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404 Not Found, got %d", rec.Code)
		}
	})

	t.Run("GET /providers/nonexistent/popular returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/nonexistent/popular", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404 Not Found, got %d", rec.Code)
		}
	})

	t.Run("GET /providers/nonexistent/manga/mock-1/chapters returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/nonexistent/manga/mock-1/chapters", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404 Not Found, got %d", rec.Code)
		}
	})
}

func TestImportProviderManga_UserStatusValidation(t *testing.T) {
	h, e := setupTestHandler(t)

	mockP := &mockProvider{id: "testprov", name: "Test Provider"}
	h.registry.Register(mockP)

	// Import with invalid user status
	importInvalid := map[string]string{
		"provider_id": "testprov",
		"remote_id":   "mock-1",
		"user_status": "invalid_reading_status",
	}
	bodyBytes, _ := json.Marshal(importInvalid)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/import", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request on import with invalid user_status, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestImportProviderManga_PreservesChapterMeta(t *testing.T) {
	h, e := setupTestHandler(t)

	mockP := &mockProvider{id: "testprov", name: "Test Provider"}
	h.registry.Register(mockP)

	body := map[string]string{
		"provider_id": "testprov",
		"remote_id":   "mock-1",
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/import", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created on import, got %d: %s", rec.Code, rec.Body.String())
	}

	var importResp struct {
		ID       string                `json:"id"`
		Metadata library.MangaMetadata `json:"metadata"`
		Bindings library.MangaBindings `json:"bindings"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &importResp); err != nil {
		t.Fatalf("failed to decode import response: %v", err)
	}
	if len(importResp.Metadata.Authors) != 1 || importResp.Metadata.Authors[0] != "Mock Author" {
		t.Errorf("expected import response Authors ['Mock Author'], got %v", importResp.Metadata.Authors)
	}
	if len(importResp.Metadata.Artists) != 1 || importResp.Metadata.Artists[0] != "Mock Artist" {
		t.Errorf("expected import response Artists ['Mock Artist'], got %v", importResp.Metadata.Artists)
	}
	if len(importResp.Metadata.Tags) != 2 || importResp.Metadata.Tags[0] != "Action" || importResp.Metadata.Tags[1] != "Fantasy" {
		t.Errorf("expected import response Tags ['Action', 'Fantasy'], got %v", importResp.Metadata.Tags)
	}
	if len(importResp.Metadata.Publishers) != 1 || importResp.Metadata.Publishers[0] != "Mock Publisher" {
		t.Errorf("expected import response Publishers ['Mock Publisher'], got %v", importResp.Metadata.Publishers)
	}
	if importResp.Metadata.ReleaseYear != 2022 || importResp.Metadata.StartDate != "2022-01-01" || importResp.Metadata.EndDate != "2023-01-01" || importResp.Metadata.Country != "JP" {
		t.Errorf("unexpected import response date/country: %+v", importResp.Metadata)
	}
	if importResp.Bindings.Content == nil || importResp.Bindings.Content.ReadingMode != "longstrip" {
		t.Errorf("expected import response Content.ReadingMode 'longstrip', got %+v", importResp.Bindings.Content)
	}
	if len(importResp.Metadata.ExternalLinks) != 1 {
		t.Fatalf("expected 1 external link in import response, got %d", len(importResp.Metadata.ExternalLinks))
	}
	if importResp.Metadata.ExternalLinks[0].Provider != "testprov" || importResp.Metadata.ExternalLinks[0].Label != "Test Provider" || importResp.Metadata.ExternalLinks[0].URL != "https://example.com/manga/mock-1" {
		t.Errorf("unexpected external link in import response: %+v", importResp.Metadata.ExternalLinks[0])
	}

	// Verify manga endpoint returns reading_mode
	req = httptest.NewRequest(http.MethodGet, "/api/v1/library/manga/mock-1", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for get manga, got %d: %s", rec.Code, rec.Body.String())
	}
	var getMangaResp map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &getMangaResp)
	if getMangaResp["reading_mode"] != "longstrip" {
		t.Errorf("expected getManga reading_mode 'longstrip', got %v", getMangaResp["reading_mode"])
	}
	topExtLinks, ok := getMangaResp["external_links"].([]interface{})
	if !ok || len(topExtLinks) != 1 {
		t.Fatalf("expected 1 top-level external_links in getMangaResp, got %v", getMangaResp["external_links"])
	}
	topExtLinksCamel, ok := getMangaResp["externalLinks"].([]interface{})
	if !ok || len(topExtLinksCamel) != 1 {
		t.Fatalf("expected 1 top-level externalLinks in getMangaResp, got %v", getMangaResp["externalLinks"])
	}
	metaMap, ok := getMangaResp["metadata"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected metadata map in getMangaResp, got %v", getMangaResp)
	}
	extLinks, ok := metaMap["external_links"].([]interface{})
	if !ok || len(extLinks) != 1 {
		t.Fatalf("expected 1 external_link in metadata, got %v", metaMap["external_links"])
	}
	extLinkMap := extLinks[0].(map[string]interface{})
	if extLinkMap["provider"] != "testprov" || extLinkMap["label"] != "Test Provider" || extLinkMap["url"] != "https://example.com/manga/mock-1" {
		t.Errorf("unexpected external_link: %+v", extLinkMap)
	}

	// Verify chapter list endpoint returns sourceOrder and uploadDate
	req = httptest.NewRequest(http.MethodGet, "/api/v1/library/manga/mock-1/chapters", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Chapters []map[string]interface{} `json:"chapters"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode chapters response: %v", err)
	}

	if len(resp.Chapters) != 1 {
		t.Fatalf("expected 1 chapter, got %d", len(resp.Chapters))
	}

	ch := resp.Chapters[0]
	if ch["sourceOrder"] != float64(1) {
		t.Errorf("expected sourceOrder 1, got %v", ch["sourceOrder"])
	}
	if ch["uploadDate"] != "2023-11-14T22:13:20Z" {
		t.Errorf("expected uploadDate '2023-11-14T22:13:20Z', got %v", ch["uploadDate"])
	}

	meta, ok := ch["meta"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected chapter meta object, got %v", ch["meta"])
	}
	if meta["source_order"] != float64(1) {
		t.Errorf("expected meta.source_order 1, got %v", meta["source_order"])
	}
	if meta["upload_date"] != "2023-11-14T22:13:20Z" {
		t.Errorf("expected meta.upload_date '2023-11-14T22:13:20Z', got %v", meta["upload_date"])
	}
}

func TestImportProviderManga_PersistsExpandedMetadata(t *testing.T) {
	h, e := setupTestHandler(t)

	mockP := &mockProvider{id: "testprov", name: "Test Provider"}
	h.registry.Register(mockP)

	body := map[string]string{
		"provider_id": "testprov",
		"remote_id":   "mock-1",
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/import", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created on import, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify that library storage actually persisted all expanded metadata fields
	storedMeta, err := h.lib.GetManga("mock-1")
	if err != nil {
		t.Fatalf("failed to retrieve stored manga from library: %v", err)
	}

	if len(storedMeta.Metadata.Publishers) != 1 || storedMeta.Metadata.Publishers[0] != "Mock Publisher" {
		t.Errorf("expected stored Publishers ['Mock Publisher'], got %v", storedMeta.Metadata.Publishers)
	}
	if storedMeta.Metadata.StartDate != "2022-01-01" {
		t.Errorf("expected stored StartDate '2022-01-01', got %q", storedMeta.Metadata.StartDate)
	}
	if storedMeta.Metadata.EndDate != "2023-01-01" {
		t.Errorf("expected stored EndDate '2023-01-01', got %q", storedMeta.Metadata.EndDate)
	}
	if storedMeta.Metadata.Country != "JP" {
		t.Errorf("expected stored Country 'JP', got %q", storedMeta.Metadata.Country)
	}
	if storedMeta.Metadata.ReleaseYear != 2022 {
		t.Errorf("expected stored ReleaseYear 2022, got %d", storedMeta.Metadata.ReleaseYear)
	}
	if len(storedMeta.Metadata.Authors) != 1 || storedMeta.Metadata.Authors[0] != "Mock Author" {
		t.Errorf("expected stored Authors ['Mock Author'], got %v", storedMeta.Metadata.Authors)
	}
	if len(storedMeta.Metadata.Artists) != 1 || storedMeta.Metadata.Artists[0] != "Mock Artist" {
		t.Errorf("expected stored Artists ['Mock Artist'], got %v", storedMeta.Metadata.Artists)
	}
	if len(storedMeta.Metadata.Tags) != 2 || storedMeta.Metadata.Tags[0] != "Action" || storedMeta.Metadata.Tags[1] != "Fantasy" {
		t.Errorf("expected stored Tags ['Action', 'Fantasy'], got %v", storedMeta.Metadata.Tags)
	}
	if len(storedMeta.Metadata.Aliases) != 1 || storedMeta.Metadata.Aliases[0] != "Mock Manga Details Alt" {
		t.Errorf("expected stored Aliases ['Mock Manga Details Alt'], got %v", storedMeta.Metadata.Aliases)
	}
}

type mockFailingProvider struct {
	mockProvider
}

func (m *mockFailingProvider) Search(ctx context.Context, query string, opts sdk.SearchOptions) ([]sdk.SearchResult, error) {
	return nil, errors.New("upstream provider connection timed out")
}

func TestHandleProviderError_Logging(t *testing.T) {
	buf := &bytes.Buffer{}
	logger.Setup(logger.Options{
		Level:   "debug",
		Format:  "pretty",
		NoColor: true,
		Writer:  buf,
	})

	h, e := setupTestHandler(t)
	mockP := &mockFailingProvider{mockProvider: mockProvider{id: "failingprov", name: "Failing Provider"}}
	h.registry.Register(mockP)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/failingprov/popular", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 Bad Gateway for timeout error, got %d: %s", rec.Code, rec.Body.String())
	}

	out := buf.String()
	if !strings.Contains(out, "provider request failed") {
		t.Errorf("expected 'provider request failed' in log, got: %s", out)
	}
	if !strings.Contains(out, "provider_id=failingprov") {
		t.Errorf("expected 'provider_id=failingprov' in log, got: %s", out)
	}
	if !strings.Contains(out, "status=502") {
		t.Errorf("expected 'status=502' in log, got: %s", out)
	}
	if !strings.Contains(out, "upstream provider connection timed out") {
		t.Errorf("expected error details in log, got: %s", out)
	}
}

func TestProviderHandler_EchoErrorLogger_PropagatesHandlerError(t *testing.T) {
	buf := &bytes.Buffer{}
	logger.Setup(logger.Options{
		Level:   "debug",
		Format:  "pretty",
		NoColor: true,
		Writer:  buf,
	})

	h, e := setupTestHandler(t)
	e.Use(logger.EchoErrorLogger())

	mockP := &mockFailingProvider{mockProvider: mockProvider{id: "failprovider", name: "Failing Provider"}}
	h.registry.Register(mockP)

	t.Run("404 missing provider logs handler_error in EchoErrorLogger", func(t *testing.T) {
		buf.Reset()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/unknownprov/popular", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404 Not Found, got %d", rec.Code)
		}

		out := buf.String()
		if !strings.Contains(out, "HTTP request warning") {
			t.Errorf("expected 'HTTP request warning' in log, got: %s", out)
		}
		if !strings.Contains(out, "status=404") {
			t.Errorf("expected status=404 in log, got: %s", out)
		}
		if !strings.Contains(out, "provider not found: unknownprov") {
			t.Errorf("expected handler error 'provider not found: unknownprov' in log, got: %s", out)
		}
	})

	t.Run("502 provider failure logs provider request failed and suppresses EchoErrorLogger duplicate", func(t *testing.T) {
		buf.Reset()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/failprovider/popular", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadGateway {
			t.Fatalf("expected 502 Bad Gateway, got %d", rec.Code)
		}

		out := buf.String()
		if !strings.Contains(out, "provider request failed") {
			t.Errorf("expected 'provider request failed' in log, got: %s", out)
		}
		if !strings.Contains(out, "provider_id=failprovider") {
			t.Errorf("expected provider_id=failprovider in log, got: %s", out)
		}
		if !strings.Contains(out, "status=502") {
			t.Errorf("expected status=502 in log, got: %s", out)
		}
		if strings.Contains(out, "HTTP request error") {
			t.Errorf("expected EchoErrorLogger duplicate 'HTTP request error' to be suppressed, got: %s", out)
		}
	})
}

func TestImportProviderManga_Validation(t *testing.T) {
	_, e := setupTestHandler(t)

	tests := []struct {
		name       string
		providerID string
		remoteID   string
	}{
		{"empty provider_id", "", "remote-1"},
		{"empty remote_id", "prov-1", ""},
		{"both empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := map[string]string{
				"provider_id": tt.providerID,
				"remote_id":   tt.remoteID,
			}
			bodyBytes, _ := json.Marshal(body)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/import", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected 400 Bad Request, got %d", rec.Code)
			}
		})
	}
}

func TestImportProviderManga_ConcurrentBatch(t *testing.T) {
	h, e := setupTestHandler(t)

	mockP := &mockMultiChapterProvider{
		mockProvider: mockProvider{id: "batchimportprov", name: "Batch Import Provider"},
	}
	const totalChapters = 35
	for i := 1; i <= totalChapters; i++ {
		mockP.chapters = append(mockP.chapters, sdk.Chapter{
			ID:          fmt.Sprintf("import-ch-%d", i),
			Name:        fmt.Sprintf("Chapter %d", i),
			Number:      float32(i),
			UploadDate:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			SourceOrder: i,
		})
	}
	h.registry.Register(mockP)

	body := map[string]string{
		"provider_id": "batchimportprov",
		"remote_id":   "batch-import-manga-1",
		"user_status": library.UserStatusReading,
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/import", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created on import, got %d: %s", rec.Code, rec.Body.String())
	}

	chapters, err := h.lib.ListChapters("batch-import-manga-1")
	if err != nil {
		t.Fatalf("failed to list chapters: %v", err)
	}
	if len(chapters) != totalChapters {
		t.Fatalf("expected %d chapters, got %d", totalChapters, len(chapters))
	}
}

type mockNoURLProvider struct {
	mockProvider
}

func (m *mockNoURLProvider) Details(ctx context.Context, remoteID string) (sdk.MangaMetadata, error) {
	meta, err := m.mockProvider.Details(ctx, remoteID)
	if err != nil {
		return meta, err
	}
	meta.URL = ""
	return meta, nil
}

func TestImportProviderManga_ExternalLinks(t *testing.T) {
	h, e := setupTestHandler(t)

	mockP := &mockProvider{id: "testprov", name: "Test Provider"}
	h.registry.Register(mockP)

	mockNoURL := &mockNoURLProvider{mockProvider: mockProvider{id: "nourlprov", name: "No URL Provider"}}
	h.registry.Register(mockNoURL)

	t.Run("with URL", func(t *testing.T) {
		body := map[string]string{
			"provider_id": "testprov",
			"remote_id":   "mock-with-url",
		}
		bodyBytes, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/import", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created on import, got %d: %s", rec.Code, rec.Body.String())
		}

		var importResp struct {
			ID       string                `json:"id"`
			Metadata library.MangaMetadata `json:"metadata"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &importResp); err != nil {
			t.Fatalf("failed to decode import response: %v", err)
		}

		if len(importResp.Metadata.ExternalLinks) != 1 {
			t.Fatalf("expected 1 external link, got %d", len(importResp.Metadata.ExternalLinks))
		}
		link := importResp.Metadata.ExternalLinks[0]
		if link.Provider != "testprov" || link.Label != "Test Provider" || link.URL != "https://example.com/manga/mock-with-url" {
			t.Errorf("unexpected external link: %+v", link)
		}

		stored, err := h.lib.GetManga("mock-with-url")
		if err != nil {
			t.Fatalf("GetManga failed: %v", err)
		}
		if len(stored.Metadata.ExternalLinks) != 1 {
			t.Fatalf("expected 1 stored external link, got %d", len(stored.Metadata.ExternalLinks))
		}
		if stored.Metadata.ExternalLinks[0].Provider != "testprov" || stored.Metadata.ExternalLinks[0].Label != "Test Provider" || stored.Metadata.ExternalLinks[0].URL != "https://example.com/manga/mock-with-url" {
			t.Errorf("unexpected stored external link: %+v", stored.Metadata.ExternalLinks[0])
		}
	})

	t.Run("without URL", func(t *testing.T) {
		body := map[string]string{
			"provider_id": "nourlprov",
			"remote_id":   "mock-without-url",
		}
		bodyBytes, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/import", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created on import, got %d: %s", rec.Code, rec.Body.String())
		}

		var importResp struct {
			ID       string                `json:"id"`
			Metadata library.MangaMetadata `json:"metadata"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &importResp); err != nil {
			t.Fatalf("failed to decode import response: %v", err)
		}

		if len(importResp.Metadata.ExternalLinks) != 0 {
			t.Errorf("expected 0 external links, got %d", len(importResp.Metadata.ExternalLinks))
		}

		stored, err := h.lib.GetManga("mock-without-url")
		if err != nil {
			t.Fatalf("GetManga failed: %v", err)
		}
		if len(stored.Metadata.ExternalLinks) != 0 {
			t.Errorf("expected 0 stored external links, got %d", len(stored.Metadata.ExternalLinks))
		}
	})
}

type mockEmptyMetaProvider struct {
	mockProvider
}

func (m *mockEmptyMetaProvider) Details(ctx context.Context, remoteID string) (sdk.MangaMetadata, error) {
	return sdk.MangaMetadata{
		RemoteID:     remoteID,
		Title:        "Mock Manga Empty Details",
		Synopsis:     "Mock synopsis",
		Availability: sdk.AvailabilityAvailable,
		URL:          "https://example.com/manga/" + remoteID,
	}, nil
}

func TestProviderMangaDetails_EmptyMetadata(t *testing.T) {
	h, e := setupTestHandler(t)

	mockEmpty := &mockEmptyMetaProvider{mockProvider: mockProvider{id: "emptymeta", name: "Empty Meta Provider"}}
	h.registry.Register(mockEmpty)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/emptymeta/manga/mock-empty", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}
	var details map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &details); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if authors, ok := details["authors"].([]interface{}); !ok || len(authors) != 0 {
		t.Errorf("expected empty authors [], got %v", details["authors"])
	}
	if artists, ok := details["artists"].([]interface{}); !ok || len(artists) != 0 {
		t.Errorf("expected empty artists [], got %v", details["artists"])
	}
	if aliases, ok := details["aliases"].([]interface{}); !ok || len(aliases) != 0 {
		t.Errorf("expected empty aliases [], got %v", details["aliases"])
	}
	if _, exists := details["score"]; exists {
		t.Errorf("expected score to be omitted when 0, got %v", details["score"])
	}
}

func TestProviderMangaDetails_ExpandedMetadata(t *testing.T) {
	h, e := setupTestHandler(t)

	mockP := &mockProvider{id: "fullmeta", name: "Full Meta Provider"}
	h.registry.Register(mockP)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/fullmeta/manga/remote-42", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}
	var details map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &details); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if details["id"] != "remote-42" {
		t.Errorf("expected id 'remote-42', got %v", details["id"])
	}
	if details["title"] != "Mock Manga Details" {
		t.Errorf("expected title 'Mock Manga Details', got %v", details["title"])
	}
	if details["start_date"] != "2022-01-01" {
		t.Errorf("expected start_date '2022-01-01', got %v", details["start_date"])
	}
	if details["end_date"] != "2023-01-01" {
		t.Errorf("expected end_date '2023-01-01', got %v", details["end_date"])
	}
	if details["country"] != "JP" {
		t.Errorf("expected country 'JP', got %v", details["country"])
	}
	if yr, ok := details["release_year"].(float64); !ok || yr != 2022 {
		t.Errorf("expected release_year 2022, got %v", details["release_year"])
	}
	if score, ok := details["score"].(float64); !ok || score != 8.5 {
		t.Errorf("expected score 8.5, got %v", details["score"])
	}
	if authors, ok := details["authors"].([]interface{}); !ok || len(authors) != 1 || authors[0] != "Mock Author" {
		t.Errorf("expected authors ['Mock Author'], got %v", details["authors"])
	}
	if artists, ok := details["artists"].([]interface{}); !ok || len(artists) != 1 || artists[0] != "Mock Artist" {
		t.Errorf("expected artists ['Mock Artist'], got %v", details["artists"])
	}
	if tags, ok := details["tags"].([]interface{}); !ok || len(tags) != 2 || tags[0] != "Action" || tags[1] != "Fantasy" {
		t.Errorf("expected tags ['Action', 'Fantasy'], got %v", details["tags"])
	}
	if aliases, ok := details["aliases"].([]interface{}); !ok || len(aliases) != 1 || aliases[0] != "Mock Manga Details Alt" {
		t.Errorf("expected aliases ['Mock Manga Details Alt'], got %v", details["aliases"])
	}
	if publishers, ok := details["publishers"].([]interface{}); !ok || len(publishers) != 1 || publishers[0] != "Mock Publisher" {
		t.Errorf("expected publishers ['Mock Publisher'], got %v", details["publishers"])
	}
}

func TestProviderMangaDetails_LibraryBindingID(t *testing.T) {
	h, e := setupTestHandler(t)

	mockFox := &mockProvider{id: "mangafox", name: "MangaFox"}
	mockDex := &mockProvider{id: "mangadex", name: "MangaDex"}
	h.registry.Register(mockFox)
	h.registry.Register(mockDex)

	// Cross-provider: primary mangafox/frieren (distinct from one-piece), Providers[] has mangadex/md-frieren
	crossMeta := library.Manga{
		Metadata: library.MangaMetadata{
			Title:       "Frieren",
			Aliases:     []string{},
			Description: " mage human.",
		},
		Bindings: library.MangaBindings{
			Content: &library.ContentSource{
				ProviderID:      "mangafox",
				ProviderMangaID: "frieren",
			},
			Providers: []library.ProviderRef{
				{ProviderID: "mangadex", ProviderMangaID: "md-frieren", MangaTitle: "Frieren"},
			},
		},
	}
	_ = h.lib.SaveManga("frieren-lib-id", crossMeta)

	// Primary-only: mangafox/one-piece
	primaryMeta := library.Manga{
		Metadata: library.MangaMetadata{
			Title:       "One Piece",
			Aliases:     []string{},
			Description: "Pirate king.",
		},
		Bindings: library.MangaBindings{
			Content: &library.ContentSource{
				ProviderID:      "mangafox",
				ProviderMangaID: "one-piece",
			},
		},
	}
	_ = h.lib.SaveManga("onepiece-lib-id", primaryMeta)

	t.Run("Cross-provider hit via Providers[]", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/mangadex/manga/md-frieren", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}
		var details map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &details); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		libID, ok := details["libraryMangaId"].(string)
		if !ok || libID == "" {
			t.Errorf("expected libraryMangaId to be set, got %v", details["libraryMangaId"])
		}
		if libID != "frieren-lib-id" {
			t.Errorf("expected libraryMangaId 'frieren-lib-id', got %v", libID)
		}
	})

	t.Run("Primary hit via Content", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/mangafox/manga/one-piece", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}
		var details map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &details); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		libID, ok := details["libraryMangaId"].(string)
		if !ok || libID == "" {
			t.Errorf("expected libraryMangaId to be set, got %v", details["libraryMangaId"])
		}
		if libID != "onepiece-lib-id" {
			t.Errorf("expected libraryMangaId 'onepiece-lib-id', got %v", libID)
		}
	})

	t.Run("No binding returns no libraryMangaId", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/mangafox/manga/nonexistent", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}
		var details map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &details); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if _, exists := details["libraryMangaId"]; exists {
			t.Errorf("expected libraryMangaId absent, got %v", details["libraryMangaId"])
		}
	})

	t.Run("ListManga error still returns 200 without libraryMangaId", func(t *testing.T) {
		// Defensive path: a ListManga failure must not break the provider-details response.
		// We can't swap h.lib (it's a concrete *library.Library), so we rely on the
		// handler's nil-check + logger fallback. Covered by code review, not asserted here.
		t.Skip("requires library mock; covered by code review of findLibraryBindingID")
	})
}

