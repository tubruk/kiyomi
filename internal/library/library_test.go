package library

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
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
	info := Manga{
		ID: mangaID,
		Metadata: MangaMetadata{
			Title:         "Test Manga",
			Description:   "Sample description",
			Authors:       []string{"Author 1"},
			Artists:       []string{"Artist 1"},
			Tags:          []string{"Action", "Fantasy"},
			Aliases:       []string{"Test Manga Alias"},
			Collections:   []string{"Favorites"},
			Publishers:    []string{"Test Publisher"},
			ReleaseYear:   2021,
			StartDate:     "2021-05-10",
			EndDate:       "2022-12-25",
			Country:       "JP",
			ExternalLinks: []ExternalLink{
				{
					Provider: "mangadex",
					Label:    "MangaDex",
					URL:      "https://mangadex.org/title/remote-manga-123",
				},
			},
		},
		UserState: UserState{
			Status:   "reading",
			Rating:   9.0,
			Favorite: true,
		},
		Bindings: MangaBindings{
			Content: &ContentSource{
				ProviderID:      "mangadex",
				ProviderMangaID: "remote-manga-123",
				ReadingMode:     "rtl",
				LastSyncedAt:    time.Now().Truncate(time.Second),
			},
		},
	}

	if err := lib.SaveManga(mangaID, info); err != nil {
		t.Fatalf("failed to save manga: %v", err)
	}

	// Test GetManga
	got, err := lib.GetManga(mangaID)
	if err != nil {
		t.Fatalf("failed to get manga: %v", err)
	}

	if got.Metadata.Title != info.Metadata.Title {
		t.Errorf("expected title %q, got %q", info.Metadata.Title, got.Metadata.Title)
	}
	if got.UserState.Favorite != info.UserState.Favorite {
		t.Errorf("expected favorite %v, got %v", info.UserState.Favorite, got.UserState.Favorite)
	}
	if got.UserState.Status != info.UserState.Status {
		t.Errorf("expected status %q, got %q", info.UserState.Status, got.UserState.Status)
	}
	if got.UserState.Rating != info.UserState.Rating {
		t.Errorf("expected rating %v, got %v", info.UserState.Rating, got.UserState.Rating)
	}
	if !reflect.DeepEqual(got.Metadata.Publishers, info.Metadata.Publishers) {
		t.Errorf("expected publishers %v, got %v", info.Metadata.Publishers, got.Metadata.Publishers)
	}
	if got.Metadata.StartDate != info.Metadata.StartDate {
		t.Errorf("expected start_date %q, got %q", info.Metadata.StartDate, got.Metadata.StartDate)
	}
	if got.Metadata.EndDate != info.Metadata.EndDate {
		t.Errorf("expected end_date %q, got %q", info.Metadata.EndDate, got.Metadata.EndDate)
	}
	if got.Metadata.Country != info.Metadata.Country {
		t.Errorf("expected country %q, got %q", info.Metadata.Country, got.Metadata.Country)
	}
	if got.Metadata.ReleaseYear != info.Metadata.ReleaseYear {
		t.Errorf("expected release_year %d, got %d", info.Metadata.ReleaseYear, got.Metadata.ReleaseYear)
	}
	if !reflect.DeepEqual(got.Metadata.Aliases, info.Metadata.Aliases) {
		t.Errorf("expected aliases %v, got %v", info.Metadata.Aliases, got.Metadata.Aliases)
	}
	if !reflect.DeepEqual(got.Metadata.Collections, info.Metadata.Collections) {
		t.Errorf("expected collections %v, got %v", info.Metadata.Collections, got.Metadata.Collections)
	}
	if len(got.Metadata.ExternalLinks) != 1 || got.Metadata.ExternalLinks[0].Provider != "mangadex" || got.Metadata.ExternalLinks[0].Label != "MangaDex" || got.Metadata.ExternalLinks[0].URL != "https://mangadex.org/title/remote-manga-123" {
		t.Errorf("expected ExternalLinks %+v, got %+v", info.Metadata.ExternalLinks, got.Metadata.ExternalLinks)
	}
	if got.Bindings.Content == nil || got.Bindings.Content.ProviderMangaID != info.Bindings.Content.ProviderMangaID {
		t.Errorf("content binding mismatch: got %+v", got.Bindings.Content)
	}
	if got.Bindings.Content.ReadingMode != "rtl" {
		t.Errorf("expected reading_mode %q, got %q", "rtl", got.Bindings.Content.ReadingMode)
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

	if err := lib.SaveChapter(mangaID, "mangadex", chapterID, chapterMeta); err != nil {
		t.Fatalf("failed to save chapter: %v", err)
	}

	// Test GetChapter
	gotChapterMeta, err := lib.GetChapter(mangaID, "mangadex", chapterID)
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
	chapters, err := lib.ListChapters(mangaID, "mangadex")
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
	if err := lib.DeleteChapter(mangaID, "mangadex", chapterID); err != nil {
		t.Fatalf("failed to delete chapter: %v", err)
	}
	chaptersAfterDelete, err := lib.ListChapters(mangaID, "mangadex")
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
	info := Manga{ID: rawID, Metadata: MangaMetadata{Title: "One Piece"}}

	if err := lib.SaveManga(rawID, info); err != nil {
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
	info := Manga{Metadata: MangaMetadata{Title: "Progress Manga"}}
	if err := lib.SaveManga(mangaID, info); err != nil {
		t.Fatalf("failed to save manga: %v", err)
	}

	// Initial chapter
	chapterMeta := &ChapterMeta{
		Title:  "Chapter 1",
		Number: 1.0,
	}
	if err := lib.SaveChapter(mangaID, LocalProviderID, chapterID, chapterMeta); err != nil {
		t.Fatalf("failed to save chapter: %v", err)
	}

	// Update progress
	chInfo, err := lib.UpdateChapterProgress(mangaID, LocalProviderID, chapterID, true, 15)
	if err != nil {
		t.Fatalf("failed to update chapter progress: %v", err)
	}

	if chInfo.ID != chapterID {
		t.Errorf("expected chapter ID %q, got %q", chapterID, chInfo.ID)
	}
	if chInfo.MangaID != mangaID {
		t.Errorf("expected manga ID %q, got %q", mangaID, chInfo.MangaID)
	}
	if !chInfo.Meta.IsRead {
		t.Errorf("expected is_read to be true, got %v", chInfo.Meta.IsRead)
	}
	if chInfo.Meta.LastReadPage != 15 {
		t.Errorf("expected last_read_page to be 15, got %d", chInfo.Meta.LastReadPage)
	}
	if chInfo.Meta.LastReadAt.IsZero() {
		t.Errorf("expected last_read_at to be non-zero")
	}

	// Check chapter was persisted
	savedCh, err := lib.GetChapter(mangaID, LocalProviderID, chapterID)
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
	if savedManga.UserState.LastReadChapterID != chapterID {
		t.Errorf("expected manga UserState.LastReadChapterID %q, got %q", chapterID, savedManga.UserState.LastReadChapterID)
	}
	if savedManga.UserState.LastReadAt.IsZero() {
		t.Errorf("expected manga UserState.LastReadAt to be non-zero")
	}

	// Non-existent chapter
	if _, err := lib.UpdateChapterProgress(mangaID, LocalProviderID, "non-existent-ch", true, 1); err == nil {
		t.Errorf("expected error updating non-existent chapter, got nil")
	}
}

func TestMangaMetadata_Normalize(t *testing.T) {
	// Case 1: Nil receiver should not panic
	var nilMeta *MangaMetadata
	nilMeta.Normalize()

	// Case 2: Case-insensitive deduplication, whitespace trimming, and empty string removal
	m := MangaMetadata{
		Title:       "Normalization Test",
		Publishers:  []string{"Shueisha", "shueisha", "VIZ", "viz", "  VIZ  ", ""},
		Authors:     []string{"Eiichiro Oda", "eiichiro oda", "EIICHIRO ODA", "Akira Toriyama", "   "},
		Artists:     []string{"Yusuke Murata", "yusuke murata", "ONE", "one"},
		Tags:        []string{"Action", "action", "ACTION", "Shounen", "shounen", "Adventure"},
		Aliases:     []string{"One Piece", "one piece", "ONE PIECE", "OP"},
		Collections: []string{"Top Manga", "top manga", "Favorites", "favorites"},
	}

	m.Normalize()

	expectedPublishers := []string{"Shueisha", "VIZ"}
	if !reflect.DeepEqual(m.Publishers, expectedPublishers) {
		t.Errorf("expected publishers %v, got %v", expectedPublishers, m.Publishers)
	}

	expectedAuthors := []string{"Eiichiro Oda", "Akira Toriyama"}
	if !reflect.DeepEqual(m.Authors, expectedAuthors) {
		t.Errorf("expected authors %v, got %v", expectedAuthors, m.Authors)
	}

	expectedArtists := []string{"Yusuke Murata", "ONE"}
	if !reflect.DeepEqual(m.Artists, expectedArtists) {
		t.Errorf("expected artists %v, got %v", expectedArtists, m.Artists)
	}

	expectedTags := []string{"Action", "Shounen", "Adventure"}
	if !reflect.DeepEqual(m.Tags, expectedTags) {
		t.Errorf("expected tags %v, got %v", expectedTags, m.Tags)
	}

	expectedAliases := []string{"One Piece", "OP"}
	if !reflect.DeepEqual(m.Aliases, expectedAliases) {
		t.Errorf("expected aliases %v, got %v", expectedAliases, m.Aliases)
	}

	expectedCollections := []string{"Top Manga", "Favorites"}
	if !reflect.DeepEqual(m.Collections, expectedCollections) {
		t.Errorf("expected collections %v, got %v", expectedCollections, m.Collections)
	}
}

func TestLibrary_MetadataFields_RoundTrip(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kiyomi-metadata-roundtrip-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	lib := NewLibrary(tempDir)
	mangaID := "full-metadata-manga"
	original := Manga{
		ID: mangaID,
		Metadata: MangaMetadata{
			Title:         "Full Metadata Manga",
			Aliases:       []string{"Alias 1", "Alias 2"},
			Description:   "Comprehensive metadata description",
			Authors:       []string{"Author One", "Author Two"},
			Artists:       []string{"Artist One", "Artist Two"},
			Tags:          []string{"Action", "Adventure", "Fantasy"},
			Collections:   []string{"Classics", "Must-Read"},
			ContentRating: "safe",
			Publishers:    []string{"Shueisha", "VIZ Media"},
			ReleaseYear:   1997,
			StartDate:     "1997-07-22",
			EndDate:       "2024-12-31",
			Country:       "JP",
			CoverURL:      "https://example.com/cover.jpg",
		},
		UserState: UserState{
			Status:   "reading",
			Rating:   9.5,
			Favorite: true,
			Notes:    "Masterpiece",
			AddedAt:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	if err := lib.SaveManga(mangaID, original); err != nil {
		t.Fatalf("SaveManga failed: %v", err)
	}

	retrieved, err := lib.GetManga(mangaID)
	if err != nil {
		t.Fatalf("GetManga failed: %v", err)
	}

	if !reflect.DeepEqual(retrieved.Metadata.Publishers, original.Metadata.Publishers) {
		t.Errorf("Publishers mismatch: got %v, want %v", retrieved.Metadata.Publishers, original.Metadata.Publishers)
	}
	if retrieved.Metadata.StartDate != original.Metadata.StartDate {
		t.Errorf("StartDate mismatch: got %q, want %q", retrieved.Metadata.StartDate, original.Metadata.StartDate)
	}
	if retrieved.Metadata.EndDate != original.Metadata.EndDate {
		t.Errorf("EndDate mismatch: got %q, want %q", retrieved.Metadata.EndDate, original.Metadata.EndDate)
	}
	if retrieved.Metadata.Country != original.Metadata.Country {
		t.Errorf("Country mismatch: got %q, want %q", retrieved.Metadata.Country, original.Metadata.Country)
	}
	if retrieved.Metadata.ReleaseYear != original.Metadata.ReleaseYear {
		t.Errorf("ReleaseYear mismatch: got %d, want %d", retrieved.Metadata.ReleaseYear, original.Metadata.ReleaseYear)
	}
	if !reflect.DeepEqual(retrieved.Metadata.Authors, original.Metadata.Authors) {
		t.Errorf("Authors mismatch: got %v, want %v", retrieved.Metadata.Authors, original.Metadata.Authors)
	}
	if !reflect.DeepEqual(retrieved.Metadata.Artists, original.Metadata.Artists) {
		t.Errorf("Artists mismatch: got %v, want %v", retrieved.Metadata.Artists, original.Metadata.Artists)
	}
	if !reflect.DeepEqual(retrieved.Metadata.Tags, original.Metadata.Tags) {
		t.Errorf("Tags mismatch: got %v, want %v", retrieved.Metadata.Tags, original.Metadata.Tags)
	}
	if !reflect.DeepEqual(retrieved.Metadata.Aliases, original.Metadata.Aliases) {
		t.Errorf("Aliases mismatch: got %v, want %v", retrieved.Metadata.Aliases, original.Metadata.Aliases)
	}
	if !reflect.DeepEqual(retrieved.Metadata.Collections, original.Metadata.Collections) {
		t.Errorf("Collections mismatch: got %v, want %v", retrieved.Metadata.Collections, original.Metadata.Collections)
	}

	// Update metadata and round-trip again
	original.Metadata.Publishers = []string{"Kodansha"}
	original.Metadata.StartDate = "2000-01-01"
	original.Metadata.EndDate = "2010-01-01"
	original.Metadata.Country = "US"
	original.Metadata.ReleaseYear = 2000

	if err := lib.SaveManga(mangaID, original); err != nil {
		t.Fatalf("SaveManga update failed: %v", err)
	}

	updated, err := lib.GetManga(mangaID)
	if err != nil {
		t.Fatalf("GetManga after update failed: %v", err)
	}
	if !reflect.DeepEqual(updated.Metadata.Publishers, []string{"Kodansha"}) {
		t.Errorf("updated Publishers mismatch: got %v, want %v", updated.Metadata.Publishers, []string{"Kodansha"})
	}
	if updated.Metadata.StartDate != "2000-01-01" {
		t.Errorf("updated StartDate mismatch: got %q, want %q", updated.Metadata.StartDate, "2000-01-01")
	}
	if updated.Metadata.EndDate != "2010-01-01" {
		t.Errorf("updated EndDate mismatch: got %q, want %q", updated.Metadata.EndDate, "2010-01-01")
	}
	if updated.Metadata.Country != "US" {
		t.Errorf("updated Country mismatch: got %q, want %q", updated.Metadata.Country, "US")
	}
	if updated.Metadata.ReleaseYear != 2000 {
		t.Errorf("updated ReleaseYear mismatch: got %d, want %d", updated.Metadata.ReleaseYear, 2000)
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
		if err := lib.SaveManga(id, Manga{Metadata: MangaMetadata{Title: title}}); err != nil {
			t.Fatalf("failed to save manga %s: %v", id, err)
		}
	}

	// Add an invalid/corrupted directory (directory with invalid json)
	corruptDir := filepath.Join(tempDir, "corrupted-manga")
	if err := os.MkdirAll(corruptDir, 0o755); err != nil {
		t.Fatalf("failed to create corrupt dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(corruptDir, "metadata.json"), []byte("invalid json{"), 0o644); err != nil {
		t.Fatalf("failed to write corrupt metadata: %v", err)
	}

	// Add an empty directory (no metadata.json)
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
		prev := strings.ToLower(mangas[i-1].Metadata.Title)
		curr := strings.ToLower(mangas[i].Metadata.Title)
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

	if err := lib.SaveManga(mangaID, Manga{Metadata: MangaMetadata{Title: "Test Manga"}}); err != nil {
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
		if err := lib.SaveChapter(mangaID, LocalProviderID, chID, &ChapterMeta{
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

	chapters, err := lib.ListChapters(mangaID, LocalProviderID)
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
	chapters, err := lib.ListChapters("non-existent-manga", "")
	if err != nil {
		t.Fatalf("expected no error on non-existent manga, got: %v", err)
	}
	if chapters != nil {
		t.Errorf("expected nil chapters, got %+v", chapters)
	}

	// Manga with no chapters
	if err := lib.SaveManga("empty-manga", Manga{Metadata: MangaMetadata{Title: "Empty"}}); err != nil {
		t.Fatalf("failed to save manga: %v", err)
	}
	chapters2, err := lib.ListChapters("empty-manga", "")
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
		if err := lib.SaveManga(mangaID, Manga{Metadata: MangaMetadata{Title: fmt.Sprintf("Manga %d", i)}}); err != nil {
			t.Fatalf("failed to save manga: %v", err)
		}
		for j := 0; j < 10; j++ {
			chID := fmt.Sprintf("ch-%02d", j)
			if err := lib.SaveChapter(mangaID, LocalProviderID, chID, &ChapterMeta{Title: fmt.Sprintf("Ch %d", j), Number: float32(j)}); err != nil {
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
			_, _ = lib.ListChapters(mangaID, LocalProviderID)
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

	if err := lib.SaveChapterPages(mangaID, LocalProviderID, chapterID, pages); err != nil {
		t.Fatalf("SaveChapterPages failed: %v", err)
	}

	// Verify file was written to expected location
	pagesPath := filepath.Join(tempDir, mangaID, LocalProviderID, chapterID, "pages.json")
	if _, err := os.Stat(pagesPath); err != nil {
		t.Fatalf("expected pages.json at %s, got err: %v", pagesPath, err)
	}

	gotPages, err := lib.GetChapterPages(mangaID, LocalProviderID, chapterID)
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
	if err := lib.SaveChapterPages("", LocalProviderID, chapterID, pages); err != nil {
		t.Fatalf("SaveChapterPages with empty mangaID failed: %v", err)
	}

	pagesPath := filepath.Join(tempDir, "_pages", chapterID+".json")
	if _, err := os.Stat(pagesPath); err != nil {
		t.Fatalf("expected file at %s, got err: %v", pagesPath, err)
	}

	gotPages, err := lib.GetChapterPages("", LocalProviderID, chapterID)
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
	if err := lib.SaveManga(mangaID, Manga{Metadata: MangaMetadata{Title: "Count Test Manga"}}); err != nil {
		t.Fatalf("SaveManga failed: %v", err)
	}
	if err := lib.SaveChapter(mangaID, LocalProviderID, chapterID, &ChapterMeta{
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

	if err := lib.SaveChapterPages(mangaID, LocalProviderID, chapterID, pages); err != nil {
		t.Fatalf("SaveChapterPages failed: %v", err)
	}

	// Verify meta.json was updated with page_count = 4
	updatedMeta, err := lib.GetChapter(mangaID, LocalProviderID, chapterID)
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
	if err := lib.SaveChapterPages(mangaID, LocalProviderID, chapterID, []PageItem{}); err == nil {
		t.Errorf("expected error saving empty page slice, got nil")
	} else if err.Error() != "save chapter pages: empty page list" {
		t.Errorf("expected error 'save chapter pages: empty page list', got: %v", err)
	}

	// 2. Nil slice error
	if err := lib.SaveChapterPages(mangaID, LocalProviderID, chapterID, nil); err == nil {
		t.Errorf("expected error saving nil page slice, got nil")
	}

	// 3. GetChapterPages non-existent
	if _, err := lib.GetChapterPages(mangaID, LocalProviderID, "non-existent"); err == nil {
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

	if _, err := lib.GetChapterPages(mangaID, LocalProviderID, "corrupt-ch"); err == nil {
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

	// Create manga metadata (so GetManga works).
	if err := lib.SaveMetadata(mangaID, MangaMetadata{Title: "Provider Test Manga"}); err != nil {
		t.Fatalf("failed to save manga metadata: %v", err)
	}

	// Add first provider
	ref1 := ProviderRef{ProviderID: "mangadex", ProviderMangaID: "md-123", MangaTitle: "Provider Test Manga"}
	if err := lib.AddProvider(mangaID, ref1); err != nil {
		t.Fatalf("failed to add provider: %v", err)
	}

	got, _ := lib.GetManga(mangaID)
	if len(got.Bindings.Providers) != 1 {
		t.Errorf("expected 1 provider, got %d", len(got.Bindings.Providers))
	}
	if got.Bindings.Providers[0].ProviderID != "mangadex" {
		t.Errorf("expected provider_id mangadex, got %s", got.Bindings.Providers[0].ProviderID)
	}

	// Add second provider
	ref2 := ProviderRef{ProviderID: "mangafox", ProviderMangaID: "mf-456", MangaTitle: "Provider Test Manga"}
	if err := lib.AddProvider(mangaID, ref2); err != nil {
		t.Fatalf("failed to add second provider: %v", err)
	}

	got, _ = lib.GetManga(mangaID)
	if len(got.Bindings.Providers) != 2 {
		t.Errorf("expected 2 providers, got %d", len(got.Bindings.Providers))
	}

	// Add duplicate should succeed and update title if provided (idempotent)
	ref1Updated := ProviderRef{ProviderID: "mangadex", ProviderMangaID: "md-123", MangaTitle: "Updated Manga Title"}
	if err := lib.AddProvider(mangaID, ref1Updated); err != nil {
		t.Fatalf("expected nil error on duplicate add provider, got: %v", err)
	}

	got, _ = lib.GetManga(mangaID)
	if len(got.Bindings.Providers) != 2 {
		t.Errorf("expected 2 providers, got %d", len(got.Bindings.Providers))
	}
	if got.Bindings.Providers[0].MangaTitle != "Updated Manga Title" {
		t.Errorf("expected updated title %q, got %q", "Updated Manga Title", got.Bindings.Providers[0].MangaTitle)
	}

	// Re-add duplicate with empty title should leave existing title unchanged
	ref1NoTitle := ProviderRef{ProviderID: "mangadex", ProviderMangaID: "md-123", MangaTitle: ""}
	if err := lib.AddProvider(mangaID, ref1NoTitle); err != nil {
		t.Fatalf("expected nil error on duplicate add provider with empty title, got: %v", err)
	}
	got, _ = lib.GetManga(mangaID)
	if got.Bindings.Providers[0].MangaTitle != "Updated Manga Title" {
		t.Errorf("expected unchanged title %q, got %q", "Updated Manga Title", got.Bindings.Providers[0].MangaTitle)
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

	info := Manga{
		Metadata: MangaMetadata{Title: "Remove Provider Test"},
		Bindings: MangaBindings{
			Providers: []ProviderRef{
				{ProviderID: "mangadex", ProviderMangaID: "md-123", MangaTitle: "Remove Test"},
				{ProviderID: "mangafox", ProviderMangaID: "mf-456", MangaTitle: "Remove Test"},
			},
			Content: &ContentSource{ProviderID: "mangadex", ProviderMangaID: "md-123"},
		},
	}
	if err := lib.SaveManga(mangaID, info); err != nil {
		t.Fatalf("failed to save manga: %v", err)
	}

	// Remove mangafox (not content provider) should succeed
	if err := lib.RemoveProvider(mangaID, "mangafox", "mf-456", capLookup); err != nil {
		t.Fatalf("failed to remove non-content provider: %v", err)
	}

	got, _ := lib.GetManga(mangaID)
	if len(got.Bindings.Providers) != 1 {
		t.Errorf("expected 1 provider after removal, got %d", len(got.Bindings.Providers))
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

	info := Manga{
		Metadata: MangaMetadata{Title: "Switch Provider Test"},
		Bindings: MangaBindings{
			Content: &ContentSource{ProviderID: "mangadex", ProviderMangaID: "md-123"},
			Providers: []ProviderRef{
				{ProviderID: "mangadex", ProviderMangaID: "md-123", MangaTitle: "Switch Test"},
			},
		},
	}
	if err := lib.SaveManga(mangaID, info); err != nil {
		t.Fatalf("failed to save manga: %v", err)
	}

	// Switch to mangafox (already in providers)
	if err := lib.SwitchContentProvider(mangaID, "mangafox", "mf-456", ""); err != nil {
		t.Fatalf("failed to switch content provider: %v", err)
	}

	got, _ := lib.GetManga(mangaID)
	if got.Bindings.Content.ProviderID != "mangafox" || got.Bindings.Content.ProviderMangaID != "mf-456" {
		t.Errorf("expected content provider mangafox/mf-456, got %s/%s", got.Bindings.Content.ProviderID, got.Bindings.Content.ProviderMangaID)
	}
	// mangafox should be added to providers
	if len(got.Bindings.Providers) != 2 {
		t.Errorf("expected 2 providers after switch, got %d", len(got.Bindings.Providers))
	}

	// Switch to mangadex (existing provider in list)
	if err := lib.SwitchContentProvider(mangaID, "mangadex", "md-123", ""); err != nil {
		t.Fatalf("failed to switch back to existing provider: %v", err)
	}

	got, _ = lib.GetManga(mangaID)
	if got.Bindings.Content.ProviderID != "mangadex" {
		t.Errorf("expected content provider mangadex, got %s", got.Bindings.Content.ProviderID)
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

	info := Manga{
		Metadata: MangaMetadata{Title: "Has Content Test"},
		Bindings: MangaBindings{
			Providers: []ProviderRef{
				{ProviderID: "mangadex", ProviderMangaID: "md-123", MangaTitle: "Has Content Test"},
			},
		},
	}
	if err := lib.SaveManga(mangaID, info); err != nil {
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

	// Add another content provider via direct SaveBindings.
	if err := lib.SaveBindings(mangaID, MangaBindings{
		Providers: append(info.Bindings.Providers, ProviderRef{ProviderID: "mangafox", ProviderMangaID: "mf-456", MangaTitle: "Has Content Test"}),
	}); err != nil {
		t.Fatalf("failed to save bindings: %v", err)
	}

	// Now excluding mangadex should return true (mangafox is still there)
	has, err = lib.HasContentProvider(mangaID, "mangadex", "md-123", capLookup)
	if err != nil {
		t.Fatalf("HasContentProvider failed: %v", err)
	}
	if !has {
		t.Errorf("expected true when mangafox still provides content, got false")
	}
}

func TestSyncCover_Success(t *testing.T) {
	coverBytes := []byte("FAKE-JPEG-BYTES")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write(coverBytes)
	}))
	defer srv.Close()

	tempDir, err := os.MkdirTemp("", "kiyomi-cover-test-*")
	if err != nil {
		t.Fatalf("mkdir temp: %v", err)
	}
	defer os.RemoveAll(tempDir)

	lib := NewLibrary(tempDir)
	lib.SetAllowPrivateNetworks(true) // httptest binds 127.0.0.1
	mangaID := "manga-cover-1"

	if err := lib.SyncCover(mangaID, srv.URL+"/cover.jpg"); err != nil {
		t.Fatalf("SyncCover failed: %v", err)
	}

	// Verify file written at expected path
	target := filepath.Join(tempDir, mangaID, "cover.jpg")
	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("expected cover at %s: %v", target, err)
	}
	if info.Size() != int64(len(coverBytes)) {
		t.Errorf("expected cover size %d, got %d", len(coverBytes), info.Size())
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read cover: %v", err)
	}
	if string(got) != string(coverBytes) {
		t.Errorf("cover content mismatch: got %q want %q", got, coverBytes)
	}
}

func TestSyncCover_ExtensionFromURL(t *testing.T) {
	coverBytes := []byte("PNG-BYTES")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream") // force URL-based detection
		_, _ = w.Write(coverBytes)
	}))
	defer srv.Close()

	tempDir, err := os.MkdirTemp("", "kiyomi-cover-ext-*")
	if err != nil {
		t.Fatalf("mkdir temp: %v", err)
	}
	defer os.RemoveAll(tempDir)

	lib := NewLibrary(tempDir)
	lib.SetAllowPrivateNetworks(true) // httptest binds 127.0.0.1
	mangaID := "manga-png"

	if err := lib.SyncCover(mangaID, srv.URL+"/cover.png"); err != nil {
		t.Fatalf("SyncCover failed: %v", err)
	}

	target := filepath.Join(tempDir, mangaID, "cover.png")
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("expected cover at %s: %v", target, err)
	}
}

func TestSyncCover_ExtensionFromContentType(t *testing.T) {
	// URL has no extension, so we rely on Content-Type detection.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/webp")
		_, _ = w.Write([]byte("WEBP-BYTES"))
	}))
	defer srv.Close()

	tempDir, err := os.MkdirTemp("", "kiyomi-cover-ct-*")
	if err != nil {
		t.Fatalf("mkdir temp: %v", err)
	}
	defer os.RemoveAll(tempDir)

	lib := NewLibrary(tempDir)
	lib.SetAllowPrivateNetworks(true) // httptest binds 127.0.0.1
	mangaID := "manga-webp"

	if err := lib.SyncCover(mangaID, srv.URL+"/cover"); err != nil {
		t.Fatalf("SyncCover failed: %v", err)
	}

	target := filepath.Join(tempDir, mangaID, "cover.webp")
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("expected cover at %s: %v", target, err)
	}
}

func TestSyncCover_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	tempDir, err := os.MkdirTemp("", "kiyomi-cover-err-*")
	if err != nil {
		t.Fatalf("mkdir temp: %v", err)
	}
	defer os.RemoveAll(tempDir)

	lib := NewLibrary(tempDir)
	mangaID := "manga-404"

	if err := lib.SyncCover(mangaID, srv.URL+"/cover.jpg"); err == nil {
		t.Errorf("expected error on 404 response, got nil")
	}
}

func TestSyncCover_EmptyURL(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kiyomi-cover-empty-*")
	if err != nil {
		t.Fatalf("mkdir temp: %v", err)
	}
	defer os.RemoveAll(tempDir)

	lib := NewLibrary(tempDir)
	if err := lib.SyncCover("any", ""); err == nil {
		t.Errorf("expected error on empty URL, got nil")
	}
}

func TestHasCover(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kiyomi-hascover-*")
	if err != nil {
		t.Fatalf("mkdir temp: %v", err)
	}
	defer os.RemoveAll(tempDir)

	lib := NewLibrary(tempDir)
	mangaID := "manga-hascover"

	if lib.HasCover(mangaID) {
		t.Errorf("expected HasCover to be false for missing cover")
	}

	// Create a cover.jpg
	mangaDir := filepath.Join(tempDir, mangaID)
	if err := os.MkdirAll(mangaDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(mangaDir, "cover.jpg"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write cover: %v", err)
	}

	if !lib.HasCover(mangaID) {
		t.Errorf("expected HasCover to be true after creating cover.jpg")
	}
	if p := lib.CoverPath(mangaID); p != filepath.Join(mangaDir, "cover.jpg") {
		t.Errorf("CoverPath mismatch: got %q want %q", p, filepath.Join(mangaDir, "cover.jpg"))
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
		if err := lib.SaveManga(mangaID, Manga{Metadata: MangaMetadata{Title: fmt.Sprintf("Title %d", i)}}); err != nil {
			t.Fatalf("failed to save manga: %v", err)
		}
		for j := 0; j < 5; j++ {
			chID := fmt.Sprintf("ch-%d", j)
			if err := lib.SaveChapter(mangaID, LocalProviderID, chID, &ChapterMeta{Title: fmt.Sprintf("Ch %d", j), Number: float32(j)}); err != nil {
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
				_, _ = lib.ListChapters(mangaID, LocalProviderID)
				_, _ = lib.GetManga(mangaID)
				_, _ = lib.GetChapter(mangaID, LocalProviderID, "ch-0")
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
				_, _ = lib.UpdateChapterProgress(mangaID, LocalProviderID, "ch-0", true, k)
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
				_ = lib.SaveChapterPages(mangaID, LocalProviderID, "ch-0", []PageItem{{Index: 0, URL: "http://example.com/p0.jpg"}})
			}
		}(i)
	}

	wg.Wait()
}

func TestBatchUpdateChapterProgress(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kiyomi-library-batch-progress-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	lib := NewLibrary(tempDir)
	mangaID := "batch-progress-manga"

	if err := lib.SaveManga(mangaID, Manga{Metadata: MangaMetadata{Title: "Batch Progress Manga"}}); err != nil {
		t.Fatalf("failed to save manga: %v", err)
	}

	chapterIDs := []string{"ch-01", "ch-02", "ch-03"}
	for i, chID := range chapterIDs {
		if err := lib.SaveChapter(mangaID, LocalProviderID, chID, &ChapterMeta{
			Title:  fmt.Sprintf("Chapter %d", i+1),
			Number: float32(i + 1),
		}); err != nil {
			t.Fatalf("failed to save chapter %s: %v", chID, err)
		}
	}

	// 1. Batch update progress for all 3 chapters
	updated, err := lib.BatchUpdateChapterProgress(mangaID, LocalProviderID, chapterIDs, true, 20)
	if err != nil {
		t.Fatalf("BatchUpdateChapterProgress failed: %v", err)
	}

	if len(updated) != 3 {
		t.Fatalf("expected 3 updated chapters, got %d", len(updated))
	}

	for i, u := range updated {
		if u.ID != chapterIDs[i] {
			t.Errorf("expected chapter ID %q, got %q", chapterIDs[i], u.ID)
		}
		if !u.Meta.IsRead {
			t.Errorf("chapter %s: expected is_read true, got %v", u.ID, u.Meta.IsRead)
		}
		if u.Meta.LastReadPage != 20 {
			t.Errorf("chapter %s: expected last_read_page 20, got %d", u.ID, u.Meta.LastReadPage)
		}
		if u.Meta.LastReadAt.IsZero() {
			t.Errorf("chapter %s: expected last_read_at to be non-zero, got %v", u.ID, u.Meta.LastReadAt)
		}
	}

	// Verify parent manga has LastReadChapterID set to last chapter in list ("ch-03")
	manga, err := lib.GetManga(mangaID)
	if err != nil {
		t.Fatalf("GetManga failed: %v", err)
	}
	if manga.UserState.LastReadChapterID != "ch-03" {
		t.Errorf("expected UserState.LastReadChapterID 'ch-03', got %q", manga.UserState.LastReadChapterID)
	}
	if manga.UserState.LastReadAt.IsZero() {
		t.Errorf("expected UserState.LastReadAt to be set")
	}

	// 2. Error case: non-existent chapter
	_, err = lib.BatchUpdateChapterProgress(mangaID, LocalProviderID, []string{"ch-01", "non-existent-ch"}, true, 1)
	if err == nil {
		t.Fatalf("expected error for non-existent chapter, got nil")
	}
}

func TestBatchDeleteChapters(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kiyomi-library-batch-del-chapters-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	lib := NewLibrary(tempDir)
	mangaID := "batch-del-manga"

	if err := lib.SaveManga(mangaID, Manga{Metadata: MangaMetadata{Title: "Batch Delete Manga"}}); err != nil {
		t.Fatalf("failed to save manga: %v", err)
	}

	chapterIDs := []string{"ch-01", "ch-02", "ch-03", "ch-04"}
	for i, chID := range chapterIDs {
		if err := lib.SaveChapter(mangaID, LocalProviderID, chID, &ChapterMeta{
			Title:  fmt.Sprintf("Chapter %d", i+1),
			Number: float32(i + 1),
		}); err != nil {
			t.Fatalf("failed to save chapter %s: %v", chID, err)
		}
	}

	chapters, _ := lib.ListChapters(mangaID, LocalProviderID)
	if len(chapters) != 4 {
		t.Fatalf("expected 4 chapters initially, got %d", len(chapters))
	}

	// Delete ch-02 and ch-04
	if err := lib.BatchDeleteChapters(mangaID, LocalProviderID, []string{"ch-02", "ch-04"}); err != nil {
		t.Fatalf("BatchDeleteChapters failed: %v", err)
	}

	remaining, err := lib.ListChapters(mangaID, LocalProviderID)
	if err != nil {
		t.Fatalf("ListChapters failed: %v", err)
	}
	if len(remaining) != 2 {
		t.Fatalf("expected 2 remaining chapters, got %d", len(remaining))
	}
	if remaining[0].ID != "ch-01" || remaining[1].ID != "ch-03" {
		t.Errorf("unexpected remaining chapters: %+v", remaining)
	}
}

func TestBatchDeleteChapterFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kiyomi-library-batch-del-files-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	lib := NewLibrary(tempDir)
	mangaID := "batch-files-manga"

	if err := lib.SaveManga(mangaID, Manga{Metadata: MangaMetadata{Title: "Batch Files Manga"}}); err != nil {
		t.Fatalf("failed to save manga: %v", err)
	}

	chapterIDs := []string{"ch-01", "ch-02"}
	for i, chID := range chapterIDs {
		if err := lib.SaveChapter(mangaID, LocalProviderID, chID, &ChapterMeta{
			Title:  fmt.Sprintf("Chapter %d", i+1),
			Number: float32(i + 1),
		}); err != nil {
			t.Fatalf("failed to save chapter %s: %v", chID, err)
		}

		chDir := filepath.Join(tempDir, mangaID, LocalProviderID, chID)
		_ = os.WriteFile(filepath.Join(chDir, "001.jpg"), []byte("jpgdata"), 0o644)
		_ = os.WriteFile(filepath.Join(chDir, "002.png"), []byte("pngdata"), 0o644)
		_ = os.WriteFile(filepath.Join(chDir, "pages.json"), []byte("[]"), 0o644)
	}

	// Delete files for both chapters
	if err := lib.BatchDeleteChapterFiles(mangaID, LocalProviderID, chapterIDs); err != nil {
		t.Fatalf("BatchDeleteChapterFiles failed: %v", err)
	}

	for _, chID := range chapterIDs {
		chDir := filepath.Join(tempDir, mangaID, LocalProviderID, chID)
		// meta.json must still exist
		if _, err := os.Stat(filepath.Join(chDir, "meta.json")); err != nil {
			t.Errorf("chapter %s: meta.json was unexpectedly deleted", chID)
		}
		// 001.jpg, 002.png, pages.json must not exist
		if _, err := os.Stat(filepath.Join(chDir, "001.jpg")); !os.IsNotExist(err) {
			t.Errorf("chapter %s: 001.jpg still exists", chID)
		}
		if _, err := os.Stat(filepath.Join(chDir, "pages.json")); !os.IsNotExist(err) {
			t.Errorf("chapter %s: pages.json still exists", chID)
		}
	}

	// Error when chapter dir doesn't exist
	if err := lib.BatchDeleteChapterFiles(mangaID, LocalProviderID, []string{"non-existent-ch"}); err == nil {
		t.Fatalf("expected error for non-existent chapter dir, got nil")
	}
}

func TestChapterDownloadedPagesTracking(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kiyomi-chapter-download-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	lib := NewLibrary(tempDir)
	mangaID := "manga-track"
	provID := "prov-test"

	if err := lib.SaveManga(mangaID, Manga{Metadata: MangaMetadata{Title: "Tracking Manga"}}); err != nil {
		t.Fatalf("failed to save manga: %v", err)
	}

	// 1. Chapter with 0 pages (no files on disk)
	ch0 := "ch-zero"
	if err := lib.SaveChapter(mangaID, provID, ch0, &ChapterMeta{
		Title:     "Chapter 0",
		Number:    0.0,
		PageCount: 5,
	}); err != nil {
		t.Fatalf("failed to save ch0: %v", err)
	}

	// 2. Chapter with partial pages (3 files on disk, PageCount = 5)
	chPartial := "ch-partial"
	if err := lib.SaveChapter(mangaID, provID, chPartial, &ChapterMeta{
		Title:     "Chapter Partial",
		Number:    1.0,
		PageCount: 5,
	}); err != nil {
		t.Fatalf("failed to save chPartial: %v", err)
	}
	chPartialDir := filepath.Join(tempDir, mangaID, provID, chPartial)
	_ = os.WriteFile(filepath.Join(chPartialDir, "1.jpg"), []byte("data"), 0o644)
	_ = os.WriteFile(filepath.Join(chPartialDir, "2.jpeg"), []byte("data"), 0o644)
	_ = os.WriteFile(filepath.Join(chPartialDir, "3.png"), []byte("data"), 0o644)
	// Non-images to exclude:
	_ = os.WriteFile(filepath.Join(chPartialDir, "pages.json"), []byte("[]"), 0o644)
	_ = os.WriteFile(filepath.Join(chPartialDir, "1.jpg.tmp.1234"), []byte("temp"), 0o644)
	_ = os.WriteFile(filepath.Join(chPartialDir, "notes.txt"), []byte("text"), 0o644)
	_ = os.MkdirAll(filepath.Join(chPartialDir, "subfolder"), 0o755)

	// 3. Chapter with complete pages (5 files on disk, PageCount = 5)
	chComplete := "ch-complete"
	if err := lib.SaveChapter(mangaID, provID, chComplete, &ChapterMeta{
		Title:        "Chapter Complete",
		Number:       2.0,
		PageCount:    5,
		DownloadedAt: time.Now(),
	}); err != nil {
		t.Fatalf("failed to save chComplete: %v", err)
	}
	chCompleteDir := filepath.Join(tempDir, mangaID, provID, chComplete)
	_ = os.WriteFile(filepath.Join(chCompleteDir, "1.webp"), []byte("data"), 0o644)
	_ = os.WriteFile(filepath.Join(chCompleteDir, "2.gif"), []byte("data"), 0o644)
	_ = os.WriteFile(filepath.Join(chCompleteDir, "3.avif"), []byte("data"), 0o644)
	_ = os.WriteFile(filepath.Join(chCompleteDir, "4.jpg"), []byte("data"), 0o644)
	_ = os.WriteFile(filepath.Join(chCompleteDir, "5.png"), []byte("data"), 0o644)

	// 4. Chapter with 0 page count but downloaded files
	chZeroMetaCount := "ch-zero-count"
	if err := lib.SaveChapter(mangaID, provID, chZeroMetaCount, &ChapterMeta{
		Title:     "Chapter Zero Count",
		Number:    3.0,
		PageCount: 0,
	}); err != nil {
		t.Fatalf("failed to save chZeroMetaCount: %v", err)
	}
	chZeroDir := filepath.Join(tempDir, mangaID, provID, chZeroMetaCount)
	_ = os.WriteFile(filepath.Join(chZeroDir, "1.jpg"), []byte("data"), 0o644)

	// Test CountDownloadedPages directly
	if count := lib.CountDownloadedPages(mangaID, provID, ch0); count != 0 {
		t.Errorf("ch0 expected 0 downloaded pages, got %d", count)
	}
	if count := lib.CountDownloadedPages(mangaID, provID, chPartial); count != 3 {
		t.Errorf("chPartial expected 3 downloaded pages, got %d", count)
	}
	if count := lib.CountDownloadedPages(mangaID, provID, chComplete); count != 5 {
		t.Errorf("chComplete expected 5 downloaded pages, got %d", count)
	}
	if count := lib.CountDownloadedPages(mangaID, provID, chZeroMetaCount); count != 1 {
		t.Errorf("chZeroMetaCount expected 1 downloaded page, got %d", count)
	}
	if count := lib.CountDownloadedPages("nonexistent", provID, "nonexistent"); count != 0 {
		t.Errorf("nonexistent chapter expected 0 downloaded pages, got %d", count)
	}

	// Test GetChapterFileStatus directly
	tests := []struct {
		mangaID       string
		chapterID     string
		metaPageCount int
		wantPages     int
		wantDownload  bool
	}{
		{mangaID, ch0, 5, 0, false},
		{mangaID, chPartial, 5, 3, false},
		{mangaID, chComplete, 5, 5, true},
		{mangaID, chComplete, 3, 5, true},      // downloaded > metaPageCount
		{mangaID, chZeroMetaCount, 0, 1, true}, // metaPageCount == 0, downloaded > 0
		{mangaID, ch0, 0, 0, false},            // metaPageCount == 0, downloaded == 0
	}

	for _, tt := range tests {
		pages, downloaded := lib.GetChapterFileStatus(tt.mangaID, provID, tt.chapterID, tt.metaPageCount)
		if pages != tt.wantPages || downloaded != tt.wantDownload {
			t.Errorf("GetChapterFileStatus(%s, %d) = (%d, %v), want (%d, %v)",
				tt.chapterID, tt.metaPageCount, pages, downloaded, tt.wantPages, tt.wantDownload)
		}
	}

	// Test ListChapters populates DownloadedPages and IsDownloaded
	chapters, err := lib.ListChapters(mangaID, provID)
	if err != nil {
		t.Fatalf("ListChapters failed: %v", err)
	}
	if len(chapters) != 4 {
		t.Fatalf("expected 4 chapters, got %d", len(chapters))
	}

	chMap := make(map[string]ChapterInfo)
	for _, ch := range chapters {
		chMap[ch.ID] = ch
	}

	if chMap[ch0].Meta.DownloadedPages != 0 || chMap[ch0].Meta.IsDownloaded != false {
		t.Errorf("ch0 meta status mismatch: pages=%d, is_downloaded=%v",
			chMap[ch0].Meta.DownloadedPages, chMap[ch0].Meta.IsDownloaded)
	}
	if chMap[chPartial].Meta.DownloadedPages != 3 || chMap[chPartial].Meta.IsDownloaded != false {
		t.Errorf("chPartial meta status mismatch: pages=%d, is_downloaded=%v",
			chMap[chPartial].Meta.DownloadedPages, chMap[chPartial].Meta.IsDownloaded)
	}
	if chMap[chComplete].Meta.DownloadedPages != 5 || chMap[chComplete].Meta.IsDownloaded != true {
		t.Errorf("chComplete meta status mismatch: pages=%d, is_downloaded=%v",
			chMap[chComplete].Meta.DownloadedPages, chMap[chComplete].Meta.IsDownloaded)
	}
	if chMap[chZeroMetaCount].Meta.DownloadedPages != 1 || chMap[chZeroMetaCount].Meta.IsDownloaded != true {
		t.Errorf("chZeroMetaCount meta status mismatch: pages=%d, is_downloaded=%v",
			chMap[chZeroMetaCount].Meta.DownloadedPages, chMap[chZeroMetaCount].Meta.IsDownloaded)
	}

	// Test DeleteChapterFiles resets DownloadedAt and meta.json
	if err := lib.DeleteChapterFiles(mangaID, provID, chComplete); err != nil {
		t.Fatalf("DeleteChapterFiles failed: %v", err)
	}

	metaAfterDelete, err := lib.GetChapter(mangaID, provID, chComplete)
	if err != nil {
		t.Fatalf("GetChapter failed after delete: %v", err)
	}
	if !metaAfterDelete.DownloadedAt.IsZero() {
		t.Errorf("expected DownloadedAt to be zero after delete, got %v", metaAfterDelete.DownloadedAt)
	}
	if metaAfterDelete.DownloadedPages != 0 {
		t.Errorf("expected DownloadedPages to be 0 after delete, got %d", metaAfterDelete.DownloadedPages)
	}
	if metaAfterDelete.IsDownloaded != false {
		t.Errorf("expected IsDownloaded to be false after delete, got %v", metaAfterDelete.IsDownloaded)
	}
}

func TestLibrary_PathTraversal(t *testing.T) {
	tempDir := t.TempDir()
	lib := NewLibrary(tempDir)

	traversalIDs := []string{
		"../outside",
		"../../etc/passwd",
		"..",
		"",
		"/../../secret",
	}

	for _, badID := range traversalIDs {
		if _, err := lib.GetManga(badID); err == nil {
			t.Errorf("expected GetManga(%q) to fail with path traversal, got nil", badID)
		}
		if err := lib.SaveManga(badID, Manga{Metadata: MangaMetadata{Title: "Bad"}}); err == nil {
			t.Errorf("expected SaveManga(%q) to fail with path traversal, got nil", badID)
		}
		if err := lib.DeleteManga(badID); err == nil {
			t.Errorf("expected DeleteManga(%q) to fail with path traversal, got nil", badID)
		}
		if _, err := lib.ListChapters(badID); err == nil {
			t.Errorf("expected ListChapters(%q) to fail with path traversal, got nil", badID)
		}
		if _, _, err := lib.FindChapter(badID, "ch-1"); err == nil {
			t.Errorf("expected FindChapter(%q) to fail with path traversal, got nil", badID)
		}
		if err := lib.SaveChapter(badID, "prov", "ch-1", &ChapterMeta{Title: "Ch"}); err == nil {
			t.Errorf("expected SaveChapter(%q) to fail with path traversal, got nil", badID)
		}
		if _, err := lib.GetChapter(badID, "prov", "ch-1"); err == nil {
			t.Errorf("expected GetChapter(%q) to fail with path traversal, got nil", badID)
		}
	}
}

func TestLibrary_ExternalLinks(t *testing.T) {
	tempDir := t.TempDir()
	lib := NewLibrary(tempDir)

	info := Manga{
		Metadata: MangaMetadata{
			Title: "External Links Test Manga",
			ExternalLinks: []ExternalLink{
				{
					Provider: "mangadex",
					Label:    "MangaDex",
					URL:      "https://mangadex.org/title/manga-1",
				},
				{
					Provider: "anilist",
					Label:    "AniList",
					URL:      "https://anilist.co/manga/1",
				},
			},
		},
	}

	if err := lib.SaveManga("manga-links", info); err != nil {
		t.Fatalf("SaveManga failed: %v", err)
	}

	got, err := lib.GetManga("manga-links")
	if err != nil {
		t.Fatalf("GetManga failed: %v", err)
	}

	if len(got.Metadata.ExternalLinks) != 2 {
		t.Fatalf("expected 2 external links, got %d", len(got.Metadata.ExternalLinks))
	}

	if got.Metadata.ExternalLinks[0].Provider != "mangadex" || got.Metadata.ExternalLinks[0].Label != "MangaDex" || got.Metadata.ExternalLinks[0].URL != "https://mangadex.org/title/manga-1" {
		t.Errorf("unexpected first external link: %+v", got.Metadata.ExternalLinks[0])
	}
	if got.Metadata.ExternalLinks[1].Provider != "anilist" || got.Metadata.ExternalLinks[1].Label != "AniList" || got.Metadata.ExternalLinks[1].URL != "https://anilist.co/manga/1" {
		t.Errorf("unexpected second external link: %+v", got.Metadata.ExternalLinks[1])
	}
}

func TestLibrary_GetMetadata_SaveMetadata(t *testing.T) {
	tempDir := t.TempDir()
	lib := NewLibrary(tempDir)
	id := "manga-meta"

	// Round-trip empty
	if err := lib.SaveMetadata(id, MangaMetadata{}); err != nil {
		t.Fatalf("SaveMetadata empty: %v", err)
	}
	m, err := lib.GetMetadata(id)
	if err != nil {
		t.Fatalf("GetMetadata: %v", err)
	}
	if m.Title != "" {
		t.Errorf("expected empty title, got %q", m.Title)
	}

	// Round-trip populated
	want := MangaMetadata{
		Title:       "Berserk",
		Authors:     []string{"Kentarou Miura"},
		Tags:        []string{"fantasy", "Fantasy", "seinen"}, // test dedup
		CoverURL:    "https://example.com/cover.jpg",
		ReleaseYear: 1989,
		ExternalLinks: []ExternalLink{
			{Provider: "mangadex", Label: "MangaDex", URL: "https://mangadex.org/title/1"},
		},
	}
	if err := lib.SaveMetadata(id, want); err != nil {
		t.Fatalf("SaveMetadata populated: %v", err)
	}
	got, err := lib.GetMetadata(id)
	if err != nil {
		t.Fatalf("GetMetadata populated: %v", err)
	}
	if got.Title != want.Title {
		t.Errorf("Title mismatch: %q != %q", got.Title, want.Title)
	}
	if !reflect.DeepEqual(got.Authors, want.Authors) {
		t.Errorf("Authors mismatch: %v != %v", got.Authors, want.Authors)
	}
	if !reflect.DeepEqual(got.Tags, []string{"fantasy", "seinen"}) {
		t.Errorf("Tags dedup mismatch: %v", got.Tags)
	}
	if got.ReleaseYear != 1989 {
		t.Errorf("ReleaseYear mismatch: %d", got.ReleaseYear)
	}
	if !reflect.DeepEqual(got.ExternalLinks, want.ExternalLinks) {
		t.Errorf("ExternalLinks mismatch: %v != %v", got.ExternalLinks, want.ExternalLinks)
	}

	// Missing file returns zero value + nil
	empty, err := lib.GetMetadata("nonexistent")
	if err != nil {
		t.Fatalf("GetMetadata missing: %v", err)
	}
	if !reflect.DeepEqual(empty, MangaMetadata{}) {
		t.Errorf("expected zero MangaMetadata for missing file, got %+v", empty)
	}
}

func TestLibrary_GetUserState_SaveUserState(t *testing.T) {
	tempDir := t.TempDir()
	lib := NewLibrary(tempDir)
	id := "manga-userstate"

	// First save populates AddedAt and UpdatedAt.
	beforeSave := time.Now()
	if err := lib.SaveUserState(id, UserState{Status: UserStatusReading, Rating: 9.5, Favorite: true, Notes: "great"}); err != nil {
		t.Fatalf("SaveUserState: %v", err)
	}
	afterSave := time.Now()

	got, err := lib.GetUserState(id)
	if err != nil {
		t.Fatalf("GetUserState: %v", err)
	}
	if got.Status != UserStatusReading {
		t.Errorf("Status mismatch: %q", got.Status)
	}
	if got.Rating != 9.5 {
		t.Errorf("Rating mismatch: %v", got.Rating)
	}
	if !got.Favorite {
		t.Errorf("Favorite mismatch: %v", got.Favorite)
	}
	if got.Notes != "great" {
		t.Errorf("Notes mismatch: %q", got.Notes)
	}
	if got.AddedAt.Before(beforeSave) || got.AddedAt.After(afterSave) {
		t.Errorf("AddedAt %v not in [%v, %v]", got.AddedAt, beforeSave, afterSave)
	}
	if got.UpdatedAt.Before(beforeSave) || got.UpdatedAt.After(afterSave) {
		t.Errorf("UpdatedAt %v not in [%v, %v]", got.UpdatedAt, beforeSave, afterSave)
	}
	if got.AddedAt.IsZero() {
		t.Errorf("AddedAt should be auto-set")
	}
	if got.UpdatedAt.IsZero() {
		t.Errorf("UpdatedAt should be auto-set")
	}

	// Second save should bump UpdatedAt but keep AddedAt.
	firstAddedAt := got.AddedAt
	firstUpdatedAt := got.UpdatedAt
	time.Sleep(2 * time.Millisecond) // ensure clock advances
	if err := lib.SaveUserState(id, UserState{Status: UserStatusCompleted, AddedAt: firstAddedAt}); err != nil {
		t.Fatalf("SaveUserState second: %v", err)
	}
	got2, err := lib.GetUserState(id)
	if err != nil {
		t.Fatalf("GetUserState second: %v", err)
	}
	if !got2.AddedAt.Equal(firstAddedAt) {
		t.Errorf("AddedAt should be preserved: was %v, now %v", firstAddedAt, got2.AddedAt)
	}
	if !got2.UpdatedAt.After(firstUpdatedAt) {
		t.Errorf("UpdatedAt should bump forward: was %v, now %v", firstUpdatedAt, got2.UpdatedAt)
	}

	// Missing file returns zero value + nil.
	empty, err := lib.GetUserState("nonexistent")
	if err != nil {
		t.Fatalf("GetUserState missing: %v", err)
	}
	if !reflect.DeepEqual(empty, UserState{}) {
		t.Errorf("expected zero UserState, got %+v", empty)
	}
}

func TestLibrary_GetBindings_SaveBindings(t *testing.T) {
	tempDir := t.TempDir()
	lib := NewLibrary(tempDir)
	id := "manga-bindings"

	want := MangaBindings{
		Providers: []ProviderRef{
			{ProviderID: "mangadex", ProviderMangaID: "md-1", MangaTitle: "Title"},
			{ProviderID: "mangadex", ProviderMangaID: "md-1", MangaTitle: "Dup"}, // duplicate
			{ProviderID: "anilist", ProviderMangaID: "al-1", MangaTitle: "Title"},
		},
		Content: &ContentSource{ProviderID: "mangadex", ProviderMangaID: "md-1", ReadingMode: "rtl"},
	}
	if err := lib.SaveBindings(id, want); err != nil {
		t.Fatalf("SaveBindings: %v", err)
	}

	got, err := lib.GetBindings(id)
	if err != nil {
		t.Fatalf("GetBindings: %v", err)
	}
	if len(got.Providers) != 2 {
		t.Fatalf("expected dedup to 2 providers, got %d: %+v", len(got.Providers), got.Providers)
	}
	if got.Providers[0].ProviderID != "mangadex" || got.Providers[0].ProviderMangaID != "md-1" {
		t.Errorf("first provider mismatch: %+v", got.Providers[0])
	}
	if got.Providers[0].MangaTitle != "Title" {
		t.Errorf("expected first-seen title preserved, got %q", got.Providers[0].MangaTitle)
	}
	if got.Providers[1].ProviderID != "anilist" {
		t.Errorf("second provider mismatch: %+v", got.Providers[1])
	}
	if got.Content == nil || got.Content.ProviderID != "mangadex" {
		t.Errorf("Content mismatch: %+v", got.Content)
	}

	// Missing file returns zero value + nil.
	empty, err := lib.GetBindings("nonexistent")
	if err != nil {
		t.Fatalf("GetBindings missing: %v", err)
	}
	if !reflect.DeepEqual(empty, MangaBindings{}) {
		t.Errorf("expected zero MangaBindings, got %+v", empty)
	}
}

func TestLibrary_GetManga_ComposesAllThree(t *testing.T) {
	tempDir := t.TempDir()
	lib := NewLibrary(tempDir)
	id := "manga-compose"

	if err := lib.SaveMetadata(id, MangaMetadata{Title: "Composed", Authors: []string{"Author"}}); err != nil {
		t.Fatalf("SaveMetadata: %v", err)
	}
	if err := lib.SaveUserState(id, UserState{Status: UserStatusReading}); err != nil {
		t.Fatalf("SaveUserState: %v", err)
	}
	if err := lib.SaveBindings(id, MangaBindings{
		Providers: []ProviderRef{{ProviderID: "mangadex", ProviderMangaID: "x"}},
		Content:   &ContentSource{ProviderID: "mangadex", ProviderMangaID: "x"},
	}); err != nil {
		t.Fatalf("SaveBindings: %v", err)
	}

	info, err := lib.GetManga(id)
	if err != nil {
		t.Fatalf("GetManga: %v", err)
	}
	if info.ID != id {
		t.Errorf("ID mismatch: %q", info.ID)
	}
	if info.Metadata.Title != "Composed" {
		t.Errorf("Metadata.Title mismatch: %q", info.Metadata.Title)
	}
	if info.UserState.Status != UserStatusReading {
		t.Errorf("UserState.Status mismatch: %q", info.UserState.Status)
	}
	if len(info.Bindings.Providers) != 1 {
		t.Errorf("Bindings.Providers mismatch: %+v", info.Bindings.Providers)
	}

	// Missing metadata.json → error
	if _, err := lib.GetManga("nonexistent"); err == nil {
		t.Errorf("expected error getting nonexistent manga, got nil")
	}
}

func TestLibrary_ListManga_ReadsThreeFiles(t *testing.T) {
	tempDir := t.TempDir()
	lib := NewLibrary(tempDir)

	// Set up two manga with distinct state
	for _, tt := range []struct {
		id, title, status string
		providers         []ProviderRef
	}{
		{"a", "Alpha", UserStatusReading, []ProviderRef{{ProviderID: "md", ProviderMangaID: "1"}}},
		{"b", "Beta", UserStatusCompleted, nil},
	} {
		if err := lib.SaveMetadata(tt.id, MangaMetadata{Title: tt.title}); err != nil {
			t.Fatalf("SaveMetadata %s: %v", tt.id, err)
		}
		if err := lib.SaveUserState(tt.id, UserState{Status: tt.status}); err != nil {
			t.Fatalf("SaveUserState %s: %v", tt.id, err)
		}
		if err := lib.SaveBindings(tt.id, MangaBindings{Providers: tt.providers}); err != nil {
			t.Fatalf("SaveBindings %s: %v", tt.id, err)
		}
	}

	list, err := lib.ListManga()
	if err != nil {
		t.Fatalf("ListManga: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 manga, got %d", len(list))
	}
	// Sorted case-insensitive by title: Alpha, Beta
	if list[0].Metadata.Title != "Alpha" || list[1].Metadata.Title != "Beta" {
		t.Errorf("sort mismatch: %q, %q", list[0].Metadata.Title, list[1].Metadata.Title)
	}
	if list[0].UserState.Status != UserStatusReading {
		t.Errorf("Alpha user state mismatch: %q", list[0].UserState.Status)
	}
	if list[1].UserState.Status != UserStatusCompleted {
		t.Errorf("Beta user state mismatch: %q", list[1].UserState.Status)
	}
	if len(list[0].Bindings.Providers) != 1 {
		t.Errorf("Alpha bindings.Providers mismatch: %+v", list[0].Bindings.Providers)
	}
	if len(list[1].Bindings.Providers) != 0 {
		t.Errorf("Beta bindings.Providers mismatch: %+v", list[1].Bindings.Providers)
	}
}

func TestLibrary_SaveJSONAtomic_Atomic(t *testing.T) {
	tempDir := t.TempDir()
	lib := NewLibrary(tempDir)
	id := "atomic-manga"
	target := filepath.Join(tempDir, id, "metadata.json")

	// Happy path
	if err := lib.SaveMetadata(id, MangaMetadata{Title: "Atomic"}); err != nil {
		t.Fatalf("SaveMetadata: %v", err)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("expected file at %s: %v", target, err)
	}
	// Verify no leftover temp files
	entries, _ := os.ReadDir(filepath.Join(tempDir, id))
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp.") {
			t.Errorf("leftover temp file: %s", e.Name())
		}
	}

	// Error path: invalid JSON value (channel is not JSON-serializable)
	//   We can't easily feed a non-marshalable value via the public API,
	//   so we exercise the helper directly to confirm cleanup.
	badPath := filepath.Join(tempDir, id, "bad.json")
	if err := saveJSONAtomic(badPath, make(chan int)); err == nil {
		t.Errorf("expected error marshaling channel, got nil")
	}
	if _, err := os.Stat(badPath); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("expected no file at %s after encode error, got err=%v", badPath, err)
	}
}
