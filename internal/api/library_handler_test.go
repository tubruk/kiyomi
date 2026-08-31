package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tubruk/kiyomi/internal/library"
	"github.com/tubruk/kiyomi/internal/queue"
	"github.com/tubruk/kiyomi/internal/queue/inmemory"
	"github.com/tubruk/kiyomi/pkg/provider/sdk"
)

func TestLibraryHandler_PatchUserMetadata(t *testing.T) {
	_, e := setupTestHandler(t)

	// Create initial manga
	createBody := map[string]interface{}{
		"id": "patch-manga-1",
		"meta": map[string]interface{}{
			"title":         "Patch Test Manga",
			"user_status":   library.UserStatusUnread,
			"user_favorite": false,
			"user_rating":   0,
			"user_notes":    "",
		},
	}
	createBytes, _ := json.Marshal(createBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga", bytes.NewReader(createBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}

	// 1. Patch user_favorite -> true
	patch1 := map[string]interface{}{
		"user_favorite": true,
	}
	p1Bytes, _ := json.Marshal(patch1)
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/library/manga/patch-manga-1", bytes.NewReader(p1Bytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp1 struct {
		ID   string            `json:"id"`
		Meta library.MangaMeta `json:"meta"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp1); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !resp1.Meta.UserFavorite {
		t.Errorf("expected UserFavorite to be true, got %v", resp1.Meta.UserFavorite)
	}
	if resp1.Meta.UserStatus != library.UserStatusUnread {
		t.Errorf("expected UserStatus %q, got %q", library.UserStatusUnread, resp1.Meta.UserStatus)
	}

	// 2. Patch user_status -> reading
	patch2 := map[string]interface{}{
		"user_status": library.UserStatusReading,
	}
	p2Bytes, _ := json.Marshal(patch2)
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/library/manga/patch-manga-1", bytes.NewReader(p2Bytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp2 struct {
		ID   string            `json:"id"`
		Meta library.MangaMeta `json:"meta"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp2)
	if resp2.Meta.UserStatus != library.UserStatusReading {
		t.Errorf("expected UserStatus %q, got %q", library.UserStatusReading, resp2.Meta.UserStatus)
	}
	if !resp2.Meta.UserFavorite {
		t.Errorf("expected UserFavorite to remain true, got %v", resp2.Meta.UserFavorite)
	}

	// 3. Patch user_rating and user_notes
	patch3 := map[string]interface{}{
		"user_rating": 8.5,
		"user_notes":  "Loved chapter 10!",
	}
	p3Bytes, _ := json.Marshal(patch3)
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/library/manga/patch-manga-1", bytes.NewReader(p3Bytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp3 struct {
		ID   string            `json:"id"`
		Meta library.MangaMeta `json:"meta"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp3)
	if resp3.Meta.UserRating != 8.5 {
		t.Errorf("expected UserRating 8.5, got %v", resp3.Meta.UserRating)
	}
	if resp3.Meta.UserNotes != "Loved chapter 10!" {
		t.Errorf("expected UserNotes 'Loved chapter 10!', got %q", resp3.Meta.UserNotes)
	}
	if resp3.Meta.UserStatus != library.UserStatusReading {
		t.Errorf("expected UserStatus to remain reading, got %q", resp3.Meta.UserStatus)
	}
	if !resp3.Meta.UserFavorite {
		t.Errorf("expected UserFavorite to remain true, got %v", resp3.Meta.UserFavorite)
	}

	// 4. Patch with invalid user_status -> should fail with 400 Bad Request
	patchInvalid := map[string]interface{}{
		"user_status": "non_existent_status",
	}
	pInvBytes, _ := json.Marshal(patchInvalid)
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/library/manga/patch-manga-1", bytes.NewReader(pInvBytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for invalid user_status, got %d: %s", rec.Code, rec.Body.String())
	}

	// 5. Patch non-existent manga -> should return 404
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/library/manga/non-existent-manga", bytes.NewReader(p1Bytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestLibraryHandler_UserStatusValidation(t *testing.T) {
	_, e := setupTestHandler(t)

	// Create with invalid status
	createInvalid := map[string]interface{}{
		"id": "invalid-status-manga",
		"meta": map[string]interface{}{
			"title":       "Invalid Status Manga",
			"user_status": "invalid_status_enum",
		},
	}
	ciBytes, _ := json.Marshal(createInvalid)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga", bytes.NewReader(ciBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request on create with invalid user_status, got %d: %s", rec.Code, rec.Body.String())
	}

	// Create with valid status
	createValid := map[string]interface{}{
		"id": "valid-status-manga",
		"meta": map[string]interface{}{
			"title":       "Valid Status Manga",
			"user_status": library.UserStatusPlanToRead,
		},
	}
	cvBytes, _ := json.Marshal(createValid)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/library/manga", bytes.NewReader(cvBytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created on create with valid user_status, got %d: %s", rec.Code, rec.Body.String())
	}

	// Update with invalid status
	updateInvalid := map[string]interface{}{
		"title":       "Valid Status Manga Updated",
		"user_status": "corrupted_status",
	}
	uiBytes, _ := json.Marshal(updateInvalid)
	req = httptest.NewRequest(http.MethodPut, "/api/v1/library/manga/valid-status-manga", bytes.NewReader(uiBytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request on update with invalid user_status, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestLibraryHandler_RefreshLibraryManga(t *testing.T) {
	h, e := setupTestHandler(t)

	mockP := &mockProvider{id: "mockprov", name: "Mock Provider"}
	h.registry.Register(mockP)

	// 1. Sync non-existent manga -> 404 Not Found
	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/non-existent/refresh", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found for non-existent manga, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. Sync manga without connected provider -> 400 Bad Request
	noProvManga := map[string]interface{}{
		"id": "no-provider-manga",
		"meta": map[string]interface{}{
			"title": "No Provider Manga",
		},
	}
	npBytes, _ := json.Marshal(noProvManga)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/library/manga", bytes.NewReader(npBytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/no-provider-manga/refresh", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for manga with no provider, got %d: %s", rec.Code, rec.Body.String())
	}

	// 3. Sync valid manga with mock provider -> 200 OK with added chapters
	validManga := map[string]interface{}{
		"id": "refresh-manga-1",
		"meta": map[string]interface{}{
			"title": "Refresh Manga 1",
			"content": map[string]interface{}{
				"provider_id":       "mockprov",
				"provider_manga_id": "mock-1",
			},
		},
	}
	vmBytes, _ := json.Marshal(validManga)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/library/manga", bytes.NewReader(vmBytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/refresh-manga-1/refresh", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on sync, got %d: %s", rec.Code, rec.Body.String())
	}

	var refreshResp struct {
		Added      int    `json:"added"`
		Updated    int    `json:"updated"`
		ProviderID string `json:"provider_id"`
		MangaID    string `json:"manga_id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &refreshResp); err != nil {
		t.Fatalf("failed to decode sync response: %v", err)
	}

	if refreshResp.Added != 1 {
		t.Errorf("expected added=1, got %d", refreshResp.Added)
	}
	if refreshResp.ProviderID != "mockprov" {
		t.Errorf("expected providerId='mockprov', got %s", refreshResp.ProviderID)
	}
	if refreshResp.MangaID != "refresh-manga-1" {
		t.Errorf("expected mangaId='refresh-manga-1', got %s", refreshResp.MangaID)
	}

	// Verify chapter exists in library
	chapters, err := h.lib.ListChapters("refresh-manga-1")
	if err != nil {
		t.Fatalf("failed to list chapters: %v", err)
	}
	if len(chapters) != 1 {
		t.Fatalf("expected 1 chapter, got %d", len(chapters))
	}
	if chapters[0].ID != "ch-1" || chapters[0].Meta.Title != "Chapter 1" {
		t.Errorf("unexpected chapter data: %+v", chapters[0])
	}
	if chapters[0].Meta.SourceOrder != 1 {
		t.Errorf("expected chapter sourceOrder 1, got %d", chapters[0].Meta.SourceOrder)
	}
	if chapters[0].Meta.UploadDate.IsZero() {
		t.Errorf("expected chapter uploadDate to not be zero")
	}

	// Verify GET /library/manga/refresh-manga-1/chapters returns sourceOrder and uploadDate
	req = httptest.NewRequest(http.MethodGet, "/api/v1/library/manga/refresh-manga-1/chapters", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on get chapters, got %d: %s", rec.Code, rec.Body.String())
	}
	var listResp struct {
		Chapters []map[string]interface{} `json:"chapters"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("failed to decode chapters response: %v", err)
	}
	if len(listResp.Chapters) != 1 {
		t.Fatalf("expected 1 chapter in list response, got %d", len(listResp.Chapters))
	}
	if listResp.Chapters[0]["sourceOrder"] != float64(1) {
		t.Errorf("expected sourceOrder 1, got %v", listResp.Chapters[0]["sourceOrder"])
	}
	if listResp.Chapters[0]["uploadDate"] != "2023-11-14T22:13:20Z" {
		t.Errorf("expected uploadDate '2023-11-14T22:13:20Z', got %v", listResp.Chapters[0]["uploadDate"])
	}

	// 4. Syncing again skips already existing chapters -> added=0
	req = httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/refresh-manga-1/refresh", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on second sync, got %d: %s", rec.Code, rec.Body.String())
	}

	var refreshResp2 struct {
		Added int `json:"added"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &refreshResp2)
	if refreshResp2.Added != 0 {
		t.Errorf("expected added=0 on second sync, got %d", refreshResp2.Added)
	}
}

func TestLibraryHandler_PatchChapterProgress(t *testing.T) {
	h, e := setupTestHandler(t)

	// Create a manga
	mangaID := "progress-manga-1"
	chapterID := "progress-ch-1"
	mangaMeta := &library.MangaMeta{
		Title: "Progress Manga 1",
	}
	if err := h.lib.SaveManga(mangaID, mangaMeta); err != nil {
		t.Fatalf("failed to save manga: %v", err)
	}

	// Create a chapter
	chMeta := &library.ChapterMeta{
		Title:  "Chapter 1",
		Number: 1.0,
	}
	if err := h.lib.SaveChapter(mangaID, library.LocalProviderID, chapterID, chMeta); err != nil {
		t.Fatalf("failed to save chapter: %v", err)
	}

	// 1. Full update: is_read=true, last_read_page=5
	body1 := map[string]interface{}{
		"is_read":        true,
		"last_read_page": 5,
	}
	b1Bytes, _ := json.Marshal(body1)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/library/manga/"+mangaID+"/providers/local/chapters/"+chapterID+"/progress", bytes.NewReader(b1Bytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp1 library.ChapterInfo
	if err := json.Unmarshal(rec.Body.Bytes(), &resp1); err != nil {
		t.Fatalf("failed to unmarshal ChapterInfo response: %v", err)
	}
	if resp1.ID != chapterID {
		t.Errorf("expected ID %q, got %q", chapterID, resp1.ID)
	}
	if resp1.MangaID != mangaID {
		t.Errorf("expected MangaID %q, got %q", mangaID, resp1.MangaID)
	}
	if !resp1.Meta.IsRead {
		t.Errorf("expected IsRead to be true, got %v", resp1.Meta.IsRead)
	}
	if resp1.Meta.LastReadPage != 5 {
		t.Errorf("expected LastReadPage to be 5, got %d", resp1.Meta.LastReadPage)
	}
	if resp1.Meta.LastReadAt.IsZero() {
		t.Errorf("expected LastReadAt to be non-zero")
	}

	// Verify manga metadata was updated
	savedManga, err := h.lib.GetManga(mangaID)
	if err != nil {
		t.Fatalf("failed to get manga: %v", err)
	}
	if savedManga.LastReadChapterID != chapterID {
		t.Errorf("expected manga LastReadChapterID %q, got %q", chapterID, savedManga.LastReadChapterID)
	}
	if savedManga.LastReadAt.IsZero() {
		t.Errorf("expected manga LastReadAt to be non-zero")
	}

	// 2. Partial update: only last_read_page=12 (is_read should remain true)
	body2 := map[string]interface{}{
		"last_read_page": 12,
	}
	b2Bytes, _ := json.Marshal(body2)
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/library/manga/"+mangaID+"/providers/local/chapters/"+chapterID+"/progress", bytes.NewReader(b2Bytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on partial update, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp2 library.ChapterInfo
	_ = json.Unmarshal(rec.Body.Bytes(), &resp2)
	if !resp2.Meta.IsRead {
		t.Errorf("expected IsRead to remain true, got %v", resp2.Meta.IsRead)
	}
	if resp2.Meta.LastReadPage != 12 {
		t.Errorf("expected LastReadPage to be 12, got %d", resp2.Meta.LastReadPage)
	}

	// 3. Partial update: only is_read=false (last_read_page should remain 12)
	body3 := map[string]interface{}{
		"is_read": false,
	}
	b3Bytes, _ := json.Marshal(body3)
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/library/manga/"+mangaID+"/providers/local/chapters/"+chapterID+"/progress", bytes.NewReader(b3Bytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on partial update is_read=false, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp3 library.ChapterInfo
	_ = json.Unmarshal(rec.Body.Bytes(), &resp3)
	if resp3.Meta.IsRead {
		t.Errorf("expected IsRead to be false, got %v", resp3.Meta.IsRead)
	}
	if resp3.Meta.LastReadPage != 12 {
		t.Errorf("expected LastReadPage to remain 12, got %d", resp3.Meta.LastReadPage)
	}

	// 4. Non-existent chapter -> 404
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/library/manga/"+mangaID+"/providers/local/chapters/non-existent-ch/progress", bytes.NewReader(b1Bytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found for non-existent chapter, got %d: %s", rec.Code, rec.Body.String())
	}

	// 5. Non-existent manga -> 404
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/library/manga/non-existent-manga/providers/local/chapters/"+chapterID+"/progress", bytes.NewReader(b1Bytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found for non-existent manga, got %d: %s", rec.Code, rec.Body.String())
	}

	// 6. Invalid JSON payload -> 400
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/library/manga/"+mangaID+"/providers/local/chapters/"+chapterID+"/progress", bytes.NewReader([]byte("{invalid-json")))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for invalid json, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestLibraryHandler_ReadingModeOperations(t *testing.T) {
	_, e := setupTestHandler(t)

	// 1. Create manga with reading_mode in content
	createBody := map[string]interface{}{
		"id": "rm-manga-1",
		"meta": map[string]interface{}{
			"title": "Reading Mode Test Manga",
			"content": map[string]interface{}{
				"provider_id":       "mangadex",
				"provider_manga_id": "remote-rm-1",
				"reading_mode":      "rtl",
			},
		},
	}
	createBytes, _ := json.Marshal(createBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga", bytes.NewReader(createBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. GET /library/manga/rm-manga-1 returns reading_mode
	req = httptest.NewRequest(http.MethodGet, "/api/v1/library/manga/rm-manga-1", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}
	var getResp map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &getResp)
	if getResp["reading_mode"] != "rtl" {
		t.Errorf("expected reading_mode 'rtl', got %v", getResp["reading_mode"])
	}

	// 3. GET /library/manga (list) returns reading_mode
	req = httptest.NewRequest(http.MethodGet, "/api/v1/library/manga", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}
	var listResp []map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &listResp)
	var foundRM string
	for _, item := range listResp {
		if item["id"] == "rm-manga-1" {
			if rm, ok := item["reading_mode"].(string); ok {
				foundRM = rm
			}
		}
	}
	if foundRM != "rtl" {
		t.Errorf("expected list reading_mode 'rtl', got %q", foundRM)
	}

	// 4. PATCH with top-level reading_mode: "longstrip"
	patchBody := map[string]interface{}{
		"reading_mode": "longstrip",
	}
	pBytes, _ := json.Marshal(patchBody)
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/library/manga/rm-manga-1", bytes.NewReader(pBytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on patch reading_mode, got %d: %s", rec.Code, rec.Body.String())
	}

	var patchResp struct {
		ID   string            `json:"id"`
		Meta library.MangaMeta `json:"meta"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &patchResp)
	if patchResp.Meta.Content == nil {
		t.Fatalf("expected Content to not be nil")
	}
	if patchResp.Meta.Content.ReadingMode != "longstrip" {
		t.Errorf("expected Content.ReadingMode 'longstrip', got %q", patchResp.Meta.Content.ReadingMode)
	}
	if patchResp.Meta.Content.ProviderID != "mangadex" {
		t.Errorf("expected Content.ProviderID to be preserved as 'mangadex', got %q", patchResp.Meta.Content.ProviderID)
	}

	// 5. PATCH with nested content.reading_mode: "vertical"
	patchNestedBody := map[string]interface{}{
		"content": map[string]interface{}{
			"reading_mode": "vertical",
		},
	}
	pnBytes, _ := json.Marshal(patchNestedBody)
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/library/manga/rm-manga-1", bytes.NewReader(pnBytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on patch nested reading_mode, got %d: %s", rec.Code, rec.Body.String())
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &patchResp)
	if patchResp.Meta.Content.ReadingMode != "vertical" {
		t.Errorf("expected Content.ReadingMode 'vertical', got %q", patchResp.Meta.Content.ReadingMode)
	}
}

func TestLibraryHandler_RefreshLibraryManga_ConcurrentBatch(t *testing.T) {
	h, e := setupTestHandler(t)

	mockP := &mockMultiChapterProvider{
		mockProvider: mockProvider{id: "batchrefreshprov", name: "Batch Refresh Provider"},
	}
	const totalChapters = 50
	for i := 1; i <= totalChapters; i++ {
		mockP.chapters = append(mockP.chapters, sdk.Chapter{
			ID:          fmt.Sprintf("batch-ch-%d", i),
			Name:        fmt.Sprintf("Chapter %d", i),
			Number:      float32(i),
			SourceOrder: i,
		})
	}
	h.registry.Register(mockP)

	validManga := map[string]interface{}{
		"id": "batch-refresh-manga-1",
		"meta": map[string]interface{}{
			"title": "Batch Refresh Manga 1",
			"content": map[string]interface{}{
				"provider_id":       "batchrefreshprov",
				"provider_manga_id": "batch-manga-1",
			},
		},
	}
	vmBytes, _ := json.Marshal(validManga)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga", bytes.NewReader(vmBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/batch-refresh-manga-1/refresh", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on sync, got %d: %s", rec.Code, rec.Body.String())
	}

	var refreshResp struct {
		Added int `json:"added"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &refreshResp); err != nil {
		t.Fatalf("failed to decode sync response: %v", err)
	}
	if refreshResp.Added != totalChapters {
		t.Errorf("expected added=%d, got %d", totalChapters, refreshResp.Added)
	}

	chapters, err := h.lib.ListChapters("batch-refresh-manga-1")
	if err != nil {
		t.Fatalf("failed to list chapters: %v", err)
	}
	if len(chapters) != totalChapters {
		t.Fatalf("expected %d chapters saved, got %d", totalChapters, len(chapters))
	}

	// Second sync: all chapters already exist -> added should be 0
	req = httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/batch-refresh-manga-1/refresh", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on second sync, got %d: %s", rec.Code, rec.Body.String())
	}

	var refreshResp2 struct {
		Added int `json:"added"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &refreshResp2)
	if refreshResp2.Added != 0 {
		t.Errorf("expected added=0 on second sync, got %d", refreshResp2.Added)
	}
}

type mockRefreshPagesProvider struct {
	mockProvider
	fetchCalls atomic.Int32
	pageUrls   []string
}

type mockNoContentProvider struct {
	mockProvider
}

func (m *mockNoContentProvider) Capabilities() []string { return []string{"metadata"} }

func (m *mockRefreshPagesProvider) FetchPages(ctx context.Context, mangaRef, chapterRef string) ([]sdk.Page, error) {
	m.fetchCalls.Add(1)
	pages := make([]sdk.Page, len(m.pageUrls))
	for i, u := range m.pageUrls {
		pages[i] = sdk.Page{Index: i + 1, URL: u}
	}
	return pages, nil
}

func TestLibraryHandler_PullChapter(t *testing.T) {
	h, e, store := setupJobTestHandler(t)

	mockP := &mockRefreshPagesProvider{
		mockProvider: mockProvider{id: "pullpagesprov", name: "Refresh Pages Provider"},
		pageUrls: []string{
			"https://example.com/fresh_p1.jpg",
			"https://example.com/fresh_p2.jpg",
		},
	}
	h.registry.Register(mockP)

	mockNoContent := &mockNoContentProvider{mockProvider: mockProvider{id: "nocontentprov", name: "No Content Provider"}}
	h.registry.Register(mockNoContent)

	mangaID := "manga-ref-1"
	chapterID := "ch-ref-1"

	// Setup manga and chapter in library
	_ = h.lib.SaveManga(mangaID, &library.MangaMeta{
		Title: "Refresh Manga",
		Content: &library.ContentSource{
			ProviderID:      "pullpagesprov",
			ProviderMangaID: "remote-manga-1",
		},
	})
	_ = h.lib.SaveChapter(mangaID, "pullpagesprov", chapterID, &library.ChapterMeta{
		Title: "Refresh Chapter",
		Content: &library.ContentSource{
			ProviderID: "pullpagesprov",
			ChapterRef: "remote-ch-1",
		},
	})

	// 1. Successful pull chapter: enqueues pull_chapter job and returns 202 Accepted
	req1 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/library/manga/%s/providers/pullpagesprov/chapters/%s/pull", mangaID, chapterID), nil)
	rec1 := httptest.NewRecorder()
	e.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted, got %d: %s", rec1.Code, rec1.Body.String())
	}

	var resp1 struct {
		JobID     string `json:"job_id"`
		ChapterID string `json:"chapter_id"`
	}
	if err := json.Unmarshal(rec1.Body.Bytes(), &resp1); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp1.JobID == "" {
		t.Fatalf("expected non-empty job_id")
	}
	if resp1.ChapterID != chapterID {
		t.Errorf("expected chapter_id %q, got %q", chapterID, resp1.ChapterID)
	}

	// Verify enqueued job in jobStore
	job, err := store.GetJob(context.Background(), resp1.JobID)
	if err != nil {
		t.Fatalf("job %s not found in store: %v", resp1.JobID, err)
	}
	if job.Type != queue.JobTypePullChapter {
		t.Errorf("expected job type %q, got %q", queue.JobTypePullChapter, job.Type)
	}
	if job.Status != queue.StatusPending {
		t.Errorf("expected job status %q, got %q", queue.StatusPending, job.Status)
	}
	if job.MaxRetries != 3 {
		t.Errorf("expected max retries 3, got %d", job.MaxRetries)
	}
	if job.ConcurrencyGroup != "pull:pullpagesprov" {
		t.Errorf("expected concurrency group 'pull:pullpagesprov', got %q", job.ConcurrencyGroup)
	}
	if job.Metadata["manga_id"] != mangaID {
		t.Errorf("expected manga_id %q, got %q", mangaID, job.Metadata["manga_id"])
	}
	if job.Metadata["provider_id"] != "pullpagesprov" {
		t.Errorf("expected provider_id 'pullpagesprov', got %q", job.Metadata["provider_id"])
	}
	if job.Metadata["chapter_id"] != chapterID {
		t.Errorf("expected chapter_id %q, got %q", chapterID, job.Metadata["chapter_id"])
	}

	var payload queue.PullChapterPayload
	if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil {
		t.Fatalf("failed to unmarshal job payload: %v", err)
	}
	if payload.MangaID != mangaID || payload.ProviderID != "pullpagesprov" || payload.ChapterID != chapterID {
		t.Errorf("unexpected payload: %+v", payload)
	}

	// 2. Error case: provider lacks content capability -> 400 Bad Request
	reqNoContent := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/library/manga/%s/providers/nocontentprov/chapters/%s/pull", mangaID, chapterID), nil)
	recNoContent := httptest.NewRecorder()
	e.ServeHTTP(recNoContent, reqNoContent)
	if recNoContent.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for no content capability, got %d: %s", recNoContent.Code, recNoContent.Body.String())
	}

	// 3. Error case: job queue not configured -> 503 Service Unavailable
	hNoQueue, eNoQueue := setupTestHandler(t)
	_ = hNoQueue.lib.SaveManga(mangaID, &library.MangaMeta{
		Title: "Pull Test Manga",
	})
	hNoQueue.registry.Register(mockP)
	reqNoQueue := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/library/manga/%s/providers/pullpagesprov/chapters/%s/pull", mangaID, chapterID), nil)
	recNoQueue := httptest.NewRecorder()
	eNoQueue.ServeHTTP(recNoQueue, reqNoQueue)
	if recNoQueue.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 Service Unavailable when queue not configured, got %d: %s", recNoQueue.Code, recNoQueue.Body.String())
	}
}

func TestLibraryHandler_PullChapter_PageJobMetadata(t *testing.T) {
	h, e, store := setupJobTestHandler(t)

	mockP := &mockRefreshPagesProvider{
		mockProvider: mockProvider{id: "jobmetaprov", name: "Job Meta Provider"},
		pageUrls: []string{
			"https://example.com/p0.jpg",
			"https://example.com/p1.jpg",
		},
	}
	h.registry.Register(mockP)

	mangaID := "manga-meta-1"
	chapterID := "ch-meta-1"

	_ = h.lib.SaveManga(mangaID, &library.MangaMeta{
		Title: "Meta Test Manga",
		Content: &library.ContentSource{
			ProviderID:      "jobmetaprov",
			ProviderMangaID: "remote-manga-meta",
		},
	})
	_ = h.lib.SaveChapter(mangaID, "jobmetaprov", chapterID, &library.ChapterMeta{
		Title: "Meta Test Chapter",
		Content: &library.ContentSource{
			ProviderID: "jobmetaprov",
			ChapterRef: "remote-ch-meta",
		},
	})

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/library/manga/%s/providers/jobmetaprov/chapters/%s/pull", mangaID, chapterID), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		JobID     string `json:"job_id"`
		ChapterID string `json:"chapter_id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode resp: %v", err)
	}
	if resp.JobID == "" {
		t.Fatalf("expected non-empty job_id")
	}
	if resp.ChapterID != chapterID {
		t.Errorf("expected chapter_id %q, got %q", chapterID, resp.ChapterID)
	}

	job, err := store.GetJob(context.Background(), resp.JobID)
	if err != nil {
		t.Fatalf("get job %s: %v", resp.JobID, err)
	}
	if job.Type != queue.JobTypePullChapter {
		t.Errorf("expected job type %q, got %q", queue.JobTypePullChapter, job.Type)
	}
	if job.Metadata["manga_id"] != mangaID {
		t.Errorf("expected manga_id %q, got %q", mangaID, job.Metadata["manga_id"])
	}
	if job.Metadata["provider_id"] != "jobmetaprov" {
		t.Errorf("expected provider_id 'jobmetaprov', got %q", job.Metadata["provider_id"])
	}
	if job.Metadata["chapter_id"] != chapterID {
		t.Errorf("expected chapter_id %q, got %q", chapterID, job.Metadata["chapter_id"])
	}
}

func TestLibraryHandler_ProviderBindings(t *testing.T) {
	h, e := setupTestHandler(t)

	// Register providers used in this test
	// mangafox is content-capable (mockProvider default: metadata+content)
	h.registry.Register(&mockProvider{id: "mangafox", name: "MangaFox"})
	// mangadex is NOT content-capable for this test (only metadata)
	h.registry.Register(&mockNoContentProvider{mockProvider: mockProvider{id: "mangadex", name: "MangaDex"}})

	// Create manga first
	createBody := map[string]interface{}{
		"id": "provider-test-manga",
		"meta": map[string]interface{}{
			"title": "Provider Binding Test",
		},
	}
	createBytes, _ := json.Marshal(createBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga", bytes.NewReader(createBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}

	// 1. List providers (empty initially)
	req = httptest.NewRequest(http.MethodGet, "/api/v1/library/manga/provider-test-manga/providers", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}
	var listResp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &listResp)
	providers := listResp["providers"].([]interface{})
	if len(providers) != 0 {
		t.Errorf("expected 0 providers initially, got %d", len(providers))
	}

	// 2. Add a provider binding
	addBody := map[string]interface{}{
		"provider_id":       "mangadex",
		"provider_manga_id": "md-123",
		"manga_title":       "Provider Binding Test",
	}
	addBytes, _ := json.Marshal(addBody)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/provider-test-manga/providers", bytes.NewReader(addBytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}
	var addResp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &addResp)
	addedProviders := addResp["providers"].([]interface{})
	if len(addedProviders) != 1 {
		t.Errorf("expected 1 provider after add, got %d", len(addedProviders))
	}

	// 3. Add duplicate should return 409 Conflict
	req = httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/provider-test-manga/providers", bytes.NewReader(addBytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict for duplicate, got %d: %s", rec.Code, rec.Body.String())
	}

	// 4. Add another provider
	addBody2 := map[string]interface{}{
		"provider_id":       "mangafox",
		"provider_manga_id": "mf-456",
		"manga_title":       "Provider Binding Test Fox",
	}
	addBytes2, _ := json.Marshal(addBody2)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/provider-test-manga/providers", bytes.NewReader(addBytes2))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for second add, got %d: %s", rec.Code, rec.Body.String())
	}

	// 5. Switch content provider (PATCH /content)
	switchBody := map[string]interface{}{
		"provider_id":       "mangafox",
		"provider_manga_id": "mf-456",
	}
	switchBytes, _ := json.Marshal(switchBody)
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/library/manga/provider-test-manga/content", bytes.NewReader(switchBytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for switch content, got %d: %s", rec.Code, rec.Body.String())
	}
	var switchResp struct {
		Meta struct {
			Content *library.ContentSource `json:"content"`
		} `json:"meta"`
	}
	json.Unmarshal(rec.Body.Bytes(), &switchResp)
	if switchResp.Meta.Content.ProviderID != "mangafox" {
		t.Errorf("expected content provider mangafox, got %s", switchResp.Meta.Content.ProviderID)
	}

	// 6. Try to remove mangafox provider (content provider) - should fail with 409
	// because mangadex is not a registered provider with content capability
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/library/manga/provider-test-manga/providers/mangafox/mf-456", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict for removing content provider without backup, got %d: %s", rec.Code, rec.Body.String())
	}

	// 7. Remove mangadex provider (non-content provider) - should succeed
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/library/manga/provider-test-manga/providers/mangadex/md-123", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 No Content for remove non-content provider, got %d: %s", rec.Code, rec.Body.String())
	}

	// 8. Remove non-existent provider -> 404
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/library/manga/provider-test-manga/providers/nonexistent/nx-999", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found for non-existent, got %d: %s", rec.Code, rec.Body.String())
	}

	// 8. GET non-existent manga -> 404
	req = httptest.NewRequest(http.MethodGet, "/api/v1/library/manga/nonexistent/providers", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found for non-existent manga, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestLibraryHandler_ProviderBindings_AddRemoteManga(t *testing.T) {
	h, e := setupTestHandler(t)

	mockP := &mockProvider{id: "importprov", name: "Import Provider"}
	h.registry.Register(mockP)

	// Import a manga from provider
	importBody := map[string]interface{}{
		"provider_id": "importprov",
		"remote_id":   "imported-manga-1",
		"user_status": "reading",
	}
	importBytes, _ := json.Marshal(importBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/import", bytes.NewReader(importBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}

	var importResp struct {
		Meta struct {
			Providers []library.ProviderRef `json:"providers"`
		} `json:"meta"`
	}
	json.Unmarshal(rec.Body.Bytes(), &importResp)

	// Verify Providers was populated with the import
	if len(importResp.Meta.Providers) != 1 {
		t.Fatalf("expected 1 provider in imported manga, got %d", len(importResp.Meta.Providers))
	}
	if importResp.Meta.Providers[0].ProviderID != "importprov" {
		t.Errorf("expected provider_id importprov, got %s", importResp.Meta.Providers[0].ProviderID)
	}
	if importResp.Meta.Providers[0].ProviderMangaID != "imported-manga-1" {
		t.Errorf("expected provider_manga_id imported-manga-1, got %s", importResp.Meta.Providers[0].ProviderMangaID)
	}
}

func TestLibraryHandler_AddProvider_SetAsContent(t *testing.T) {
	h, e := setupTestHandler(t)

	mockP := &mockProvider{id: "contentprov", name: "Content Provider"}
	h.registry.Register(mockP)

	mockNoContent := &mockNoContentProvider{mockProvider: mockProvider{id: "nocontentprov", name: "No Content Provider"}}
	h.registry.Register(mockNoContent)

	// Create manga first
	createBody := map[string]interface{}{
		"id": "setascontent-manga",
		"meta": map[string]interface{}{
			"title": "SetAsContent Test",
		},
	}
	createBytes, _ := json.Marshal(createBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga", bytes.NewReader(createBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}

	// 1. Add provider with set_as_content=true (content-capable) -> 201, content pointer set
	addBody := map[string]interface{}{
		"provider_id":       "contentprov",
		"provider_manga_id": "cp-123",
		"manga_title":       "SetAsContent Test",
		"set_as_content":    true,
	}
	addBytes, _ := json.Marshal(addBody)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/setascontent-manga/providers", bytes.NewReader(addBytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created with set_as_content, got %d: %s", rec.Code, rec.Body.String())
	}
	var addResp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &addResp)
	providers := addResp["providers"].([]interface{})
	if len(providers) != 1 {
		t.Errorf("expected 1 provider, got %d", len(providers))
	}

	// Verify content pointer was set
	updated, _ := h.lib.GetManga("setascontent-manga")
	if updated.Content == nil {
		t.Fatalf("expected Content to be set")
	}
	if updated.Content.ProviderID != "contentprov" {
		t.Errorf("expected content provider contentprov, got %s", updated.Content.ProviderID)
	}

	// 2. Add another provider with set_as_content=true -> 201, content flipped
	addBody2 := map[string]interface{}{
		"provider_id":       "contentprov",
		"provider_manga_id": "cp-456",
		"set_as_content":    true,
	}
	addBytes2, _ := json.Marshal(addBody2)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/setascontent-manga/providers", bytes.NewReader(addBytes2))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for second set_as_content, got %d: %s", rec.Code, rec.Body.String())
	}

	updated2, _ := h.lib.GetManga("setascontent-manga")
	if updated2.Content.ProviderID != "contentprov" || updated2.Content.ProviderMangaID != "cp-456" {
		t.Errorf("expected content flipped to cp-456, got %s/%s", updated2.Content.ProviderID, updated2.Content.ProviderMangaID)
	}

	// 3. Add provider without set_as_content -> 201, content unchanged
	addBody3 := map[string]interface{}{
		"provider_id":       "contentprov",
		"provider_manga_id": "cp-789",
		"set_as_content":    false,
	}
	addBytes3, _ := json.Marshal(addBody3)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/setascontent-manga/providers", bytes.NewReader(addBytes3))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created without set_as_content, got %d: %s", rec.Code, rec.Body.String())
	}

	updated3, _ := h.lib.GetManga("setascontent-manga")
	if updated3.Content.ProviderMangaID != "cp-456" {
		t.Errorf("expected content unchanged after non-set_as_content add, got %s", updated3.Content.ProviderMangaID)
	}

	// 4. Add non-content provider with set_as_content=true -> 400
	addBodyBad := map[string]interface{}{
		"provider_id":       "nocontentprov",
		"provider_manga_id": "ncp-999",
		"set_as_content":    true,
	}
	addBytesBad, _ := json.Marshal(addBodyBad)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/setascontent-manga/providers", bytes.NewReader(addBytesBad))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for non-content with set_as_content, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestLibraryHandler_SwitchContentProvider_CapabilityValidation(t *testing.T) {
	h, e := setupTestHandler(t)

	mockP := &mockProvider{id: "contentprov", name: "Content Provider"}
	h.registry.Register(mockP)

	mockNoContent := &mockNoContentProvider{mockProvider: mockProvider{id: "nocontentprov", name: "No Content Provider"}}
	h.registry.Register(mockNoContent)

	// Create manga with existing providers
	createBody := map[string]interface{}{
		"id": "switchcap-manga",
		"meta": map[string]interface{}{
			"title":     "Switch Capability Test",
			"providers": []library.ProviderRef{{ProviderID: "contentprov", ProviderMangaID: "existing-1"}},
		},
	}
	createBytes, _ := json.Marshal(createBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga", bytes.NewReader(createBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}

	// 1. Switch to content-capable provider -> 200
	switchBody := map[string]interface{}{
		"provider_id":       "contentprov",
		"provider_manga_id": "existing-1",
	}
	switchBytes, _ := json.Marshal(switchBody)
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/library/manga/switchcap-manga/content", bytes.NewReader(switchBytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for switch to content-capable, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. Switch to non-content provider -> 400
	switchBodyBad := map[string]interface{}{
		"provider_id":       "nocontentprov",
		"provider_manga_id": "ncp-1",
	}
	switchBytesBad, _ := json.Marshal(switchBodyBad)
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/library/manga/switchcap-manga/content", bytes.NewReader(switchBytesBad))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for switch to non-content, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSyncChapters_ContextCanceled(t *testing.T) {
	h, _ := setupTestHandler(t)

	mockP := &mockMultiChapterProvider{
		mockProvider: mockProvider{id: "cancelprov", name: "Cancel Provider"},
	}
	for i := 1; i <= 20; i++ {
		mockP.chapters = append(mockP.chapters, sdk.Chapter{
			ID:          fmt.Sprintf("cancel-ch-%d", i),
			Name:        fmt.Sprintf("Chapter %d", i),
			Number:      float32(i),
			SourceOrder: i,
		})
	}
	h.registry.Register(mockP)

	mangaID := "cancel-manga"
	if err := h.lib.SaveManga(mangaID, &library.MangaMeta{
		Title: "Cancel Manga",
		Content: &library.ContentSource{
			ProviderID:      "cancelprov",
			ProviderMangaID: "cancel-remote",
		},
	}); err != nil {
		t.Fatalf("failed to save manga: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, _, err := h.refreshChaptersFromContent(ctx, "cancelprov", mangaID)
	if err == nil {
		t.Fatalf("expected context cancellation error, got nil")
	}
	if !errors.Is(err, context.Canceled) && !strings.Contains(err.Error(), "canceled") {
		t.Fatalf("expected context canceled error, got %v", err)
	}
}

func TestLibraryHandler_BatchPatchChapterProgress(t *testing.T) {
	h, e := setupTestHandler(t)

	mangaID := "batch-patch-manga-1"
	if err := h.lib.SaveManga(mangaID, &library.MangaMeta{
		Title: "Batch Patch Manga",
	}); err != nil {
		t.Fatalf("failed to save manga: %v", err)
	}

	chapterIDs := []string{"ch-1", "ch-2", "ch-3"}
	for i, chID := range chapterIDs {
		if err := h.lib.SaveChapter(mangaID, library.LocalProviderID, chID, &library.ChapterMeta{
			Title:  fmt.Sprintf("Chapter %d", i+1),
			Number: float32(i + 1),
		}); err != nil {
			t.Fatalf("failed to save chapter: %v", err)
		}
	}

	// 1. Valid batch update: ch-1 and ch-2
	body1 := map[string]interface{}{
		"chapter_ids":    []string{"ch-1", "ch-2"},
		"is_read":        true,
		"last_read_page": 10,
	}
	b1Bytes, _ := json.Marshal(body1)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/library/manga/"+mangaID+"/providers/local/chapters/progress", bytes.NewReader(b1Bytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp1 struct {
		Updated    int      `json:"updated"`
		ChapterIDs []string `json:"chapter_ids"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp1); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp1.Updated != 2 {
		t.Errorf("expected updated 2, got %d", resp1.Updated)
	}
	if len(resp1.ChapterIDs) != 2 || resp1.ChapterIDs[0] != "ch-1" || resp1.ChapterIDs[1] != "ch-2" {
		t.Errorf("unexpected chapter_ids: %+v", resp1.ChapterIDs)
	}

	// Verify ch-1 and ch-2 updated in library
	ch1, _ := h.lib.GetChapter(mangaID, library.LocalProviderID, "ch-1")
	if !ch1.IsRead || ch1.LastReadPage != 10 {
		t.Errorf("ch-1 unexpected meta: %+v", ch1)
	}
	ch2, _ := h.lib.GetChapter(mangaID, library.LocalProviderID, "ch-2")
	if !ch2.IsRead || ch2.LastReadPage != 10 {
		t.Errorf("ch-2 unexpected meta: %+v", ch2)
	}
	ch3, _ := h.lib.GetChapter(mangaID, library.LocalProviderID, "ch-3")
	if ch3.IsRead || ch3.LastReadPage != 0 {
		t.Errorf("ch-3 should remain untouched: %+v", ch3)
	}

	// Verify manga last read
	manga, _ := h.lib.GetManga(mangaID)
	if manga.LastReadChapterID != "ch-2" {
		t.Errorf("expected manga LastReadChapterID 'ch-2', got %q", manga.LastReadChapterID)
	}

	// 2. Empty chapter_ids -> 400
	bodyEmpty := map[string]interface{}{
		"chapter_ids": []string{},
		"is_read":     true,
	}
	beBytes, _ := json.Marshal(bodyEmpty)
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/library/manga/"+mangaID+"/providers/local/chapters/progress", bytes.NewReader(beBytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request on empty chapter_ids, got %d", rec.Code)
	}

	// 3. Invalid JSON -> 400
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/library/manga/"+mangaID+"/providers/local/chapters/progress", bytes.NewReader([]byte("{invalid")))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request on invalid json, got %d", rec.Code)
	}

	// 4. Non-existent chapter -> 404
	bodyNonExistent := map[string]interface{}{
		"chapter_ids": []string{"ch-1", "non-existent-ch"},
		"is_read":     true,
	}
	bneBytes, _ := json.Marshal(bodyNonExistent)
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/library/manga/"+mangaID+"/providers/local/chapters/progress", bytes.NewReader(bneBytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found on non-existent chapter, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestLibraryHandler_BatchPullChapters(t *testing.T) {
	h, e := setupTestHandler(t)
	inmemDriver := inmemory.NewDriver(10)
	h.jobStore = inmemDriver
	h.enqueuer = inmemDriver

	mockP := &mockRefreshPagesProvider{
		mockProvider: mockProvider{id: "batchpullprov", name: "Batch Pull Provider"},
		pageUrls: []string{
			"https://example.com/bp_p1.jpg",
			"https://example.com/bp_p2.jpg",
		},
	}
	h.registry.Register(mockP)

	mockNoContent := &mockNoContentProvider{mockProvider: mockProvider{id: "nocontentbatchprov", name: "No Content"}}
	h.registry.Register(mockNoContent)

	mangaID := "batch-pull-manga"
	_ = h.lib.SaveManga(mangaID, &library.MangaMeta{
		Title: "Batch Pull Manga",
		Content: &library.ContentSource{
			ProviderID:      "batchpullprov",
			ProviderMangaID: "remote-bp-manga",
		},
	})

	_ = h.lib.SaveChapter(mangaID, "batchpullprov", "ch-1", &library.ChapterMeta{
		Title: "Chapter 1",
		Content: &library.ContentSource{
			ProviderID: "batchpullprov",
			ChapterRef: "remote-bp-ch-1",
		},
	})
	_ = h.lib.SaveChapter(mangaID, "batchpullprov", "ch-2", &library.ChapterMeta{
		Title: "Chapter 2",
		Content: &library.ContentSource{
			ProviderID: "batchpullprov",
			ChapterRef: "remote-bp-ch-2",
		},
	})

	// 1. Valid batch pull
	body1 := map[string]interface{}{
		"chapter_ids": []string{"ch-1", "ch-2"},
	}
	b1Bytes, _ := json.Marshal(body1)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/"+mangaID+"/providers/batchpullprov/chapters/pull", bytes.NewReader(b1Bytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp1 struct {
		ChapterCount int      `json:"chapter_count"`
		JobIDs       []string `json:"job_ids"`
		JobCount     int      `json:"job_count"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp1); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp1.ChapterCount != 2 {
		t.Errorf("expected chapter_count 2, got %d", resp1.ChapterCount)
	}
	if resp1.JobCount != 2 {
		t.Errorf("expected job_count 2, got %d", resp1.JobCount)
	}
	if len(resp1.JobIDs) != 2 {
		t.Errorf("expected 2 job_ids, got %v", resp1.JobIDs)
	}

	// Verify jobs created in jobStore
	for i, jID := range resp1.JobIDs {
		job, err := inmemDriver.GetJob(context.Background(), jID)
		if err != nil {
			t.Fatalf("job %s not found in store: %v", jID, err)
		}
		if job.Type != queue.JobTypePullChapter {
			t.Errorf("expected job type %q, got %q", queue.JobTypePullChapter, job.Type)
		}
		if job.ConcurrencyGroup != "pull:batchpullprov" {
			t.Errorf("expected concurrency group pull:batchpullprov, got %q", job.ConcurrencyGroup)
		}
		expectedChID := fmt.Sprintf("ch-%d", i+1)
		if job.Metadata["chapter_id"] != expectedChID {
			t.Errorf("expected chapter_id metadata %s, got %s", expectedChID, job.Metadata["chapter_id"])
		}
		if job.Metadata["manga_id"] != mangaID {
			t.Errorf("expected manga_id metadata %s, got %s", mangaID, job.Metadata["manga_id"])
		}
		if job.Metadata["provider_id"] != "batchpullprov" {
			t.Errorf("expected provider_id metadata batchpullprov, got %s", job.Metadata["provider_id"])
		}

		var payload queue.PullChapterPayload
		if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil {
			t.Fatalf("failed to unmarshal job payload: %v", err)
		}
		if payload.MangaID != mangaID || payload.ProviderID != "batchpullprov" || payload.ChapterID != expectedChID {
			t.Errorf("unexpected payload: %+v", payload)
		}
	}

	// 2. Empty chapter_ids -> 400
	bodyEmpty := map[string]interface{}{
		"chapter_ids": []string{},
	}
	beBytes, _ := json.Marshal(bodyEmpty)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/"+mangaID+"/providers/batchpullprov/chapters/pull", bytes.NewReader(beBytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on empty chapter_ids, got %d", rec.Code)
	}

	// 3. Provider without content capability -> 400
	req = httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/"+mangaID+"/providers/nocontentbatchprov/chapters/pull", bytes.NewReader(b1Bytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for provider without content capability, got %d", rec.Code)
	}
}

func TestLibraryHandler_BatchDeleteChapterFiles(t *testing.T) {
	h, e := setupTestHandler(t)

	mangaID := "batch-del-files-manga"
	_ = h.lib.SaveManga(mangaID, &library.MangaMeta{
		Title: "Batch Delete Files Manga",
	})

	for _, chID := range []string{"ch-1", "ch-2"} {
		_ = h.lib.SaveChapter(mangaID, library.LocalProviderID, chID, &library.ChapterMeta{
			Title: "Chapter " + chID,
		})
		chDir := h.lib.ProviderChapterDir(mangaID, library.LocalProviderID, chID)
		_ = os.WriteFile(chDir+"/001.jpg", []byte("img"), 0o644)
		_ = os.WriteFile(chDir+"/pages.json", []byte("[]"), 0o644)
	}

	// 1. POST route: /files/delete
	body1 := map[string]interface{}{
		"chapter_ids": []string{"ch-1"},
	}
	b1Bytes, _ := json.Marshal(body1)
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/"+mangaID+"/providers/local/chapters/files/delete", bytes.NewReader(b1Bytes))
	req1.Header.Set("Content-Type", "application/json")
	rec1 := httptest.NewRecorder()
	e.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Fatalf("POST files/delete expected 200 OK, got %d: %s", rec1.Code, rec1.Body.String())
	}
	var resp1 struct {
		Deleted int `json:"deleted"`
	}
	_ = json.Unmarshal(rec1.Body.Bytes(), &resp1)
	if resp1.Deleted != 1 {
		t.Errorf("expected deleted 1, got %d", resp1.Deleted)
	}

	// Verify ch-1 files gone, meta.json intact
	ch1Dir := h.lib.ProviderChapterDir(mangaID, library.LocalProviderID, "ch-1")
	if _, err := os.Stat(ch1Dir + "/meta.json"); err != nil {
		t.Errorf("ch-1 meta.json should exist")
	}
	if _, err := os.Stat(ch1Dir + "/001.jpg"); !os.IsNotExist(err) {
		t.Errorf("ch-1 001.jpg should be deleted")
	}

	// 2. DELETE route: /files
	body2 := map[string]interface{}{
		"chapter_ids": []string{"ch-2"},
	}
	b2Bytes, _ := json.Marshal(body2)
	req2 := httptest.NewRequest(http.MethodDelete, "/api/v1/library/manga/"+mangaID+"/providers/local/chapters/files", bytes.NewReader(b2Bytes))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	e.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("DELETE files expected 200 OK, got %d: %s", rec2.Code, rec2.Body.String())
	}
	var resp2 struct {
		Deleted int `json:"deleted"`
	}
	_ = json.Unmarshal(rec2.Body.Bytes(), &resp2)
	if resp2.Deleted != 1 {
		t.Errorf("expected deleted 1, got %d", resp2.Deleted)
	}

	// 3. Empty chapter_ids -> 400
	req3 := httptest.NewRequest(http.MethodDelete, "/api/v1/library/manga/"+mangaID+"/providers/local/chapters/files", bytes.NewReader([]byte(`{"chapter_ids":[]}`)))
	req3.Header.Set("Content-Type", "application/json")
	rec3 := httptest.NewRecorder()
	e.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on empty chapter_ids, got %d", rec3.Code)
	}

	// 4. Non-existent chapter dir -> 404
	req4 := httptest.NewRequest(http.MethodDelete, "/api/v1/library/manga/"+mangaID+"/providers/local/chapters/files", bytes.NewReader([]byte(`{"chapter_ids":["non-existent-ch"]}`)))
	req4.Header.Set("Content-Type", "application/json")
	rec4 := httptest.NewRecorder()
	e.ServeHTTP(rec4, req4)
	if rec4.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for non-existent chapter dir, got %d: %s", rec4.Code, rec4.Body.String())
	}
}

func TestLibraryHandler_BatchDeleteChapters(t *testing.T) {
	h, e := setupTestHandler(t)

	mangaID := "batch-del-chapters-manga"
	_ = h.lib.SaveManga(mangaID, &library.MangaMeta{
		Title: "Batch Delete Chapters Manga",
	})

	for _, chID := range []string{"ch-1", "ch-2", "ch-3"} {
		_ = h.lib.SaveChapter(mangaID, library.LocalProviderID, chID, &library.ChapterMeta{
			Title: "Chapter " + chID,
		})
	}

	// 1. POST route: /chapters/delete
	body1 := map[string]interface{}{
		"chapter_ids": []string{"ch-1"},
	}
	b1Bytes, _ := json.Marshal(body1)
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/"+mangaID+"/providers/local/chapters/delete", bytes.NewReader(b1Bytes))
	req1.Header.Set("Content-Type", "application/json")
	rec1 := httptest.NewRecorder()
	e.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Fatalf("POST chapters/delete expected 200 OK, got %d: %s", rec1.Code, rec1.Body.String())
	}
	var resp1 struct {
		Removed int `json:"removed"`
	}
	_ = json.Unmarshal(rec1.Body.Bytes(), &resp1)
	if resp1.Removed != 1 {
		t.Errorf("expected removed 1, got %d", resp1.Removed)
	}

	// 2. DELETE route: /chapters
	body2 := map[string]interface{}{
		"chapter_ids": []string{"ch-2"},
	}
	b2Bytes, _ := json.Marshal(body2)
	req2 := httptest.NewRequest(http.MethodDelete, "/api/v1/library/manga/"+mangaID+"/providers/local/chapters", bytes.NewReader(b2Bytes))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	e.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("DELETE chapters expected 200 OK, got %d: %s", rec2.Code, rec2.Body.String())
	}
	var resp2 struct {
		Removed int `json:"removed"`
	}
	_ = json.Unmarshal(rec2.Body.Bytes(), &resp2)
	if resp2.Removed != 1 {
		t.Errorf("expected removed 1, got %d", resp2.Removed)
	}

	// Verify only ch-3 remains
	remaining, err := h.lib.ListChapters(mangaID, library.LocalProviderID)
	if err != nil {
		t.Fatalf("ListChapters failed: %v", err)
	}
	if len(remaining) != 1 || remaining[0].ID != "ch-3" {
		t.Errorf("expected only ch-3 remaining, got %+v", remaining)
	}

	// 3. Empty chapter_ids -> 400
	req3 := httptest.NewRequest(http.MethodDelete, "/api/v1/library/manga/"+mangaID+"/providers/local/chapters", bytes.NewReader([]byte(`{"chapter_ids":[]}`)))
	req3.Header.Set("Content-Type", "application/json")
	rec3 := httptest.NewRecorder()
	e.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on empty chapter_ids, got %d", rec3.Code)
	}
}

type mockBatchRefreshProvider struct {
	mockProvider
	chapters []sdk.Chapter
}

func (m *mockBatchRefreshProvider) FetchChapters(ctx context.Context, mangaRef string) ([]sdk.Chapter, error) {
	return m.chapters, nil
}

func TestBatchRefreshChapters(t *testing.T) {
	h, e := setupTestHandler(t)

	upDate1 := time.Date(2025, 5, 1, 12, 0, 0, 0, time.UTC)
	upDate2 := time.Date(2025, 5, 2, 12, 0, 0, 0, time.UTC)

	mockP := &mockBatchRefreshProvider{
		mockProvider: mockProvider{id: "batchrefreshprov", name: "Batch Refresh Provider"},
		chapters: []sdk.Chapter{
			{ID: "ch-1", Name: "Updated Chapter 1 Title", Number: 1.5, UploadDate: upDate1, SourceOrder: 10},
			{ID: "ch-2", Name: "Updated Chapter 2 Title", Number: 2.5, UploadDate: upDate2, SourceOrder: 20},
		},
	}
	h.registry.Register(mockP)

	mockNoContent := &mockNoContentProvider{mockProvider: mockProvider{id: "nocontentrefreshprov", name: "No Content"}}
	h.registry.Register(mockNoContent)

	mangaID := "batch-refresh-manga"
	if err := h.lib.SaveManga(mangaID, &library.MangaMeta{
		Title: "Batch Refresh Manga",
		Content: &library.ContentSource{
			ProviderID:      "batchrefreshprov",
			ProviderMangaID: "remote-br-manga",
		},
	}); err != nil {
		t.Fatalf("failed to save manga: %v", err)
	}

	readAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Save ch-1 with reading progress and old metadata
	if err := h.lib.SaveChapter(mangaID, "batchrefreshprov", "ch-1", &library.ChapterMeta{
		Title:        "Old Chapter 1",
		Number:       1.0,
		UploadDate:   time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		SourceOrder:  1,
		IsRead:       true,
		LastReadPage: 12,
		LastReadAt:   readAt,
		Content: &library.ContentSource{
			ProviderID: "batchrefreshprov",
			ChapterRef: "ch-1",
		},
	}); err != nil {
		t.Fatalf("failed to save ch-1: %v", err)
	}

	// Save ch-2 with reading progress
	if err := h.lib.SaveChapter(mangaID, "batchrefreshprov", "ch-2", &library.ChapterMeta{
		Title:        "Old Chapter 2",
		Number:       2.0,
		UploadDate:   time.Date(2020, 2, 1, 0, 0, 0, 0, time.UTC),
		SourceOrder:  2,
		IsRead:       false,
		LastReadPage: 3,
		LastReadAt:   readAt,
		Content: &library.ContentSource{
			ProviderID: "batchrefreshprov",
			ChapterRef: "ch-2",
		},
	}); err != nil {
		t.Fatalf("failed to save ch-2: %v", err)
	}

	// Save mock pages to verify page files are preserved
	if err := h.lib.SaveChapterPages(mangaID, "batchrefreshprov", "ch-1", []library.PageItem{
		{Index: 1, URL: "https://example.com/p1.jpg"},
		{Index: 2, URL: "https://example.com/p2.jpg"},
	}); err != nil {
		t.Fatalf("failed to save chapter pages: %v", err)
	}

	// 1. Valid batch refresh
	body1 := map[string]interface{}{
		"chapter_ids": []string{"ch-1", "ch-2"},
	}
	b1Bytes, _ := json.Marshal(body1)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/"+mangaID+"/providers/batchrefreshprov/chapters/refresh", bytes.NewReader(b1Bytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Refreshed  int      `json:"refreshed"`
		ChapterIDs []string `json:"chapter_ids"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Refreshed != 2 {
		t.Errorf("expected refreshed 2, got %d", resp.Refreshed)
	}
	if len(resp.ChapterIDs) != 2 || resp.ChapterIDs[0] != "ch-1" || resp.ChapterIDs[1] != "ch-2" {
		t.Errorf("unexpected chapter_ids in response: %+v", resp.ChapterIDs)
	}

	// Verify ch-1 metadata updated while preserving reading progress
	ch1, err := h.lib.GetChapter(mangaID, "batchrefreshprov", "ch-1")
	if err != nil {
		t.Fatalf("failed to get ch-1: %v", err)
	}
	if ch1.Title != "Updated Chapter 1 Title" {
		t.Errorf("expected title 'Updated Chapter 1 Title', got %q", ch1.Title)
	}
	if ch1.Number != 1.5 {
		t.Errorf("expected number 1.5, got %f", ch1.Number)
	}
	if !ch1.UploadDate.Equal(upDate1) {
		t.Errorf("expected upload date %v, got %v", upDate1, ch1.UploadDate)
	}
	if ch1.SourceOrder != 10 {
		t.Errorf("expected source order 10, got %d", ch1.SourceOrder)
	}
	if !ch1.IsRead {
		t.Errorf("expected is_read to remain true")
	}
	if ch1.LastReadPage != 12 {
		t.Errorf("expected last_read_page to remain 12, got %d", ch1.LastReadPage)
	}
	if !ch1.LastReadAt.Equal(readAt) {
		t.Errorf("expected last_read_at %v, got %v", readAt, ch1.LastReadAt)
	}
	if ch1.Content == nil || ch1.Content.LastSyncedAt.IsZero() {
		t.Errorf("expected Content.LastSyncedAt to be populated")
	}

	// Verify pages preserved
	pages1, err := h.lib.GetChapterPages(mangaID, "batchrefreshprov", "ch-1")
	if err != nil || len(pages1) != 2 {
		t.Errorf("expected 2 pages preserved for ch-1, got %v (err: %v)", pages1, err)
	}

	// Verify ch-2 metadata updated while preserving reading progress
	ch2, err := h.lib.GetChapter(mangaID, "batchrefreshprov", "ch-2")
	if err != nil {
		t.Fatalf("failed to get ch-2: %v", err)
	}
	if ch2.Title != "Updated Chapter 2 Title" {
		t.Errorf("expected title 'Updated Chapter 2 Title', got %q", ch2.Title)
	}
	if ch2.Number != 2.5 {
		t.Errorf("expected number 2.5, got %f", ch2.Number)
	}
	if !ch2.UploadDate.Equal(upDate2) {
		t.Errorf("expected upload date %v, got %v", upDate2, ch2.UploadDate)
	}
	if ch2.SourceOrder != 20 {
		t.Errorf("expected source order 20, got %d", ch2.SourceOrder)
	}
	if ch2.IsRead {
		t.Errorf("expected is_read to remain false")
	}
	if ch2.LastReadPage != 3 {
		t.Errorf("expected last_read_page to remain 3, got %d", ch2.LastReadPage)
	}

	// 2. Empty chapter_ids -> 400
	bodyEmpty := map[string]interface{}{
		"chapter_ids": []string{},
	}
	beBytes, _ := json.Marshal(bodyEmpty)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/"+mangaID+"/providers/batchrefreshprov/chapters/refresh", bytes.NewReader(beBytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on empty chapter_ids, got %d", rec.Code)
	}

	// 3. Provider without content capability -> 400
	req = httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/"+mangaID+"/providers/nocontentrefreshprov/chapters/refresh", bytes.NewReader(b1Bytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for provider without content capability, got %d", rec.Code)
	}

	// 4. Unknown chapter in upstream -> ignored in count
	bodyPartial := map[string]interface{}{
		"chapter_ids": []string{"ch-1", "non-existent-ch"},
	}
	bpBytes, _ := json.Marshal(bodyPartial)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/"+mangaID+"/providers/batchrefreshprov/chapters/refresh", bytes.NewReader(bpBytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for partial match, got %d", rec.Code)
	}
	var respPartial struct {
		Refreshed  int      `json:"refreshed"`
		ChapterIDs []string `json:"chapter_ids"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &respPartial)
	if respPartial.Refreshed != 1 || len(respPartial.ChapterIDs) != 1 || respPartial.ChapterIDs[0] != "ch-1" {
		t.Errorf("expected only 1 chapter refreshed (ch-1), got %+v", respPartial)
	}
}

func TestLibraryHandler_ListChaptersDownloadedStatus(t *testing.T) {
	h, e := setupTestHandler(t)

	mangaID := "download-status-manga"
	_ = h.lib.SaveManga(mangaID, &library.MangaMeta{
		Title: "Download Status Manga",
		Content: &library.ContentSource{
			ProviderID: "local",
		},
	})

	// Chapter 1: 0 downloaded pages, page_count = 10
	ch1ID := "ch-1"
	_ = h.lib.SaveChapter(mangaID, "local", ch1ID, &library.ChapterMeta{
		Title:     "Chapter 1",
		Number:    1.0,
		PageCount: 10,
	})

	// Chapter 2: 5 downloaded pages, page_count = 10 (partial)
	ch2ID := "ch-2"
	_ = h.lib.SaveChapter(mangaID, "local", ch2ID, &library.ChapterMeta{
		Title:     "Chapter 2",
		Number:    2.0,
		PageCount: 10,
	})
	ch2Dir := h.lib.ProviderChapterDir(mangaID, "local", ch2ID)
	_ = os.MkdirAll(ch2Dir, 0o755)
	for i := 1; i <= 5; i++ {
		_ = os.WriteFile(filepath.Join(ch2Dir, fmt.Sprintf("%d.jpg", i)), []byte("img"), 0o644)
	}

	// Chapter 3: 10 downloaded pages, page_count = 10 (complete)
	ch3ID := "ch-3"
	now := time.Now().Truncate(time.Second)
	_ = h.lib.SaveChapter(mangaID, "local", ch3ID, &library.ChapterMeta{
		Title:        "Chapter 3",
		Number:       3.0,
		PageCount:    10,
		DownloadedAt: now,
	})
	ch3Dir := h.lib.ProviderChapterDir(mangaID, "local", ch3ID)
	_ = os.MkdirAll(ch3Dir, 0o755)
	for i := 1; i <= 10; i++ {
		_ = os.WriteFile(filepath.Join(ch3Dir, fmt.Sprintf("%d.webp", i)), []byte("img"), 0o644)
	}

	// GET /library/manga/:mangaId/chapters
	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/manga/"+mangaID+"/chapters", nil)
	rec := httptest.NewRecorder()
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
	if len(resp.Chapters) != 3 {
		t.Fatalf("expected 3 chapters, got %d", len(resp.Chapters))
	}

	// Verify Chapter 1
	c1 := resp.Chapters[0]
	if c1["id"] != ch1ID {
		t.Errorf("expected ch1 ID %q, got %v", ch1ID, c1["id"])
	}
	if c1["is_downloaded"] != false {
		t.Errorf("ch1 expected is_downloaded=false, got %v", c1["is_downloaded"])
	}
	if c1["downloaded_pages"] != float64(0) {
		t.Errorf("ch1 expected downloaded_pages=0, got %v", c1["downloaded_pages"])
	}
	if c1["page_count"] != float64(10) {
		t.Errorf("ch1 expected page_count=10, got %v", c1["page_count"])
	}
	meta1, ok := c1["meta"].(map[string]interface{})
	if !ok {
		t.Fatalf("ch1 meta missing")
	}
	if meta1["is_downloaded"] != nil && meta1["is_downloaded"] != false {
		t.Errorf("ch1 meta is_downloaded mismatch: %v", meta1["is_downloaded"])
	}
	if meta1["downloaded_pages"] != nil && meta1["downloaded_pages"] != float64(0) {
		t.Errorf("ch1 meta downloaded_pages mismatch: %v", meta1["downloaded_pages"])
	}

	// Verify Chapter 2
	c2 := resp.Chapters[1]
	if c2["id"] != ch2ID {
		t.Errorf("expected ch2 ID %q, got %v", ch2ID, c2["id"])
	}
	if c2["is_downloaded"] != false {
		t.Errorf("ch2 expected is_downloaded=false, got %v", c2["is_downloaded"])
	}
	if c2["downloaded_pages"] != float64(5) {
		t.Errorf("ch2 expected downloaded_pages=5, got %v", c2["downloaded_pages"])
	}
	if c2["page_count"] != float64(10) {
		t.Errorf("ch2 expected page_count=10, got %v", c2["page_count"])
	}
	meta2, ok := c2["meta"].(map[string]interface{})
	if !ok {
		t.Fatalf("ch2 meta missing")
	}
	if meta2["downloaded_pages"] != float64(5) {
		t.Errorf("ch2 meta expected downloaded_pages=5, got %v", meta2["downloaded_pages"])
	}

	// Verify Chapter 3
	c3 := resp.Chapters[2]
	if c3["id"] != ch3ID {
		t.Errorf("expected ch3 ID %q, got %v", ch3ID, c3["id"])
	}
	if c3["is_downloaded"] != true {
		t.Errorf("ch3 expected is_downloaded=true, got %v", c3["is_downloaded"])
	}
	if c3["downloaded_pages"] != float64(10) {
		t.Errorf("ch3 expected downloaded_pages=10, got %v", c3["downloaded_pages"])
	}
	if c3["page_count"] != float64(10) {
		t.Errorf("ch3 expected page_count=10, got %v", c3["page_count"])
	}
	if c3["downloaded_at"] == "" || c3["downloaded_at"] == nil {
		t.Errorf("ch3 expected non-empty downloaded_at, got %v", c3["downloaded_at"])
	}
	meta3, ok := c3["meta"].(map[string]interface{})
	if !ok {
		t.Fatalf("ch3 meta missing")
	}
	if meta3["is_downloaded"] != true {
		t.Errorf("ch3 meta expected is_downloaded=true, got %v", meta3["is_downloaded"])
	}
	if meta3["downloaded_pages"] != float64(10) {
		t.Errorf("ch3 meta expected downloaded_pages=10, got %v", meta3["downloaded_pages"])
	}

	// Test GET /library/manga/:mangaId/providers/:providerId/chapters/:chapterId
	reqGet := httptest.NewRequest(http.MethodGet, "/api/v1/library/manga/"+mangaID+"/providers/local/chapters/"+ch3ID, nil)
	recGet := httptest.NewRecorder()
	e.ServeHTTP(recGet, reqGet)
	if recGet.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on getChapter, got %d", recGet.Code)
	}
	var getChapterResp struct {
		ID   string              `json:"id"`
		Meta library.ChapterMeta `json:"meta"`
	}
	if err := json.Unmarshal(recGet.Body.Bytes(), &getChapterResp); err != nil {
		t.Fatalf("unmarshal getChapter response failed: %v", err)
	}
	if !getChapterResp.Meta.IsDownloaded {
		t.Errorf("getChapter expected IsDownloaded=true, got false")
	}
	if getChapterResp.Meta.DownloadedPages != 10 {
		t.Errorf("getChapter expected DownloadedPages=10, got %d", getChapterResp.Meta.DownloadedPages)
	}
}

func TestListLibraryManga_EmptyReturnsEmptyArray(t *testing.T) {
	_, e := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/manga", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}
	body := strings.TrimSpace(rec.Body.String())
	if body != "[]" {
		t.Errorf("expected empty JSON array '[]', got %q", body)
	}
}

func TestPullChapter_MangaNotFound(t *testing.T) {
	h, e := setupTestHandler(t)
	inmemDriver := inmemory.NewDriver(10)
	h.jobStore = inmemDriver
	h.enqueuer = inmemDriver
	mockP := &mockProvider{id: "prov-1", name: "Provider 1"}
	h.registry.Register(mockP)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/nonexistent/providers/prov-1/chapters/ch-1/pull", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 Not Found for nonexistent manga, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPullChaptersBatch_MangaNotFound(t *testing.T) {
	h, e := setupTestHandler(t)
	inmemDriver := inmemory.NewDriver(10)
	h.jobStore = inmemDriver
	h.enqueuer = inmemDriver
	mockP := &mockProvider{id: "prov-1", name: "Provider 1"}
	h.registry.Register(mockP)

	body := map[string]interface{}{
		"chapter_ids": []string{"ch-1", "ch-2"},
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/nonexistent/providers/prov-1/chapters/pull", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 Not Found for nonexistent manga, got %d: %s", rec.Code, rec.Body.String())
	}
}
