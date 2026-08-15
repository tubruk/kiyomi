package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tubruk/kiyomi/internal/library"
)

func TestStandardizedLibraryRoutes(t *testing.T) {
	_, e := setupTestHandler(t)

	// Create Manga
	createBody := map[string]interface{}{
		"id": "test-manga-1",
		"meta": map[string]interface{}{
			"title":       "Test Manga 1",
			"description": "Test Description",
		},
	}
	bodyBytes, _ := json.Marshal(createBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}

	// GET /library/manga/:mangaId
	req = httptest.NewRequest(http.MethodGet, "/api/v1/library/manga/test-manga-1", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}
	var getResp map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &getResp)
	if getResp["id"] != "test-manga-1" {
		t.Errorf("expected manga id 'test-manga-1', got %v", getResp["id"])
	}

	// Save chapter: POST /library/manga/:mangaId/chapters/:chapterId
	chMeta := library.ChapterMeta{
		Title:  "Chapter 1",
		Number: 1,
	}
	chBytes, _ := json.Marshal(chMeta)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/test-manga-1/chapters/ch-100", bytes.NewReader(chBytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	// GET /library/manga/:mangaId/chapters
	req = httptest.NewRequest(http.MethodGet, "/api/v1/library/manga/test-manga-1/chapters", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	// GET /library/manga/:mangaId/chapters/:chapterId
	req = httptest.NewRequest(http.MethodGet, "/api/v1/library/manga/test-manga-1/chapters/ch-100", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	// DELETE /library/manga/:mangaId/chapters/:chapterId
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/library/manga/test-manga-1/chapters/ch-100", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 No Content, got %d: %s", rec.Code, rec.Body.String())
	}

	// DELETE /library/manga/:mangaId
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/library/manga/test-manga-1", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 No Content, got %d: %s", rec.Code, rec.Body.String())
	}
}
