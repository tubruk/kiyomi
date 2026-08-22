package library

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestLibraryCRUD(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kiyomi-library-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	lib := NewLibrary(tempDir)

	// Test SaveManga
	mangaID := "manga-01"
	mangaMeta := &MangaMeta{
		Title:       "Test Manga",
		Description: "Sample description",
		Authors:     []string{"Author 1"},
		Artists:     []string{"Artist 1"},
		Tags:        []string{"Action", "Fantasy"},
		UserStatus:   "reading",
		UserRating:   9.0,
		UserFavorite: true,
		Content: &ContentSource{
			ProviderID:      "mangadex",
			ProviderMangaID: "remote-manga-123",
			ReadingMode:     "rtl",
			LastSyncedAt:    time.Now().Truncate(time.Second),
		},
	}

	if err := lib.SaveManga(mangaID, mangaMeta); err != nil {
		t.Fatalf("failed to save manga: %v", err)
	}

	// Test GetManga
	gotMeta, err := lib.GetManga(mangaID)
	if err != nil {
		t.Fatalf("failed to get manga: %v", err)
	}

	if gotMeta.Title != mangaMeta.Title {
		t.Errorf("expected title %q, got %q", mangaMeta.Title, gotMeta.Title)
	}
	if gotMeta.UserFavorite != mangaMeta.UserFavorite {
		t.Errorf("expected user_favorite %v, got %v", mangaMeta.UserFavorite, gotMeta.UserFavorite)
	}
	if gotMeta.UserStatus != mangaMeta.UserStatus {
		t.Errorf("expected user_status %q, got %q", mangaMeta.UserStatus, gotMeta.UserStatus)
	}
	if gotMeta.UserRating != mangaMeta.UserRating {
		t.Errorf("expected user_rating %v, got %v", mangaMeta.UserRating, gotMeta.UserRating)
	}
	if gotMeta.Content == nil || gotMeta.Content.ProviderMangaID != mangaMeta.Content.ProviderMangaID {
		t.Errorf("content binding mismatch: got %+v", gotMeta.Content)
	}
	if gotMeta.Content.ReadingMode != "rtl" {
		t.Errorf("expected reading_mode %q, got %q", "rtl", gotMeta.Content.ReadingMode)
	}

	// Test SaveChapter
	chapterID := "ch-01"
	uploadTime := time.Date(2023, 11, 14, 22, 13, 20, 0, time.UTC)
	chapterMeta := &ChapterMeta{
		Title:       "Chapter 1",
		Number:      1.0,
		UploadDate:  uploadTime,
		SourceOrder: 1,
		PageCount:   20,
		PageFormat:  "jpg",
		Content: &ContentSource{
			ProviderID:   "mangadex",
			ChapterRef:   "remote-ch-456",
			LastSyncedAt: time.Now().Truncate(time.Second),
		},
	}

	if err := lib.SaveChapter(mangaID, chapterID, chapterMeta); err != nil {
		t.Fatalf("failed to save chapter: %v", err)
	}

	// Test GetChapter
	gotChapterMeta, err := lib.GetChapter(mangaID, chapterID)
	if err != nil {
		t.Fatalf("failed to get chapter: %v", err)
	}

	if gotChapterMeta.Title != chapterMeta.Title {
		t.Errorf("expected chapter title %q, got %q", chapterMeta.Title, gotChapterMeta.Title)
	}
	if !gotChapterMeta.UploadDate.Equal(uploadTime) {
		t.Errorf("expected upload date %v, got %v", uploadTime, gotChapterMeta.UploadDate)
	}
	if gotChapterMeta.SourceOrder != 1 {
		t.Errorf("expected source order 1, got %d", gotChapterMeta.SourceOrder)
	}
	if gotChapterMeta.Content == nil || gotChapterMeta.Content.ChapterRef != chapterMeta.Content.ChapterRef {
		t.Errorf("chapter content binding mismatch")
	}

	// Test ListManga
	mangas, err := lib.ListManga()
	if err != nil {
		t.Fatalf("failed to list manga: %v", err)
	}
	if len(mangas) != 1 {
		t.Errorf("expected 1 manga, got %d", len(mangas))
	}
	if mangas[0].ID != mangaID {
		t.Errorf("expected manga ID %q, got %q", mangaID, mangas[0].ID)
	}

	// Test ListChapters
	chapters, err := lib.ListChapters(mangaID)
	if err != nil {
		t.Fatalf("failed to list chapters: %v", err)
	}
	if len(chapters) != 1 {
		t.Errorf("expected 1 chapter, got %d", len(chapters))
	}
	if chapters[0].ID != chapterID {
		t.Errorf("expected chapter ID %q, got %q", chapterID, chapters[0].ID)
	}

	// Test DeleteChapter
	if err := lib.DeleteChapter(mangaID, chapterID); err != nil {
		t.Fatalf("failed to delete chapter: %v", err)
	}
	chaptersAfterDelete, err := lib.ListChapters(mangaID)
	if err != nil {
		t.Fatalf("failed to list chapters after delete: %v", err)
	}
	if len(chaptersAfterDelete) != 0 {
		t.Errorf("expected 0 chapters, got %d", len(chaptersAfterDelete))
	}

	// Test DeleteManga
	if err := lib.DeleteManga(mangaID); err != nil {
		t.Fatalf("failed to delete manga: %v", err)
	}
	mangasAfterDelete, err := lib.ListManga()
	if err != nil {
		t.Fatalf("failed to list manga after delete: %v", err)
	}
	if len(mangasAfterDelete) != 0 {
		t.Errorf("expected 0 mangas, got %d", len(mangasAfterDelete))
	}
}

func TestSanitizeID(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"one_piece", "one_piece"},
		{"/manga/one_piece/", "one_piece"},
		{"manga/one_piece/", "one_piece"},
		{"/one_piece/", "one_piece"},
		{"https://fanfox.net/manga/one_piece/", "one_piece"},
		{"manga-01", "manga-01"},
	}

	for _, tt := range tests {
		got := sanitizeID(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeID(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}

	tempDir, err := os.MkdirTemp("", "kiyomi-library-sanitize-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	lib := NewLibrary(tempDir)
	rawID := "/manga/one_piece/"
	meta := &MangaMeta{Title: "One Piece"}

	if err := lib.SaveManga(rawID, meta); err != nil {
		t.Fatalf("SaveManga failed: %v", err)
	}

	mangas, err := lib.ListManga()
	if err != nil {
		t.Fatalf("ListManga failed: %v", err)
	}
	if len(mangas) != 1 {
		t.Fatalf("expected 1 manga, got %d", len(mangas))
	}
	if mangas[0].ID != "one_piece" {
		t.Errorf("expected manga ID %q, got %q", "one_piece", mangas[0].ID)
	}
}

func TestIsValidUserStatus(t *testing.T) {
	validStatuses := []string{
		UserStatusUnread,
		UserStatusReading,
		UserStatusCompleted,
		UserStatusOnHold,
		UserStatusDropped,
		UserStatusPlanToRead,
	}

	for _, s := range validStatuses {
		if !IsValidUserStatus(s) {
			t.Errorf("expected IsValidUserStatus(%q) to be true, got false", s)
		}
	}

	invalidStatuses := []string{
		"",
		"reading_now",
		"READING",
		"Plan_To_Read",
		"finished",
		"unknown",
		"invalid",
		" ",
	}

	for _, s := range invalidStatuses {
		if IsValidUserStatus(s) {
			t.Errorf("expected IsValidUserStatus(%q) to be false, got true", s)
		}
	}
}

func TestUpdateChapterProgress(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kiyomi-library-progress-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	lib := NewLibrary(tempDir)
	mangaID := "manga-progress-01"
	chapterID := "ch-progress-01"

	// Initial manga
	mangaMeta := &MangaMeta{
		Title: "Progress Manga",
	}
	if err := lib.SaveManga(mangaID, mangaMeta); err != nil {
		t.Fatalf("failed to save manga: %v", err)
	}

	// Initial chapter
	chapterMeta := &ChapterMeta{
		Title:  "Chapter 1",
		Number: 1.0,
	}
	if err := lib.SaveChapter(mangaID, chapterID, chapterMeta); err != nil {
		t.Fatalf("failed to save chapter: %v", err)
	}

	// Update progress
	info, err := lib.UpdateChapterProgress(mangaID, chapterID, true, 15)
	if err != nil {
		t.Fatalf("failed to update chapter progress: %v", err)
	}

	if info.ID != chapterID {
		t.Errorf("expected chapter ID %q, got %q", chapterID, info.ID)
	}
	if info.MangaID != mangaID {
		t.Errorf("expected manga ID %q, got %q", mangaID, info.MangaID)
	}
	if !info.Meta.IsRead {
		t.Errorf("expected is_read to be true, got %v", info.Meta.IsRead)
	}
	if info.Meta.LastReadPage != 15 {
		t.Errorf("expected last_read_page to be 15, got %d", info.Meta.LastReadPage)
	}
	if info.Meta.LastReadAt.IsZero() {
		t.Errorf("expected last_read_at to be non-zero")
	}

	// Check chapter was persisted
	savedCh, err := lib.GetChapter(mangaID, chapterID)
	if err != nil {
		t.Fatalf("failed to get saved chapter: %v", err)
	}
	if !savedCh.IsRead || savedCh.LastReadPage != 15 || savedCh.LastReadAt.IsZero() {
		t.Errorf("persisted chapter mismatch: %+v", savedCh)
	}

	// Check parent manga was updated with last read info
	savedManga, err := lib.GetManga(mangaID)
	if err != nil {
		t.Fatalf("failed to get saved manga: %v", err)
	}
	if savedManga.LastReadChapterID != chapterID {
		t.Errorf("expected manga LastReadChapterID %q, got %q", chapterID, savedManga.LastReadChapterID)
	}
	if savedManga.LastReadAt.IsZero() {
		t.Errorf("expected manga LastReadAt to be non-zero")
	}

	// Non-existent chapter
	if _, err := lib.UpdateChapterProgress(mangaID, "non-existent-ch", true, 1); err == nil {
		t.Errorf("expected error updating non-existent chapter, got nil")
	}
}

func TestMangaMeta_UnmarshalLegacyReadingDirection(t *testing.T) {
	// Case 1: Legacy reading_direction with no content struct
	jsonData1 := []byte(`{
		"title": "Legacy Manga",
		"reading_direction": "vertical"
	}`)

	var meta1 MangaMeta
	if err := json.Unmarshal(jsonData1, &meta1); err != nil {
		t.Fatalf("failed to unmarshal legacy manga: %v", err)
	}
	if meta1.Content == nil || meta1.Content.ReadingMode != "vertical" {
		t.Fatalf("expected content.reading_mode 'vertical', got %+v", meta1.Content)
	}

	// Case 2: Explicit content.reading_mode takes precedence over legacy reading_direction
	jsonData2 := []byte(`{
		"title": "New Manga",
		"reading_direction": "rtl",
		"content": {
			"provider_id": "mangadex",
			"reading_mode": "longstrip"
		}
	}`)

	var meta2 MangaMeta
	if err := json.Unmarshal(jsonData2, &meta2); err != nil {
		t.Fatalf("failed to unmarshal manga: %v", err)
	}
	if meta2.Content == nil || meta2.Content.ReadingMode != "longstrip" {
		t.Fatalf("expected content.reading_mode 'longstrip', got %+v", meta2.Content)
	}

	// Case 3: Top-level reading_mode with no content struct
	jsonData3 := []byte(`{
		"title": "Direct Reading Mode Manga",
		"reading_mode": "longstrip"
	}`)

	var meta3 MangaMeta
	if err := json.Unmarshal(jsonData3, &meta3); err != nil {
		t.Fatalf("failed to unmarshal direct reading_mode manga: %v", err)
	}
	if meta3.Content == nil || meta3.Content.ReadingMode != "longstrip" {
		t.Fatalf("expected content.reading_mode 'longstrip', got %+v", meta3.Content)
	}

	// Case 4: Marshaling does not output top-level reading_direction or reading_mode
	bytes, err := json.Marshal(&meta1)
	if err != nil {
		t.Fatalf("failed to marshal meta: %v", err)
	}
	var rawMap map[string]any
	if err := json.Unmarshal(bytes, &rawMap); err != nil {
		t.Fatalf("failed to unmarshal into map: %v", err)
	}
	if _, ok := rawMap["reading_direction"]; ok {
		t.Errorf("marshal output should not contain top-level reading_direction: %s", string(bytes))
	}
	if _, ok := rawMap["reading_mode"]; ok {
		t.Errorf("marshal output should not contain top-level reading_mode: %s", string(bytes))
	}
}

func TestListManga_ParallelAndSorted(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kiyomi-list-manga-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	lib := NewLibrary(tempDir)

	// Create 25 manga entries with varying titles
	titles := []string{
		"Solo Leveling", "One Piece", "Attack on Titan", "Berserk", "Naruto",
		"Bleach", "Dragon Ball", "Death Note", "Fullmetal Alchemist", "Hunter x Hunter",
		"Tokyo Ghoul", "Chainsaw Man", "Jujutsu Kaisen", "Demon Slayer", "My Hero Academia",
		"Vinland Saga", "Vagabond", "Monster", "Kingdom", "Haikyuu!!",
		"Slam Dunk", "Gintama", "JoJo's Bizarre Adventure", "Mob Psycho 100", "One Punch Man",
	}

	for i, title := range titles {
		id := fmt.Sprintf("manga-%02d", i)
		if err := lib.SaveManga(id, &MangaMeta{Title: title}); err != nil {
			t.Fatalf("failed to save manga %s: %v", id, err)
		}
	}

	// Add an invalid/corrupted directory (directory with invalid json)
	corruptDir := filepath.Join(tempDir, "corrupted-manga")
	if err := os.MkdirAll(corruptDir, 0o755); err != nil {
		t.Fatalf("failed to create corrupt dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(corruptDir, "meta.json"), []byte("invalid json{"), 0o644); err != nil {
		t.Fatalf("failed to write corrupt meta: %v", err)
	}

	// Add an empty directory (no meta.json)
	emptyDir := filepath.Join(tempDir, "empty-manga")
	if err := os.MkdirAll(emptyDir, 0o755); err != nil {
		t.Fatalf("failed to create empty dir: %v", err)
	}

	// Add a non-directory regular file
	regularFile := filepath.Join(tempDir, "ignore-me.txt")
	if err := os.WriteFile(regularFile, []byte("hello"), 0o644); err != nil {
		t.Fatalf("failed to create regular file: %v", err)
	}

	mangas, err := lib.ListManga()
	if err != nil {
		t.Fatalf("ListManga failed: %v", err)
	}

	if len(mangas) != len(titles) {
		t.Fatalf("expected %d manga, got %d", len(titles), len(mangas))
	}

	// Verify titles are sorted case-insensitively
	for i := 1; i < len(mangas); i++ {
		prev := strings.ToLower(mangas[i-1].Meta.Title)
		curr := strings.ToLower(mangas[i].Meta.Title)
		if prev > curr {
			t.Errorf("manga not sorted at index %d: %q > %q", i, prev, curr)
		}
	}
}

func TestListChapters_ParallelAndSorted(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kiyomi-list-chapters-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	lib := NewLibrary(tempDir)
	mangaID := "test-manga-chapters"

	if err := lib.SaveManga(mangaID, &MangaMeta{Title: "Test Manga"}); err != nil {
		t.Fatalf("failed to save manga: %v", err)
	}

	// Chapter numbers in unsorted order
	chapNums := []float32{
		100.5, 1.0, 0.5, 2.0, 99.0, 3.5, 50.0, 10.0, 20.5, 30.0,
		4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 11.0, 12.0, 13.0, 14.0,
		15.0, 16.0, 17.0, 18.0, 19.0,
	}

	for i, num := range chapNums {
		chID := fmt.Sprintf("ch-%02d", i)
		if err := lib.SaveChapter(mangaID, chID, &ChapterMeta{
			Title:  fmt.Sprintf("Chapter %v", num),
			Number: num,
		}); err != nil {
			t.Fatalf("failed to save chapter %s: %v", chID, err)
		}
	}

	// Add an invalid subdirectory (corrupt meta.json)
	corruptChDir := filepath.Join(tempDir, mangaID, "corrupted-ch")
	if err := os.MkdirAll(corruptChDir, 0o755); err != nil {
		t.Fatalf("failed to create corrupt ch dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(corruptChDir, "meta.json"), []byte("bad-json"), 0o644); err != nil {
		t.Fatalf("failed to write corrupt chapter meta: %v", err)
	}

	// Add an empty chapter directory
	emptyChDir := filepath.Join(tempDir, mangaID, "empty-ch")
	if err := os.MkdirAll(emptyChDir, 0o755); err != nil {
		t.Fatalf("failed to create empty ch dir: %v", err)
	}

	// Add a regular file inside manga directory
	regFile := filepath.Join(tempDir, mangaID, "chapter-notes.txt")
	if err := os.WriteFile(regFile, []byte("notes"), 0o644); err != nil {
		t.Fatalf("failed to create regular file: %v", err)
	}

	chapters, err := lib.ListChapters(mangaID)
	if err != nil {
		t.Fatalf("ListChapters failed: %v", err)
	}

	if len(chapters) != len(chapNums) {
		t.Fatalf("expected %d chapters, got %d", len(chapNums), len(chapters))
	}

	// Verify chapters are sorted by Number ascending
	for i := 1; i < len(chapters); i++ {
		prev := chapters[i-1].Meta.Number
		curr := chapters[i].Meta.Number
		if prev > curr {
			t.Errorf("chapters not sorted at index %d: %v > %v", i, prev, curr)
		}
	}
}

func TestListManga_EmptyAndNonExistent(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kiyomi-empty-manga-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	lib := NewLibrary(tempDir)

	// Empty library
	mangas, err := lib.ListManga()
	if err != nil {
		t.Fatalf("expected no error on empty dir, got: %v", err)
	}
	if len(mangas) != 0 {
		t.Errorf("expected 0 mangas, got %d", len(mangas))
	}

	// Non-existent library root
	nonExistentLib := NewLibrary(filepath.Join(tempDir, "does-not-exist"))
	mangas2, err := nonExistentLib.ListManga()
	if err != nil {
		t.Fatalf("expected no error on non-existent dir, got: %v", err)
	}
	if mangas2 != nil {
		t.Errorf("expected nil mangas, got %+v", mangas2)
	}
}

func TestListChapters_EmptyAndNonExistent(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kiyomi-empty-chapters-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	lib := NewLibrary(tempDir)

	// Non-existent manga
	chapters, err := lib.ListChapters("non-existent-manga")
	if err != nil {
		t.Fatalf("expected no error on non-existent manga, got: %v", err)
	}
	if chapters != nil {
		t.Errorf("expected nil chapters, got %+v", chapters)
	}

	// Manga with no chapters
	if err := lib.SaveManga("empty-manga", &MangaMeta{Title: "Empty"}); err != nil {
		t.Fatalf("failed to save manga: %v", err)
	}
	chapters2, err := lib.ListChapters("empty-manga")
	if err != nil {
		t.Fatalf("expected no error on manga with no chapters, got: %v", err)
	}
	if len(chapters2) != 0 {
		t.Errorf("expected 0 chapters, got %d", len(chapters2))
	}
}

func TestLibrary_ConcurrentRaceCheck(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kiyomi-race-check-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	lib := NewLibrary(tempDir)

	// Setup initial data
	for i := 0; i < 20; i++ {
		mangaID := fmt.Sprintf("race-manga-%02d", i)
		if err := lib.SaveManga(mangaID, &MangaMeta{Title: fmt.Sprintf("Manga %d", i)}); err != nil {
			t.Fatalf("failed to save manga: %v", err)
		}
		for j := 0; j < 10; j++ {
			chID := fmt.Sprintf("ch-%02d", j)
			if err := lib.SaveChapter(mangaID, chID, &ChapterMeta{Title: fmt.Sprintf("Ch %d", j), Number: float32(j)}); err != nil {
				t.Fatalf("failed to save chapter: %v", err)
			}
		}
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, _ = lib.ListManga()
		}()
		go func(idx int) {
			defer wg.Done()
			mangaID := fmt.Sprintf("race-manga-%02d", idx%20)
			_, _ = lib.ListChapters(mangaID)
		}(i)
	}
	wg.Wait()
}

func TestChapterPages_SaveAndGet(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kiyomi-chapter-pages-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	lib := NewLibrary(tempDir)
	mangaID := "manga-test-pages"
	chapterID := "ch-test-pages"

	pages := []PageItem{
		{Index: 0, URL: "https://example.com/p0.jpg", Source: "remote"},
		{Index: 1, URL: "https://example.com/p1.jpg", Source: "remote"},
		{Index: 2, URL: "https://example.com/p2.jpg"},
	}

	if err := lib.SaveChapterPages(mangaID, chapterID, pages); err != nil {
		t.Fatalf("SaveChapterPages failed: %v", err)
	}

	// Verify file was written to expected location
	pagesPath := filepath.Join(tempDir, mangaID, chapterID, "pages.json")
	if _, err := os.Stat(pagesPath); err != nil {
		t.Fatalf("expected pages.json at %s, got err: %v", pagesPath, err)
	}

	gotPages, err := lib.GetChapterPages(mangaID, chapterID)
	if err != nil {
		t.Fatalf("GetChapterPages failed: %v", err)
	}

	if len(gotPages) != len(pages) {
		t.Fatalf("expected %d pages, got %d", len(pages), len(gotPages))
	}

	for i := range pages {
		if gotPages[i].Index != pages[i].Index || gotPages[i].URL != pages[i].URL || gotPages[i].Source != pages[i].Source {
			t.Errorf("page mismatch at %d: expected %+v, got %+v", i, pages[i], gotPages[i])
		}
	}
}

func TestChapterPages_FallbackToPagesDir(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kiyomi-chapter-pages-fallback-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	lib := NewLibrary(tempDir)
	chapterID := "standalone-ch-01"

	pages := []PageItem{
		{Index: 0, URL: "https://cdn.example.com/page1.webp"},
		{Index: 1, URL: "https://cdn.example.com/page2.webp"},
	}

	// Empty mangaID should store in _pages/<chapter_id>.json
	if err := lib.SaveChapterPages("", chapterID, pages); err != nil {
		t.Fatalf("SaveChapterPages with empty mangaID failed: %v", err)
	}

	pagesPath := filepath.Join(tempDir, "_pages", chapterID+".json")
	if _, err := os.Stat(pagesPath); err != nil {
		t.Fatalf("expected file at %s, got err: %v", pagesPath, err)
	}

	gotPages, err := lib.GetChapterPages("", chapterID)
	if err != nil {
		t.Fatalf("GetChapterPages with empty mangaID failed: %v", err)
	}

	if len(gotPages) != len(pages) {
		t.Fatalf("expected %d pages, got %d", len(pages), len(gotPages))
	}
	if gotPages[0].URL != pages[0].URL || gotPages[1].URL != pages[1].URL {
		t.Errorf("page content mismatch: %+v", gotPages)
	}
}

func TestChapterPages_MetaPageCountUpdate(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kiyomi-chapter-pages-count-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	lib := NewLibrary(tempDir)
	mangaID := "manga-count-test"
	chapterID := "ch-count-test"

	// Create manga and chapter meta with 0 page count
	if err := lib.SaveManga(mangaID, &MangaMeta{Title: "Count Test Manga"}); err != nil {
		t.Fatalf("SaveManga failed: %v", err)
	}
	if err := lib.SaveChapter(mangaID, chapterID, &ChapterMeta{
		Title:     "Chapter 1",
		Number:    1.0,
		PageCount: 0,
	}); err != nil {
		t.Fatalf("SaveChapter failed: %v", err)
	}

	pages := []PageItem{
		{Index: 0, URL: "https://example.com/1.png"},
		{Index: 1, URL: "https://example.com/2.png"},
		{Index: 2, URL: "https://example.com/3.png"},
		{Index: 3, URL: "https://example.com/4.png"},
	}

	if err := lib.SaveChapterPages(mangaID, chapterID, pages); err != nil {
		t.Fatalf("SaveChapterPages failed: %v", err)
	}

	// Verify meta.json was updated with page_count = 4
	updatedMeta, err := lib.GetChapter(mangaID, chapterID)
	if err != nil {
		t.Fatalf("GetChapter failed: %v", err)
	}
	if updatedMeta.PageCount != 4 {
		t.Errorf("expected PageCount to be updated to 4, got %d", updatedMeta.PageCount)
	}
}

func TestChapterPages_Errors(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kiyomi-chapter-pages-errors-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	lib := NewLibrary(tempDir)
	mangaID := "manga-err-test"
	chapterID := "ch-err-test"

	// 1. Empty slice error
	if err := lib.SaveChapterPages(mangaID, chapterID, []PageItem{}); err == nil {
		t.Errorf("expected error saving empty page slice, got nil")
	} else if err.Error() != "save chapter pages: empty page list" {
		t.Errorf("expected error 'save chapter pages: empty page list', got: %v", err)
	}

	// 2. Nil slice error
	if err := lib.SaveChapterPages(mangaID, chapterID, nil); err == nil {
		t.Errorf("expected error saving nil page slice, got nil")
	}

	// 3. GetChapterPages non-existent
	if _, err := lib.GetChapterPages(mangaID, "non-existent"); err == nil {
		t.Errorf("expected error getting non-existent pages, got nil")
	}

	// 4. GetChapterPages corrupt/invalid JSON
	corruptDir := filepath.Join(tempDir, mangaID, "corrupt-ch")
	if err := os.MkdirAll(corruptDir, 0o755); err != nil {
		t.Fatalf("failed to create corrupt dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(corruptDir, "pages.json"), []byte("{not valid json"), 0o644); err != nil {
		t.Fatalf("failed to write corrupt pages file: %v", err)
	}

	if _, err := lib.GetChapterPages(mangaID, "corrupt-ch"); err == nil {
		t.Errorf("expected unmarshal error on corrupt pages.json, got nil")
	}
}

func TestAddProvider(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kiyomi-add-provider-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	lib := NewLibrary(tempDir)
	mangaID := "manga-prov-1"

	// Create manga
	if err := lib.SaveManga(mangaID, &MangaMeta{Title: "Provider Test Manga"}); err != nil {
		t.Fatalf("failed to save manga: %v", err)
	}

	// Add first provider
	ref1 := ProviderRef{ProviderID: "mangadex", ProviderMangaID: "md-123", MangaTitle: "Provider Test Manga"}
	if err := lib.AddProvider(mangaID, ref1); err != nil {
		t.Fatalf("failed to add provider: %v", err)
	}

	meta, _ := lib.GetManga(mangaID)
	if len(meta.Providers) != 1 {
		t.Errorf("expected 1 provider, got %d", len(meta.Providers))
	}
	if meta.Providers[0].ProviderID != "mangadex" {
		t.Errorf("expected provider_id mangadex, got %s", meta.Providers[0].ProviderID)
	}

	// Add second provider
	ref2 := ProviderRef{ProviderID: "mangafox", ProviderMangaID: "mf-456", MangaTitle: "Provider Test Manga"}
	if err := lib.AddProvider(mangaID, ref2); err != nil {
		t.Fatalf("failed to add second provider: %v", err)
	}

	meta, _ = lib.GetManga(mangaID)
	if len(meta.Providers) != 2 {
		t.Errorf("expected 2 providers, got %d", len(meta.Providers))
	}

	// Add duplicate should fail
	if err := lib.AddProvider(mangaID, ref1); err == nil {
		t.Errorf("expected error adding duplicate provider, got nil")
	}
}

func TestRemoveProvider(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kiyomi-remove-provider-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	capLookup := func(providerID string) []string {
		if providerID == "mangadex" || providerID == "mangafox" {
			return []string{"metadata", "content", "tracking"}
		}
		return nil
	}

	lib := NewLibrary(tempDir)
	mangaID := "manga-remove-prov"

	// Create manga with multiple providers
	meta := &MangaMeta{
		Title: "Remove Provider Test",
		Providers: []ProviderRef{
			{ProviderID: "mangadex", ProviderMangaID: "md-123", MangaTitle: "Remove Test"},
			{ProviderID: "mangafox", ProviderMangaID: "mf-456", MangaTitle: "Remove Test"},
		},
		Content: &ContentSource{ProviderID: "mangadex", ProviderMangaID: "md-123"},
	}
	if err := lib.SaveManga(mangaID, meta); err != nil {
		t.Fatalf("failed to save manga: %v", err)
	}

	// Remove mangafox (not content provider) should succeed
	if err := lib.RemoveProvider(mangaID, "mangafox", "mf-456", capLookup); err != nil {
		t.Fatalf("failed to remove non-content provider: %v", err)
	}

	meta, _ = lib.GetManga(mangaID)
	if len(meta.Providers) != 1 {
		t.Errorf("expected 1 provider after removal, got %d", len(meta.Providers))
	}

	// Remove mangadex (content provider) should fail since it's the last content-capable one
	if err := lib.RemoveProvider(mangaID, "mangadex", "md-123", capLookup); err == nil {
		t.Errorf("expected error removing last content-capable provider, got nil")
	}

	// Remove non-existent provider should fail
	if err := lib.RemoveProvider(mangaID, "nonexistent", "nx-999", capLookup); err == nil {
		t.Errorf("expected error removing non-existent provider, got nil")
	}
}

func TestSwitchContentProvider(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kiyomi-switch-provider-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	lib := NewLibrary(tempDir)
	mangaID := "manga-switch-prov"

	// Create manga with content provider
	meta := &MangaMeta{
		Title: "Switch Provider Test",
		Content: &ContentSource{ProviderID: "mangadex", ProviderMangaID: "md-123"},
		Providers: []ProviderRef{
			{ProviderID: "mangadex", ProviderMangaID: "md-123", MangaTitle: "Switch Test"},
		},
	}
	if err := lib.SaveManga(mangaID, meta); err != nil {
		t.Fatalf("failed to save manga: %v", err)
	}

	// Switch to mangafox (already in providers)
	if err := lib.SwitchContentProvider(mangaID, "mangafox", "mf-456", ""); err != nil {
		t.Fatalf("failed to switch content provider: %v", err)
	}

	meta, _ = lib.GetManga(mangaID)
	if meta.Content.ProviderID != "mangafox" || meta.Content.ProviderMangaID != "mf-456" {
		t.Errorf("expected content provider mangafox/mf-456, got %s/%s", meta.Content.ProviderID, meta.Content.ProviderMangaID)
	}
	// mangafox should be added to providers
	if len(meta.Providers) != 2 {
		t.Errorf("expected 2 providers after switch, got %d", len(meta.Providers))
	}

	// Switch to mangadex (existing provider in list)
	if err := lib.SwitchContentProvider(mangaID, "mangadex", "md-123", ""); err != nil {
		t.Fatalf("failed to switch back to existing provider: %v", err)
	}

	meta, _ = lib.GetManga(mangaID)
	if meta.Content.ProviderID != "mangadex" {
		t.Errorf("expected content provider mangadex, got %s", meta.Content.ProviderID)
	}
}

func TestHasContentProvider(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kiyomi-has-content-prov-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	capLookup := func(providerID string) []string {
		if providerID == "mangadex" || providerID == "mangafox" {
			return []string{"metadata", "content", "tracking"}
		}
		return nil
	}

	lib := NewLibrary(tempDir)
	mangaID := "manga-has-content"

	// Create manga with one provider
	meta := &MangaMeta{
		Title: "Has Content Test",
		Providers: []ProviderRef{
			{ProviderID: "mangadex", ProviderMangaID: "md-123", MangaTitle: "Has Content Test"},
		},
	}
	if err := lib.SaveManga(mangaID, meta); err != nil {
		t.Fatalf("failed to save manga: %v", err)
	}

	// Check excluding mangadex - should return false (no other content providers)
	has, err := lib.HasContentProvider(mangaID, "mangadex", "md-123", capLookup)
	if err != nil {
		t.Fatalf("HasContentProvider failed: %v", err)
	}
	if has {
		t.Errorf("expected false when excluding only content provider, got true")
	}

	// Add another content provider
	meta.Providers = append(meta.Providers, ProviderRef{ProviderID: "mangafox", ProviderMangaID: "mf-456", MangaTitle: "Has Content Test"})
	lib.SaveManga(mangaID, meta)

	// Now excluding mangadex should return true (mangafox is still there)
	has, err = lib.HasContentProvider(mangaID, "mangadex", "md-123", capLookup)
	if err != nil {
		t.Fatalf("HasContentProvider failed: %v", err)
	}
	if !has {
		t.Errorf("expected true when mangafox still provides content, got false")
	}
}

func TestLibrary_ConcurrentReadWrite(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kiyomi-rw-race-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	lib := NewLibrary(tempDir)

	const numManga = 10
	for i := 0; i < numManga; i++ {
		mangaID := fmt.Sprintf("manga-%d", i)
		if err := lib.SaveManga(mangaID, &MangaMeta{Title: fmt.Sprintf("Title %d", i)}); err != nil {
			t.Fatalf("failed to save manga: %v", err)
		}
		for j := 0; j < 5; j++ {
			chID := fmt.Sprintf("ch-%d", j)
			if err := lib.SaveChapter(mangaID, chID, &ChapterMeta{Title: fmt.Sprintf("Ch %d", j), Number: float32(j)}); err != nil {
				t.Fatalf("failed to save chapter: %v", err)
			}
		}
	}

	var wg sync.WaitGroup
	// Launch manga list readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for k := 0; k < 20; k++ {
				_, _ = lib.ListManga()
			}
		}()
	}

	// Launch chapter and manga readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			mangaID := fmt.Sprintf("manga-%d", idx%numManga)
			for k := 0; k < 20; k++ {
				_, _ = lib.ListChapters(mangaID)
				_, _ = lib.GetManga(mangaID)
				_, _ = lib.GetChapter(mangaID, "ch-0")
			}
		}(i)
	}

	// Launch writers updating progress
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			mangaID := fmt.Sprintf("manga-%d", idx%numManga)
			for k := 0; k < 20; k++ {
				_, _ = lib.UpdateChapterProgress(mangaID, "ch-0", true, k)
			}
		}(i)
	}

	// Launch writers updating pages
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			mangaID := fmt.Sprintf("manga-%d", idx%numManga)
			for k := 0; k < 20; k++ {
				_ = lib.SaveChapterPages(mangaID, "ch-0", []PageItem{{Index: 0, URL: "http://example.com/p0.jpg"}})
			}
		}(i)
	}

	wg.Wait()
}



