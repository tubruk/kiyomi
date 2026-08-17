package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/tubruk/kiyomi/internal/cache"
)

func TestSystemCacheHandler_NilCache(t *testing.T) {
	e := echo.New()
	h := &Handler{
		imageCache: nil,
	}

	// 1. GET /api/v1/system/cache with nil imageCache
	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/cache", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.getCacheStats(c); err != nil {
		t.Fatalf("getCacheStats returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var statsResp CacheStatsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &statsResp); err != nil {
		t.Fatalf("failed to decode stats response: %v", err)
	}
	if statsResp.SizeBytes != 0 || statsResp.ItemCount != 0 {
		t.Errorf("expected size 0 and count 0, got %d bytes and %d items", statsResp.SizeBytes, statsResp.ItemCount)
	}

	// 2. POST /api/v1/system/cache/clear with nil imageCache
	req = httptest.NewRequest(http.MethodPost, "/api/v1/system/cache/clear", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)

	if err := h.clearCache(c); err != nil {
		t.Fatalf("clearCache returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var clearResp ClearCacheResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &clearResp); err != nil {
		t.Fatalf("failed to decode clear response: %v", err)
	}
	if clearResp.Status != "ok" {
		t.Errorf("expected status 'ok', got %q", clearResp.Status)
	}
}

func TestSystemCacheHandler_Workflow(t *testing.T) {
	h, e := setupTestHandler(t)

	// 1. Initially empty cache stats
	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/cache", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var statsResp CacheStatsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &statsResp); err != nil {
		t.Fatalf("failed to decode stats response: %v", err)
	}
	if statsResp.SizeBytes != 0 || statsResp.ItemCount != 0 {
		t.Fatalf("expected initial size 0 and count 0, got %d bytes and %d items", statsResp.SizeBytes, statsResp.ItemCount)
	}

	// 2. Add dummy images to image cache
	imgData1 := []byte("fake-image-data-page-001")
	imgData2 := []byte("fake-image-data-page-002-longer-payload")

	rc1, _, err := h.ImageCache().GetOrFetch(context.Background(), "https://example.com/ch1/001.jpg", func(ctx context.Context) (io.ReadCloser, cache.Meta, error) {
		return io.NopCloser(bytes.NewReader(imgData1)), cache.Meta{ContentType: "image/jpeg"}, nil
	})
	if err != nil {
		t.Fatalf("failed to seed cache item 1: %v", err)
	}
	_ = rc1.Close()

	rc2, _, err := h.ImageCache().GetOrFetch(context.Background(), "https://example.com/ch1/002.jpg", func(ctx context.Context) (io.ReadCloser, cache.Meta, error) {
		return io.NopCloser(bytes.NewReader(imgData2)), cache.Meta{ContentType: "image/jpeg"}, nil
	})
	if err != nil {
		t.Fatalf("failed to seed cache item 2: %v", err)
	}
	_ = rc2.Close()

	// 3. Query stats after adding images
	req = httptest.NewRequest(http.MethodGet, "/api/v1/system/cache", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var populatedStats CacheStatsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &populatedStats); err != nil {
		t.Fatalf("failed to decode stats response: %v", err)
	}

	expectedSize := int64(len(imgData1) + len(imgData2))
	if populatedStats.ItemCount != 2 {
		t.Errorf("expected 2 items, got %d", populatedStats.ItemCount)
	}
	if populatedStats.SizeBytes != expectedSize {
		t.Errorf("expected size %d, got %d", expectedSize, populatedStats.SizeBytes)
	}

	// 4. Clear cache via POST /api/v1/system/cache/clear
	req = httptest.NewRequest(http.MethodPost, "/api/v1/system/cache/clear", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 on clear, got %d: %s", rec.Code, rec.Body.String())
	}

	var clearResp ClearCacheResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &clearResp); err != nil {
		t.Fatalf("failed to decode clear response: %v", err)
	}
	if clearResp.Status != "ok" {
		t.Errorf("expected status 'ok', got %q", clearResp.Status)
	}

	// 5. Query stats again -> should be 0 items, 0 bytes
	req = httptest.NewRequest(http.MethodGet, "/api/v1/system/cache", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var clearedStats CacheStatsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &clearedStats); err != nil {
		t.Fatalf("failed to decode stats response: %v", err)
	}
	if clearedStats.SizeBytes != 0 || clearedStats.ItemCount != 0 {
		t.Errorf("expected 0 bytes and 0 count after clear, got %d bytes and %d items", clearedStats.SizeBytes, clearedStats.ItemCount)
	}
}
