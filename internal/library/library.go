package library

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/tubruk/kiyomi/internal/security/ssrfguard"
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

// LocalProviderID is the provider ID used for locally-managed chapters.
const LocalProviderID = "local"

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

// ProviderRef represents a provider binding for a manga.
type ProviderRef struct {
	ProviderID      string `json:"provider_id"`
	ProviderMangaID string `json:"provider_manga_id"`
	MangaTitle      string `json:"manga_title,omitempty"`
}

// ContentSource represents the upstream primary content provider binding.
type ContentSource struct {
	ProviderID      string    `json:"provider_id"`
	ProviderMangaID string    `json:"provider_manga_id,omitempty"`
	ChapterRef      string    `json:"chapter_ref,omitempty"`
	ReadingMode     string    `json:"reading_mode,omitempty"`
	LastSyncedAt    time.Time `json:"last_synced_at,omitempty"`
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
	Providers         []ProviderRef  `json:"providers,omitempty"`
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
	PageCount       int            `json:"page_count"`
	PageFormat      string         `json:"page_format"`
	DownloadedAt    time.Time      `json:"downloaded_at,omitempty"`
	DownloadedPages int            `json:"downloaded_pages,omitempty"`
	IsDownloaded    bool           `json:"is_downloaded,omitempty"`
	Orphaned        bool           `json:"orphaned,omitempty"`
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
	root                 string
	mu                   sync.RWMutex
	allowPrivateNetworks bool
}

// SetAllowPrivateNetworks toggles the SSRF guard used by outbound cover and
// image fetches. When false (the default), URLs that resolve to private,
// loopback, or link-local address ranges are rejected before any request is
// made. Flip to true in local development against 127.0.0.1 or LAN-hosted
// providers.
func (l *Library) SetAllowPrivateNetworks(allow bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.allowPrivateNetworks = allow
}

// NewLibrary initializes a new Library manager pointing to the given root directory.
func NewLibrary(root string) *Library {
	return &Library{root: root}
}

// mangaDir returns the directory path for a manga ID, validating that the cleaned
// directory path stays within the library root to prevent path traversal.
func (l *Library) mangaDir(mangaID string) (string, error) {
	cleanRoot := filepath.Clean(l.root)
	id := sanitizeID(mangaID)
	if id == "" {
		return "", fmt.Errorf("invalid manga id: empty")
	}
	dir := filepath.Clean(filepath.Join(cleanRoot, id))
	rel, err := filepath.Rel(cleanRoot, dir)
	if err != nil || strings.HasPrefix(rel, "..") || rel == "." {
		return "", fmt.Errorf("invalid manga id %q: path traversal outside root", mangaID)
	}
	return dir, nil
}

// providerChapterDir returns the directory path for a chapter within a provider,
// ensuring the path stays within the library root.
func (l *Library) providerChapterDir(mangaID, providerID, chapterID string) (string, error) {
	mDir, err := l.mangaDir(mangaID)
	if err != nil {
		return "", err
	}
	cleanRoot := filepath.Clean(l.root)
	pID := sanitizeID(providerID)
	cID := sanitizeID(chapterID)
	if pID == "" || cID == "" {
		return "", fmt.Errorf("invalid provider or chapter id: empty")
	}
	dir := filepath.Clean(filepath.Join(mDir, pID, cID))
	rel, err := filepath.Rel(cleanRoot, dir)
	if err != nil || strings.HasPrefix(rel, "..") || rel == "." {
		return "", fmt.Errorf("invalid chapter path: path traversal outside root")
	}
	return dir, nil
}

// ProviderChapterDir returns the on-disk directory path for a chapter within a
// provider namespace. This does not create the directory.
func (l *Library) ProviderChapterDir(mangaID, providerID, chapterID string) string {
	dir, err := l.providerChapterDir(mangaID, providerID, chapterID)
	if err != nil {
		return filepath.Join(l.root, sanitizeID(mangaID), sanitizeID(providerID), sanitizeID(chapterID))
	}
	return dir
}

// Root returns the library root directory.
func (l *Library) Root() string {
	return l.root
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
	l.mu.RLock()
	defer l.mu.RUnlock()

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
				meta, err := l.getManga(id)
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
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.getManga(id)
}

func (l *Library) getManga(id string) (*MangaMeta, error) {
	dir, err := l.mangaDir(id)
	if err != nil {
		return nil, fmt.Errorf("get manga %s: %w", id, err)
	}
	path := filepath.Join(dir, "meta.json")
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

// SaveManga writes or updates library/<manga_id>/meta.json atomically.
func (l *Library) SaveManga(id string, meta *MangaMeta) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.saveManga(id, meta)
}

func (l *Library) saveManga(id string, meta *MangaMeta) error {
	dir, err := l.mangaDir(id)
	if err != nil {
		return fmt.Errorf("save manga %s: %w", id, err)
	}
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

	tmpFile, err := os.CreateTemp(dir, "meta.json.tmp.*")
	if err != nil {
		return fmt.Errorf("save manga %s: create temp file: %w", id, err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if _, err := tmpFile.Write(bytes); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("save manga %s: write temp file: %w", id, err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("save manga %s: close temp file: %w", id, err)
	}

	targetFile := filepath.Join(dir, "meta.json")
	if err := os.Rename(tmpPath, targetFile); err != nil {
		return fmt.Errorf("save manga %s: rename temp file: %w", id, err)
	}

	return nil
}

// DeleteManga removes the library/<manga_id> directory and all its contents.
func (l *Library) DeleteManga(id string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.deleteManga(id)
}

func (l *Library) deleteManga(id string) error {
	dir, err := l.mangaDir(id)
	if err != nil {
		return fmt.Errorf("delete manga %s: %w", id, err)
	}
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("delete manga %s: %w", id, err)
	}
	return nil
}

// ListChapters lists all chapters for a manga by reading directories containing meta.json.
// If providerIDs is empty, lists chapters from all provider subdirectories under mangaID.
// Otherwise, lists chapters only from the specified providers.
func (l *Library) ListChapters(mangaID string, providerIDs ...string) ([]ChapterInfo, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	mangaDir, err := l.mangaDir(mangaID)
	if err != nil {
		return nil, fmt.Errorf("list chapters for manga %s: %w", mangaID, err)
	}

	// If no providers specified, walk all provider subdirectories
	if len(providerIDs) == 0 {
		entries, err := os.ReadDir(mangaDir)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil, nil
			}
			return nil, fmt.Errorf("list chapters for manga %s: %w", mangaID, err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				providerIDs = append(providerIDs, entry.Name())
			}
		}
		if len(providerIDs) == 0 {
			return nil, nil
		}
	}

	// Collect all chapters from specified providers
	var allChapters []ChapterInfo
	for _, providerID := range providerIDs {
		providerDir := filepath.Join(mangaDir, providerID)
		entries, err := os.ReadDir(providerDir)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			chapterID := entry.Name()
			meta, err := l.getChapter(mangaID, providerID, chapterID)
			if err != nil {
				continue
			}
			allChapters = append(allChapters, ChapterInfo{
				ID:      chapterID,
				MangaID: sanitizeID(mangaID),
				Meta:    *meta,
			})
		}
	}

	// Sort chapters by chapter number ascending
	sort.Slice(allChapters, func(i, j int) bool {
		return allChapters[i].Meta.Number < allChapters[j].Meta.Number
	})

	return allChapters, nil
}

// FindChapter searches all provider subdirectories for a chapter by its ID.
// Returns the chapter meta and the provider ID it was found under, or an error if not found.
// This is used by API handlers that need to locate a chapter when the provider is unknown.
func (l *Library) FindChapter(mangaID, chapterID string) (*ChapterMeta, string, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	mangaDir, err := l.mangaDir(mangaID)
	if err != nil {
		return nil, "", fmt.Errorf("find chapter %s/%s: %w", mangaID, chapterID, err)
	}

	entries, err := os.ReadDir(mangaDir)
	if err != nil {
		return nil, "", fmt.Errorf("find chapter %s/%s: %w", mangaID, chapterID, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		providerID := entry.Name()
		meta, err := l.getChapter(mangaID, providerID, chapterID)
		if err == nil {
			return meta, providerID, nil
		}
	}

	return nil, "", fmt.Errorf("find chapter %s/%s: not found", mangaID, chapterID)
}

// CountDownloadedPages inspects the chapter directory and counts image files with
// extensions .jpg, .jpeg, .png, .webp, .gif, .avif.
func (l *Library) CountDownloadedPages(mangaID, providerID, chapterID string) int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.countDownloadedPages(mangaID, providerID, chapterID)
}

func (l *Library) countDownloadedPages(mangaID, providerID, chapterID string) int {
	dir, err := l.providerChapterDir(mangaID, providerID, chapterID)
	if err != nil {
		return 0
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.Contains(name, ".tmp.") {
			continue
		}
		ext := strings.ToLower(filepath.Ext(name))
		switch ext {
		case ".jpg", ".jpeg", ".png", ".webp", ".gif", ".avif":
			count++
		}
	}
	return count
}

// GetChapterFileStatus checks on-disk chapter files and computes download status.
func (l *Library) GetChapterFileStatus(mangaID, providerID, chapterID string, metaPageCount int) (downloadedPages int, isDownloaded bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.getChapterFileStatus(mangaID, providerID, chapterID, metaPageCount)
}

func (l *Library) getChapterFileStatus(mangaID, providerID, chapterID string, metaPageCount int) (downloadedPages int, isDownloaded bool) {
	downloadedPages = l.countDownloadedPages(mangaID, providerID, chapterID)
	isDownloaded = downloadedPages > 0 && (metaPageCount == 0 || downloadedPages >= metaPageCount)
	return downloadedPages, isDownloaded
}

// GetChapter reads library/<manga_id>/<provider_id>/<chapter_id>/meta.json.
func (l *Library) GetChapter(mangaID, providerID, chapterID string) (*ChapterMeta, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.getChapter(mangaID, providerID, chapterID)
}

func (l *Library) getChapter(mangaID, providerID, chapterID string) (*ChapterMeta, error) {
	dir, err := l.providerChapterDir(mangaID, providerID, chapterID)
	if err != nil {
		return nil, fmt.Errorf("get chapter %s/%s/%s: %w", mangaID, providerID, chapterID, err)
	}
	path := filepath.Join(dir, "meta.json")
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("get chapter %s/%s/%s: %w", mangaID, providerID, chapterID, err)
	}

	var meta ChapterMeta
	if err := json.Unmarshal(bytes, &meta); err != nil {
		return nil, fmt.Errorf("get chapter %s/%s/%s: unmarshal error: %w", mangaID, providerID, chapterID, err)
	}

	meta.DownloadedPages, meta.IsDownloaded = l.getChapterFileStatus(mangaID, providerID, chapterID, meta.PageCount)

	return &meta, nil
}

// SaveChapter writes or updates library/<manga_id>/<provider_id>/<chapter_id>/meta.json atomically.
func (l *Library) SaveChapter(mangaID, providerID, chapterID string, meta *ChapterMeta) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.saveChapter(mangaID, providerID, chapterID, meta)
}

func (l *Library) saveChapter(mangaID, providerID, chapterID string, meta *ChapterMeta) error {
	// Default to LocalProviderID if no provider specified
	if meta.Content != nil && meta.Content.ProviderID == "" {
		meta.Content.ProviderID = LocalProviderID
	}

	dir, err := l.providerChapterDir(mangaID, providerID, chapterID)
	if err != nil {
		return fmt.Errorf("save chapter %s/%s/%s: %w", mangaID, providerID, chapterID, err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("save chapter %s/%s/%s: create dir: %w", mangaID, providerID, chapterID, err)
	}

	bytes, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("save chapter %s/%s/%s: marshal error: %w", mangaID, providerID, chapterID, err)
	}

	tmpFile, err := os.CreateTemp(dir, "meta.json.tmp.*")
	if err != nil {
		return fmt.Errorf("save chapter %s/%s/%s: create temp file: %w", mangaID, providerID, chapterID, err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if _, err := tmpFile.Write(bytes); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("save chapter %s/%s/%s: write temp file: %w", mangaID, providerID, chapterID, err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("save chapter %s/%s/%s: close temp file: %w", mangaID, providerID, chapterID, err)
	}

	targetFile := filepath.Join(dir, "meta.json")
	if err := os.Rename(tmpPath, targetFile); err != nil {
		return fmt.Errorf("save chapter %s/%s/%s: rename temp file: %w", mangaID, providerID, chapterID, err)
	}

	return nil
}

// DeleteChapter deletes the chapter directory library/<manga_id>/<provider_id>/<chapter_id>.
func (l *Library) DeleteChapter(mangaID, providerID, chapterID string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.deleteChapter(mangaID, providerID, chapterID)
}

func (l *Library) deleteChapter(mangaID, providerID, chapterID string) error {
	dir, err := l.providerChapterDir(mangaID, providerID, chapterID)
	if err != nil {
		return fmt.Errorf("delete chapter %s/%s/%s: %w", mangaID, providerID, chapterID, err)
	}
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("delete chapter %s/%s/%s: %w", mangaID, providerID, chapterID, err)
	}
	return nil
}

// BatchDeleteChapters deletes multiple chapter directories under library/<manga_id>/<provider_id>/.
func (l *Library) BatchDeleteChapters(mangaID, providerID string, chapterIDs []string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, chID := range chapterIDs {
		if err := l.deleteChapter(mangaID, providerID, chID); err != nil {
			return fmt.Errorf("batch delete chapters %s/%s/%s: %w", mangaID, providerID, chID, err)
		}
	}
	return nil
}

// DeleteChapterFiles removes page images (*.jpg, *.png, *.webp, *.gif) and pages.json
// from the chapter directory. Does NOT remove meta.json or the chapter directory itself.
func (l *Library) DeleteChapterFiles(mangaID, providerID, chapterID string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.deleteChapterFiles(mangaID, providerID, chapterID)
}

func (l *Library) deleteChapterFiles(mangaID, providerID, chapterID string) error {
	dir, err := l.providerChapterDir(mangaID, providerID, chapterID)
	if err != nil {
		return fmt.Errorf("delete chapter files %s/%s/%s: %w", mangaID, providerID, chapterID, err)
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("delete chapter files %s/%s/%s: chapter dir does not exist", mangaID, providerID, chapterID)
	}

	exts := []string{".jpg", ".jpeg", ".png", ".webp", ".gif", ".avif"}
	for _, ext := range exts {
		pattern := filepath.Join(dir, "*"+ext)
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return fmt.Errorf("delete chapter files %s/%s/%s: glob %s: %w", mangaID, providerID, chapterID, ext, err)
		}
		for _, f := range matches {
			if err := os.Remove(f); err != nil {
				return fmt.Errorf("delete chapter files %s/%s/%s: remove %s: %w", mangaID, providerID, chapterID, f, err)
			}
		}
	}

	pagesFile := filepath.Join(dir, "pages.json")
	if err := os.Remove(pagesFile); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("delete chapter files %s/%s/%s: remove pages.json: %w", mangaID, providerID, chapterID, err)
	}

	metaPath := filepath.Join(dir, "meta.json")
	if _, err := os.Stat(metaPath); err == nil {
		meta, err := l.getChapter(mangaID, providerID, chapterID)
		if err == nil {
			meta.DownloadedAt = time.Time{}
			if err := l.saveChapter(mangaID, providerID, chapterID, meta); err != nil {
				return fmt.Errorf("delete chapter files %s/%s/%s: save meta: %w", mangaID, providerID, chapterID, err)
			}
		}
	}

	return nil
}

// BatchDeleteChapterFiles removes page images and pages.json for multiple chapters.
func (l *Library) BatchDeleteChapterFiles(mangaID, providerID string, chapterIDs []string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, chID := range chapterIDs {
		if err := l.deleteChapterFiles(mangaID, providerID, chID); err != nil {
			return fmt.Errorf("batch delete chapter files %s/%s/%s: %w", mangaID, providerID, chID, err)
		}
	}
	return nil
}

// DeleteAllChapters removes chapter subdirectories under library/<mangaID>/.
// If providerIDs is empty, removes all provider subdirectories.
// Otherwise, removes only the specified provider subdirectories.
// Manga meta.json is preserved. Safe no-op if no chapters exist.
func (l *Library) DeleteAllChapters(mangaID string, providerIDs ...string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.deleteAllChapters(mangaID, providerIDs...)
}

func (l *Library) deleteAllChapters(mangaID string, providerIDs ...string) error {
	mangaDir, err := l.mangaDir(mangaID)
	if err != nil {
		return fmt.Errorf("delete all chapters for manga %s: %w", mangaID, err)
	}

	// If no providers specified, remove all provider subdirectories
	if len(providerIDs) == 0 {
		entries, err := os.ReadDir(mangaDir)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return fmt.Errorf("delete all chapters for manga %s: %w", mangaID, err)
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			providerDir := filepath.Join(mangaDir, entry.Name())
			if err := os.RemoveAll(providerDir); err != nil {
				return fmt.Errorf("delete provider %s: %w", entry.Name(), err)
			}
		}
		return nil
	}

	// Remove only specified provider subdirectories
	for _, providerID := range providerIDs {
		providerDir := filepath.Join(mangaDir, providerID)
		if err := os.RemoveAll(providerDir); err != nil {
			return fmt.Errorf("delete provider %s: %w", providerID, err)
		}
	}
	return nil
}

// UpdateChapterProgress updates the reading progress for a chapter and updates the parent manga's last read metadata.
func (l *Library) UpdateChapterProgress(mangaID, providerID, chapterID string, isRead bool, lastReadPage int) (*ChapterInfo, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	mangaID = sanitizeID(mangaID)
	chapterID = sanitizeID(chapterID)

	chMeta, err := l.getChapter(mangaID, providerID, chapterID)
	if err != nil {
		return nil, fmt.Errorf("update chapter progress %s/%s/%s: %w", mangaID, providerID, chapterID, err)
	}

	now := time.Now()
	chMeta.IsRead = isRead
	chMeta.LastReadPage = lastReadPage
	chMeta.LastReadAt = now

	if err := l.saveChapter(mangaID, providerID, chapterID, chMeta); err != nil {
		return nil, fmt.Errorf("update chapter progress %s/%s/%s: save chapter: %w", mangaID, providerID, chapterID, err)
	}

	mangaMeta, err := l.getManga(mangaID)
	if err != nil {
		return nil, fmt.Errorf("update chapter progress %s/%s/%s: get manga: %w", mangaID, providerID, chapterID, err)
	}

	mangaMeta.LastReadChapterID = chapterID
	mangaMeta.LastReadAt = now

	if err := l.saveManga(mangaID, mangaMeta); err != nil {
		return nil, fmt.Errorf("update chapter progress %s/%s/%s: save manga: %w", mangaID, providerID, chapterID, err)
	}

	return &ChapterInfo{
		ID:      chapterID,
		MangaID: mangaID,
		Meta:    *chMeta,
	}, nil
}

// BatchUpdateChapterProgress updates the reading progress for multiple chapters and updates the parent manga's last read metadata.
func (l *Library) BatchUpdateChapterProgress(mangaID, providerID string, chapterIDs []string, isRead bool, lastReadPage int) ([]*ChapterInfo, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	mangaID = sanitizeID(mangaID)
	now := time.Now()
	var updated []*ChapterInfo
	var lastUpdatedChapterID string

	for _, chID := range chapterIDs {
		cleanID := sanitizeID(chID)
		chMeta, err := l.getChapter(mangaID, providerID, cleanID)
		if err != nil {
			return nil, fmt.Errorf("batch update chapter progress %s/%s/%s: %w", mangaID, providerID, cleanID, err)
		}

		chMeta.IsRead = isRead
		chMeta.LastReadPage = lastReadPage
		chMeta.LastReadAt = now

		if err := l.saveChapter(mangaID, providerID, cleanID, chMeta); err != nil {
			return nil, fmt.Errorf("batch update chapter progress %s/%s/%s: save chapter: %w", mangaID, providerID, cleanID, err)
		}

		lastUpdatedChapterID = cleanID
		updated = append(updated, &ChapterInfo{
			ID:      cleanID,
			MangaID: mangaID,
			Meta:    *chMeta,
		})
	}

	if len(updated) > 0 {
		mangaMeta, err := l.getManga(mangaID)
		if err != nil {
			return nil, fmt.Errorf("batch update chapter progress %s: get manga: %w", mangaID, err)
		}

		mangaMeta.LastReadChapterID = lastUpdatedChapterID
		mangaMeta.LastReadAt = now

		if err := l.saveManga(mangaID, mangaMeta); err != nil {
			return nil, fmt.Errorf("batch update chapter progress %s: save manga: %w", mangaID, err)
		}
	}

	return updated, nil
}

// GetChapterPages reads and parses the stored page list for a chapter.
// If mangaID is non-empty, pages are read from library/<manga_id>/<provider_id>/<chapter_id>/pages.json.
// If mangaID is empty, pages are read from library/_pages/<chapter_id>.json.
func (l *Library) GetChapterPages(mangaID, providerID, chapterID string) ([]PageItem, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.getChapterPages(mangaID, providerID, chapterID)
}

func (l *Library) getChapterPages(mangaID, providerID, chapterID string) ([]PageItem, error) {
	chapterID = sanitizeID(chapterID)
	cleanRoot := filepath.Clean(l.root)

	var path string
	if mangaID != "" {
		dir, err := l.providerChapterDir(mangaID, providerID, chapterID)
		if err != nil {
			return nil, fmt.Errorf("get chapter pages %s/%s/%s: %w", mangaID, providerID, chapterID, err)
		}
		path = filepath.Join(dir, "pages.json")
	} else {
		if chapterID == "" {
			return nil, fmt.Errorf("get chapter pages: empty chapter ID")
		}
		path = filepath.Clean(filepath.Join(cleanRoot, "_pages", chapterID+".json"))
		rel, err := filepath.Rel(cleanRoot, path)
		if err != nil || strings.HasPrefix(rel, "..") {
			return nil, fmt.Errorf("get chapter pages: path traversal outside root")
		}
	}

	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("get chapter pages %s/%s/%s: %w", mangaID, providerID, chapterID, err)
	}

	var pages []PageItem
	if err := json.Unmarshal(bytes, &pages); err != nil {
		return nil, fmt.Errorf("get chapter pages %s/%s/%s: unmarshal error: %w", mangaID, providerID, chapterID, err)
	}

	return pages, nil
}

// SaveChapterPages stores the page list for a chapter atomically.
// If mangaID is non-empty, pages are written to library/<manga_id>/<provider_id>/<chapter_id>/pages.json
// and if meta.json exists, its page_count is updated.
// If mangaID is empty, pages are written to library/_pages/<chapter_id>.json.
func (l *Library) SaveChapterPages(mangaID, providerID, chapterID string, pages []PageItem) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.saveChapterPages(mangaID, providerID, chapterID, pages)
}

func (l *Library) saveChapterPages(mangaID, providerID, chapterID string, pages []PageItem) error {
	if len(pages) == 0 {
		return fmt.Errorf("save chapter pages: empty page list")
	}

	chapterID = sanitizeID(chapterID)
	cleanRoot := filepath.Clean(l.root)

	var targetDir, targetFile string
	if mangaID != "" {
		dir, err := l.providerChapterDir(mangaID, providerID, chapterID)
		if err != nil {
			return fmt.Errorf("save chapter pages %s/%s/%s: %w", mangaID, providerID, chapterID, err)
		}
		targetDir = dir
		targetFile = filepath.Join(targetDir, "pages.json")
	} else {
		if chapterID == "" {
			return fmt.Errorf("save chapter pages: empty chapter ID")
		}
		targetDir = filepath.Clean(filepath.Join(cleanRoot, "_pages"))
		targetFile = filepath.Clean(filepath.Join(targetDir, chapterID+".json"))
		rel, err := filepath.Rel(cleanRoot, targetFile)
		if err != nil || strings.HasPrefix(rel, "..") {
			return fmt.Errorf("save chapter pages: path traversal outside root")
		}
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
		metaPath := filepath.Join(targetDir, "meta.json")
		if _, err := os.Stat(metaPath); err == nil {
			chMeta, err := l.getChapter(mangaID, providerID, chapterID)
			if err != nil {
				return fmt.Errorf("save chapter pages: get chapter meta: %w", err)
			}
			chMeta.PageCount = len(pages)
			if err := l.saveChapter(mangaID, providerID, chapterID, chMeta); err != nil {
				return fmt.Errorf("save chapter pages: update chapter meta: %w", err)
			}
		}
	}

	return nil
}

// AddProvider appends a provider binding to manga. Dedup by (provider_id, provider_manga_id).
func (l *Library) AddProvider(mangaID string, ref ProviderRef) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	mangaID = sanitizeID(mangaID)
	meta, err := l.getManga(mangaID)
	if err != nil {
		return fmt.Errorf("add provider: %w", err)
	}

	// Check for duplicate
	for _, p := range meta.Providers {
		if p.ProviderID == ref.ProviderID && p.ProviderMangaID == ref.ProviderMangaID {
			return fmt.Errorf("add provider: duplicate provider binding (%s, %s)", ref.ProviderID, ref.ProviderMangaID)
		}
	}

	meta.Providers = append(meta.Providers, ref)
	if err := l.saveManga(mangaID, meta); err != nil {
		return fmt.Errorf("add provider: save manga: %w", err)
	}
	return nil
}

// RemoveProvider removes the (provider_id, provider_manga_id) entry.
// Returns error if it would leave no Content-capable provider.
func (l *Library) RemoveProvider(mangaID string, providerID, providerMangaID string, capabilityLookup func(providerID string) []string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	mangaID = sanitizeID(mangaID)
	meta, err := l.getManga(mangaID)
	if err != nil {
		return fmt.Errorf("remove provider: %w", err)
	}

	// Find and remove the provider ref
	found := false
	var newProviders []ProviderRef
	for _, p := range meta.Providers {
		if p.ProviderID == providerID && p.ProviderMangaID == providerMangaID {
			found = true
			continue
		}
		newProviders = append(newProviders, p)
	}
	if !found {
		return fmt.Errorf("remove provider: provider binding not found")
	}

	// Check if removal would leave no content-capable provider.
	// Only enforce this constraint if the provider being removed is the current content provider.
	if meta.Content != nil && meta.Content.ProviderID == providerID && meta.Content.ProviderMangaID == providerMangaID {
		// Check if any OTHER remaining provider has content capability
		hasContent := false
		for _, p := range newProviders {
			caps := capabilityLookup(p.ProviderID)
			for _, cap := range caps {
				if cap == "content" {
					hasContent = true
					break
				}
			}
			if hasContent {
				break
			}
		}
		if !hasContent {
			return fmt.Errorf("remove provider: cannot remove last content-capable provider")
		}
		// Clear content since we're removing the content provider
		meta.Content = nil
	}

	meta.Providers = newProviders
	if err := l.saveManga(mangaID, meta); err != nil {
		return fmt.Errorf("remove provider: save manga: %w", err)
	}
	return nil
}

// SwitchContentProvider sets content provider and re-correlates chapters.
// If provider not in providers[], add it first.
func (l *Library) SwitchContentProvider(mangaID string, providerID, providerMangaID string, mangaTitle string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	mangaID = sanitizeID(mangaID)
	meta, err := l.getManga(mangaID)
	if err != nil {
		return fmt.Errorf("switch content provider: %w", err)
	}

	// Check if provider is already in providers list
	found := false
	for _, p := range meta.Providers {
		if p.ProviderID == providerID && p.ProviderMangaID == providerMangaID {
			found = true
			break
		}
	}

	// If not found, add it
	if !found {
		title := mangaTitle
		if title == "" {
			title = meta.Title
		}
		meta.Providers = append(meta.Providers, ProviderRef{
			ProviderID:      providerID,
			ProviderMangaID: providerMangaID,
			MangaTitle:      title,
		})
	}

	// Set content provider
	if meta.Content == nil {
		meta.Content = &ContentSource{}
	}
	meta.Content.ProviderID = providerID
	meta.Content.ProviderMangaID = providerMangaID

	if err := l.saveManga(mangaID, meta); err != nil {
		return fmt.Errorf("switch content provider: save manga: %w", err)
	}
	return nil
}

// HasContentProvider returns true if any provider in providers[] (excluding the given pair) has Content capability.
func (l *Library) HasContentProvider(mangaID string, excludingProviderID, excludingMangaID string, capabilityLookup func(providerID string) []string) (bool, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	mangaID = sanitizeID(mangaID)
	meta, err := l.getManga(mangaID)
	if err != nil {
		return false, fmt.Errorf("has content provider: %w", err)
	}

	for _, p := range meta.Providers {
		if p.ProviderID == excludingProviderID && p.ProviderMangaID == excludingMangaID {
			continue
		}
		caps := capabilityLookup(p.ProviderID)
		for _, cap := range caps {
			if cap == "content" {
				return true, nil
			}
		}
	}
	return false, nil
}

// HasCover reports whether a cover image file exists for the given manga.
func (l *Library) HasCover(mangaID string) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.hasCover(mangaID)
}

func (l *Library) hasCover(mangaID string) bool {
	mangaDir, err := l.mangaDir(mangaID)
	if err != nil {
		return false
	}
	for _, ext := range []string{".jpg", ".jpeg", ".png", ".webp", ".gif"} {
		if _, err := os.Stat(filepath.Join(mangaDir, "cover"+ext)); err == nil {
			return true
		}
	}
	return false
}

// ClaimCoverEnqueue atomically decides whether the caller should enqueue a
// pull_cover job for mangaID. Returns (true, nil) when:
//   - no cover file is on disk yet, AND
//   - the caller successfully created a ".cover-pending" sentinel via O_EXCL,
//     which acts as a TOCTOU-safe lock against concurrent enqueues.
//
// Returns (false, nil) when either a cover is already on disk or another
// caller already claimed the sentinel. The claim is released by the
// pull_cover handler via ReleaseCoverClaim after SyncCover completes, so a
// failed cover fetch can be re-enqueued.
func (l *Library) ClaimCoverEnqueue(mangaID string) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.claimCoverEnqueue(mangaID)
}

func (l *Library) claimCoverEnqueue(mangaID string) (bool, error) {
	if l.hasCover(mangaID) {
		return false, nil
	}
	mangaDir, err := l.mangaDir(mangaID)
	if err != nil {
		return false, fmt.Errorf("claim cover enqueue %s: %w", mangaID, err)
	}
	if err := os.MkdirAll(mangaDir, 0o755); err != nil {
		return false, fmt.Errorf("claim cover enqueue %s: mkdir: %w", mangaID, err)
	}
	flagPath := filepath.Join(mangaDir, ".cover-pending")
	f, err := os.OpenFile(flagPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			return false, nil
		}
		return false, fmt.Errorf("claim cover enqueue %s: %w", mangaID, err)
	}
	_ = f.Close()
	return true, nil
}

// ReleaseCoverClaim removes the ".cover-pending" sentinel placed by
// ClaimCoverEnqueue. Safe to call when no sentinel exists.
func (l *Library) ReleaseCoverClaim(mangaID string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.releaseCoverClaim(mangaID)
}

func (l *Library) releaseCoverClaim(mangaID string) error {
	mangaDir, err := l.mangaDir(mangaID)
	if err != nil {
		return fmt.Errorf("release cover claim %s: %w", mangaID, err)
	}
	flagPath := filepath.Join(mangaDir, ".cover-pending")
	if err := os.Remove(flagPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("release cover claim %s: %w", mangaID, err)
	}
	return nil
}

// CoverPath returns the absolute path of the cover file for a manga, or empty string if none exists.
func (l *Library) CoverPath(mangaID string) string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.coverPath(mangaID)
}

func (l *Library) coverPath(mangaID string) string {
	mangaDir, err := l.mangaDir(mangaID)
	if err != nil {
		return ""
	}
	for _, ext := range []string{".jpg", ".jpeg", ".png", ".webp", ".gif"} {
		p := filepath.Join(mangaDir, "cover"+ext)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// extensionFromCoverResponse picks the file extension for a cover image based on
// the URL path and HTTP Content-Type header. Falls back to ".jpg" when no
// recognized extension can be determined.
func extensionFromCoverResponse(rawURL, contentType string) string {
	if u, err := url.Parse(rawURL); err == nil {
		if idx := strings.LastIndex(u.Path, "."); idx >= 0 {
			ext := strings.ToLower(u.Path[idx:])
			switch ext {
			case ".jpg", ".jpeg", ".png", ".webp", ".gif":
				return ext
			}
		}
	}
	switch strings.ToLower(contentType) {
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	case "image/jpeg", "image/jpg":
		return ".jpg"
	}
	return ".jpg"
}

// SyncCover downloads a cover image from coverURL and atomically writes it to
// <root>/<mangaID>/cover.<ext>. The extension is inferred from the URL path, then
// the HTTP Content-Type header, and finally defaults to ".jpg". Returns an error
// if the URL is empty, the download fails, or the HTTP status is >= 400.
func (l *Library) SyncCover(mangaID string, coverURL string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.syncCover(mangaID, coverURL)
}

func (l *Library) syncCover(mangaID string, coverURL string) error {
	if coverURL == "" {
		return fmt.Errorf("refresh cover %s: empty cover URL", mangaID)
	}

	if _, err := ssrfguard.ValidateURL(coverURL, l.allowPrivateNetworks); err != nil {
		return fmt.Errorf("refresh cover %s: %w", mangaID, err)
	}

	mangaDir, err := l.mangaDir(mangaID)
	if err != nil {
		return fmt.Errorf("refresh cover %s: %w", mangaID, err)
	}
	if err := os.MkdirAll(mangaDir, 0o755); err != nil {
		return fmt.Errorf("refresh cover %s: create dir: %w", mangaID, err)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(coverURL)
	if err != nil {
		return fmt.Errorf("refresh cover %s: fetch: %w", mangaID, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("refresh cover %s: HTTP status %d", mangaID, resp.StatusCode)
	}

	ext := extensionFromCoverResponse(coverURL, resp.Header.Get("Content-Type"))
	target := filepath.Join(mangaDir, "cover"+ext)

	tmpFile, err := os.CreateTemp(mangaDir, "cover.tmp.*")
	if err != nil {
		return fmt.Errorf("refresh cover %s: create temp file: %w", mangaID, err)
	}
	tmpPath := tmpFile.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("refresh cover %s: copy body: %w", mangaID, err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("refresh cover %s: close temp file: %w", mangaID, err)
	}

	if err := os.Rename(tmpPath, target); err != nil {
		return fmt.Errorf("refresh cover %s: rename temp file: %w", mangaID, err)
	}

	return nil
}
