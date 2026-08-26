package queue_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/tubruk/kiyomi/internal/library"
	"github.com/tubruk/kiyomi/internal/queue"
	"github.com/tubruk/kiyomi/internal/queue/inmemory"
	"github.com/tubruk/kiyomi/pkg/fingerprint"
	"github.com/tubruk/kiyomi/pkg/provider"
	"github.com/tubruk/kiyomi/pkg/provider/sdk"
)

// fakeContentProvider is a minimal sdk.Content implementation for testing.
type fakeContentProvider struct {
	id       string
	chapters []sdk.Chapter
	pages    map[string][]sdk.Page
}

func (p *fakeContentProvider) ID() string        { return p.id }
func (p *fakeContentProvider) Name() string      { return p.id }
func (p *fakeContentProvider) Icon() string      { return "" }
func (p *fakeContentProvider) Capabilities() []string { return []string{"content"} }
func (p *fakeContentProvider) ConfigKeys() []sdk.ConfigKeySpec {
	return nil
}
func (p *fakeContentProvider) RequiresAuth() bool             { return false }
func (p *fakeContentProvider) State() sdk.ProviderState       { return sdk.StateActive }
func (p *fakeContentProvider) HasStableChapterID() bool       { return true }
func (p *fakeContentProvider) RateLimit() sdk.RateLimitHint   { return sdk.RateLimitHint{} }
func (p *fakeContentProvider) FetchChapters(_ context.Context, _ string) ([]sdk.Chapter, error) {
	return p.chapters, nil
}
func (p *fakeContentProvider) FetchPages(_ context.Context, _, chapterRef string) ([]sdk.Page, error) {
	if pages, ok := p.pages[chapterRef]; ok {
		return pages, nil
	}
	return nil, nil
}
func (p *fakeContentProvider) FetchPageStream(_ context.Context, _ sdk.Page) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}

// newTestHandlers wires a queue.PullHandlers backed by an in-memory queue
// driver and an httptest server that serves page images. Returns the handlers,
// the driver (for assertions), the page-server base URL, and a cleanup func.
//
// The SSRF guard is opted out via SetAllowPrivateNetworks on both the queue
// handlers and the underlying Library because the in-process test server
// binds 127.0.0.1. Tests that exercise the SSRF guard itself flip it back on.
func newTestHandlers(t *testing.T, prov sdk.Content) (*queue.PullHandlers, *inmemory.Driver, string, func()) {
	t.Helper()

	libRoot, err := os.MkdirTemp("", "kiyomi-pull-handler-*")
	if err != nil {
		t.Fatalf("mkdir temp: %v", err)
	}
	lib := library.NewLibrary(libRoot)
	lib.SetAllowPrivateNetworks(true)

	driver := inmemory.NewDriver(64)

	pageSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte("PAGE-IMAGE-BYTES"))
	}))

	httpClient := &http.Client{Timeout: 5 * time.Second}

	reg := provider.NewRegistry()
	reg.Register(prov)

	dh := queue.NewPullHandlers(lib, reg, httpClient, nil, driver, driver)
	dh.SetAllowPrivateNetworks(true)

	cleanup := func() {
		pageSrv.Close()
		_ = os.RemoveAll(libRoot)
	}

	return dh, driver, pageSrv.URL, cleanup
}

func TestHandlePullChapter_PersistsPagesJSON(t *testing.T) {
	provID := "fakeprov"
	pages := []sdk.Page{
		{Index: 0, URL: "http://example.com/page-0.jpg"},
		{Index: 1, URL: "http://example.com/page-1.jpg"},
	}
	prov := &fakeContentProvider{
		id:    provID,
		pages: map[string][]sdk.Page{"ch-1": pages},
	}

	dh, driver, _, cleanup := newTestHandlers(t, prov)
	defer cleanup()

	mangaID := "manga-A"
	chapterID := "ch-1"
	providerID := provID

	if err := dh.Lib().SaveChapter(mangaID, providerID, chapterID, &library.ChapterMeta{
		Title:  "Chapter 1",
		Number: 1.0,
		Content: &library.ContentSource{
			ProviderID: providerID,
			ChapterRef: chapterID,
		},
	}); err != nil {
		t.Fatalf("seed chapter: %v", err)
	}

	payload, _ := json.Marshal(queue.PullChapterPayload{
		MangaID: mangaID, ProviderID: providerID, ChapterID: chapterID,
	})
	job := &queue.Job{
		ID:               uuid.New().String(),
		Type:             queue.JobTypePullChapter,
		Payload:          string(payload),
		ConcurrencyGroup: "pull:" + providerID,
	}

	if err := driver.Register(queue.JobTypePullChapter, queue.JobHandlerFunc(dh.HandlePullChapter)); err != nil {
		t.Fatalf("register handler: %v", err)
	}
	if err := driver.Start(context.Background()); err != nil {
		t.Fatalf("start driver: %v", err)
	}
	if err := driver.Enqueue(context.Background(), job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	if !waitForStatus(t, driver, job.ID, queue.StatusCompleted, 3*time.Second) {
		t.Fatalf("job did not complete")
	}

	pagesPath := filepath.Join(dh.Lib().ProviderChapterDir(mangaID, providerID, chapterID), "pages.json")
	if _, err := os.Stat(pagesPath); err != nil {
		t.Fatalf("expected pages.json at %s: %v", pagesPath, err)
	}

	got, err := dh.Lib().GetChapterPages(mangaID, providerID, chapterID)
	if err != nil {
		t.Fatalf("GetChapterPages: %v", err)
	}
	if len(got) != len(pages) {
		t.Fatalf("expected %d pages, got %d", len(pages), len(got))
	}
	for i, p := range got {
		if p.Index != pages[i].Index || p.URL != pages[i].URL {
			t.Errorf("page %d mismatch: got %+v want %+v", i, p, pages[i])
		}
	}

	childJobs, err := driver.ListJobs(context.Background(), queue.JobFilter{Type: queue.JobTypePullPage})
	if err != nil {
		t.Fatalf("ListJobs pull_page: %v", err)
	}
	if len(childJobs) != len(pages) {
		t.Fatalf("expected %d pull_page child jobs, got %d", len(pages), len(childJobs))
	}
	for i, j := range childJobs {
		if j.Metadata["manga_id"] != mangaID {
			t.Errorf("job %d: expected manga_id %q, got %q", i, mangaID, j.Metadata["manga_id"])
		}
		if j.Metadata["provider_id"] != providerID {
			t.Errorf("job %d: expected provider_id %q, got %q", i, providerID, j.Metadata["provider_id"])
		}
		if j.Metadata["chapter_id"] != chapterID {
			t.Errorf("job %d: expected chapter_id %q, got %q", i, chapterID, j.Metadata["chapter_id"])
		}
		if j.Metadata["page_index"] != strconv.Itoa(pages[i].Index) {
			t.Errorf("job %d: expected page_index %q, got %q", i, strconv.Itoa(pages[i].Index), j.Metadata["page_index"])
		}
		if _, ok := j.Metadata["page_url"]; ok {
			t.Errorf("job %d: metadata should not contain page_url", i)
		}
	}
}

func TestHandlePullPage_WritesImageFile(t *testing.T) {
	provID := "fakeprov"
	prov := &fakeContentProvider{id: provID}

	dh, driver, pageURL, cleanup := newTestHandlers(t, prov)
	defer cleanup()

	mangaID := "manga-B"
	providerID := provID
	chapterID := "ch-1"
	index := 3
	pageURLFull := pageURL + "/pages/3.jpg"

	dir := dh.Lib().ProviderChapterDir(mangaID, providerID, chapterID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir chapter dir: %v", err)
	}

	// Seed chapter meta so HandlePullPage can stamp DownloadedAt after the
	// image write.
	if err := dh.Lib().SaveChapter(mangaID, providerID, chapterID, &library.ChapterMeta{
		Title:  "Chapter 1",
		Number: 1.0,
		Content: &library.ContentSource{
			ProviderID: providerID,
			ChapterRef: chapterID,
		},
	}); err != nil {
		t.Fatalf("seed chapter: %v", err)
	}

	payload, _ := json.Marshal(queue.PullPagePayload{
		MangaID: mangaID, ProviderID: providerID, ChapterID: chapterID,
		PageIndex: index, PageURL: pageURLFull,
	})
	job := &queue.Job{
		ID:               uuid.New().String(),
		Type:             queue.JobTypePullPage,
		Payload:          string(payload),
		ConcurrencyGroup: "pull:" + providerID,
	}

	if err := driver.Register(queue.JobTypePullPage, queue.JobHandlerFunc(dh.HandlePullPage)); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := driver.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := driver.Enqueue(context.Background(), job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	if !waitForStatus(t, driver, job.ID, queue.StatusCompleted, 3*time.Second) {
		t.Fatalf("job did not complete")
	}

	target := filepath.Join(dir, "3.jpg")
	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("expected image at %s: %v", target, err)
	}
	if info.Size() == 0 {
		t.Errorf("expected non-empty image at %s", target)
	}
}

func TestHandlePullManga_SkipsExistingChapters(t *testing.T) {
	provID := "fakeprov"
	chapters := []sdk.Chapter{
		{ID: "ch-1", Name: "Chapter 1", Number: 1.0, SourceOrder: 1},
		{ID: "ch-2", Name: "Chapter 2", Number: 2.0, SourceOrder: 2},
	}
	prov := &fakeContentProvider{id: provID, chapters: chapters}

	dh, driver, _, cleanup := newTestHandlers(t, prov)
	defer cleanup()

	mangaID := "manga-C"
	providerID := provID
	providerMangaID := "remote-C"

	if err := dh.Lib().SaveManga(mangaID, &library.MangaMeta{
		Title: "Manga C",
		Content: &library.ContentSource{
			ProviderID:      providerID,
			ProviderMangaID: providerMangaID,
		},
	}); err != nil {
		t.Fatalf("seed manga: %v", err)
	}

	if err := dh.Lib().SaveChapter(mangaID, providerID, "ch-1", &library.ChapterMeta{
		Title:  "Chapter 1",
		Number: 1.0,
	}); err != nil {
		t.Fatalf("seed ch-1: %v", err)
	}

	payload, _ := json.Marshal(queue.PullMangaPayload{
		MangaID:         mangaID,
		ProviderID:      providerID,
		ProviderMangaID: providerMangaID,
	})
	job := &queue.Job{
		ID:               uuid.New().String(),
		Type:             queue.JobTypePullManga,
		Payload:          string(payload),
		ConcurrencyGroup: "pull:" + providerID,
	}

	if err := driver.Register(queue.JobTypePullManga, queue.JobHandlerFunc(dh.HandlePullManga)); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := driver.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := driver.Enqueue(context.Background(), job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	if !waitForStatus(t, driver, job.ID, queue.StatusCompleted, 3*time.Second) {
		t.Fatalf("job did not complete")
	}

	got, err := dh.Lib().ListChapters(mangaID, providerID)
	if err != nil {
		t.Fatalf("ListChapters: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 chapters on disk, got %d", len(got))
	}

	ch1Meta, err := dh.Lib().GetChapter(mangaID, providerID, "ch-1")
	if err != nil {
		t.Fatalf("GetChapter ch-1: %v", err)
	}
	if ch1Meta.Content != nil && !ch1Meta.Content.LastSyncedAt.IsZero() {
		t.Errorf("expected ch-1.LastSyncedAt unchanged, got %v", ch1Meta.Content.LastSyncedAt)
	}

	ch2Meta, err := dh.Lib().GetChapter(mangaID, providerID, "ch-2")
	if err != nil {
		t.Fatalf("GetChapter ch-2: %v", err)
	}
	if ch2Meta.Title != "Chapter 2" {
		t.Errorf("expected ch-2.Title 'Chapter 2', got %q", ch2Meta.Title)
	}

	pending, err := driver.ListJobs(context.Background(), queue.JobFilter{Type: queue.JobTypePullChapter})
	if err != nil {
		t.Fatalf("ListJobs pull_chapter: %v", err)
	}
	count := 0
	for _, j := range pending {
		var p queue.PullChapterPayload
		if err := json.Unmarshal([]byte(j.Payload), &p); err == nil {
			if p.MangaID == mangaID {
				count++
				if j.Metadata["manga_id"] != mangaID {
					t.Errorf("expected pull_chapter metadata manga_id %q, got %q", mangaID, j.Metadata["manga_id"])
				}
				if j.Metadata["provider_id"] != providerID {
					t.Errorf("expected pull_chapter metadata provider_id %q, got %q", providerID, j.Metadata["provider_id"])
				}
				if j.Metadata["chapter_id"] != "ch-2" {
					t.Errorf("expected pull_chapter metadata chapter_id 'ch-2', got %q", j.Metadata["chapter_id"])
				}
			}
		}
	}
	if count != 1 {
		t.Errorf("expected 1 pull_chapter job enqueued, got %d", count)
	}
}

// waitForStatus polls the in-memory driver until the job reaches the target
// status or the timeout elapses. Returns true if observed.
func waitForStatus(t *testing.T, d *inmemory.Driver, jobID string, target queue.JobStatus, timeout time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		j, err := d.GetJob(context.Background(), jobID)
		if err == nil && j.Status == target {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

func TestHandlePullPage_SetsDownloadedAt(t *testing.T) {
	provID := "fakeprov"
	prov := &fakeContentProvider{id: provID}

	dh, driver, pageURL, cleanup := newTestHandlers(t, prov)
	defer cleanup()

	mangaID := "manga-downloaded"
	providerID := provID
	chapterID := "ch-1"
	pageURLFull := pageURL + "/pages/0.jpg"

	if err := dh.Lib().SaveChapter(mangaID, providerID, chapterID, &library.ChapterMeta{
		Title:  "Chapter 1",
		Number: 1.0,
		Content: &library.ContentSource{
			ProviderID: providerID,
			ChapterRef: chapterID,
		},
	}); err != nil {
		t.Fatalf("seed chapter: %v", err)
	}

	payload, _ := json.Marshal(queue.PullPagePayload{
		MangaID:    mangaID,
		ProviderID: providerID,
		ChapterID:  chapterID,
		PageIndex:  0,
		PageURL:    pageURLFull,
	})
	job := &queue.Job{
		ID:               uuid.New().String(),
		Type:             queue.JobTypePullPage,
		Payload:          string(payload),
		ConcurrencyGroup: "pull:" + providerID,
	}

	before := time.Now().Add(-time.Second)
	if err := driver.Register(queue.JobTypePullPage, queue.JobHandlerFunc(dh.HandlePullPage)); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := driver.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := driver.Enqueue(context.Background(), job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if !waitForStatus(t, driver, job.ID, queue.StatusCompleted, 3*time.Second) {
		t.Fatalf("job did not complete")
	}

	chMeta, err := dh.Lib().GetChapter(mangaID, providerID, chapterID)
	if err != nil {
		t.Fatalf("GetChapter: %v", err)
	}
	if chMeta.DownloadedAt.IsZero() {
		t.Fatalf("expected DownloadedAt to be set, got zero time")
	}
	if chMeta.DownloadedAt.Before(before) {
		t.Errorf("DownloadedAt %v is before enqueue time %v", chMeta.DownloadedAt, before)
	}
	if chMeta.DownloadedAt.After(time.Now().Add(time.Second)) {
		t.Errorf("DownloadedAt %v is in the future", chMeta.DownloadedAt)
	}
}

func TestHandlePullManga_UpdatesLastSyncedAt(t *testing.T) {
	provID := "fakeprov"
	chapters := []sdk.Chapter{
		{ID: "ch-1", Name: "Chapter 1", Number: 1.0, SourceOrder: 1},
	}
	prov := &fakeContentProvider{id: provID, chapters: chapters}

	dh, driver, _, cleanup := newTestHandlers(t, prov)
	defer cleanup()

	mangaID := "manga-synced"
	providerID := provID
	providerMangaID := "remote-synced"

	// Seed manga meta with an existing LastSyncedAt so we can verify the
	// pull_manga handler overwrites it with a fresh timestamp.
	pastSync := time.Now().Add(-24 * time.Hour)
	if err := dh.Lib().SaveManga(mangaID, &library.MangaMeta{
		Title: "Manga Synced",
		Content: &library.ContentSource{
			ProviderID:      providerID,
			ProviderMangaID: providerMangaID,
			LastSyncedAt:    pastSync,
		},
	}); err != nil {
		t.Fatalf("seed manga: %v", err)
	}

	before := time.Now().Add(-time.Second)
	payload, _ := json.Marshal(queue.PullMangaPayload{
		MangaID:         mangaID,
		ProviderID:      providerID,
		ProviderMangaID: providerMangaID,
	})
	job := &queue.Job{
		ID:               uuid.New().String(),
		Type:             queue.JobTypePullManga,
		Payload:          string(payload),
		ConcurrencyGroup: "pull:" + providerID,
	}

	if err := driver.Register(queue.JobTypePullManga, queue.JobHandlerFunc(dh.HandlePullManga)); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := driver.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := driver.Enqueue(context.Background(), job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if !waitForStatus(t, driver, job.ID, queue.StatusCompleted, 3*time.Second) {
		t.Fatalf("job did not complete")
	}

	mangaMeta, err := dh.Lib().GetManga(mangaID)
	if err != nil {
		t.Fatalf("GetManga: %v", err)
	}
	if mangaMeta.Content == nil {
		t.Fatalf("expected manga Content to remain set")
	}
	if mangaMeta.Content.LastSyncedAt.Equal(pastSync) {
		t.Errorf("expected LastSyncedAt to be updated away from %v, got unchanged", pastSync)
	}
	if mangaMeta.Content.LastSyncedAt.Before(before) {
		t.Errorf("LastSyncedAt %v is before pull start %v", mangaMeta.Content.LastSyncedAt, before)
	}
}

// recordingWorker implements queue.Worker for tests that need to observe which
// group limits were applied without spinning up a real driver. Only
// SetGroupLimit is exercised; the other methods panic if called.
type recordingWorker struct {
	limits map[string]int
}

func newRecordingWorker() *recordingWorker {
	return &recordingWorker{limits: make(map[string]int)}
}

func (r *recordingWorker) Register(string, queue.JobHandler) error { panic("unused") }
func (r *recordingWorker) Start(context.Context) error           { panic("unused") }
func (r *recordingWorker) Stop(context.Context) error            { panic("unused") }
func (r *recordingWorker) SetGroupLimit(group string, limit int) { r.limits[group] = limit }

// rateLimitedProvider wraps fakeContentProvider to override RateLimit().
type rateLimitedProvider struct {
	*fakeContentProvider
	rateLimit sdk.RateLimitHint
}

func (p *rateLimitedProvider) RateLimit() sdk.RateLimitHint { return p.rateLimit }

// TestApplyPullGroupLimits_AppliesSensibleDefaults verifies that
// ApplyPullGroupLimits writes the expected cover limit and per-provider limits
// derived from RateLimit().
func TestApplyPullGroupLimits_AppliesSensibleDefaults(t *testing.T) {
	fast := &rateLimitedProvider{
		fakeContentProvider: &fakeContentProvider{id: "fastprov"},
		rateLimit:           sdk.RateLimitHint{RequestsPerSecond: 5},
	}
	slow := &rateLimitedProvider{
		fakeContentProvider: &fakeContentProvider{id: "slowprov"},
		rateLimit:           sdk.RateLimitHint{RequestsPerSecond: 1},
	}
	defaultProv := &fakeContentProvider{id: "defaultprov"} // zero hint

	w := newRecordingWorker()
	queue.ApplyPullGroupLimits(w, []sdk.Content{fast, slow, defaultProv})

	if got, want := w.limits[queue.CoverConcurrencyGroup], queue.DefaultCoverGroupLimit; got != want {
		t.Errorf("pull:cover limit = %d, want %d", got, want)
	}
	// 5 rps → ~10 concurrent (5*2 burst), bounded by max 16.
	if got, want := w.limits[queue.PullGroup("fastprov")], 10; got != want {
		t.Errorf("pull:fastprov limit = %d, want %d", got, want)
	}
	// 1 rps → ~2 concurrent, bounded by min 1.
	if got, want := w.limits[queue.PullGroup("slowprov")], 2; got != want {
		t.Errorf("pull:slowprov limit = %d, want %d", got, want)
	}
	// Zero hint → fallback default.
	if got, want := w.limits[queue.PullGroup("defaultprov")], queue.DefaultContentGroupLimit; got != want {
		t.Errorf("pull:defaultprov limit = %d, want %d", got, want)
	}
}

func TestHandlePullCover_DownloadsCoverFile(t *testing.T) {
	provID := "fakeprov"
	prov := &fakeContentProvider{id: provID}

	dh, driver, pageURL, cleanup := newTestHandlers(t, prov)
	defer cleanup()

	mangaID := "manga-cover"
	coverURL := pageURL + "/cover.jpg"

	if err := dh.Lib().SaveManga(mangaID, &library.MangaMeta{
		Title: "Cover Manga",
	}); err != nil {
		t.Fatalf("seed manga: %v", err)
	}

	claimed, err := dh.Lib().ClaimCoverEnqueue(mangaID)
	if err != nil || !claimed {
		t.Fatalf("claim cover enqueue failed: %v", err)
	}

	payload, _ := json.Marshal(queue.PullCoverPayload{
		MangaID:  mangaID,
		CoverURL: coverURL,
	})
	job := &queue.Job{
		ID:               uuid.New().String(),
		Type:             queue.JobTypePullCover,
		Payload:          string(payload),
		ConcurrencyGroup: queue.CoverConcurrencyGroup,
	}

	if err := driver.Register(queue.JobTypePullCover, queue.JobHandlerFunc(dh.HandlePullCover)); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := driver.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := driver.Enqueue(context.Background(), job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	if !waitForStatus(t, driver, job.ID, queue.StatusCompleted, 3*time.Second) {
		t.Fatalf("job did not complete")
	}

	if !dh.Lib().HasCover(mangaID) {
		t.Errorf("expected cover to exist for manga %s", mangaID)
	}

	coverPath := dh.Lib().CoverPath(mangaID)
	if coverPath == "" {
		t.Fatalf("expected non-empty CoverPath")
	}
	info, err := os.Stat(coverPath)
	if err != nil {
		t.Fatalf("stat coverPath %s: %v", coverPath, err)
	}
	if info.Size() == 0 {
		t.Errorf("expected non-empty cover file at %s", coverPath)
	}
}

type mockConfigContentProvider struct {
	fakeContentProvider
	baseURL string
}

func (m *mockConfigContentProvider) GetConfig() sdk.ProviderConfig {
	return sdk.ProviderConfig{
		ID:      m.id,
		Name:    m.id,
		BaseURL: m.baseURL,
	}
}

func TestHandlePullPage_WithClientProfileAndHeaders(t *testing.T) {
	var (
		receivedUA      string
		receivedReferer string
		receivedClientUA string
		receivedCookie  string
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUA = r.Header.Get("User-Agent")
		receivedReferer = r.Header.Get("Referer")
		receivedClientUA = r.Header.Get("Sec-Ch-Ua")
		receivedCookie = r.Header.Get("Cookie")
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte("IMG-BYTES"))
	}))
	defer srv.Close()

	libRoot, err := os.MkdirTemp("", "kiyomi-pull-headers-*")
	if err != nil {
		t.Fatalf("mkdir temp: %v", err)
	}
	defer func() { _ = os.RemoveAll(libRoot) }()

	lib := library.NewLibrary(libRoot)
	lib.SetAllowPrivateNetworks(true)

	fpStore := fingerprint.NewMemoryStore()
	provID := "prov-header"
	_ = fpStore.Set(provID, fingerprint.Profile{
		UserAgent: "Mozilla/5.0 (Custom Pull Profile)",
		ClientHints: &fingerprint.ClientHints{
			UA: `"CustomBrowser";v="1"`,
		},
		Cookies: map[string]string{
			srv.URL: "auth=secret123",
		},
	})

	prov := &mockConfigContentProvider{
		fakeContentProvider: fakeContentProvider{id: provID},
		baseURL:             "https://example-prov.org",
	}

	reg := provider.NewRegistry()
	reg.Register(prov)

	driver := inmemory.NewDriver(16)
	httpClient := &http.Client{Timeout: 5 * time.Second}

	dh := queue.NewPullHandlers(lib, reg, httpClient, fpStore, driver, driver)
	dh.SetAllowPrivateNetworks(true)

	mangaID := "manga-hdr"
	chapterID := "ch-1"
	dir := dh.Lib().ProviderChapterDir(mangaID, provID, chapterID)
	_ = os.MkdirAll(dir, 0o755)
	_ = dh.Lib().SaveChapter(mangaID, provID, chapterID, &library.ChapterMeta{
		Title: "Chapter 1",
	})

	payload, _ := json.Marshal(queue.PullPagePayload{
		MangaID:    mangaID,
		ProviderID: provID,
		ChapterID:  chapterID,
		PageIndex:  1,
		PageURL:    srv.URL + "/page1.jpg",
	})
	job := &queue.Job{
		ID:               uuid.New().String(),
		Type:             queue.JobTypePullPage,
		Payload:          string(payload),
		ConcurrencyGroup: "pull:" + provID,
	}

	if err := driver.Register(queue.JobTypePullPage, queue.JobHandlerFunc(dh.HandlePullPage)); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := driver.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := driver.Enqueue(context.Background(), job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	if !waitForStatus(t, driver, job.ID, queue.StatusCompleted, 3*time.Second) {
		t.Fatalf("job did not complete")
	}

	if receivedUA != "Mozilla/5.0 (Custom Pull Profile)" {
		t.Errorf("expected custom User-Agent, got %q", receivedUA)
	}
	if receivedReferer != "https://example-prov.org/" {
		t.Errorf("expected Referer https://example-prov.org/, got %q", receivedReferer)
	}
	if receivedClientUA != `"CustomBrowser";v="1"` {
		t.Errorf("expected Sec-Ch-Ua ClientHint, got %q", receivedClientUA)
	}
	if !strings.Contains(receivedCookie, "auth=secret123") {
		t.Errorf("expected Cookie to contain auth=secret123, got %q", receivedCookie)
	}
}

func TestHandlePullCover_WithProviderHeaders(t *testing.T) {
	var receivedReferer string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedReferer = r.Header.Get("Referer")
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("COVER-BYTES"))
	}))
	defer srv.Close()

	libRoot, err := os.MkdirTemp("", "kiyomi-pull-cover-hdr-*")
	if err != nil {
		t.Fatalf("mkdir temp: %v", err)
	}
	defer func() { _ = os.RemoveAll(libRoot) }()

	lib := library.NewLibrary(libRoot)
	lib.SetAllowPrivateNetworks(true)

	provID := "cover-prov"
	prov := &mockConfigContentProvider{
		fakeContentProvider: fakeContentProvider{id: provID},
		baseURL:             "https://cover-source.net/",
	}

	reg := provider.NewRegistry()
	reg.Register(prov)

	driver := inmemory.NewDriver(16)
	httpClient := &http.Client{Timeout: 5 * time.Second}

	dh := queue.NewPullHandlers(lib, reg, httpClient, nil, driver, driver)
	dh.SetAllowPrivateNetworks(true)

	mangaID := "manga-cover-bound"
	_ = dh.Lib().SaveManga(mangaID, &library.MangaMeta{
		Title: "Cover Bound",
		Content: &library.ContentSource{
			ProviderID: provID,
		},
	})
	_, _ = dh.Lib().ClaimCoverEnqueue(mangaID)

	payload, _ := json.Marshal(queue.PullCoverPayload{
		MangaID:  mangaID,
		CoverURL: srv.URL + "/cover.png",
	})
	job := &queue.Job{
		ID:               uuid.New().String(),
		Type:             queue.JobTypePullCover,
		Payload:          string(payload),
		ConcurrencyGroup: queue.CoverConcurrencyGroup,
	}

	if err := driver.Register(queue.JobTypePullCover, queue.JobHandlerFunc(dh.HandlePullCover)); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := driver.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := driver.Enqueue(context.Background(), job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	if !waitForStatus(t, driver, job.ID, queue.StatusCompleted, 3*time.Second) {
		t.Fatalf("job did not complete")
	}

	if receivedReferer != "https://cover-source.net/" {
		t.Errorf("expected Referer https://cover-source.net/, got %q", receivedReferer)
	}
}

func TestHandlePullManga_EnqueuesCoverWithProviderID(t *testing.T) {
	provID := "fakeprov"
	chapters := []sdk.Chapter{
		{ID: "ch-1", Name: "Chapter 1", Number: 1.0, SourceOrder: 1},
	}
	prov := &fakeContentProvider{id: provID, chapters: chapters}

	dh, driver, _, cleanup := newTestHandlers(t, prov)
	defer cleanup()

	mangaID := "manga-cover-enqueue"
	providerID := provID
	providerMangaID := "remote-cover-enqueue"
	coverURL := "http://example.com/cover.jpg"

	if err := dh.Lib().SaveManga(mangaID, &library.MangaMeta{
		Title: "Manga Cover Enqueue",
	}); err != nil {
		t.Fatalf("seed manga: %v", err)
	}

	payload, _ := json.Marshal(queue.PullMangaPayload{
		MangaID:         mangaID,
		ProviderID:      providerID,
		ProviderMangaID: providerMangaID,
		CoverURL:        coverURL,
	})
	job := &queue.Job{
		ID:               uuid.New().String(),
		Type:             queue.JobTypePullManga,
		Payload:          string(payload),
		ConcurrencyGroup: "pull:" + providerID,
	}

	if err := driver.Register(queue.JobTypePullManga, queue.JobHandlerFunc(dh.HandlePullManga)); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := driver.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := driver.Enqueue(context.Background(), job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	if !waitForStatus(t, driver, job.ID, queue.StatusCompleted, 3*time.Second) {
		t.Fatalf("job did not complete")
	}

	jobs, err := driver.ListJobs(context.Background(), queue.JobFilter{Type: queue.JobTypePullCover})
	if err != nil {
		t.Fatalf("list jobs: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 pull_cover job, got %d", len(jobs))
	}

	var coverPayload queue.PullCoverPayload
	if err := json.Unmarshal([]byte(jobs[0].Payload), &coverPayload); err != nil {
		t.Fatalf("unmarshal cover payload: %v", err)
	}
	if coverPayload.ProviderID != providerID {
		t.Errorf("expected cover payload ProviderID %q, got %q", providerID, coverPayload.ProviderID)
	}
	if coverPayload.CoverURL != coverURL {
		t.Errorf("expected cover payload CoverURL %q, got %q", coverURL, coverPayload.CoverURL)
	}
	if jobs[0].Metadata["manga_id"] != mangaID {
		t.Errorf("expected cover metadata manga_id %q, got %q", mangaID, jobs[0].Metadata["manga_id"])
	}
	if jobs[0].Metadata["provider_id"] != providerID {
		t.Errorf("expected cover metadata provider_id %q, got %q", providerID, jobs[0].Metadata["provider_id"])
	}
}

func TestHandlePullCover_FallbackToProvidersList(t *testing.T) {
	var receivedReferer string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedReferer = r.Header.Get("Referer")
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("COVER-BYTES"))
	}))
	defer srv.Close()

	libRoot, err := os.MkdirTemp("", "kiyomi-pull-cover-provlist-*")
	if err != nil {
		t.Fatalf("mkdir temp: %v", err)
	}
	defer func() { _ = os.RemoveAll(libRoot) }()

	lib := library.NewLibrary(libRoot)
	lib.SetAllowPrivateNetworks(true)

	provID := "providers-list-prov"
	prov := &mockConfigContentProvider{
		fakeContentProvider: fakeContentProvider{id: provID},
		baseURL:             "https://providers-list.org/",
	}

	reg := provider.NewRegistry()
	reg.Register(prov)

	driver := inmemory.NewDriver(16)
	httpClient := &http.Client{Timeout: 5 * time.Second}

	dh := queue.NewPullHandlers(lib, reg, httpClient, nil, driver, driver)
	dh.SetAllowPrivateNetworks(true)

	mangaID := "manga-cover-providers-list"
	// Content is nil, but Providers list is populated
	_ = dh.Lib().SaveManga(mangaID, &library.MangaMeta{
		Title: "Cover Providers List",
		Providers: []library.ProviderRef{
			{ProviderID: provID, ProviderMangaID: "rem-1"},
		},
	})
	_, _ = dh.Lib().ClaimCoverEnqueue(mangaID)

	// Payload without ProviderID
	payload, _ := json.Marshal(queue.PullCoverPayload{
		MangaID:  mangaID,
		CoverURL: srv.URL + "/cover.png",
	})
	job := &queue.Job{
		ID:               uuid.New().String(),
		Type:             queue.JobTypePullCover,
		Payload:          string(payload),
		ConcurrencyGroup: queue.CoverConcurrencyGroup,
	}

	if err := driver.Register(queue.JobTypePullCover, queue.JobHandlerFunc(dh.HandlePullCover)); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := driver.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := driver.Enqueue(context.Background(), job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	if !waitForStatus(t, driver, job.ID, queue.StatusCompleted, 3*time.Second) {
		t.Fatalf("job did not complete")
	}

	if receivedReferer != "https://providers-list.org/" {
		t.Errorf("expected Referer https://providers-list.org/, got %q", receivedReferer)
	}
}

func TestHandlePullCover_WithExplicitPayloadProviderID(t *testing.T) {
	var receivedReferer string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedReferer = r.Header.Get("Referer")
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("COVER-BYTES"))
	}))
	defer srv.Close()

	libRoot, err := os.MkdirTemp("", "kiyomi-pull-cover-explicit-*")
	if err != nil {
		t.Fatalf("mkdir temp: %v", err)
	}
	defer func() { _ = os.RemoveAll(libRoot) }()

	lib := library.NewLibrary(libRoot)
	lib.SetAllowPrivateNetworks(true)

	provID := "explicit-cover-prov"
	prov := &mockConfigContentProvider{
		fakeContentProvider: fakeContentProvider{id: provID},
		baseURL:             "https://explicit-cover.org/",
	}

	reg := provider.NewRegistry()
	reg.Register(prov)

	driver := inmemory.NewDriver(16)
	httpClient := &http.Client{Timeout: 5 * time.Second}

	dh := queue.NewPullHandlers(lib, reg, httpClient, nil, driver, driver)
	dh.SetAllowPrivateNetworks(true)

	mangaID := "manga-cover-explicit"
	// MangaMeta has no Content or Providers
	_ = dh.Lib().SaveManga(mangaID, &library.MangaMeta{
		Title: "Cover Explicit",
	})
	_, _ = dh.Lib().ClaimCoverEnqueue(mangaID)

	// Payload with explicit ProviderID
	payload, _ := json.Marshal(queue.PullCoverPayload{
		MangaID:    mangaID,
		CoverURL:   srv.URL + "/cover.png",
		ProviderID: provID,
	})
	job := &queue.Job{
		ID:               uuid.New().String(),
		Type:             queue.JobTypePullCover,
		Payload:          string(payload),
		ConcurrencyGroup: queue.CoverConcurrencyGroup,
	}

	if err := driver.Register(queue.JobTypePullCover, queue.JobHandlerFunc(dh.HandlePullCover)); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := driver.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := driver.Enqueue(context.Background(), job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	if !waitForStatus(t, driver, job.ID, queue.StatusCompleted, 3*time.Second) {
		t.Fatalf("job did not complete")
	}

	if receivedReferer != "https://explicit-cover.org/" {
		t.Errorf("expected Referer https://explicit-cover.org/, got %q", receivedReferer)
	}
}

