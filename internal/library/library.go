package library

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Standard user reading status constants.
const (
	UserStatusUnread     = "unread"
	UserStatusReading    = "reading"
	UserStatusCompleted  = "completed"
	UserStatusOnHold     = "on_hold"
	UserStatusDropped    = "dropped"
	UserStatusPlanToRead = "plan_to_read"
)

// IsValidUserStatus reports whether status is a known valid user reading status.
func IsValidUserStatus(status string) bool {
	switch status {
	case UserStatusUnread,
		UserStatusReading,
		UserStatusCompleted,
		UserStatusOnHold,
		UserStatusDropped,
		UserStatusPlanToRead:
		return true
	default:
		return false
	}
}

// ContentSource represents the upstream primary content provider binding.
type ContentSource struct {
	ProviderID   string    `json:"provider_id"`
	MangaID      string    `json:"manga_id,omitempty"`
	ChapterRef   string    `json:"chapter_ref,omitempty"`
	ReadingMode  string    `json:"reading_mode,omitempty"`
	LastSyncedAt time.Time `json:"last_synced_at,omitempty"`
}

// MangaMeta matches the schema defined in docs/design/library.md
type MangaMeta struct {
	Title             string         `json:"title"`
	Aliases           []string       `json:"aliases"`
	Description       string         `json:"description"`
	Authors           []string       `json:"authors"`
	Artists           []string       `json:"artists"`
	Tags              []string       `json:"tags"`
	Collections       []string       `json:"collections"`
	ContentRating     string         `json:"content_rating"`
	Publisher         string         `json:"publisher"`
	ReleaseYear       int            `json:"release_year"`
	StartDate         string         `json:"start_date"`
	EndDate           string         `json:"end_date"`
	Country           string         `json:"country"`
	CoverURL          string         `json:"cover_url,omitempty"`
	Content           *ContentSource `json:"content,omitempty"`
	UserStatus        string         `json:"user_status"`
	UserRating        float64        `json:"user_rating"`
	UserFavorite      bool           `json:"user_favorite"`
	UserNotes         string         `json:"user_notes"`
	LastReadChapterID string         `json:"last_read_chapter_id,omitempty"`
	LastReadAt        time.Time      `json:"last_read_at,omitempty"`
	AddedAt           time.Time      `json:"added_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

// UnmarshalJSON implements custom unmarshaling for MangaMeta to support backward compatibility
// for the deprecated top-level reading_direction field and top-level reading_mode.
func (m *MangaMeta) UnmarshalJSON(data []byte) error {
	type Alias MangaMeta
	aux := struct {
		ReadingDirection string `json:"reading_direction"`
		ReadingMode      string `json:"reading_mode"`
		*Alias
	}{
		Alias: (*Alias)(m),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.ReadingMode != "" {
		if m.Content == nil {
			m.Content = &ContentSource{}
		}
		m.Content.ReadingMode = aux.ReadingMode
	} else if aux.ReadingDirection != "" {
		if m.Content == nil {
			m.Content = &ContentSource{}
		}
		if m.Content.ReadingMode == "" {
			m.Content.ReadingMode = aux.ReadingDirection
		}
	}
	return nil
}

// ChapterMeta matches the schema defined in docs/design/library.md
type ChapterMeta struct {
	Title        string         `json:"title"`
	Number       float32        `json:"number"`
	Volume       int            `json:"volume"`
	Language     string         `json:"language"`
	UploadDate   time.Time      `json:"upload_date,omitempty"`
	SourceOrder  int            `json:"source_order"`
	Content      *ContentSource `json:"content,omitempty"`
	PageCount    int            `json:"page_count"`
	PageFormat   string         `json:"page_format"`
	DownloadedAt time.Time      `json:"downloaded_at,omitempty"`
	IsRead       bool           `json:"is_read"`
	LastReadPage int            `json:"last_read_page"`
	LastReadAt   time.Time      `json:"last_read_at,omitempty"`
}

// PageItem represents a single page in a chapter.
type PageItem struct {
	Index  int    `json:"index"`
	URL    string `json:"url"`
	Source string `json:"source,omitempty"`
}

// MangaInfo contains the ID and its parsed metadata.
type MangaInfo struct {
	ID   string    `json:"id"`
	Meta MangaMeta `json:"meta"`
}

// ChapterInfo contains the ID, its parent manga ID, and its parsed metadata.
type ChapterInfo struct {
	ID      string      `json:"id"`
	MangaID string      `json:"manga_id"`
	Meta    ChapterMeta `json:"meta"`
}

// Library handles direct filesystem operations on the manga library directory.
type Library struct {
	root string
}

// NewLibrary initializes a new Library manager pointing to the given root directory.
func NewLibrary(root string) *Library {
	return &Library{root: root}
}

// sanitizeID trims leading/trailing slashes and strips /manga/ prefixes to prevent nested directory creation.
func sanitizeID(id string) string {
	id = strings.TrimSpace(id)
	if u, err := url.Parse(id); err == nil && (u.Scheme != "" || u.Host != "") {
		id = u.Path
	}
	id = strings.Trim(id, "/")
	if strings.HasPrefix(id, "manga/") {
		id = strings.TrimPrefix(id, "manga/")
		id = strings.Trim(id, "/")
	}
	return id
}

// ListManga walks the library directory and lists all manga with their meta.json content.
func (l *Library) ListManga() ([]MangaInfo, error) {
	entries, err := os.ReadDir(l.root)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("list manga: read root: %w", err)
	}

	var dirNames []string
	for _, entry := range entries {
		if entry.IsDir() {
			dirNames = append(dirNames, entry.Name())
		}
	}
	if len(dirNames) == 0 {
		return nil, nil
	}

	const maxWorkers = 16
	numWorkers := len(dirNames)
	if numWorkers > maxWorkers {
		numWorkers = maxWorkers
	}

	jobs := make(chan string, len(dirNames))
	for _, name := range dirNames {
		jobs <- name
	}
	close(jobs)

	var (
		mu   sync.Mutex
		wg   sync.WaitGroup
		list []MangaInfo
	)

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for id := range jobs {
				meta, err := l.GetManga(id)
				if err != nil {
					// Skip invalid/corrupted directories
					continue
				}
				mu.Lock()
				list = append(list, MangaInfo{
					ID:   id,
					Meta: *meta,
				})
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	// Sort manga alphabetically by title
	sort.Slice(list, func(i, j int) bool {
		return strings.ToLower(list[i].Meta.Title) < strings.ToLower(list[j].Meta.Title)
	})

	return list, nil
}

// GetManga reads and parses library/<manga_id>/meta.json.
func (l *Library) GetManga(id string) (*MangaMeta, error) {
	id = sanitizeID(id)
	path := filepath.Join(l.root, id, "meta.json")
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("get manga %s: %w", id, err)
	}

	var meta MangaMeta
	if err := json.Unmarshal(bytes, &meta); err != nil {
		return nil, fmt.Errorf("get manga %s: unmarshal error: %w", id, err)
	}

	return &meta, nil
}

// SaveManga writes or updates library/<manga_id>/meta.json.
func (l *Library) SaveManga(id string, meta *MangaMeta) error {
	id = sanitizeID(id)
	dir := filepath.Join(l.root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("save manga %s: create dir: %w", id, err)
	}

	if meta.AddedAt.IsZero() {
		meta.AddedAt = time.Now()
	}
	meta.UpdatedAt = time.Now()

	bytes, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("save manga %s: marshal error: %w", id, err)
	}

	path := filepath.Join(dir, "meta.json")
	if err := os.WriteFile(path, bytes, 0o644); err != nil {
		return fmt.Errorf("save manga %s: write file: %w", id, err)
	}

	return nil
}

// DeleteManga removes the library/<manga_id> directory and all its contents.
func (l *Library) DeleteManga(id string) error {
	id = sanitizeID(id)
	dir := filepath.Join(l.root, id)
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("delete manga %s: %w", id, err)
	}
	return nil
}

// ListChapters lists all chapters for a manga by reading directories containing meta.json.
func (l *Library) ListChapters(mangaID string) ([]ChapterInfo, error) {
	mangaID = sanitizeID(mangaID)
	mangaDir := filepath.Join(l.root, mangaID)

	entries, err := os.ReadDir(mangaDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("list chapters for manga %s: %w", mangaID, err)
	}

	var dirNames []string
	for _, entry := range entries {
		if entry.IsDir() {
			dirNames = append(dirNames, entry.Name())
		}
	}
	if len(dirNames) == 0 {
		return nil, nil
	}

	const maxWorkers = 16
	numWorkers := len(dirNames)
	if numWorkers > maxWorkers {
		numWorkers = maxWorkers
	}

	jobs := make(chan string, len(dirNames))
	for _, name := range dirNames {
		jobs <- name
	}
	close(jobs)

	var (
		mu   sync.Mutex
		wg   sync.WaitGroup
		list []ChapterInfo
	)

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for chapterID := range jobs {
				meta, err := l.GetChapter(mangaID, chapterID)
				if err != nil {
					// Skip invalid subdirectories
					continue
				}
				mu.Lock()
				list = append(list, ChapterInfo{
					ID:      chapterID,
					MangaID: mangaID,
					Meta:    *meta,
				})
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	// Sort chapters by chapter number ascending
	sort.Slice(list, func(i, j int) bool {
		return list[i].Meta.Number < list[j].Meta.Number
	})

	return list, nil
}

// GetChapter reads library/<manga_id>/<chapter_id>/meta.json.
func (l *Library) GetChapter(mangaID, chapterID string) (*ChapterMeta, error) {
	mangaID = sanitizeID(mangaID)
	path := filepath.Join(l.root, mangaID, chapterID, "meta.json")
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("get chapter %s/%s: %w", mangaID, chapterID, err)
	}

	var meta ChapterMeta
	if err := json.Unmarshal(bytes, &meta); err != nil {
		return nil, fmt.Errorf("get chapter %s/%s: unmarshal error: %w", mangaID, chapterID, err)
	}

	return &meta, nil
}

// SaveChapter writes or updates library/<manga_id>/<chapter_id>/meta.json.
func (l *Library) SaveChapter(mangaID, chapterID string, meta *ChapterMeta) error {
	mangaID = sanitizeID(mangaID)
	dir := filepath.Join(l.root, mangaID, chapterID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("save chapter %s/%s: create dir: %w", mangaID, chapterID, err)
	}

	bytes, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("save chapter %s/%s: marshal error: %w", mangaID, chapterID, err)
	}

	path := filepath.Join(dir, "meta.json")
	if err := os.WriteFile(path, bytes, 0o644); err != nil {
		return fmt.Errorf("save chapter %s/%s: write file: %w", mangaID, chapterID, err)
	}

	return nil
}

// DeleteChapter deletes the chapter directory library/<manga_id>/<chapter_id>.
func (l *Library) DeleteChapter(mangaID, chapterID string) error {
	mangaID = sanitizeID(mangaID)
	dir := filepath.Join(l.root, mangaID, chapterID)
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("delete chapter %s/%s: %w", mangaID, chapterID, err)
	}
	return nil
}

// UpdateChapterProgress updates the reading progress for a chapter and updates the parent manga's last read metadata.
func (l *Library) UpdateChapterProgress(mangaID, chapterID string, isRead bool, lastReadPage int) (*ChapterInfo, error) {
	mangaID = sanitizeID(mangaID)
	chapterID = sanitizeID(chapterID)

	chMeta, err := l.GetChapter(mangaID, chapterID)
	if err != nil {
		return nil, fmt.Errorf("update chapter progress %s/%s: %w", mangaID, chapterID, err)
	}

	now := time.Now()
	chMeta.IsRead = isRead
	chMeta.LastReadPage = lastReadPage
	chMeta.LastReadAt = now

	if err := l.SaveChapter(mangaID, chapterID, chMeta); err != nil {
		return nil, fmt.Errorf("update chapter progress %s/%s: save chapter: %w", mangaID, chapterID, err)
	}

	mangaMeta, err := l.GetManga(mangaID)
	if err != nil {
		return nil, fmt.Errorf("update chapter progress %s/%s: get manga: %w", mangaID, chapterID, err)
	}

	mangaMeta.LastReadChapterID = chapterID
	mangaMeta.LastReadAt = now

	if err := l.SaveManga(mangaID, mangaMeta); err != nil {
		return nil, fmt.Errorf("update chapter progress %s/%s: save manga: %w", mangaID, chapterID, err)
	}

	return &ChapterInfo{
		ID:      chapterID,
		MangaID: mangaID,
		Meta:    *chMeta,
	}, nil
}

// GetChapterPages reads and parses the stored page list for a chapter.
// If mangaID is non-empty, pages are read from library/<manga_id>/<chapter_id>/pages.json.
// If mangaID is empty, pages are read from library/_pages/<chapter_id>.json.
func (l *Library) GetChapterPages(mangaID, chapterID string) ([]PageItem, error) {
	if mangaID != "" {
		mangaID = sanitizeID(mangaID)
	}
	chapterID = sanitizeID(chapterID)

	var path string
	if mangaID != "" {
		path = filepath.Join(l.root, mangaID, chapterID, "pages.json")
	} else {
		path = filepath.Join(l.root, "_pages", chapterID+".json")
	}

	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("get chapter pages %s/%s: %w", mangaID, chapterID, err)
	}

	var pages []PageItem
	if err := json.Unmarshal(bytes, &pages); err != nil {
		return nil, fmt.Errorf("get chapter pages %s/%s: unmarshal error: %w", mangaID, chapterID, err)
	}

	return pages, nil
}

// SaveChapterPages stores the page list for a chapter atomically.
// If mangaID is non-empty, pages are written to library/<manga_id>/<chapter_id>/pages.json
// and if meta.json exists, its page_count is updated.
// If mangaID is empty, pages are written to library/_pages/<chapter_id>.json.
func (l *Library) SaveChapterPages(mangaID, chapterID string, pages []PageItem) error {
	if len(pages) == 0 {
		return fmt.Errorf("save chapter pages: empty page list")
	}

	if mangaID != "" {
		mangaID = sanitizeID(mangaID)
	}
	chapterID = sanitizeID(chapterID)

	var targetDir, targetFile string
	if mangaID != "" {
		targetDir = filepath.Join(l.root, mangaID, chapterID)
		targetFile = filepath.Join(targetDir, "pages.json")
	} else {
		targetDir = filepath.Join(l.root, "_pages")
		targetFile = filepath.Join(targetDir, chapterID+".json")
	}

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("save chapter pages: create dir: %w", err)
	}

	bytes, err := json.MarshalIndent(pages, "", "  ")
	if err != nil {
		return fmt.Errorf("save chapter pages: marshal error: %w", err)
	}

	tmpFile, err := os.CreateTemp(targetDir, "pages.json.tmp.*")
	if err != nil {
		return fmt.Errorf("save chapter pages: create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if _, err := tmpFile.Write(bytes); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("save chapter pages: write temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("save chapter pages: close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, targetFile); err != nil {
		return fmt.Errorf("save chapter pages: rename temp file: %w", err)
	}

	if mangaID != "" {
		metaPath := filepath.Join(l.root, mangaID, chapterID, "meta.json")
		if _, err := os.Stat(metaPath); err == nil {
			chMeta, err := l.GetChapter(mangaID, chapterID)
			if err != nil {
				return fmt.Errorf("save chapter pages: get chapter meta: %w", err)
			}
			chMeta.PageCount = len(pages)
			if err := l.SaveChapter(mangaID, chapterID, chMeta); err != nil {
				return fmt.Errorf("save chapter pages: update chapter meta: %w", err)
			}
		}
	}

	return nil
}

