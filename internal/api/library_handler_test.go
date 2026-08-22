package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/tubruk/kiyomi/internal/library"
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

	// 1. Refresh non-existent manga -> 404 Not Found
	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/non-existent/refresh", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found for non-existent manga, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. Refresh manga without connected provider -> 400 Bad Request
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

	// 3. Refresh valid manga with mock provider -> 200 OK with added chapters
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
		t.Fatalf("expected 200 OK on refresh, got %d: %s", rec.Code, rec.Body.String())
	}

	var refreshResp struct {
		Added      int    `json:"added"`
		Updated    int    `json:"updated"`
		ProviderID string `json:"provider_id"`
		MangaID    string `json:"manga_id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &refreshResp); err != nil {
		t.Fatalf("failed to decode refresh response: %v", err)
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

	// 4. Refreshing again skips already existing chapters -> added=0
	req = httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/refresh-manga-1/refresh", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on second refresh, got %d: %s", rec.Code, rec.Body.String())
	}

	var refreshResp2 struct {
		Added int `json:"added"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &refreshResp2)
	if refreshResp2.Added != 0 {
		t.Errorf("expected added=0 on second refresh, got %d", refreshResp2.Added)
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
	if err := h.lib.SaveChapter(mangaID, chapterID, chMeta); err != nil {
		t.Fatalf("failed to save chapter: %v", err)
	}

	// 1. Full update: is_read=true, last_read_page=5
	body1 := map[string]interface{}{
		"is_read":        true,
		"last_read_page": 5,
	}
	b1Bytes, _ := json.Marshal(body1)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/library/manga/"+mangaID+"/chapters/"+chapterID+"/progress", bytes.NewReader(b1Bytes))
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
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/library/manga/"+mangaID+"/chapters/"+chapterID+"/progress", bytes.NewReader(b2Bytes))
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
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/library/manga/"+mangaID+"/chapters/"+chapterID+"/progress", bytes.NewReader(b3Bytes))
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
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/library/manga/"+mangaID+"/chapters/non-existent-ch/progress", bytes.NewReader(b1Bytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found for non-existent chapter, got %d: %s", rec.Code, rec.Body.String())
	}

	// 5. Non-existent manga -> 404
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/library/manga/non-existent-manga/chapters/"+chapterID+"/progress", bytes.NewReader(b1Bytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found for non-existent manga, got %d: %s", rec.Code, rec.Body.String())
	}

	// 6. Invalid JSON payload -> 400
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/library/manga/"+mangaID+"/chapters/"+chapterID+"/progress", bytes.NewReader([]byte("{invalid-json")))
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
		t.Fatalf("expected 200 OK on refresh, got %d: %s", rec.Code, rec.Body.String())
	}

	var refreshResp struct {
		Added int `json:"added"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &refreshResp); err != nil {
		t.Fatalf("failed to decode refresh response: %v", err)
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

	// Second refresh: all chapters already exist -> added should be 0
	req = httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/batch-refresh-manga-1/refresh", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on second refresh, got %d: %s", rec.Code, rec.Body.String())
	}

	var refreshResp2 struct {
		Added int `json:"added"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &refreshResp2)
	if refreshResp2.Added != 0 {
		t.Errorf("expected added=0 on second refresh, got %d", refreshResp2.Added)
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

func TestLibraryHandler_RefreshChapterPages(t *testing.T) {
	h, e := setupTestHandler(t)

	mockP := &mockRefreshPagesProvider{
		mockProvider: mockProvider{id: "refreshpagesprov", name: "Refresh Pages Provider"},
		pageUrls: []string{
			"https://example.com/fresh_p1.jpg",
			"https://example.com/fresh_p2.jpg",
		},
	}
	h.registry.Register(mockP)

	mangaID := "manga-ref-1"
	chapterID := "ch-ref-1"

	// Setup manga and chapter in library
	_ = h.lib.SaveManga(mangaID, &library.MangaMeta{
		Title: "Refresh Manga",
		Content: &library.ContentSource{
			ProviderID:      "refreshpagesprov",
			ProviderMangaID: "remote-manga-1",
		},
	})
	_ = h.lib.SaveChapter(mangaID, chapterID, &library.ChapterMeta{
		Title: "Refresh Chapter",
		Content: &library.ContentSource{
			ProviderID: "refreshpagesprov",
			ChapterRef: "remote-ch-1",
		},
	})

	// Pre-populate old pages
	oldPages := []library.PageItem{
		{Index: 1, URL: "https://example.com/old_p1.jpg", Source: "library"},
	}
	_ = h.lib.SaveChapterPages(mangaID, chapterID, oldPages)

	// Verify old pages exist
	stored, err := h.lib.GetChapterPages(mangaID, chapterID)
	if err != nil || len(stored) != 1 || stored[0].URL != "https://example.com/old_p1.jpg" {
		t.Fatalf("expected old pages in library, got %v", stored)
	}

	// 1. Refresh via /library/manga/:mangaId/chapters/:chapterId/pages/refresh
	req1 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/library/manga/%s/chapters/%s/pages/refresh", mangaID, chapterID), nil)
	rec1 := httptest.NewRecorder()
	e.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec1.Code, rec1.Body.String())
	}

	var resp1 struct {
		Message string `json:"message"`
		Pages   []struct {
			Index  int    `json:"index"`
			URL    string `json:"url"`
			Source string `json:"source"`
		} `json:"pages"`
	}
	if err := json.Unmarshal(rec1.Body.Bytes(), &resp1); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp1.Message != "chapter pages refreshed successfully" {
		t.Errorf("expected message 'chapter pages refreshed successfully', got %q", resp1.Message)
	}
	if len(resp1.Pages) != 2 {
		t.Fatalf("expected 2 pages, got %d", len(resp1.Pages))
	}
	if resp1.Pages[0].URL != "https://example.com/fresh_p1.jpg" {
		t.Errorf("expected refreshed URL, got %q", resp1.Pages[0].URL)
	}
	if resp1.Pages[0].Source != "provider" {
		t.Errorf("expected source 'provider', got %q", resp1.Pages[0].Source)
	}

	// Verify pages updated in library storage
	updatedStored, err := h.lib.GetChapterPages(mangaID, chapterID)
	if err != nil {
		t.Fatalf("failed to get updated pages: %v", err)
	}
	if len(updatedStored) != 2 || updatedStored[0].URL != "https://example.com/fresh_p1.jpg" {
		t.Fatalf("expected updated pages in library, got %v", updatedStored)
	}

	// 2. Refresh via alias route /chapters/:chapterId/pages/refresh
	mockP.pageUrls = []string{
		"https://example.com/fresh_v2_p1.jpg",
		"https://example.com/fresh_v2_p2.jpg",
		"https://example.com/fresh_v2_p3.jpg",
	}

	req2 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/chapters/%s/pages/refresh", chapterID), nil)
	rec2 := httptest.NewRecorder()
	e.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("alias refresh expected 200 OK, got %d: %s", rec2.Code, rec2.Body.String())
	}

	var resp2 struct {
		Message string `json:"message"`
		Pages   []struct {
			Index  int    `json:"index"`
			URL    string `json:"url"`
			Source string `json:"source"`
		} `json:"pages"`
	}
	if err := json.Unmarshal(rec2.Body.Bytes(), &resp2); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp2.Pages) != 3 {
		t.Fatalf("expected 3 refreshed pages, got %d", len(resp2.Pages))
	}
	if resp2.Pages[0].URL != "https://example.com/fresh_v2_p1.jpg" {
		t.Errorf("expected v2 refreshed URL, got %q", resp2.Pages[0].URL)
	}

	// Verify library updated again
	v2Stored, err := h.lib.GetChapterPages(mangaID, chapterID)
	if err != nil {
		t.Fatalf("failed to get v2 updated pages: %v", err)
	}
	if len(v2Stored) != 3 {
		t.Fatalf("expected 3 stored pages, got %d", len(v2Stored))
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

func TestRefreshChapters_ContextCanceled(t *testing.T) {
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

	_, err := h.refreshChaptersFromContent(ctx, mangaID)
	if err == nil {
		t.Fatalf("expected context cancellation error, got nil")
	}
	if !errors.Is(err, context.Canceled) && !strings.Contains(err.Error(), "canceled") {
		t.Fatalf("expected context canceled error, got %v", err)
	}
}


