package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/tubruk/kiyomi/internal/cache"
	"github.com/tubruk/kiyomi/internal/library"
	"github.com/tubruk/kiyomi/internal/security/ssrfguard"
	"github.com/tubruk/kiyomi/internal/upstream"
	"github.com/tubruk/kiyomi/pkg/fingerprint"
	"github.com/tubruk/kiyomi/pkg/provider"
	"github.com/tubruk/kiyomi/pkg/provider/sdk"
)

// Job type names for the pull pipeline.
const (
	JobTypePullManga   = "pull_manga"
	JobTypePullChapter = "pull_chapter"
	JobTypePullPage    = "pull_page"
	JobTypePullCover   = "pull_cover"
)

// Concurrency group prefixes. Provider-scoped jobs share "pull:<providerID>"
// to throttle per-provider fan-out; cover jobs use a dedicated bucket.
const (
	coverConcurrencyGroup = "pull:cover"
)

// PullMangaPayload is the JSON payload for pull_manga jobs.
type PullMangaPayload struct {
	MangaID         string `json:"manga_id"`
	ProviderID      string `json:"provider_id"`
	ProviderMangaID string `json:"provider_manga_id"`
	CoverURL        string `json:"cover_url,omitempty"`
}

// PullChapterPayload is the JSON payload for pull_chapter jobs.
type PullChapterPayload struct {
	MangaID    string `json:"manga_id"`
	ProviderID string `json:"provider_id"`
	ChapterID  string `json:"chapter_id"`
}

// PullPagePayload is the JSON payload for pull_page jobs.
type PullPagePayload struct {
	MangaID    string `json:"manga_id"`
	ProviderID string `json:"provider_id"`
	ChapterID  string `json:"chapter_id"`
	PageIndex  int    `json:"page_index"`
	PageURL    string `json:"page_url"`
}

// PullCoverPayload is the JSON payload for pull_cover jobs.
type PullCoverPayload struct {
	MangaID    string `json:"manga_id"`
	CoverURL   string `json:"cover_url"`
	ProviderID string `json:"provider_id,omitempty"`
}

// PullHandlers implements queue.JobHandler for the four pull_* job types
// and orchestrates fetching chapters, pages, and covers into the local library.
type PullHandlers struct {
	lib                  *library.Library
	registry             *provider.Registry
	httpClient           *http.Client
	fpStore              fingerprint.Store
	reqBuilder           *upstream.RequestBuilder
	enqueuer             Enqueuer
	jobStore             JobStore
	logger               *slog.Logger
	imageCache           *cache.DiskCache
	allowPrivateNetworks bool

	mangaLocks [64]sync.Mutex
}

// NewPullHandlers constructs a PullHandlers. If httpClient is nil, a
// default client with a 15s timeout is used. The enqueuer and jobStore are
// required so that child jobs can be scheduled.
func NewPullHandlers(lib *library.Library, reg *provider.Registry, httpClient *http.Client, fpStore fingerprint.Store, enqueuer Enqueuer, jobStore JobStore) *PullHandlers {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &PullHandlers{
		lib:        lib,
		registry:   reg,
		httpClient: httpClient,
		fpStore:    fpStore,
		reqBuilder: upstream.NewRequestBuilder(fpStore, reg, httpClient),
		enqueuer:   enqueuer,
		jobStore:   jobStore,
		logger:     slog.Default(),
	}
}

// SetImageCache configures the disk image cache used for hardlinking or copying
// cached assets before falling back to upstream network fetches.
func (d *PullHandlers) SetImageCache(ic *cache.DiskCache) {
	d.imageCache = ic
}

// ImageCache returns the configured disk image cache, or nil if none.
func (d *PullHandlers) ImageCache() *cache.DiskCache {
	return d.imageCache
}

// SetRequestBuilder replaces the request builder used for downloading images.
func (d *PullHandlers) SetRequestBuilder(builder *upstream.RequestBuilder) {
	d.reqBuilder = builder
}

// RequestBuilder returns the current request builder.
func (d *PullHandlers) RequestBuilder() *upstream.RequestBuilder {
	return d.reqBuilder
}

// getMangaLock returns a striped mutex for the given mangaID based on fnv32a hash.
// Callers must hold the returned mutex while running fan-out operations to prevent
// duplicate child jobs from concurrent pull_manga invocations.
func (d *PullHandlers) getMangaLock(mangaID string) *sync.Mutex {
	h := fnv.New32a()
	_, _ = h.Write([]byte(mangaID))
	idx := h.Sum32() % uint32(len(d.mangaLocks))
	return &d.mangaLocks[idx]
}

// SetAllowPrivateNetworks toggles the SSRF guard used by outbound cover and
// page fetches. When false (the default), URLs that resolve to private,
// loopback, or link-local address ranges are rejected before any request is
// made. Flip to true in local development against 127.0.0.1 or LAN-hosted
// providers.
func (d *PullHandlers) SetAllowPrivateNetworks(allow bool) {
	d.allowPrivateNetworks = allow
}

// Lib returns the underlying library instance. Exposed primarily for tests and
// for callers that need to seed state before scheduling pull jobs.
func (d *PullHandlers) Lib() *library.Library {
	return d.lib
}

// HandlePullManga fetches the chapter list from the provider, persists chapter
// metadata for any missing chapters, and enqueues a pull_chapter job per new
// chapter. If the manga has no cover file on disk, a pull_cover job is enqueued.
func (d *PullHandlers) HandlePullManga(ctx context.Context, job *Job) error {
	var p PullMangaPayload
	if err := json.Unmarshal([]byte(job.Payload), &p); err != nil {
		return Permanent(fmt.Errorf("pull_manga: decode payload: %w", err))
	}
	if p.MangaID == "" || p.ProviderID == "" || p.ProviderMangaID == "" {
		return Permanent(fmt.Errorf("pull_manga: missing required fields"))
	}

	group := pullGroup(p.ProviderID)
	job.ConcurrencyGroup = group

	content, ok := d.registry.GetContent(p.ProviderID)
	if !ok {
		return Permanent(fmt.Errorf("pull_manga: provider %q not available", p.ProviderID))
	}

	chapters, err := content.FetchChapters(ctx, p.ProviderMangaID)
	if err != nil {
		return fmt.Errorf("pull_manga: fetch chapters: %w", err)
	}

	mu := d.getMangaLock(p.MangaID)
	mu.Lock()
	defer mu.Unlock()

	existing, err := d.lib.ListChapters(p.MangaID, p.ProviderID)
	if err != nil {
		return fmt.Errorf("pull_manga: list chapters: %w", err)
	}
	existingMap := make(map[string]bool, len(existing))
	for _, ch := range existing {
		existingMap[ch.ID] = true
	}

	now := time.Now()
	for _, ch := range chapters {
		if existingMap[ch.ID] {
			continue
		}

		chMeta := &library.ChapterMeta{
			Title:       ch.Name,
			Number:      ch.Number,
			UploadDate:  ch.UploadDate,
			SourceOrder: ch.SourceOrder,
			Content: &library.ContentSource{
				ProviderID:   p.ProviderID,
				ChapterRef:   ch.ID,
				LastSyncedAt: now,
			},
		}
		if err := d.lib.SaveChapter(p.MangaID, p.ProviderID, ch.ID, chMeta); err != nil {
			return fmt.Errorf("pull_manga: save chapter %s: %w", ch.ID, err)
		}

		if err := d.enqueueChild(ctx, &job.ID, JobTypePullChapter, PullChapterPayload{
			MangaID:    p.MangaID,
			ProviderID: p.ProviderID,
			ChapterID:  ch.ID,
		}, group, map[string]string{
			"manga_id":    p.MangaID,
			"provider_id": p.ProviderID,
			"chapter_id":  ch.ID,
		}); err != nil {
			return fmt.Errorf("pull_manga: enqueue pull_chapter for %s: %w", ch.ID, err)
		}
	}

	if p.CoverURL != "" {
		claimed, err := d.lib.ClaimCoverEnqueue(p.MangaID)
		if err != nil {
			return Permanent(fmt.Errorf("pull_manga: claim cover enqueue: %w", err))
		}
		if claimed {
			if err := d.enqueueChild(ctx, &job.ID, JobTypePullCover, PullCoverPayload{
				MangaID:    p.MangaID,
				CoverURL:   p.CoverURL,
				ProviderID: p.ProviderID,
			}, coverConcurrencyGroup, map[string]string{
				"manga_id":    p.MangaID,
				"provider_id": p.ProviderID,
			}); err != nil {
				_ = d.lib.ReleaseCoverClaim(p.MangaID)
				return fmt.Errorf("pull_manga: enqueue pull_cover: %w", err)
			}
		}
	}

	// Stamp the manga's Content.LastSyncedAt so callers can tell when the
	// upstream chapter list was last reconciled. Only update if the manga is
	// already bound to a content provider — pull_manga may run before the
	// binding is set, in which case there's nothing to stamp yet.
	mangaMeta, err := d.lib.GetManga(p.MangaID)
	if err != nil {
		return fmt.Errorf("pull_manga: get manga meta: %w", err)
	}
	if mangaMeta.Content != nil {
		mangaMeta.Content.LastSyncedAt = now
		if err := d.lib.SaveManga(p.MangaID, mangaMeta); err != nil {
			return fmt.Errorf("pull_manga: save manga meta: %w", err)
		}
	}

	return nil
}

// HandlePullChapter resolves the page list for a chapter, persists it via
// SaveChapterPages, and enqueues a pull_page job per page.
func (d *PullHandlers) HandlePullChapter(ctx context.Context, job *Job) error {
	var p PullChapterPayload
	if err := json.Unmarshal([]byte(job.Payload), &p); err != nil {
		return Permanent(fmt.Errorf("pull_chapter: decode payload: %w", err))
	}
	if p.MangaID == "" || p.ProviderID == "" || p.ChapterID == "" {
		return Permanent(fmt.Errorf("pull_chapter: missing required fields"))
	}

	group := pullGroup(p.ProviderID)
	job.ConcurrencyGroup = group

	content, ok := d.registry.GetContent(p.ProviderID)
	if !ok {
		return Permanent(fmt.Errorf("pull_chapter: provider %q not available", p.ProviderID))
	}

	// Resolve the provider-side chapter ref stored in chapter meta if available.
	chapterRef := p.ChapterID
	if chMeta, err := d.lib.GetChapter(p.MangaID, p.ProviderID, p.ChapterID); err == nil && chMeta.Content != nil && chMeta.Content.ChapterRef != "" {
		chapterRef = chMeta.Content.ChapterRef
	}

	mangaRef := p.MangaID
	if mangaMeta, err := d.lib.GetManga(p.MangaID); err == nil && mangaMeta.Content != nil && mangaMeta.Content.ProviderMangaID != "" {
		mangaRef = mangaMeta.Content.ProviderMangaID
	}

	pages, err := content.FetchPages(ctx, mangaRef, chapterRef)
	if err != nil {
		return fmt.Errorf("pull_chapter: fetch pages: %w", err)
	}

	pageItems := make([]library.PageItem, 0, len(pages))
	for _, pg := range pages {
		pageItems = append(pageItems, library.PageItem{
			Index:  pg.Index,
			URL:    pg.URL,
			Source: "provider",
		})
	}

	if err := d.lib.SaveChapterPages(p.MangaID, p.ProviderID, p.ChapterID, pageItems); err != nil {
		return fmt.Errorf("pull_chapter: save pages: %w", err)
	}

	for _, pg := range pages {
		if err := d.enqueueChild(ctx, &job.ID, JobTypePullPage, PullPagePayload{
			MangaID:    p.MangaID,
			ProviderID: p.ProviderID,
			ChapterID:  p.ChapterID,
			PageIndex:  pg.Index,
			PageURL:    pg.URL,
		}, group, map[string]string{
			"manga_id":    p.MangaID,
			"provider_id": p.ProviderID,
			"chapter_id":  p.ChapterID,
			"page_index":  strconv.Itoa(pg.Index),
		}); err != nil {
			return fmt.Errorf("pull_chapter: enqueue pull_page %d: %w", pg.Index, err)
		}
	}

	return nil
}

// HandlePullPage downloads a single page image and writes it atomically to
// <root>/<mangaID>/<providerID>/<chapterID>/<index>.<ext>. On success, the
// parent chapter's DownloadedAt timestamp is updated under the per-manga lock
// so the UI can detect that files are present.
func (d *PullHandlers) HandlePullPage(ctx context.Context, job *Job) error {
	var p PullPagePayload
	if err := json.Unmarshal([]byte(job.Payload), &p); err != nil {
		return Permanent(fmt.Errorf("pull_page: decode payload: %w", err))
	}
	if p.MangaID == "" || p.ProviderID == "" || p.ChapterID == "" || p.PageURL == "" {
		return Permanent(fmt.Errorf("pull_page: missing required fields"))
	}

	group := pullGroup(p.ProviderID)
	job.ConcurrencyGroup = group

	if _, err := ssrfguard.ValidateURL(p.PageURL, d.allowPrivateNetworks); err != nil {
		return Permanent(fmt.Errorf("pull_page %d: %w", p.PageIndex, err))
	}

	if err := d.downloadImage(ctx, p.PageURL, p.ProviderID, func(ext string) (string, error) {
		dir := d.lib.ProviderChapterDir(p.MangaID, p.ProviderID, p.ChapterID)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", fmt.Errorf("create chapter dir: %w", err)
		}
		return filepath.Join(dir, fmt.Sprintf("%d%s", p.PageIndex, ext)), nil
	}); err != nil {
		return fmt.Errorf("pull_page %d: %w", p.PageIndex, err)
	}

	// Stamp the chapter's DownloadedAt so the UI can tell that files are on
	// disk. Performed under the per-manga lock so concurrent pull_page jobs for
	// the same manga serialize their meta writes alongside pull_manga.
	mu := d.getMangaLock(p.MangaID)
	mu.Lock()
	defer mu.Unlock()

	chMeta, err := d.lib.GetChapter(p.MangaID, p.ProviderID, p.ChapterID)
	if err != nil {
		return fmt.Errorf("pull_page %d: get chapter meta: %w", p.PageIndex, err)
	}
	chMeta.DownloadedAt = time.Now()
	if err := d.lib.SaveChapter(p.MangaID, p.ProviderID, p.ChapterID, chMeta); err != nil {
		return fmt.Errorf("pull_page %d: save chapter meta: %w", p.PageIndex, err)
	}

	return nil
}

// HandlePullCover downloads a cover image and writes it to <root>/<mangaID>/cover.<ext>.
func (d *PullHandlers) HandlePullCover(ctx context.Context, job *Job) error {
	var p PullCoverPayload
	if err := json.Unmarshal([]byte(job.Payload), &p); err != nil {
		return Permanent(fmt.Errorf("pull_cover: decode payload: %w", err))
	}
	if p.MangaID == "" || p.CoverURL == "" {
		return Permanent(fmt.Errorf("pull_cover: missing required fields"))
	}

	job.ConcurrencyGroup = coverConcurrencyGroup

	if _, err := ssrfguard.ValidateURL(p.CoverURL, d.allowPrivateNetworks); err != nil {
		_ = d.lib.ReleaseCoverClaim(p.MangaID)
		return Permanent(fmt.Errorf("pull_cover: %w", err))
	}

	providerID := p.ProviderID
	if providerID == "" {
		if mangaMeta, err := d.lib.GetManga(p.MangaID); err == nil && mangaMeta != nil {
			if mangaMeta.Content != nil && mangaMeta.Content.ProviderID != "" {
				providerID = mangaMeta.Content.ProviderID
			} else if len(mangaMeta.Providers) > 0 && mangaMeta.Providers[0].ProviderID != "" {
				providerID = mangaMeta.Providers[0].ProviderID
			}
		}
	}

	if err := d.downloadImage(ctx, p.CoverURL, providerID, func(ext string) (string, error) {
		mangaDir := filepath.Join(d.lib.Root(), p.MangaID)
		if err := os.MkdirAll(mangaDir, 0o755); err != nil {
			return "", fmt.Errorf("create manga dir: %w", err)
		}
		return filepath.Join(mangaDir, "cover"+ext), nil
	}); err != nil {
		_ = d.lib.ReleaseCoverClaim(p.MangaID)
		return fmt.Errorf("pull_cover: %w", err)
	}
	if err := d.lib.ReleaseCoverClaim(p.MangaID); err != nil {
		return fmt.Errorf("pull_cover: release claim: %w", err)
	}
	return nil
}

// enqueueChild builds, persists, and schedules a child job with the given
// payload type, payload body, concurrency group, and metadata.
func (d *PullHandlers) enqueueChild(ctx context.Context, parentID *string, jobType string, payload any, group string, meta map[string]string) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	if meta == nil {
		meta = map[string]string{}
	}

	now := time.Now()
	child := &Job{
		ID:               uuid.New().String(),
		ParentID:         parentID,
		Type:             jobType,
		Payload:          string(payloadBytes),
		MaxRetries:       3,
		ConcurrencyGroup: group,
		Metadata:         meta,
		CreatedAt:        now,
		UpdatedAt:        now,
		Status:           StatusPending,
	}

	if d.jobStore != nil {
		if err := d.jobStore.CreateJob(ctx, child); err != nil {
			return fmt.Errorf("persist child job: %w", err)
		}
	}

	if d.enqueuer == nil {
		return fmt.Errorf("no enqueuer configured")
	}
	if err := d.enqueuer.Enqueue(ctx, child); err != nil {
		return fmt.Errorf("enqueue child job: %w", err)
	}
	return nil
}

// downloadImage fetches rawURL via the configured httpClient using the upstream
// request builder (applying client profile, session cookies, and appropriate headers)
// and atomically writes the body to the path returned by resolveTarget.
// If imageCache is configured and rawURL is already cached, it attempts to hardlink
// or copy the file from cache into target to avoid a network roundtrip.
func (d *PullHandlers) downloadImage(ctx context.Context, rawURL string, providerID string, resolveTarget func(ext string) (string, error)) error {
	if d.imageCache != nil {
		if cachedPath, meta, ok := d.imageCache.GetPath(rawURL); ok {
			ext := extensionFromImageResponse(rawURL, meta.ContentType)
			target, err := resolveTarget(ext)
			if err != nil {
				return err
			}

			dir := filepath.Dir(target)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return fmt.Errorf("create target dir: %w", err)
			}

			_ = os.Remove(target)

			// Attempt hardlink first to avoid duplicate disk space.
			if err := os.Link(cachedPath, target); err == nil {
				return nil
			}

			// Fallback: atomic byte copy if hardlink fails (e.g. cross-filesystem boundary).
			if err := copyFileAtomically(cachedPath, target); err == nil {
				return nil
			}
		}
	}

	var req *http.Request
	var err error
	if d.reqBuilder != nil {
		req, err = d.reqBuilder.BuildRequest(ctx, rawURL, providerID, "")
	} else {
		req, err = upstream.NewRequest(ctx, rawURL, providerID, "")
	}
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		switch resp.StatusCode {
		case http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusGone:
			return Permanent(fmt.Errorf("HTTP status %d", resp.StatusCode))
		default:
			return fmt.Errorf("HTTP status %d", resp.StatusCode)
		}
	}

	ext := extensionFromImageResponse(rawURL, resp.Header.Get("Content-Type"))
	target, err := resolveTarget(ext)
	if err != nil {
		return err
	}

	dir := filepath.Dir(target)
	tmpFile, err := os.CreateTemp(dir, filepath.Base(target)+".tmp.*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("copy body: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, target); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}
	return nil
}

func copyFileAtomically(srcPath, dstPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open source file: %w", err)
	}
	defer src.Close()

	dir := filepath.Dir(dstPath)
	tmpFile, err := os.CreateTemp(dir, filepath.Base(dstPath)+".tmp.*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err := io.Copy(tmpFile, src); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("copy bytes: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	return os.Rename(tmpPath, dstPath)
}

// extensionFromImageResponse picks the file extension for an image response
// based on the URL path and HTTP Content-Type header, defaulting to ".jpg".
func extensionFromImageResponse(rawURL, contentType string) string {
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

// pullGroup returns the per-provider concurrency group name.
func pullGroup(providerID string) string {
	return "pull:" + providerID
}

// Default concurrency limits applied to the pull pipeline. The cover bucket is
// shared across providers and kept modest so cover refreshes don't starve the
// per-provider page-fetch buckets.
const (
	DefaultCoverGroupLimit   = 4
	DefaultContentGroupLimit = 8
	maxContentGroupLimit     = 16
	contentBurstMultiplier   = 2.0
)

// ApplyPullGroupLimits wires per-group concurrency limits onto worker for the
// pull pipeline. Cover jobs share a "pull:cover" bucket; each content provider
// gets its own "pull:<id>" bucket sized from RateLimit().RequestsPerSecond
// with sensible fallbacks. providers is the list returned by
// provider.Registry.ListContent(). It is safe to call with a nil worker.
func ApplyPullGroupLimits(worker Worker, providers []sdk.Content) {
	if worker == nil {
		return
	}
	worker.SetGroupLimit(CoverConcurrencyGroup, DefaultCoverGroupLimit)
	for _, p := range providers {
		if p == nil {
			continue
		}
		worker.SetGroupLimit(PullGroup(p.ID()), computeContentGroupLimit(p.RateLimit()))
	}
}

// CoverConcurrencyGroup is the concurrency group name shared by all
// pull_cover jobs.
const CoverConcurrencyGroup = "pull:cover"

// PullGroup returns the per-provider concurrency group name.
func PullGroup(providerID string) string {
	return "pull:" + providerID
}

// computeContentGroupLimit returns a sensible concurrency limit for a content
// provider based on its rate limit hint. Providers that do not advertise a
// request rate fall back to DefaultContentGroupLimit.
func computeContentGroupLimit(hint sdk.RateLimitHint) int {
	if hint.RequestsPerSecond <= 0 {
		return DefaultContentGroupLimit
	}
	n := int(hint.RequestsPerSecond * contentBurstMultiplier)
	if n < 1 {
		n = 1
	}
	if n > maxContentGroupLimit {
		n = maxContentGroupLimit
	}
	return n
}
