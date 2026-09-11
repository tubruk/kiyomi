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
		"id":         "patch-manga-1",
		"metadata":   map[string]interface{}{"title": "Patch Test Manga"},
		"user_state": map[string]interface{}{"status": library.UserStatusUnread, "favorite": false, "rating": 0.0, "notes": ""},
	}
	createBytes, _ := json.Marshal(createBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga", bytes.NewReader(createBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}

	// 1. Patch via per-concern user_state endpoint with full state. The new
	// PATCH semantics REPLACE the user_state (not merge), so each patch
	// includes the full target state.
	patch1 := map[string]interface{}{
		"user_state": map[string]interface{}{"status": library.UserStatusUnread, "favorite": true, "rating": 0.0, "notes": ""},
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
		ID        string                `json:"id"`
		Metadata  library.MangaMetadata `json:"metadata"`
		UserState library.UserState     `json:"user_state"`
		Bindings  library.MangaBindings `json:"bindings"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp1); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !resp1.UserState.Favorite {
		t.Errorf("expected Favorite to be true, got %v", resp1.UserState.Favorite)
	}
	if resp1.UserState.Status != library.UserStatusUnread {
		t.Errorf("expected Status %q, got %q", library.UserStatusUnread, resp1.UserState.Status)
	}

	// 2. Patch user_status -> reading (full state in body)
	patch2 := map[string]interface{}{
		"user_state": map[string]interface{}{"status": library.UserStatusReading, "favorite": true, "rating": 0.0, "notes": ""},
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
		ID        string                `json:"id"`
		Metadata  library.MangaMetadata `json:"metadata"`
		UserState library.UserState     `json:"user_state"`
		Bindings  library.MangaBindings `json:"bindings"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp2)
	if resp2.UserState.Status != library.UserStatusReading {
		t.Errorf("expected Status %q, got %q", library.UserStatusReading, resp2.UserState.Status)
	}
	if !resp2.UserState.Favorite {
		t.Errorf("expected Favorite to remain true, got %v", resp2.UserState.Favorite)
	}

	// 3. Patch user_rating and user_notes (full state in body)
	patch3 := map[string]interface{}{
		"user_state": map[string]interface{}{"status": library.UserStatusReading, "favorite": true, "rating": 8.5, "notes": "Loved chapter 10!"},
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
		ID        string                `json:"id"`
		Metadata  library.MangaMetadata `json:"metadata"`
		UserState library.UserState     `json:"user_state"`
		Bindings  library.MangaBindings `json:"bindings"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp3)
	if resp3.UserState.Rating != 8.5 {
		t.Errorf("expected Rating 8.5, got %v", resp3.UserState.Rating)
	}
	if resp3.UserState.Notes != "Loved chapter 10!" {
		t.Errorf("expected Notes 'Loved chapter 10!', got %q", resp3.UserState.Notes)
	}
	if resp3.UserState.Status != library.UserStatusReading {
		t.Errorf("expected Status to remain reading, got %q", resp3.UserState.Status)
	}
	if !resp3.UserState.Favorite {
		t.Errorf("expected Favorite to remain true, got %v", resp3.UserState.Favorite)
	}

	// 4. Patch with invalid user_status -> should fail with 400 Bad Request
	patchInvalid := map[string]interface{}{
		"user_state": map[string]interface{}{"status": "non_existent_status"},
	}
	pInvBytes, _ := json.Marshal(patchInvalid)
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/library/manga/patch-manga-1", bytes.NewReader(pInvBytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for invalid user_status, got %d: %s", rec.Code, rec.Body.String())
	}

	// 5. Patch non-existent manga: in the per-concern refactor, GetMetadata
	// returns zero-value (not 404), so PATCH succeeds with empty fields.
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/library/manga/non-existent-manga", bytes.NewReader(p1Bytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK (new PATCH semantics don't 404), got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestLibraryHandler_UserStatusValidation(t *testing.T) {
	_, e := setupTestHandler(t)

	// Create with invalid status
	createInvalid := map[string]interface{}{
		"id":         "invalid-status-manga",
		"metadata":   map[string]interface{}{"title": "Invalid Status Manga"},
		"user_state": map[string]interface{}{"status": "invalid_status_enum"},
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
		"id":         "valid-status-manga",
		"metadata":   map[string]interface{}{"title": "Valid Status Manga"},
		"user_state": map[string]interface{}{"status": library.UserStatusPlanToRead},
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
		"metadata":   map[string]interface{}{"title": "Valid Status Manga Updated"},
		"user_state": map[string]interface{}{"status": "corrupted_status"},
	}
	uiBytes, _ := json.Marshal(updateInvalid)
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/library/manga/valid-status-manga", bytes.NewReader(uiBytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request on update with invalid user_status, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestLibraryHandler_RefreshLibraryManga(t *testing.T) {
	h, e := setupTestHandler(t)
	inmemDriver := inmemory.NewDriver(10)
	h.jobStore = inmemDriver
	h.enqueuer = inmemDriver

	mockP := &mockProvider{id: "mockprov", name: "Mock Provider"}
	h.registry.Register(mockP)

	// 1. Sync non-existent manga -> 404 Not Found
	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/non-existent/metadata/refresh", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found for non-existent manga, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. Sync manga without connected provider -> 400 Bad Request
	noProvManga := map[string]interface{}{
		"id":       "no-provider-manga",
		"metadata": map[string]interface{}{"title": "No Provider Manga"},
	}
	npBytes, _ := json.Marshal(noProvManga)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/library/manga", bytes.NewReader(npBytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/no-provider-manga/metadata/refresh", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for manga with no provider, got %d: %s", rec.Code, rec.Body.String())
	}

	// 3. Sync valid manga with mock provider -> 202 Accepted on metadata refresh
	validManga := map[string]interface{}{
		"id":       "refresh-manga-1",
		"metadata": map[string]interface{}{"title": "Refresh Manga 1"},
		"bindings": map[string]interface{}{
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

	req = httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/refresh-manga-1/metadata/refresh", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted on metadata refresh, got %d: %s", rec.Code, rec.Body.String())
	}

	var refreshResp struct {
		JobID      string `json:"job_id"`
		ProviderID string `json:"provider_id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &refreshResp); err != nil {
		t.Fatalf("failed to decode metadata refresh response: %v", err)
	}
	if refreshResp.JobID == "" {
		t.Errorf("expected non-empty job_id")
	}
	if refreshResp.ProviderID != "mockprov" {
		t.Errorf("expected providerId='mockprov', got %s", refreshResp.ProviderID)
	}

	// 4. Test chapter syncing via refreshChaptersFromContent
	added, orphaned, err := h.refreshChaptersFromContent(context.Background(), "mockprov", "refresh-manga-1")
	if err != nil {
		t.Fatalf("refreshChaptersFromContent failed: %v", err)
	}
	if added != 1 {
		t.Errorf("expected added=1, got %d", added)
	}
	_ = orphaned

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

	// Syncing again skips already existing chapters -> added=0
	added2, _, err := h.refreshChaptersFromContent(context.Background(), "mockprov", "refresh-manga-1")
	if err != nil {
		t.Fatalf("second refreshChaptersFromContent failed: %v", err)
	}
	if added2 != 0 {
		t.Errorf("expected added=0 on second sync, got %d", added2)
	}
}

func TestLibraryHandler_PatchChapterProgress(t *testing.T) {
	h, e := setupTestHandler(t)

	// Create a manga
	mangaID := "progress-manga-1"
	chapterID := "progress-ch-1"
	mangaMeta := library.Manga{
		Metadata: library.MangaMetadata{Title: "Progress Manga 1"},
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
	if savedManga.UserState.LastReadChapterID != chapterID {
		t.Errorf("expected manga LastReadChapterID %q, got %q", chapterID, savedManga.UserState.LastReadChapterID)
	}
	if savedManga.UserState.LastReadAt.IsZero() {
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
		"id":       "rm-manga-1",
		"metadata": map[string]interface{}{"title": "Reading Mode Test Manga"},
		"bindings": map[string]interface{}{
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

	// 4. PATCH with bindings.content.reading_mode: "longstrip"
	// The new PATCH semantics REPLACE the entire bindings, so the body must
	// include the full Content (provider_id, provider_manga_id, etc.).
	patchBody := map[string]interface{}{
		"bindings": map[string]interface{}{
			"content": map[string]interface{}{
				"provider_id":       "mangadex",
				"provider_manga_id": "remote-rm-1",
				"reading_mode":      "longstrip",
			},
		},
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
		ID       string                `json:"id"`
		Metadata library.MangaMetadata `json:"metadata"`
		Bindings library.MangaBindings `json:"bindings"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &patchResp)
	if patchResp.Bindings.Content == nil {
		t.Fatalf("expected Content to not be nil")
	}
	if patchResp.Bindings.Content.ReadingMode != "longstrip" {
		t.Errorf("expected Content.ReadingMode 'longstrip', got %q", patchResp.Bindings.Content.ReadingMode)
	}
	if patchResp.Bindings.Content.ProviderID != "mangadex" {
		t.Errorf("expected Content.ProviderID to be preserved as 'mangadex', got %q", patchResp.Bindings.Content.ProviderID)
	}

	// 5. PATCH with nested content.reading_mode: "vertical"
	// Full content body to preserve provider binding.
	patchNestedBody := map[string]interface{}{
		"bindings": map[string]interface{}{
			"content": map[string]interface{}{
				"provider_id":       "mangadex",
				"provider_manga_id": "remote-rm-1",
				"reading_mode":      "vertical",
			},
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
	if patchResp.Bindings.Content.ReadingMode != "vertical" {
		t.Errorf("expected Content.ReadingMode 'vertical', got %q", patchResp.Bindings.Content.ReadingMode)
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
		"id":       "batch-refresh-manga-1",
		"metadata": map[string]interface{}{"title": "Batch Refresh Manga 1"},
		"bindings": map[string]interface{}{
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

	added, _, err := h.refreshChaptersFromContent(context.Background(), "batchrefreshprov", "batch-refresh-manga-1")
	if err != nil {
		t.Fatalf("failed on sync: %v", err)
	}
	if added != totalChapters {
		t.Errorf("expected added=%d, got %d", totalChapters, added)
	}

	chapters, err := h.lib.ListChapters("batch-refresh-manga-1")
	if err != nil {
		t.Fatalf("failed to list chapters: %v", err)
	}
	if len(chapters) != totalChapters {
		t.Fatalf("expected %d chapters saved, got %d", totalChapters, len(chapters))
	}

	// Second sync: all chapters already exist -> added should be 0
	added2, _, err := h.refreshChaptersFromContent(context.Background(), "batchrefreshprov", "batch-refresh-manga-1")
	if err != nil {
		t.Fatalf("failed on second sync: %v", err)
	}
	if added2 != 0 {
		t.Errorf("expected added=0 on second sync, got %d", added2)
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
	_ = h.lib.SaveManga(mangaID, library.Manga{
		Metadata: library.MangaMetadata{Title: "Refresh Manga"},
		Bindings: library.MangaBindings{
			Content: &library.ContentSource{
				ProviderID:      "pullpagesprov",
				ProviderMangaID: "remote-manga-1",
			},
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
	req1 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/library/manga/%s/chapters/%s/pull", mangaID, chapterID), nil)
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
	_ = h.lib.SaveChapter(mangaID, "nocontentprov", "nocontent-ch", &library.ChapterMeta{
		Title:   "No Content Chapter",
		Content: &library.ContentSource{ProviderID: "nocontentprov"},
	})
	reqNoContent := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/library/manga/%s/chapters/nocontent-ch/pull", mangaID), nil)
	recNoContent := httptest.NewRecorder()
	e.ServeHTTP(recNoContent, reqNoContent)
	if recNoContent.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for no content capability, got %d: %s", recNoContent.Code, recNoContent.Body.String())
	}

	// 3. Error case: job queue not configured -> 503 Service Unavailable
	hNoQueue, eNoQueue := setupTestHandler(t)
	_ = hNoQueue.lib.SaveManga(mangaID, library.Manga{
		Metadata: library.MangaMetadata{Title: "Pull Test Manga"},
	})
	_ = hNoQueue.lib.SaveChapter(mangaID, "pullpagesprov", chapterID, &library.ChapterMeta{
		Title:   "Refresh Chapter",
		Content: &library.ContentSource{ProviderID: "pullpagesprov"},
	})
	hNoQueue.registry.Register(mockP)
	reqNoQueue := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/library/manga/%s/chapters/%s/pull", mangaID, chapterID), nil)
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

	_ = h.lib.SaveManga(mangaID, library.Manga{
		Metadata: library.MangaMetadata{Title: "Meta Test Manga"},
		Bindings: library.MangaBindings{
			Content: &library.ContentSource{
				ProviderID:      "jobmetaprov",
				ProviderMangaID: "remote-manga-meta",
			},
		},
	})
	_ = h.lib.SaveChapter(mangaID, "jobmetaprov", chapterID, &library.ChapterMeta{
		Title: "Meta Test Chapter",
		Content: &library.ContentSource{
			ProviderID: "jobmetaprov",
			ChapterRef: "remote-ch-meta",
		},
	})

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/library/manga/%s/chapters/%s/pull", mangaID, chapterID), nil)
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
		"id":       "provider-test-manga",
		"metadata": map[string]interface{}{"title": "Provider Binding Test"},
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
	req = httptest.NewRequest(http.MethodGet, "/api/v1/library/manga/provider-test-manga/bindings", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}
	var listResp library.MangaBindings
	json.Unmarshal(rec.Body.Bytes(), &listResp)
	if len(listResp.Providers) != 0 {
		t.Errorf("expected 0 providers initially, got %d", len(listResp.Providers))
	}

	// 2. Add a provider binding
	addBody := map[string]interface{}{
		"provider_id":       "mangadex",
		"provider_manga_id": "md-123",
		"manga_title":       "Provider Binding Test",
	}
	addBytes, _ := json.Marshal(addBody)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/provider-test-manga/bindings", bytes.NewReader(addBytes))
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

	// 3. Add duplicate should succeed idempotently (201 Created) and update title
	dupBody := map[string]interface{}{
		"provider_id":       "mangadex",
		"provider_manga_id": "md-123",
		"manga_title":       "Provider Binding Test Updated",
	}
	dupBytes, _ := json.Marshal(dupBody)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/provider-test-manga/bindings", bytes.NewReader(dupBytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for duplicate add, got %d: %s", rec.Code, rec.Body.String())
	}
	var dupResp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &dupResp)
	dupProviders := dupResp["providers"].([]interface{})
	if len(dupProviders) != 1 {
		t.Errorf("expected 1 provider after duplicate add, got %d", len(dupProviders))
	}
	firstProv := dupProviders[0].(map[string]interface{})
	if firstProv["manga_title"] != "Provider Binding Test Updated" {
		t.Errorf("expected updated title %q, got %q", "Provider Binding Test Updated", firstProv["manga_title"])
	}

	// 4. Add another provider
	addBody2 := map[string]interface{}{
		"provider_id":       "mangafox",
		"provider_manga_id": "mf-456",
		"manga_title":       "Provider Binding Test Fox",
	}
	addBytes2, _ := json.Marshal(addBody2)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/provider-test-manga/bindings", bytes.NewReader(addBytes2))
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
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/library/manga/provider-test-manga/bindings/content", bytes.NewReader(switchBytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for switch content, got %d: %s", rec.Code, rec.Body.String())
	}
	var switchResp struct {
		Bindings library.MangaBindings `json:"bindings"`
	}
	json.Unmarshal(rec.Body.Bytes(), &switchResp)
	if switchResp.Bindings.Content == nil || switchResp.Bindings.Content.ProviderID != "mangafox" {
		t.Errorf("expected content provider mangafox, got %+v", switchResp.Bindings.Content)
	}

	// 6. Try to remove mangafox provider (content provider) - should fail with 409
	// because mangadex is not a registered provider with content capability
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/library/manga/provider-test-manga/bindings/mangafox/mf-456", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict for removing content provider without backup, got %d: %s", rec.Code, rec.Body.String())
	}

	// 7. Remove mangadex provider (non-content provider) - should succeed
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/library/manga/provider-test-manga/bindings/mangadex/md-123", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 No Content for remove non-content provider, got %d: %s", rec.Code, rec.Body.String())
	}

	// 8. Remove non-existent provider -> 404
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/library/manga/provider-test-manga/bindings/nonexistent/nx-999", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found for non-existent, got %d: %s", rec.Code, rec.Body.String())
	}

	// 8. GET non-existent manga: per-concern GetBindings returns zero value
	// with no error, so the handler returns 200 with empty providers.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/library/manga/nonexistent/bindings", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK with empty providers, got %d: %s", rec.Code, rec.Body.String())
	}
	var nonExistentResp library.MangaBindings
	json.Unmarshal(rec.Body.Bytes(), &nonExistentResp)
	if len(nonExistentResp.Providers) != 0 {
		t.Errorf("expected 0 providers for non-existent manga, got %d", len(nonExistentResp.Providers))
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
		Bindings library.MangaBindings `json:"bindings"`
	}
	json.Unmarshal(rec.Body.Bytes(), &importResp)

	// Verify Providers was populated with the import
	if len(importResp.Bindings.Providers) != 1 {
		t.Fatalf("expected 1 provider in imported manga, got %d", len(importResp.Bindings.Providers))
	}
	if importResp.Bindings.Providers[0].ProviderID != "importprov" {
		t.Errorf("expected provider_id importprov, got %s", importResp.Bindings.Providers[0].ProviderID)
	}
	if importResp.Bindings.Providers[0].ProviderMangaID != "imported-manga-1" {
		t.Errorf("expected provider_manga_id imported-manga-1, got %s", importResp.Bindings.Providers[0].ProviderMangaID)
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
		"id":       "setascontent-manga",
		"metadata": map[string]interface{}{"title": "SetAsContent Test"},
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
	req = httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/setascontent-manga/bindings", bytes.NewReader(addBytes))
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
	if updated.Bindings.Content == nil {
		t.Fatalf("expected Content to be set")
	}
	if updated.Bindings.Content.ProviderID != "contentprov" {
		t.Errorf("expected content provider contentprov, got %s", updated.Bindings.Content.ProviderID)
	}

	// 2. Add another provider with set_as_content=true -> 201, content flipped
	addBody2 := map[string]interface{}{
		"provider_id":       "contentprov",
		"provider_manga_id": "cp-456",
		"set_as_content":    true,
	}
	addBytes2, _ := json.Marshal(addBody2)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/setascontent-manga/bindings", bytes.NewReader(addBytes2))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for second set_as_content, got %d: %s", rec.Code, rec.Body.String())
	}

	updated2, _ := h.lib.GetManga("setascontent-manga")
	if updated2.Bindings.Content.ProviderID != "contentprov" || updated2.Bindings.Content.ProviderMangaID != "cp-456" {
		t.Errorf("expected content flipped to cp-456, got %s/%s", updated2.Bindings.Content.ProviderID, updated2.Bindings.Content.ProviderMangaID)
	}

	// 3. Add provider without set_as_content -> 201, content unchanged
	addBody3 := map[string]interface{}{
		"provider_id":       "contentprov",
		"provider_manga_id": "cp-789",
		"set_as_content":    false,
	}
	addBytes3, _ := json.Marshal(addBody3)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/setascontent-manga/bindings", bytes.NewReader(addBytes3))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created without set_as_content, got %d: %s", rec.Code, rec.Body.String())
	}

	updated3, _ := h.lib.GetManga("setascontent-manga")
	if updated3.Bindings.Content.ProviderMangaID != "cp-456" {
		t.Errorf("expected content unchanged after non-set_as_content add, got %s", updated3.Bindings.Content.ProviderMangaID)
	}

	// 4. Add non-content provider with set_as_content=true -> 400
	addBodyBad := map[string]interface{}{
		"provider_id":       "nocontentprov",
		"provider_manga_id": "ncp-999",
		"set_as_content":    true,
	}
	addBytesBad, _ := json.Marshal(addBodyBad)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/setascontent-manga/bindings", bytes.NewReader(addBytesBad))
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
		"id":       "switchcap-manga",
		"metadata": map[string]interface{}{"title": "Switch Capability Test"},
		"bindings": map[string]interface{}{
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
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/library/manga/switchcap-manga/bindings/content", bytes.NewReader(switchBytes))
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
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/library/manga/switchcap-manga/bindings/content", bytes.NewReader(switchBytesBad))
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
	if err := h.lib.SaveManga(mangaID, library.Manga{
		Metadata: library.MangaMetadata{Title: "Cancel Manga"},
		Bindings: library.MangaBindings{
			Content: &library.ContentSource{
				ProviderID:      "cancelprov",
				ProviderMangaID: "cancel-remote",
			},
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
	if err := h.lib.SaveManga(mangaID, library.Manga{
		Metadata: library.MangaMetadata{Title: "Batch Patch Manga"},
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
	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/"+mangaID+"/batch/chapters/progress", bytes.NewReader(b1Bytes))
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
	if manga.UserState.LastReadChapterID != "ch-2" {
		t.Errorf("expected manga LastReadChapterID 'ch-2', got %q", manga.UserState.LastReadChapterID)
	}

	// 2. Empty chapter_ids -> 400
	bodyEmpty := map[string]interface{}{
		"chapter_ids": []string{},
		"is_read":     true,
	}
	beBytes, _ := json.Marshal(bodyEmpty)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/"+mangaID+"/batch/chapters/progress", bytes.NewReader(beBytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request on empty chapter_ids, got %d", rec.Code)
	}

	// 3. Invalid JSON -> 400
	req = httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/"+mangaID+"/batch/chapters/progress", bytes.NewReader([]byte("{invalid")))
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
	req = httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/"+mangaID+"/batch/chapters/progress", bytes.NewReader(bneBytes))
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
	_ = h.lib.SaveManga(mangaID, library.Manga{
		Metadata: library.MangaMetadata{Title: "Batch Pull Manga"},
		Bindings: library.MangaBindings{
			Content: &library.ContentSource{
				ProviderID:      "batchpullprov",
				ProviderMangaID: "remote-bp-manga",
			},
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
	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/"+mangaID+"/batch/chapters/pull", bytes.NewReader(b1Bytes))
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
	req = httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/"+mangaID+"/batch/chapters/pull", bytes.NewReader(beBytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on empty chapter_ids, got %d", rec.Code)
	}

	// 3. Provider without content capability -> 400
	_ = h.lib.SaveChapter(mangaID, "nocontentbatchprov", "ch-nc", &library.ChapterMeta{
		Title: "Chapter NC",
		Content: &library.ContentSource{
			ProviderID: "nocontentbatchprov",
			ChapterRef: "remote-bp-ch-nc",
		},
	})
	bodyNC := map[string]interface{}{
		"chapter_ids": []string{"ch-nc"},
	}
	bncBytes, _ := json.Marshal(bodyNC)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/"+mangaID+"/batch/chapters/pull", bytes.NewReader(bncBytes))
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
	_ = h.lib.SaveManga(mangaID, library.Manga{
		Metadata: library.MangaMetadata{Title: "Batch Delete Files Manga"},
	})

	for _, chID := range []string{"ch-1", "ch-2"} {
		_ = h.lib.SaveChapter(mangaID, library.LocalProviderID, chID, &library.ChapterMeta{
			Title: "Chapter " + chID,
		})
		chDir := h.lib.ProviderChapterDir(mangaID, library.LocalProviderID, chID)
		_ = os.WriteFile(chDir+"/001.jpg", []byte("img"), 0o644)
		_ = os.WriteFile(chDir+"/pages.json", []byte("[]"), 0o644)
	}

	// 1. POST route: /batch/chapter-files/delete
	body1 := map[string]interface{}{
		"chapter_ids": []string{"ch-1"},
	}
	b1Bytes, _ := json.Marshal(body1)
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/"+mangaID+"/batch/chapter-files/delete", bytes.NewReader(b1Bytes))
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

	// 2. Second delete: /batch/chapter-files/delete
	body2 := map[string]interface{}{
		"chapter_ids": []string{"ch-2"},
	}
	b2Bytes, _ := json.Marshal(body2)
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/"+mangaID+"/batch/chapter-files/delete", bytes.NewReader(b2Bytes))
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
	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/"+mangaID+"/batch/chapter-files/delete", bytes.NewReader([]byte(`{"chapter_ids":[]}`)))
	req3.Header.Set("Content-Type", "application/json")
	rec3 := httptest.NewRecorder()
	e.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on empty chapter_ids, got %d", rec3.Code)
	}

	// 4. Non-existent chapter dir -> 404
	req4 := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/"+mangaID+"/batch/chapter-files/delete", bytes.NewReader([]byte(`{"chapter_ids":["non-existent-ch"]}`)))
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
	_ = h.lib.SaveManga(mangaID, library.Manga{
		Metadata: library.MangaMetadata{Title: "Batch Delete Chapters Manga"},
	})

	for _, chID := range []string{"ch-1", "ch-2", "ch-3"} {
		_ = h.lib.SaveChapter(mangaID, library.LocalProviderID, chID, &library.ChapterMeta{
			Title: "Chapter " + chID,
		})
	}

	// 1. POST route: /batch/chapters/delete
	body1 := map[string]interface{}{
		"chapter_ids": []string{"ch-1"},
	}
	b1Bytes, _ := json.Marshal(body1)
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/"+mangaID+"/batch/chapters/delete", bytes.NewReader(b1Bytes))
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

	// 2. Second delete: /batch/chapters/delete
	body2 := map[string]interface{}{
		"chapter_ids": []string{"ch-2"},
	}
	b2Bytes, _ := json.Marshal(body2)
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/"+mangaID+"/batch/chapters/delete", bytes.NewReader(b2Bytes))
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
	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/"+mangaID+"/batch/chapters/delete", bytes.NewReader([]byte(`{"chapter_ids":[]}`)))
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
	if err := h.lib.SaveManga(mangaID, library.Manga{
		Metadata: library.MangaMetadata{Title: "Batch Refresh Manga"},
		Bindings: library.MangaBindings{
			Content: &library.ContentSource{
				ProviderID:      "batchrefreshprov",
				ProviderMangaID: "remote-br-manga",
			},
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
	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/"+mangaID+"/batch/chapters/refresh", bytes.NewReader(b1Bytes))
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
	req = httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/"+mangaID+"/batch/chapters/refresh", bytes.NewReader(beBytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on empty chapter_ids, got %d", rec.Code)
	}

	// 3. Provider without content capability -> 400
	_ = h.lib.SaveChapter(mangaID, "nocontentrefreshprov", "ch-nc", &library.ChapterMeta{
		Title: "Old Chapter NC",
		Content: &library.ContentSource{
			ProviderID: "nocontentrefreshprov",
			ChapterRef: "ch-nc",
		},
	})
	bodyNC := map[string]interface{}{
		"chapter_ids": []string{"ch-nc"},
	}
	bncBytes, _ := json.Marshal(bodyNC)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/"+mangaID+"/batch/chapters/refresh", bytes.NewReader(bncBytes))
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
	req = httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/"+mangaID+"/batch/chapters/refresh", bytes.NewReader(bpBytes))
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

func TestLibraryHandler_RefreshChapter(t *testing.T) {
	h, e := setupTestHandler(t)

	upDate := time.Date(2025, 5, 1, 12, 0, 0, 0, time.UTC)
	readAt := time.Date(2025, 4, 15, 8, 30, 0, 0, time.UTC)

	mockP := &mockBatchRefreshProvider{
		mockProvider: mockProvider{id: "singlerefreshprov", name: "Single Refresh Provider"},
		chapters: []sdk.Chapter{
			{
				ID:          "remote-ch-1",
				Name:        "Updated Chapter 1 Single",
				Number:      1.5,
				UploadDate:  upDate,
				SourceOrder: 10,
			},
		},
	}
	h.registry.Register(mockP)

	mockNoContent := &mockNoContentProvider{mockProvider: mockProvider{id: "nocontentsingleprov", name: "No Content"}}
	h.registry.Register(mockNoContent)

	mangaID := "single-refresh-manga"
	chapterID := "ch-single-1"

	_ = h.lib.SaveManga(mangaID, library.Manga{
		Metadata: library.MangaMetadata{Title: "Single Refresh Manga"},
		Bindings: library.MangaBindings{
			Content: &library.ContentSource{
				ProviderID:      "singlerefreshprov",
				ProviderMangaID: "remote-single-manga",
			},
		},
	})

	_ = h.lib.SaveChapter(mangaID, "singlerefreshprov", chapterID, &library.ChapterMeta{
		Title:        "Old Chapter Title",
		Number:       1.0,
		UploadDate:   time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		SourceOrder:  1,
		IsRead:       true,
		LastReadPage: 12,
		LastReadAt:   readAt,
		Content: &library.ContentSource{
			ProviderID: "singlerefreshprov",
			ChapterRef: "remote-ch-1",
		},
	})

	// 1. Successful single refresh
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/library/manga/%s/chapters/%s/refresh", mangaID, chapterID), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		ID         string              `json:"id"`
		MangaID    string              `json:"manga_id"`
		ProviderID string              `json:"provider_id"`
		Meta       library.ChapterMeta `json:"meta"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode resp: %v", err)
	}
	if resp.ID != chapterID {
		t.Errorf("expected id %q, got %q", chapterID, resp.ID)
	}
	if resp.Meta.Title != "Updated Chapter 1 Single" {
		t.Errorf("expected updated title, got %q", resp.Meta.Title)
	}
	if resp.Meta.Number != 1.5 {
		t.Errorf("expected number 1.5, got %f", resp.Meta.Number)
	}
	if !resp.Meta.IsRead {
		t.Errorf("expected reading progress is_read=true preserved")
	}
	if resp.Meta.LastReadPage != 12 {
		t.Errorf("expected last_read_page=12 preserved, got %d", resp.Meta.LastReadPage)
	}

	// 2. Non-existent chapter -> 404
	req404 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/library/manga/%s/chapters/non-existent-ch/refresh", mangaID), nil)
	rec404 := httptest.NewRecorder()
	e.ServeHTTP(rec404, req404)
	if rec404.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for non-existent chapter, got %d", rec404.Code)
	}

	// 3. Provider without content capability -> 400
	_ = h.lib.SaveChapter(mangaID, "nocontentsingleprov", "ch-nc-single", &library.ChapterMeta{
		Title: "NC Single",
		Content: &library.ContentSource{
			ProviderID: "nocontentsingleprov",
			ChapterRef: "remote-ch-nc",
		},
	})
	reqNC := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/library/manga/%s/chapters/ch-nc-single/refresh", mangaID), nil)
	recNC := httptest.NewRecorder()
	e.ServeHTTP(recNC, reqNC)
	if recNC.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for provider without content capability, got %d", recNC.Code)
	}
}

func TestLibraryHandler_GetChapterFiles(t *testing.T) {
	h, e := setupTestHandler(t)

	mangaID := "files-test-manga"
	chapterID := "ch-files-1"

	_ = h.lib.SaveManga(mangaID, library.Manga{
		Metadata: library.MangaMetadata{Title: "Files Test Manga"},
	})
	_ = h.lib.SaveChapter(mangaID, library.LocalProviderID, chapterID, &library.ChapterMeta{
		Title: "Chapter Files Test",
	})

	// Create chapter directory and write image files and a non-image file
	chDir := h.lib.ProviderChapterDir(mangaID, library.LocalProviderID, chapterID)
	_ = os.WriteFile(chDir+"/001.jpg", []byte("image-data-1"), 0o644)
	_ = os.WriteFile(chDir+"/002.png", []byte("image-data-png-2"), 0o644)
	_ = os.WriteFile(chDir+"/meta.json", []byte("{}"), 0o644)

	// 1. Successful GET /files
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/library/manga/%s/chapters/%s/files", mangaID, chapterID), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Files []struct {
			Name string `json:"name"`
			Size int64  `json:"size"`
		} `json:"files"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode resp: %v", err)
	}
	if len(resp.Files) != 2 {
		t.Fatalf("expected 2 image files, got %d", len(resp.Files))
	}
	names := map[string]bool{resp.Files[0].Name: true, resp.Files[1].Name: true}
	if !names["001.jpg"] || !names["002.png"] {
		t.Errorf("unexpected files list: %+v", resp.Files)
	}

	// 2. Chapter with no files on disk returns empty array
	chapterIDEmpty := "ch-files-empty"
	_ = h.lib.SaveChapter(mangaID, library.LocalProviderID, chapterIDEmpty, &library.ChapterMeta{
		Title: "Empty Chapter",
	})
	reqEmpty := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/library/manga/%s/chapters/%s/files", mangaID, chapterIDEmpty), nil)
	recEmpty := httptest.NewRecorder()
	e.ServeHTTP(recEmpty, reqEmpty)

	if recEmpty.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on empty chapter files, got %d: %s", recEmpty.Code, recEmpty.Body.String())
	}
	var respEmpty struct {
		Files []interface{} `json:"files"`
	}
	if err := json.Unmarshal(recEmpty.Body.Bytes(), &respEmpty); err != nil {
		t.Fatalf("decode empty resp: %v", err)
	}
	if len(respEmpty.Files) != 0 {
		t.Errorf("expected 0 files, got %d", len(respEmpty.Files))
	}

	// 3. Non-existent chapter -> 404
	req404 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/library/manga/%s/chapters/non-existent-ch/files", mangaID), nil)
	rec404 := httptest.NewRecorder()
	e.ServeHTTP(rec404, req404)
	if rec404.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for non-existent chapter, got %d", rec404.Code)
	}
}

func TestLibraryHandler_ListChaptersDownloadedStatus(t *testing.T) {
	h, e := setupTestHandler(t)

	mangaID := "download-status-manga"
	_ = h.lib.SaveManga(mangaID, library.Manga{
		Metadata: library.MangaMetadata{Title: "Download Status Manga"},
		Bindings: library.MangaBindings{
			Content: &library.ContentSource{
				ProviderID: "local",
			},
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

	// Test GET /library/manga/:mangaId/chapters/:chapterId
	reqGet := httptest.NewRequest(http.MethodGet, "/api/v1/library/manga/"+mangaID+"/chapters/"+ch3ID, nil)
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

	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/nonexistent/chapters/ch-1/pull", nil)
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
	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/nonexistent/batch/chapters/pull", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 Not Found for nonexistent manga, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestLibraryHandler_ExternalLinksResponse(t *testing.T) {
	h, e := setupTestHandler(t)

	mangaID := "ext-links-manga-1"
	links := []library.ExternalLink{
		{
			Provider: "mangadex",
			Label:    "MangaDex",
			URL:      "https://mangadex.org/title/12345",
		},
		{
			Provider: "anilist",
			Label:    "AniList",
			URL:      "https://anilist.co/manga/67890",
		},
	}
	mangaMeta := library.Manga{
		Metadata: library.MangaMetadata{
			Title:         "External Links Test Manga",
			ExternalLinks: links,
		},
	}
	if err := h.lib.SaveManga(mangaID, mangaMeta); err != nil {
		t.Fatalf("failed to save manga: %v", err)
	}

	// 1. GET /api/v1/library/manga/:mangaId
	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/manga/"+mangaID, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var getResp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &getResp); err != nil {
		t.Fatalf("failed to parse get manga response: %v", err)
	}

	for _, key := range []string{"external_links", "externalLinks"} {
		rawLinks, ok := getResp[key].([]interface{})
		if !ok || len(rawLinks) != 2 {
			t.Fatalf("expected 2 items in %s, got %v", key, getResp[key])
		}
		firstLink, ok := rawLinks[0].(map[string]interface{})
		if !ok || firstLink["provider"] != "mangadex" || firstLink["label"] != "MangaDex" || firstLink["url"] != "https://mangadex.org/title/12345" {
			t.Errorf("unexpected first link in %s: %v", key, firstLink)
		}
	}

	// 2. GET /api/v1/library/manga (list)
	req = httptest.NewRequest(http.MethodGet, "/api/v1/library/manga", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var listResp []map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("failed to parse list manga response: %v", err)
	}

	var foundItem map[string]interface{}
	for _, item := range listResp {
		if item["id"] == mangaID {
			foundItem = item
			break
		}
	}
	if foundItem == nil {
		t.Fatalf("manga %s not found in list response", mangaID)
	}

	for _, key := range []string{"external_links", "externalLinks"} {
		rawLinks, ok := foundItem[key].([]interface{})
		if !ok || len(rawLinks) != 2 {
			t.Fatalf("expected 2 items in list %s, got %v", key, foundItem[key])
		}
		firstLink, ok := rawLinks[0].(map[string]interface{})
		if !ok || firstLink["provider"] != "mangadex" || firstLink["label"] != "MangaDex" || firstLink["url"] != "https://mangadex.org/title/12345" {
			t.Errorf("unexpected first link in list %s: %v", key, firstLink)
		}
	}
}

// --- Library Manga Merge Tests ---

// writeMangaWithProviders creates a manga meta with the given providers and
// saves it. Returns the manga ID.
func writeMangaWithProviders(t *testing.T, h *Handler, mangaID string, providers []library.ProviderRef, content *library.ContentSource, extras ...func(*library.Manga)) {
	t.Helper()
	meta := library.Manga{
		Metadata: library.MangaMetadata{Title: "Test " + mangaID},
		Bindings: library.MangaBindings{Providers: providers},
	}
	if content != nil {
		meta.Bindings.Content = content
	}
	for _, ex := range extras {
		ex(&meta)
	}
	if err := h.lib.SaveManga(mangaID, meta); err != nil {
		t.Fatalf("failed to seed manga %s: %v", mangaID, err)
	}
}

func TestMergeLibraryManga_HappyPath(t *testing.T) {
	h, e := setupTestHandler(t)

	keepID := "merge-keep-1"
	srcID := "merge-src-1"

	writeMangaWithProviders(t, h, keepID,
		[]library.ProviderRef{{ProviderID: "mangadex", ProviderMangaID: "md-1"}},
		&library.ContentSource{ProviderID: "mangadex", ProviderMangaID: "md-1"},
		func(m *library.Manga) {
			m.Metadata.Title = "Keep Title"
			m.Metadata.Description = "Keep Desc"
			m.Metadata.Tags = []string{"action", "drama"}
			m.Metadata.Aliases = []string{"KeepAlias"}
			m.Metadata.Authors = []string{"Author A"}
		},
	)

	writeMangaWithProviders(t, h, srcID,
		[]library.ProviderRef{{ProviderID: "mangafox", ProviderMangaID: "mf-1"}},
		&library.ContentSource{ProviderID: "mangafox", ProviderMangaID: "mf-1"},
		func(m *library.Manga) {
			m.Metadata.Title = "Source Title"
			m.Metadata.Description = "Source Desc"
			m.Metadata.Tags = []string{"comedy", "drama"}
			m.Metadata.Aliases = []string{"SourceAlias"}
			m.Metadata.Authors = []string{"Author B"}
		},
	)

	// Add a chapter to source so we can verify chapter dirs move.
	if err := h.lib.SaveChapter(srcID, "mangafox", "ch-1", &library.ChapterMeta{Title: "Ch1", Number: 1}); err != nil {
		t.Fatalf("seed chapter: %v", err)
	}

	body := map[string]interface{}{
		"keep_manga_id":    keepID,
		"source_manga_ids": []string{srcID},
		"content_provider": map[string]string{
			"provider_id":       "mangadex",
			"provider_manga_id": "md-1",
		},
		"metadata": map[string]string{
			"title":        "keep",
			"description":  "keep",
			"aliases":      "merge",
			"tags":         "merge",
			"authors":      "merge",
			"release_year": "keep",
			"cover_url":    "keep",
		},
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/merge", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		ID       string                `json:"id"`
		Metadata library.MangaMetadata `json:"metadata"`
		Bindings library.MangaBindings `json:"bindings"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Metadata.Title != "Keep Title" {
		t.Errorf("expected Keep Title, got %q", resp.Metadata.Title)
	}
	if resp.Metadata.Description != "Keep Desc" {
		t.Errorf("expected Keep Desc, got %q", resp.Metadata.Description)
	}
	// Tags merged: action, drama, comedy (dedup)
	if len(resp.Metadata.Tags) != 3 {
		t.Errorf("expected 3 tags, got %d (%v)", len(resp.Metadata.Tags), resp.Metadata.Tags)
	}
	// Authors merged
	if len(resp.Metadata.Authors) != 2 {
		t.Errorf("expected 2 authors, got %d (%v)", len(resp.Metadata.Authors), resp.Metadata.Authors)
	}
	// Aliases merged
	if len(resp.Metadata.Aliases) != 2 {
		t.Errorf("expected 2 aliases, got %d (%v)", len(resp.Metadata.Aliases), resp.Metadata.Aliases)
	}
	// Providers union: 2
	if len(resp.Bindings.Providers) != 2 {
		t.Errorf("expected 2 providers, got %d (%v)", len(resp.Bindings.Providers), resp.Bindings.Providers)
	}
	// Content provider still points at mangadex
	if resp.Bindings.Content == nil || resp.Bindings.Content.ProviderID != "mangadex" {
		t.Errorf("expected content=mangadex, got %+v", resp.Bindings.Content)
	}

	// Source directory is gone.
	srcDir, err := h.lib.MangaDir(srcID)
	if err != nil {
		t.Fatalf("MangaDir: %v", err)
	}
	if _, err := os.Stat(srcDir); !os.IsNotExist(err) {
		t.Errorf("expected source dir removed, stat err=%v", err)
	}

	// Chapter dir moved to keep.
	keepDir, err := h.lib.MangaDir(keepID)
	if err != nil {
		t.Fatalf("MangaDir keep: %v", err)
	}
	movedChapter := filepath.Join(keepDir, "mangafox", "ch-1", "meta.json")
	if _, err := os.Stat(movedChapter); err != nil {
		t.Errorf("expected moved chapter meta at %s, got err=%v", movedChapter, err)
	}
}

func TestMergeLibraryManga_KeepInSources(t *testing.T) {
	h, e := setupTestHandler(t)

	keepID := "merge-keep-2"
	writeMangaWithProviders(t, h, keepID, nil, nil)

	body := map[string]interface{}{
		"keep_manga_id":    keepID,
		"source_manga_ids": []string{keepID},
		"content_provider": map[string]string{
			"provider_id":       "mangadex",
			"provider_manga_id": "x",
		},
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/merge", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMergeLibraryManga_KeepNotFound(t *testing.T) {
	_, e := setupTestHandler(t)

	body := map[string]interface{}{
		"keep_manga_id":    "missing-keep",
		"source_manga_ids": []string{"src-x"},
		"content_provider": map[string]string{
			"provider_id":       "mangadex",
			"provider_manga_id": "x",
		},
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/merge", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMergeLibraryManga_SourceNotFound(t *testing.T) {
	h, e := setupTestHandler(t)

	keepID := "merge-keep-3"
	writeMangaWithProviders(t, h, keepID,
		[]library.ProviderRef{{ProviderID: "mangadex", ProviderMangaID: "md-1"}},
		&library.ContentSource{ProviderID: "mangadex", ProviderMangaID: "md-1"},
	)

	body := map[string]interface{}{
		"keep_manga_id":    keepID,
		"source_manga_ids": []string{"missing-src"},
		"content_provider": map[string]string{
			"provider_id":       "mangadex",
			"provider_manga_id": "md-1",
		},
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/merge", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMergeLibraryManga_InvalidContentProvider(t *testing.T) {
	h, e := setupTestHandler(t)

	keepID := "merge-keep-4"
	srcID := "merge-src-4"

	writeMangaWithProviders(t, h, keepID,
		[]library.ProviderRef{{ProviderID: "mangadex", ProviderMangaID: "md-1"}},
		&library.ContentSource{ProviderID: "mangadex", ProviderMangaID: "md-1"},
	)
	writeMangaWithProviders(t, h, srcID,
		[]library.ProviderRef{{ProviderID: "mangafox", ProviderMangaID: "mf-1"}},
		&library.ContentSource{ProviderID: "mangafox", ProviderMangaID: "mf-1"},
	)

	body := map[string]interface{}{
		"keep_manga_id":    keepID,
		"source_manga_ids": []string{srcID},
		"content_provider": map[string]string{
			"provider_id":       "unknownprov",
			"provider_manga_id": "unk-1",
		},
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/merge", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify neither side was touched.
	if _, err := h.lib.GetManga(srcID); err != nil {
		t.Errorf("source should still exist after invalid CP rejection, got err=%v", err)
	}
}

func TestMergeLibraryManga_ContentProviderInferredFromKeep(t *testing.T) {
	h, e := setupTestHandler(t)

	keepID := "merge-keep-cp-infer"
	srcID := "merge-src-cp-infer"

	writeMangaWithProviders(t, h, keepID,
		[]library.ProviderRef{{ProviderID: "mangadex", ProviderMangaID: "md-1"}},
		&library.ContentSource{ProviderID: "mangadex", ProviderMangaID: "md-1"},
	)
	writeMangaWithProviders(t, h, srcID,
		[]library.ProviderRef{{ProviderID: "mangafox", ProviderMangaID: "mf-1"}},
		nil,
	)

	// Omit content_provider; binding should be inferred from keep.Content.
	body := map[string]interface{}{
		"keep_manga_id":    keepID,
		"source_manga_ids": []string{srcID},
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/merge", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Bindings library.MangaBindings `json:"bindings"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Bindings.Content == nil || resp.Bindings.Content.ProviderID != "mangadex" || resp.Bindings.Content.ProviderMangaID != "md-1" {
		t.Errorf("expected content=mangadex/md-1 (inferred from keep), got %+v", resp.Bindings.Content)
	}
	if len(resp.Bindings.Providers) != 2 {
		t.Errorf("expected 2 providers after merge (mangadex+mangafox), got %d", len(resp.Bindings.Providers))
	}
}

func TestMergeLibraryManga_ContentProviderCannotBeInferred(t *testing.T) {
	h, e := setupTestHandler(t)

	keepID := "merge-keep-cp-missing"
	srcID := "merge-src-cp-missing"

	// Keep has no Content; explicit content_provider also omitted.
	writeMangaWithProviders(t, h, keepID,
		[]library.ProviderRef{{ProviderID: "mangadex", ProviderMangaID: "md-1"}},
		nil,
	)
	writeMangaWithProviders(t, h, srcID,
		[]library.ProviderRef{{ProviderID: "mangafox", ProviderMangaID: "mf-1"}},
		nil,
	)

	body := map[string]interface{}{
		"keep_manga_id":    keepID,
		"source_manga_ids": []string{srcID},
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/merge", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}

	wantSubstr := "cannot infer content_provider"
	if !strings.Contains(rec.Body.String(), wantSubstr) {
		t.Errorf("expected error containing %q, got %s", wantSubstr, rec.Body.String())
	}

	// Neither side should be touched.
	if _, err := h.lib.GetManga(srcID); err != nil {
		t.Errorf("source should still exist after inference failure, got err=%v", err)
	}
}

func TestMergeLibraryManga_ChapterCollision_KeepWins(t *testing.T) {
	h, e := setupTestHandler(t)

	keepID := "merge-keep-collision-1"
	srcID := "merge-src-collision-1"

	writeMangaWithProviders(t, h, keepID,
		[]library.ProviderRef{{ProviderID: "mangadex", ProviderMangaID: "md-1"}},
		&library.ContentSource{ProviderID: "mangadex", ProviderMangaID: "md-1"},
	)
	writeMangaWithProviders(t, h, srcID,
		[]library.ProviderRef{{ProviderID: "mangadex", ProviderMangaID: "md-1"}},
		&library.ContentSource{ProviderID: "mangadex", ProviderMangaID: "md-1"},
	)

	older := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	// Keep has an older chapter; source has the same (provider_id, chapter_id)
	// with a newer timestamp and distinct files. After merge, keep's chapter
	// must win — the source chapter dir is silently dropped because the
	// source manga dir is being deleted.
	if err := h.lib.SaveChapter(keepID, "mangadex", "ch-shared", &library.ChapterMeta{
		Title:      "Keep Ch",
		Number:     1,
		UploadDate: older,
	}); err != nil {
		t.Fatalf("seed keep chapter: %v", err)
	}
	if err := h.lib.SaveChapter(srcID, "mangadex", "ch-shared", &library.ChapterMeta{
		Title:      "Source Ch",
		Number:     1,
		UploadDate: newer,
	}); err != nil {
		t.Fatalf("seed src chapter: %v", err)
	}
	// Give keep's chapter a page file so we can confirm it's the kept bytes.
	keepDir, err := h.lib.MangaDir(keepID)
	if err != nil {
		t.Fatalf("MangaDir keep: %v", err)
	}
	keepPage := filepath.Join(keepDir, "mangadex", "ch-shared", "001.jpg")
	if err := os.WriteFile(keepPage, []byte("keep-page-bytes"), 0o644); err != nil {
		t.Fatalf("write keep page: %v", err)
	}
	srcDir, err := h.lib.MangaDir(srcID)
	if err != nil {
		t.Fatalf("MangaDir src: %v", err)
	}
	srcPage := filepath.Join(srcDir, "mangadex", "ch-shared", "001.jpg")
	if err := os.WriteFile(srcPage, []byte("src-page-bytes-XYZ"), 0o644); err != nil {
		t.Fatalf("write src page: %v", err)
	}

	body := map[string]interface{}{
		"keep_manga_id":    keepID,
		"source_manga_ids": []string{srcID},
		"content_provider": map[string]string{
			"provider_id":       "mangadex",
			"provider_manga_id": "md-1",
		},
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/merge", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Source dir removed (chapter dir contents swept along with it).
	if _, err := os.Stat(srcDir); !os.IsNotExist(err) {
		t.Errorf("expected source dir removed after merge, stat err=%v", err)
	}

	// Keep's chapter preserves its title and timestamp despite source being newer.
	meta, err := h.lib.GetChapter(keepID, "mangadex", "ch-shared")
	if err != nil {
		t.Fatalf("get merged chapter: %v", err)
	}
	if meta.Title != "Keep Ch" {
		t.Errorf("keep should win collision, got title=%q", meta.Title)
	}
	if !meta.UploadDate.Equal(older) {
		t.Errorf("expected upload_date=%v, got %v", older, meta.UploadDate)
	}

	// Keep's chapter page bytes are intact (not overwritten by source).
	gotBytes, err := os.ReadFile(keepPage)
	if err != nil {
		t.Fatalf("read keep page after merge: %v", err)
	}
	if string(gotBytes) != "keep-page-bytes" {
		t.Errorf("expected keep's page bytes preserved, got %q", string(gotBytes))
	}
}

func TestMergeLibraryManga_CoverReplaced(t *testing.T) {
	h, e := setupTestHandler(t)

	keepID := "merge-keep-7"
	srcID := "merge-src-7"

	writeMangaWithProviders(t, h, keepID,
		[]library.ProviderRef{{ProviderID: "mangadex", ProviderMangaID: "md-1"}},
		&library.ContentSource{ProviderID: "mangadex", ProviderMangaID: "md-1"},
	)
	writeMangaWithProviders(t, h, srcID,
		[]library.ProviderRef{{ProviderID: "mangafox", ProviderMangaID: "mf-1"}},
		&library.ContentSource{ProviderID: "mangafox", ProviderMangaID: "mf-1"},
	)

	keepDir, err := h.lib.MangaDir(keepID)
	if err != nil {
		t.Fatalf("MangaDir keep: %v", err)
	}
	srcDir, err := h.lib.MangaDir(srcID)
	if err != nil {
		t.Fatalf("MangaDir src: %v", err)
	}

	// Seed different cover files: keep is jpg, source is png.
	if err := os.WriteFile(filepath.Join(keepDir, "cover.jpg"), []byte("keep-jpg-bytes"), 0o644); err != nil {
		t.Fatalf("seed keep cover: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "cover.png"), []byte("src-png-bytes-XYZ"), 0o644); err != nil {
		t.Fatalf("seed src cover: %v", err)
	}

	body := map[string]interface{}{
		"keep_manga_id":    keepID,
		"source_manga_ids": []string{srcID},
		"content_provider": map[string]string{
			"provider_id":       "mangadex",
			"provider_manga_id": "md-1",
		},
		"metadata": map[string]string{
			"cover_url": "source:" + srcID,
		},
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/merge", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// After merge, keep should have cover.png with source's bytes.
	gotBytes, err := os.ReadFile(filepath.Join(keepDir, "cover.png"))
	if err != nil {
		t.Fatalf("read merged cover: %v", err)
	}
	if string(gotBytes) != "src-png-bytes-XYZ" {
		t.Errorf("expected source cover bytes in keep, got %q", string(gotBytes))
	}
	if _, err := os.Stat(filepath.Join(keepDir, "cover.jpg")); !os.IsNotExist(err) {
		t.Errorf("expected keep's old cover.jpg removed, stat err=%v", err)
	}
}

func TestMergeLibraryManga_ContentProviderChange(t *testing.T) {
	h, e := setupTestHandler(t)

	keepID := "merge-keep-8"
	srcID := "merge-src-8"

	writeMangaWithProviders(t, h, keepID,
		[]library.ProviderRef{{ProviderID: "mangadex", ProviderMangaID: "md-1"}},
		&library.ContentSource{ProviderID: "mangadex", ProviderMangaID: "md-1"},
	)
	writeMangaWithProviders(t, h, srcID,
		[]library.ProviderRef{
			{ProviderID: "mangadex", ProviderMangaID: "md-1"},
			{ProviderID: "mangafox", ProviderMangaID: "mf-1"},
		},
		&library.ContentSource{ProviderID: "mangafox", ProviderMangaID: "mf-1"},
	)

	// Seed chapters on source's mangafox namespace — this is the new content provider.
	if err := h.lib.SaveChapter(srcID, "mangafox", "ch-mf-1", &library.ChapterMeta{
		Title: "MangaFox Ch1", Number: 1,
	}); err != nil {
		t.Fatalf("seed src mangafox chapter: %v", err)
	}
	if err := h.lib.SaveChapter(srcID, "mangafox", "ch-mf-2", &library.ChapterMeta{
		Title: "MangaFox Ch2", Number: 2,
	}); err != nil {
		t.Fatalf("seed src mangafox chapter 2: %v", err)
	}

	body := map[string]interface{}{
		"keep_manga_id":    keepID,
		"source_manga_ids": []string{srcID},
		"content_provider": map[string]string{
			"provider_id":       "mangafox",
			"provider_manga_id": "mf-1",
		},
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/merge", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Chapters should now live under keep/mangafox/.
	keepDir, err := h.lib.MangaDir(keepID)
	if err != nil {
		t.Fatalf("MangaDir: %v", err)
	}
	for _, ch := range []string{"ch-mf-1", "ch-mf-2"} {
		p := filepath.Join(keepDir, "mangafox", ch, "meta.json")
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected chapter %s under keep, err=%v", ch, err)
		}
	}

	// Source dir removed.
	if _, err := os.Stat(srcDir(srcID, h)); !os.IsNotExist(err) {
		t.Errorf("expected source dir removed, stat err=%v", err)
	}

	// Source's chapter meta should now report from keep.
	meta, err := h.lib.GetChapter(keepID, "mangafox", "ch-mf-1")
	if err != nil {
		t.Fatalf("get chapter from keep: %v", err)
	}
	if meta.Title != "MangaFox Ch1" {
		t.Errorf("expected title=MangaFox Ch1, got %q", meta.Title)
	}
}

func TestMergeLibraryManga_MultipleSources(t *testing.T) {
	h, e := setupTestHandler(t)

	keepID := "merge-keep-9"
	srcA := "merge-src-9a"
	srcB := "merge-src-9b"

	writeMangaWithProviders(t, h, keepID,
		[]library.ProviderRef{{ProviderID: "mangadex", ProviderMangaID: "md-1"}},
		&library.ContentSource{ProviderID: "mangadex", ProviderMangaID: "md-1"},
		func(m *library.Manga) { m.Metadata.Tags = []string{"action"} },
	)
	writeMangaWithProviders(t, h, srcA,
		[]library.ProviderRef{{ProviderID: "anilist", ProviderMangaID: "al-1"}},
		nil,
		func(m *library.Manga) { m.Metadata.Tags = []string{"comedy"} },
	)
	writeMangaWithProviders(t, h, srcB,
		[]library.ProviderRef{{ProviderID: "mal", ProviderMangaID: "mal-1"}},
		nil,
		func(m *library.Manga) { m.Metadata.Tags = []string{"drama"} },
	)

	body := map[string]interface{}{
		"keep_manga_id":    keepID,
		"source_manga_ids": []string{srcA, srcB},
		"content_provider": map[string]string{
			"provider_id":       "mangadex",
			"provider_manga_id": "md-1",
		},
		"metadata": map[string]string{
			"tags": "merge",
		},
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/merge", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	updated, err := h.lib.GetManga(keepID)
	if err != nil {
		t.Fatalf("get keep: %v", err)
	}
	if len(updated.Bindings.Providers) != 3 {
		t.Errorf("expected 3 providers (mangadex+anilist+mal), got %d (%v)", len(updated.Bindings.Providers), updated.Bindings.Providers)
	}
	// Tags: action, comedy, drama (deduped)
	if len(updated.Metadata.Tags) != 3 {
		t.Errorf("expected 3 merged tags, got %d (%v)", len(updated.Metadata.Tags), updated.Metadata.Tags)
	}
	// Both source dirs gone.
	for _, sid := range []string{srcA, srcB} {
		if _, err := os.Stat(srcDir(sid, h)); !os.IsNotExist(err) {
			t.Errorf("expected %s dir removed, stat err=%v", sid, err)
		}
	}
}

func TestMergeLibraryManga_AliasesUnionDedup(t *testing.T) {
	h, e := setupTestHandler(t)

	keepID := "merge-keep-10"
	srcID := "merge-src-10"

	writeMangaWithProviders(t, h, keepID,
		[]library.ProviderRef{{ProviderID: "mangadex", ProviderMangaID: "md-1"}},
		&library.ContentSource{ProviderID: "mangadex", ProviderMangaID: "md-1"},
		func(m *library.Manga) {
			m.Metadata.Aliases = []string{"Naruto", "NARUTO", "  naruto  "}
		},
	)
	writeMangaWithProviders(t, h, srcID,
		[]library.ProviderRef{{ProviderID: "mangafox", ProviderMangaID: "mf-1"}},
		nil,
		func(m *library.Manga) {
			m.Metadata.Aliases = []string{"ナルト", "Naruto"}
		},
	)

	body := map[string]interface{}{
		"keep_manga_id":    keepID,
		"source_manga_ids": []string{srcID},
		"content_provider": map[string]string{
			"provider_id":       "mangadex",
			"provider_manga_id": "md-1",
		},
		"metadata": map[string]string{
			"aliases": "merge",
		},
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/merge", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	updated, err := h.lib.GetManga(keepID)
	if err != nil {
		t.Fatalf("get keep: %v", err)
	}
	// Expect case-insensitive dedup: Naruto (first-seen casing), ナルト.
	if len(updated.Metadata.Aliases) != 2 {
		t.Errorf("expected 2 deduped aliases (Naruto, ナルト), got %d (%v)", len(updated.Metadata.Aliases), updated.Metadata.Aliases)
	}
}

func TestMergeLibraryManga_PulledPagesMoved(t *testing.T) {
	h, e := setupTestHandler(t)

	keepID := "merge-keep-11"
	srcID := "merge-src-11"

	writeMangaWithProviders(t, h, keepID,
		[]library.ProviderRef{{ProviderID: "mangadex", ProviderMangaID: "md-1"}},
		&library.ContentSource{ProviderID: "mangadex", ProviderMangaID: "md-1"},
	)
	writeMangaWithProviders(t, h, srcID,
		[]library.ProviderRef{{ProviderID: "mangafox", ProviderMangaID: "mf-1"}},
		nil,
	)

	// Seed a chapter on source with downloaded page images and pages.json.
	srcChapterID := "ch-downloaded"
	if err := h.lib.SaveChapter(srcID, "mangafox", srcChapterID, &library.ChapterMeta{
		Title:           "Pulled Chapter",
		Number:          1,
		PageCount:       3,
		IsDownloaded:    true,
		DownloadedPages: 3,
	}); err != nil {
		t.Fatalf("seed chapter meta: %v", err)
	}
	if err := h.lib.SaveChapterPages(srcID, "mangafox", srcChapterID, []library.PageItem{
		{Index: 1, URL: "https://example.com/1.jpg"},
		{Index: 2, URL: "https://example.com/2.jpg"},
		{Index: 3, URL: "https://example.com/3.jpg"},
	}); err != nil {
		t.Fatalf("seed chapter pages: %v", err)
	}
	// Write 3 actual page image files.
	srcDirPath, err := h.lib.MangaDir(srcID)
	if err != nil {
		t.Fatalf("src MangaDir: %v", err)
	}
	pageBytes := []byte("binary-page-data-XYZ")
	for i := 1; i <= 3; i++ {
		p := filepath.Join(srcDirPath, "mangafox", srcChapterID, fmt.Sprintf("%d.jpg", i))
		if err := os.WriteFile(p, pageBytes, 0o644); err != nil {
			t.Fatalf("write page file: %v", err)
		}
	}

	body := map[string]interface{}{
		"keep_manga_id":    keepID,
		"source_manga_ids": []string{srcID},
		"content_provider": map[string]string{
			"provider_id":       "mangadex",
			"provider_manga_id": "md-1",
		},
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/manga/merge", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// After merge, page images must physically exist at the keep location with
	// identical byte content.
	keepDirPath, err := h.lib.MangaDir(keepID)
	if err != nil {
		t.Fatalf("keep MangaDir: %v", err)
	}
	for i := 1; i <= 3; i++ {
		p := filepath.Join(keepDirPath, "mangafox", srcChapterID, fmt.Sprintf("%d.jpg", i))
		got, err := os.ReadFile(p)
		if err != nil {
			t.Errorf("expected moved page file at %s, err=%v", p, err)
			continue
		}
		if string(got) != string(pageBytes) {
			t.Errorf("page %d content mismatch: got %q, want %q", i, got, pageBytes)
		}
	}

	// And pages.json should also be at the new location.
	pagesPath := filepath.Join(keepDirPath, "mangafox", srcChapterID, "pages.json")
	pagesBytes, err := os.ReadFile(pagesPath)
	if err != nil {
		t.Fatalf("read moved pages.json: %v", err)
	}
	var pages []library.PageItem
	if err := json.Unmarshal(pagesBytes, &pages); err != nil {
		t.Fatalf("unmarshal pages: %v", err)
	}
	if len(pages) != 3 {
		t.Errorf("expected 3 pages after move, got %d", len(pages))
	}

	// Chapter meta also at the new location, still reports downloaded.
	meta, err := h.lib.GetChapter(keepID, "mangafox", srcChapterID)
	if err != nil {
		t.Fatalf("get merged chapter: %v", err)
	}
	if !meta.IsDownloaded {
		t.Errorf("expected merged chapter IsDownloaded=true, got false")
	}
	if meta.DownloadedPages != 3 {
		t.Errorf("expected DownloadedPages=3, got %d", meta.DownloadedPages)
	}

	// Source dir gone.
	if _, err := os.Stat(srcDirPath); !os.IsNotExist(err) {
		t.Errorf("expected source dir removed, stat err=%v", err)
	}
}

// srcDir is a small helper for tests that need to assert on a manga's directory.
func srcDir(mangaID string, h *Handler) string {
	d, err := h.lib.MangaDir(mangaID)
	if err != nil {
		return ""
	}
	return d
}
