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
	id   string
	name string
}

func (m *mockProvider) ID() string                    { return m.id }
func (m *mockProvider) Name() string                  { return m.name }
func (m *mockProvider) Icon() string                  { return "https://example.com/icon.png" }
func (m *mockProvider) Capabilities() []string        { return []string{"metadata", "content"} }
func (m *mockProvider) ConfigKeys() []sdk.ConfigKeySpec { return nil }
func (m *mockProvider) RequiresAuth() bool           { return false }
func (m *mockProvider) State() sdk.ProviderState      { return sdk.StateActive }

func (m *mockProvider) GetConfig() sdk.ProviderConfig {
	return sdk.ProviderConfig{
		ID:       m.id,
		Name:     m.name,
		BaseURL:  "https://example.com",
		Language: "en",
	}
}

func (m *mockProvider) Search(ctx context.Context, query string, opts sdk.SearchOptions) ([]sdk.SearchResult, error) {
	return []sdk.SearchResult{
		{
			RemoteID:     "mock-1",
			Title:        "Mock Manga 1",
			CoverURL:     "https://example.com/cover1.jpg",
			Availability: sdk.AvailabilityAvailable,
		},
	}, nil
}

func (m *mockProvider) Details(ctx context.Context, remoteID string) (sdk.MangaMetadata, error) {
	return sdk.MangaMetadata{
		RemoteID:     remoteID,
		Title:        "Mock Manga Details",
		Synopsis:     "Mock synopsis",
		Author:       "Mock Author",
		Artist:       "Mock Artist",
		Genres:       []string{"Action", "Fantasy"},
		ReadingMode:  sdk.ReadingModeLongstrip,
		Availability: sdk.AvailabilityAvailable,
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

	// Should contain mangadex, mangafox, and mockprov (sorted by id)
	if len(list) < 3 {
		t.Errorf("expected at least 3 content providers, got %d", len(list))
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
		ID   string            `json:"id"`
		Meta library.MangaMeta `json:"meta"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &importResp); err != nil {
		t.Fatalf("failed to decode import response: %v", err)
	}
	if importResp.Meta.Content == nil || importResp.Meta.Content.ReadingMode != "longstrip" {
		t.Errorf("expected import response Content.ReadingMode 'longstrip', got %+v", importResp.Meta.Content)
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

	t.Run("502 provider failure logs handler_error in EchoErrorLogger", func(t *testing.T) {
		buf.Reset()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/failprovider/popular", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadGateway {
			t.Fatalf("expected 502 Bad Gateway, got %d", rec.Code)
		}

		out := buf.String()
		if !strings.Contains(out, "HTTP request error") {
			t.Errorf("expected 'HTTP request error' in log, got: %s", out)
		}
		if !strings.Contains(out, "status=502") {
			t.Errorf("expected status=502 in log, got: %s", out)
		}
		if !strings.Contains(out, "error=") {
			t.Errorf("expected error attribute in log, got: %s", out)
		}
	})
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


