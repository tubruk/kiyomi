package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/tubruk/kiyomi/internal/config"
	"github.com/tubruk/kiyomi/internal/library"
	"github.com/tubruk/kiyomi/pkg/provider/sdk"
)

func TestProviderClientIsolation(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		LibraryDir: tmpDir,
		CacheDir:   filepath.Join(tmpDir, "cache"),
	}
	lib := library.NewLibrary(tmpDir)

	h := NewHandler(cfg, lib, nil, nil)
	if h.httpClient == nil {
		t.Fatal("expected handler httpClient to be non-nil")
	}
	if h.httpClient.Transport == nil {
		t.Fatal("expected handler httpClient Transport to be non-nil")
	}
}

func TestProxyRefererHeaderSetting(t *testing.T) {
	var receivedReferer string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedReferer = r.Header.Get("Referer")
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("fake-image"))
	}))
	defer ts.Close()

	h, _ := setupTestHandler(t)
	mockFox := &mockProvider{id: "mangafox", name: "MangaFox", baseURL: "https://fanfox.net"}
	mockDex := &mockProvider{id: "mangadex", name: "MangaDex", baseURL: "https://mangadex.org"}
	h.registry.Register(mockFox)
	h.registry.Register(mockDex)
	e := echo.New()

	tests := []struct {
		name            string
		queryReferer    string
		content         *library.ContentSource
		urlPath         string
		expectedReferer string
	}{
		{
			name:            "Query param referer provided",
			queryReferer:    "https://custom-referer.example.com/",
			content:         nil,
			urlPath:         "/img-custom.png",
			expectedReferer: "https://custom-referer.example.com/",
		},
		{
			name:            "Derived from mangafox provider ID",
			queryReferer:    "",
			content:         &library.ContentSource{ProviderID: "mangafox"},
			urlPath:         "/img-mangafox.png",
			expectedReferer: "https://fanfox.net/",
		},
		{
			name:            "Derived from mangadex provider ID",
			queryReferer:    "",
			content:         &library.ContentSource{ProviderID: "mangadex"},
			urlPath:         "/img-mangadex.png",
			expectedReferer: "https://mangadex.org/",
		},
		{
			name:            "Fallback target URL containing fanfox.net",
			queryReferer:    "",
			content:         nil,
			urlPath:         "/fanfox.net/img.png",
			expectedReferer: "https://fanfox.net/",
		},
		{
			name:            "Fallback target URL containing mfcdn.net",
			queryReferer:    "",
			content:         nil,
			urlPath:         "/mfcdn.net/img.png",
			expectedReferer: "https://fanfox.net/",
		},
		{
			name:            "Fallback target URL containing mangafox.me",
			queryReferer:    "",
			content:         nil,
			urlPath:         "/mangafox.me/img.png",
			expectedReferer: "https://fanfox.net/",
		},
		{
			name:            "Fallback target URL containing zjcdn",
			queryReferer:    "",
			content:         nil,
			urlPath:         "/zjcdn/img.png",
			expectedReferer: "https://fanfox.net/",
		},
		{
			name:            "Query param referer overrides provider ID and URL match",
			queryReferer:    "https://override.example.com/",
			content:         &library.ContentSource{ProviderID: "mangadex"},
			urlPath:         "/override/fanfox.net/img.png",
			expectedReferer: "https://override.example.com/",
		},
		{
			name:            "No referer when unspecified and URL doesn't match",
			queryReferer:    "",
			content:         nil,
			urlPath:         "/unknown/img.png",
			expectedReferer: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			receivedReferer = ""
			targetURL := ts.URL + tt.urlPath
			reqURL := "/proxy/image?url=" + targetURL
			if tt.queryReferer != "" {
				reqURL += "&referer=" + tt.queryReferer
			}

			req := httptest.NewRequest(http.MethodGet, reqURL, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := h.streamRemoteImage(c, targetURL, tt.content)
			if err != nil {
				t.Fatalf("streamRemoteImage failed: %v", err)
			}

			if receivedReferer != tt.expectedReferer {
				t.Errorf("expected Referer %q, got %q", tt.expectedReferer, receivedReferer)
			}
		})
	}
}

func TestProxyHandlerRoutes(t *testing.T) {
	h, e := setupTestHandler(t)

	mockP := &mockProvider{id: "testprov", name: "Test Provider"}
	h.registry.Register(mockP)

	// Save chapter to library first for proxyPageImage test
	_ = h.lib.SaveManga("manga1", &library.MangaMeta{Title: "Manga 1"})
	_ = h.lib.SaveChapter("manga1", library.LocalProviderID, "ch1", &library.ChapterMeta{Title: "Chapter 1"})

	t.Run("GET /providers/:providerId/manga/:remoteId/chapters/:chapterId/pages", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/testprov/manga/mock-1/chapters/ch-1/pages", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("GET /providers/:providerId/manga/:remoteId/chapters/:chapterId/pages with tilde chapter ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/testprov/manga/mock-1/chapters/v01~c001~1.html/pages", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("GET /chapters/:chapterId/pages alias", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/chapters/ch-1/pages?providerId=testprov", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("GET /library/manga/:mangaId/chapters/:chapterId/pages/:pageIndex", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "image/jpeg")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("image-data"))
		}))
		defer ts.Close()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/library/manga/manga1/chapters/ch1/pages/1?url="+ts.URL, nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}

func TestGetChapterPages_ConcurrentFallbackSearch(t *testing.T) {
	h, e := setupTestHandler(t)

	mockP := &mockMultiChapterProvider{
		mockProvider: mockProvider{id: "fallbackprov", name: "Fallback Provider"},
	}
	h.registry.Register(mockP)

	// Create 20 mangas in the library
	const numManga = 20
	targetMangaID := "manga-14"
	targetChapterID := "target-ch-123"

	for i := 1; i <= numManga; i++ {
		mID := fmt.Sprintf("manga-%d", i)
		_ = h.lib.SaveManga(mID, &library.MangaMeta{Title: fmt.Sprintf("Manga %d", i)})
	}

	// Save target chapter in targetMangaID with ProviderID set
	err := h.lib.SaveChapter(targetMangaID, "fallbackprov", targetChapterID, &library.ChapterMeta{
		Title: "Target Chapter",
		Content: &library.ContentSource{
			ProviderID: "fallbackprov",
			ChapterRef: targetChapterID,
		},
	})
	if err != nil {
		t.Fatalf("failed to save chapter: %v", err)
	}

	// Request GET /chapters/:chapterId/pages without providerId or mangaId
	req := httptest.NewRequest(http.MethodGet, "/api/v1/chapters/"+targetChapterID+"/pages", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Pages []map[string]interface{} `json:"pages"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Pages) != 1 {
		t.Fatalf("expected 1 page from mock provider, got %d", len(resp.Pages))
	}
}

func TestProxyImageCaching_HitAndMiss(t *testing.T) {
	var upstreamHits int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&upstreamHits, 1)
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Length", "12")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("cached-image"))
	}))
	defer ts.Close()

	h, e := setupTestHandler(t)

	targetURL := ts.URL + "/manga/page1.png"
	reqURL := "/api/v1/proxy/image?url=" + targetURL

	// First request: Cache Miss -> Upstream hit
	req1 := httptest.NewRequest(http.MethodGet, reqURL, nil)
	rec1 := httptest.NewRecorder()
	e.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Fatalf("first request expected 200 OK, got %d: %s", rec1.Code, rec1.Body.String())
	}
	if rec1.Body.String() != "cached-image" {
		t.Errorf("first request expected body 'cached-image', got %q", rec1.Body.String())
	}
	if ct := rec1.Header().Get("Content-Type"); ct != "image/png" {
		t.Errorf("first request expected Content-Type 'image/png', got %q", ct)
	}
	if cc := rec1.Header().Get("Cache-Control"); cc != "public, max-age=86400" {
		t.Errorf("first request expected Cache-Control 'public, max-age=86400', got %q", cc)
	}
	if hits := atomic.LoadInt32(&upstreamHits); hits != 1 {
		t.Fatalf("expected 1 upstream hit after first request, got %d", hits)
	}

	// Verify image is in cache
	if h.ImageCache() == nil {
		t.Fatal("expected imageCache to be non-nil")
	}
	if _, _, ok := h.ImageCache().GetPath(targetURL); !ok {
		t.Fatal("expected targetURL to be present in disk cache")
	}

	// Second request: Cache Hit -> Upstream NOT hit
	req2 := httptest.NewRequest(http.MethodGet, reqURL, nil)
	rec2 := httptest.NewRecorder()
	e.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("second request expected 200 OK, got %d: %s", rec2.Code, rec2.Body.String())
	}
	if rec2.Body.String() != "cached-image" {
		t.Errorf("second request expected body 'cached-image', got %q", rec2.Body.String())
	}
	if ct := rec2.Header().Get("Content-Type"); ct != "image/png" {
		t.Errorf("second request expected Content-Type 'image/png', got %q", ct)
	}
	if cc := rec2.Header().Get("Cache-Control"); cc != "public, max-age=86400" {
		t.Errorf("second request expected Cache-Control 'public, max-age=86400', got %q", cc)
	}
	if hits := atomic.LoadInt32(&upstreamHits); hits != 1 {
		t.Fatalf("expected still 1 upstream hit on cache hit, got %d", hits)
	}
}

func TestProxyImageCaching_FallbackWhenCacheNil(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("fallback-image"))
	}))
	defer ts.Close()

	tmpDir := t.TempDir()
	// CacheDir is empty, so imageCache is nil
	cfg := &config.Config{LibraryDir: tmpDir, AllowPrivateNetworks: true}
	lib := library.NewLibrary(tmpDir)
	h := NewHandler(cfg, lib, nil, nil)
	e := echo.New()
	h.RegisterRoutes(e)

	if h.ImageCache() != nil {
		t.Fatal("expected imageCache to be nil when CacheDir is empty")
	}

	targetURL := ts.URL + "/page.jpg"
	req := httptest.NewRequest(http.MethodGet, "/api/v1/proxy/image?url="+targetURL, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != "fallback-image" {
		t.Errorf("expected body 'fallback-image', got %q", rec.Body.String())
	}
}

func TestProxyImageCaching_UpstreamError(t *testing.T) {
	var upstreamHits int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&upstreamHits, 1)
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found upstream"))
	}))
	defer ts.Close()

	h, e := setupTestHandler(t)

	targetURL := ts.URL + "/missing.jpg"
	req := httptest.NewRequest(http.MethodGet, "/api/v1/proxy/image?url="+targetURL, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 Bad Gateway, got %d: %s", rec.Code, rec.Body.String())
	}

	// Ensure error was not cached
	if _, _, ok := h.ImageCache().GetPath(targetURL); ok {
		t.Fatal("failed upstream request should not be cached")
	}

	// Second request should call upstream again
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/proxy/image?url="+targetURL, nil)
	rec2 := httptest.NewRecorder()
	e.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 Bad Gateway on second request, got %d", rec2.Code)
	}
	if hits := atomic.LoadInt32(&upstreamHits); hits != 2 {
		t.Fatalf("expected 2 upstream hits because error was not cached, got %d", hits)
	}
}

type mockCountingPagesProvider struct {
	mockProvider
	fetchPagesCalls atomic.Int32
}

func (m *mockCountingPagesProvider) FetchPages(ctx context.Context, mangaRef, chapterRef string) ([]sdk.Page, error) {
	m.fetchPagesCalls.Add(1)
	return []sdk.Page{
		{Index: 1, URL: "https://example.com/ch-lazy/p1.jpg"},
		{Index: 2, URL: "https://example.com/ch-lazy/p2.jpg"},
	}, nil
}

func TestGetChapterPages_LazyLoadingAndRefresh(t *testing.T) {
	h, e := setupTestHandler(t)

	mockP := &mockCountingPagesProvider{
		mockProvider: mockProvider{id: "countingprov", name: "Counting Provider"},
	}
	h.registry.Register(mockP)

	mangaID := "manga-lazy"
	chapterID := "ch-lazy"

	_ = h.lib.SaveManga(mangaID, &library.MangaMeta{
		Title: "Lazy Manga",
		Content: &library.ContentSource{
			ProviderID:      "countingprov",
			ProviderMangaID: mangaID,
		},
	})
	_ = h.lib.SaveChapter(mangaID, "countingprov", chapterID, &library.ChapterMeta{
		Title: "Lazy Chapter",
		Content: &library.ContentSource{
			ProviderID: "countingprov",
			ChapterRef: chapterID,
		},
	})

	type pageResponse struct {
		Pages []struct {
			Index  int    `json:"index"`
			URL    string `json:"url"`
			Source string `json:"source"`
		} `json:"pages"`
	}

	// 1. First request: Should fetch from provider and save to library
	req1 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/chapters/%s/pages?providerId=countingprov&mangaId=%s", chapterID, mangaID), nil)
	rec1 := httptest.NewRecorder()
	e.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Fatalf("call 1 expected 200 OK, got %d: %s", rec1.Code, rec1.Body.String())
	}

	var resp1 pageResponse
	if err := json.Unmarshal(rec1.Body.Bytes(), &resp1); err != nil {
		t.Fatalf("call 1 json unmarshal error: %v", err)
	}
	if len(resp1.Pages) != 2 {
		t.Fatalf("call 1 expected 2 pages, got %d", len(resp1.Pages))
	}
	if resp1.Pages[0].Source != "provider" {
		t.Errorf("call 1 expected source 'provider', got %q", resp1.Pages[0].Source)
	}
	if calls := mockP.fetchPagesCalls.Load(); calls != 1 {
		t.Fatalf("call 1 expected 1 provider call, got %d", calls)
	}

	// Verify library has saved pages
	saved, err := h.lib.GetChapterPages(mangaID, "countingprov", chapterID)
	if err != nil {
		t.Fatalf("expected chapter pages saved to library: %v", err)
	}
	if len(saved) != 2 {
		t.Fatalf("expected 2 pages saved in library, got %d", len(saved))
	}

	// 2. Second request: Should return from library without calling provider mock
	req2 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/chapters/%s/pages?providerId=countingprov&mangaId=%s", chapterID, mangaID), nil)
	rec2 := httptest.NewRecorder()
	e.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("call 2 expected 200 OK, got %d: %s", rec2.Code, rec2.Body.String())
	}

	var resp2 pageResponse
	if err := json.Unmarshal(rec2.Body.Bytes(), &resp2); err != nil {
		t.Fatalf("call 2 json unmarshal error: %v", err)
	}
	if len(resp2.Pages) != 2 {
		t.Fatalf("call 2 expected 2 pages, got %d", len(resp2.Pages))
	}
	if resp2.Pages[0].Source != "library" {
		t.Errorf("call 2 expected source 'library', got %q", resp2.Pages[0].Source)
	}
	if calls := mockP.fetchPagesCalls.Load(); calls != 1 {
		t.Fatalf("call 2 expected provider calls to stay 1, got %d", calls)
	}

	// 3. Third request with ?refresh=true: Should bypass library and fetch fresh from provider
	req3 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/chapters/%s/pages?providerId=countingprov&mangaId=%s&refresh=true", chapterID, mangaID), nil)
	rec3 := httptest.NewRecorder()
	e.ServeHTTP(rec3, req3)

	if rec3.Code != http.StatusOK {
		t.Fatalf("call 3 expected 200 OK, got %d: %s", rec3.Code, rec3.Body.String())
	}

	var resp3 pageResponse
	if err := json.Unmarshal(rec3.Body.Bytes(), &resp3); err != nil {
		t.Fatalf("call 3 json unmarshal error: %v", err)
	}
	if len(resp3.Pages) != 2 {
		t.Fatalf("call 3 expected 2 pages, got %d", len(resp3.Pages))
	}
	if resp3.Pages[0].Source != "provider" {
		t.Errorf("call 3 expected source 'provider', got %q", resp3.Pages[0].Source)
	}
	if calls := mockP.fetchPagesCalls.Load(); calls != 2 {
		t.Fatalf("call 3 expected provider calls to increase to 2, got %d", calls)
	}

	// 4. Provider path route: /api/v1/providers/:providerId/manga/:remoteId/chapters/:chapterId/pages
	chapterID2 := "ch-lazy-2"
	mangaID2 := "manga-lazy-2"
	providerPath := fmt.Sprintf("/api/v1/providers/countingprov/manga/%s/chapters/%s/pages", mangaID2, chapterID2)

	// Call A: 1st hit -> provider
	reqA := httptest.NewRequest(http.MethodGet, providerPath, nil)
	recA := httptest.NewRecorder()
	e.ServeHTTP(recA, reqA)
	if recA.Code != http.StatusOK {
		t.Fatalf("call A expected 200 OK, got %d: %s", recA.Code, recA.Body.String())
	}
	var respA pageResponse
	_ = json.Unmarshal(recA.Body.Bytes(), &respA)
	if respA.Pages[0].Source != "provider" {
		t.Errorf("call A expected source 'provider', got %q", respA.Pages[0].Source)
	}

	// Call B: 2nd hit -> library
	reqB := httptest.NewRequest(http.MethodGet, providerPath, nil)
	recB := httptest.NewRecorder()
	e.ServeHTTP(recB, reqB)
	if recB.Code != http.StatusOK {
		t.Fatalf("call B expected 200 OK, got %d: %s", recB.Code, recB.Body.String())
	}
	var respB pageResponse
	_ = json.Unmarshal(recB.Body.Bytes(), &respB)
	if respB.Pages[0].Source != "library" {
		t.Errorf("call B expected source 'library', got %q", respB.Pages[0].Source)
	}
}

func TestProxyPageImage_DiskCacheServing(t *testing.T) {
	h, e := setupTestHandler(t)

	mangaID := "proxy-manga"
	chapterID := "proxy-ch"
	providerID := "local"

	_ = h.lib.SaveManga(mangaID, &library.MangaMeta{Title: "Proxy Manga"})
	_ = h.lib.SaveChapter(mangaID, providerID, chapterID, &library.ChapterMeta{
		Title: "Proxy Chapter",
	})

	// Create page 1 on disk
	chDir := h.lib.ProviderChapterDir(mangaID, providerID, chapterID)
	_ = os.MkdirAll(chDir, 0o755)
	page1Content := []byte("local-image-data-page-1")
	_ = os.WriteFile(filepath.Join(chDir, "1.jpg"), page1Content, 0o644)

	// 1. Request page 1: should be served from disk, even without url param or upstream server
	req1 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/library/manga/%s/chapters/%s/pages/1?provider_id=%s", mangaID, chapterID, providerID), nil)
	rec1 := httptest.NewRecorder()
	e.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for on-disk page 1, got %d: %s", rec1.Code, rec1.Body.String())
	}
	if rec1.Body.String() != string(page1Content) {
		t.Errorf("expected body %q, got %q", string(page1Content), rec1.Body.String())
	}
	if cc := rec1.Header().Get("Cache-Control"); cc != "public, max-age=86400" {
		t.Errorf("expected Cache-Control 'public, max-age=86400', got %q", cc)
	}

	// 2. Request page 2 (not on disk): should fail without url param
	req2 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/library/manga/%s/chapters/%s/pages/2?provider_id=%s", mangaID, chapterID, providerID), nil)
	rec2 := httptest.NewRecorder()
	e.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for page 2 without url, got %d", rec2.Code)
	}

	// 3. Request page 2 with upstream url
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("upstream-page-2"))
	}))
	defer ts.Close()

	req3 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/library/manga/%s/chapters/%s/pages/2?provider_id=%s&url=%s", mangaID, chapterID, providerID, ts.URL), nil)
	rec3 := httptest.NewRecorder()
	e.ServeHTTP(rec3, req3)

	if rec3.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for page 2 from upstream, got %d: %s", rec3.Code, rec3.Body.String())
	}
	if rec3.Body.String() != "upstream-page-2" {
		t.Errorf("expected body 'upstream-page-2', got %q", rec3.Body.String())
	}

	// 4. Request page 3 (not on disk) without url in query, but with pages.json saved
	_ = h.lib.SaveChapterPages(mangaID, providerID, chapterID, []library.PageItem{
		{Index: 3, URL: ts.URL},
	})
	req4 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/library/manga/%s/chapters/%s/pages/3?provider_id=%s", mangaID, chapterID, providerID), nil)
	rec4 := httptest.NewRecorder()
	e.ServeHTTP(rec4, req4)

	if rec4.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for page 3 with url resolved from pages.json, got %d: %s", rec4.Code, rec4.Body.String())
	}
	if rec4.Body.String() != "upstream-page-2" {
		t.Errorf("expected body 'upstream-page-2', got %q", rec4.Body.String())
	}
}

func TestProxyHandler_DataURI_SVG(t *testing.T) {
	_, e := setupTestHandler(t)
	svgURI := "data:image/svg+xml,<svg xmlns='http://www.w3.org/2000/svg'><circle r='10'/></svg>"

	req := httptest.NewRequest(http.MethodGet, "/api/v1/proxy/image?url="+url.QueryEscape(svgURI), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for SVG data URI, got %d: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/svg+xml" {
		t.Errorf("expected Content-Type 'image/svg+xml', got %q", ct)
	}
}

func TestProxyHandler_SSRFGuard_Rejection(t *testing.T) {
	tmpDir := t.TempDir()
	// AllowPrivateNetworks is false
	cfg := &config.Config{
		LibraryDir:           tmpDir,
		CacheDir:             filepath.Join(tmpDir, "cache"),
		AllowPrivateNetworks: false,
	}
	lib := library.NewLibrary(tmpDir)
	h := NewHandler(cfg, lib, nil, nil)
	e := echo.New()
	h.RegisterRoutes(e)

	blockedURLs := []string{
		"http://127.0.0.1:8080/test.png",
		"http://192.168.1.1/test.png",
		"http://10.0.0.1/test.png",
		"http://localhost/test.png",
	}

	for _, u := range blockedURLs {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/proxy/image?url="+u, nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request for blocked SSRF URL %s, got %d", u, rec.Code)
		}
	}
}

func TestProxyPageImage_RetainProviderID(t *testing.T) {
	var receivedReferer string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedReferer = r.Header.Get("Referer")
		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("image-data"))
	}))
	defer ts.Close()

	h, e := setupTestHandler(t)
	mockFox := &mockProvider{id: "mangafox", name: "MangaFox", baseURL: "https://fanfox.net"}
	h.registry.Register(mockFox)

	mangaID := "retain-prov-manga"
	chapterID := "retain-prov-ch"
	providerID := "mangafox"

	_ = h.lib.SaveManga(mangaID, &library.MangaMeta{Title: "Retain Prov Manga"})
	// Chapter exists, but chMeta.Content is nil
	_ = h.lib.SaveChapter(mangaID, providerID, chapterID, &library.ChapterMeta{
		Title:   "Chapter 1",
		Content: nil,
	})

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/library/manga/%s/chapters/%s/pages/1?provider_id=%s&url=%s", mangaID, chapterID, providerID, ts.URL), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}
	if receivedReferer != "https://fanfox.net/" {
		t.Errorf("expected Referer from retained providerID 'https://fanfox.net/', got %q", receivedReferer)
	}
}

type mockCapturePagesProvider struct {
	mockProvider
	lastMangaRef   string
	lastChapterRef string
}

func (m *mockCapturePagesProvider) FetchPages(ctx context.Context, mangaRef, chapterRef string) ([]sdk.Page, error) {
	m.lastMangaRef = mangaRef
	m.lastChapterRef = chapterRef
	return []sdk.Page{
		{Index: 1, URL: "https://example.com/p1.jpg"},
	}, nil
}

func TestGetChapterPages_ResolvesRemoteIDs(t *testing.T) {
	h, e := setupTestHandler(t)

	mockP := &mockCapturePagesProvider{
		mockProvider: mockProvider{id: "remoteprov", name: "Remote Provider"},
	}
	h.registry.Register(mockP)

	// Case 1: Primary provider content in mangaMeta.Content
	localMangaID := "local-manga-1"
	localChapterID := "local-ch-1"
	remoteMangaID := "remote-manga-100"
	remoteChapterRef := "remote-ch-200"

	_ = h.lib.SaveManga(localMangaID, &library.MangaMeta{
		Title: "Local Manga 1",
		Content: &library.ContentSource{
			ProviderID:      "remoteprov",
			ProviderMangaID: remoteMangaID,
		},
	})
	_ = h.lib.SaveChapter(localMangaID, "remoteprov", localChapterID, &library.ChapterMeta{
		Title: "Local Chapter 1",
		Content: &library.ContentSource{
			ProviderID: "remoteprov",
			ChapterRef: remoteChapterRef,
		},
	})

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/chapters/%s/pages?providerId=remoteprov&mangaId=%s", localChapterID, localMangaID), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}
	if mockP.lastMangaRef != remoteMangaID {
		t.Errorf("expected FetchPages mangaRef %q, got %q", remoteMangaID, mockP.lastMangaRef)
	}
	if mockP.lastChapterRef != remoteChapterRef {
		t.Errorf("expected FetchPages chapterRef %q, got %q", remoteChapterRef, mockP.lastChapterRef)
	}

	// Verify pages saved under local manga and chapter IDs
	saved, err := h.lib.GetChapterPages(localMangaID, "remoteprov", localChapterID)
	if err != nil || len(saved) == 0 {
		t.Fatalf("expected chapter pages saved to local library IDs: %v", err)
	}

	// Case 2: Multi-provider binding in mangaMeta.Providers
	localMangaID2 := "local-manga-2"
	localChapterID2 := "local-ch-2"
	remoteMangaID2 := "remote-manga-300"
	remoteChapterRef2 := "remote-ch-400"

	_ = h.lib.SaveManga(localMangaID2, &library.MangaMeta{
		Title: "Local Manga 2",
		Providers: []library.ProviderRef{
			{
				ProviderID:      "remoteprov",
				ProviderMangaID: remoteMangaID2,
			},
		},
	})
	_ = h.lib.SaveChapter(localMangaID2, "remoteprov", localChapterID2, &library.ChapterMeta{
		Title: "Local Chapter 2",
		Content: &library.ContentSource{
			ProviderID: "remoteprov",
			ChapterRef: remoteChapterRef2,
		},
	})

	req2 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/chapters/%s/pages?providerId=remoteprov&mangaId=%s", localChapterID2, localMangaID2), nil)
	rec2 := httptest.NewRecorder()
	e.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for case 2, got %d: %s", rec2.Code, rec2.Body.String())
	}
	if mockP.lastMangaRef != remoteMangaID2 {
		t.Errorf("expected FetchPages mangaRef %q, got %q", remoteMangaID2, mockP.lastMangaRef)
	}
	if mockP.lastChapterRef != remoteChapterRef2 {
		t.Errorf("expected FetchPages chapterRef %q, got %q", remoteChapterRef2, mockP.lastChapterRef)
	}
}
