//go:build e2e

// Package mock provides an in-process mock provider for e2e testing.
// It is only compiled when the e2e build tag is set and is completely
// absent from production binaries.
package mock

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tubruk/kiyomi/pkg/provider/sdk"
)

const (
	ProviderID   = "mock"
	ProviderName = "Mock Provider"
)

var _ sdk.Provider = (*Provider)(nil)
var _ sdk.Metadata = (*Provider)(nil)
var _ sdk.Content = (*Provider)(nil)
var _ sdk.Tracking = (*Provider)(nil)

// catalogEntry represents a search result from the catalog fixture.
type catalogEntry struct {
	RemoteID     string                  `json:"remoteId"`
	Title        string                  `json:"title"`
	Aliases      []string                `json:"aliases,omitempty"`
	CoverURL     string                  `json:"coverUrl,omitempty"`
	URL          string                  `json:"url,omitempty"`
	Availability sdk.ContentAvailability `json:"availability,omitempty"`
}

// mangaFixture represents manga metadata from a per-manga fixture file.
type mangaFixture struct {
	RemoteID      string                  `json:"remoteId"`
	Title         string                  `json:"title"`
	Aliases       []string                `json:"aliases,omitempty"`
	CoverURL      string                  `json:"coverUrl,omitempty"`
	Synopsis      string                  `json:"synopsis,omitempty"`
	Status        string                  `json:"status,omitempty"`
	Author        string                  `json:"author,omitempty"`
	Artist        string                  `json:"artist,omitempty"`
	Genres        []string                `json:"genres,omitempty"`
	TotalChapters int                     `json:"totalChapters"`
	Score         float32                 `json:"score,omitempty"`
	URL           string                  `json:"url,omitempty"`
	Availability  sdk.ContentAvailability `json:"availability,omitempty"`
	Chapters      []chapterFixture        `json:"chapters,omitempty"`
}

// chapterFixture represents a chapter entry in a manga fixture.
type chapterFixture struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	URL         string  `json:"url"`
	Number      float32 `json:"number"`
	UploadDate  string  `json:"uploadDate"` // RFC3339 timestamp string, parsed at fetch time
	SourceOrder int     `json:"sourceOrder"`
}

// Provider implements all sdk interfaces for e2e testing.
type Provider struct {
	fixturesDir string
	catalog     []catalogEntry
	mangaCache  map[string]*mangaFixture
}

// New creates a new mock provider, loading fixtures from fixturesDir.
func New(fixturesDir string) *Provider {
	p := &Provider{
		fixturesDir: fixturesDir,
		mangaCache:  make(map[string]*mangaFixture),
	}
	p.loadCatalog()
	return p
}

// loadCatalog loads the catalog.json fixture.
func (p *Provider) loadCatalog() {
	if p.fixturesDir == "" {
		return
	}
	path := filepath.Join(p.fixturesDir, "catalog.json")
	data, err := os.ReadFile(path)
	if err != nil {
		path = filepath.Join(p.fixturesDir, "providers", "catalog.json")
		data, err = os.ReadFile(path)
		if err != nil {
			return
		}
	}
	var catalog []catalogEntry
	if err := json.Unmarshal(data, &catalog); err != nil {
		return
	}
	p.catalog = catalog
}

// loadManga loads a per-manga fixture file.
func (p *Provider) loadManga(remoteID string) *mangaFixture {
	if p.fixturesDir == "" {
		return nil
	}
	path := filepath.Join(p.fixturesDir, fmt.Sprintf("manga-%s.json", remoteID))
	data, err := os.ReadFile(path)
	if err != nil {
		path = filepath.Join(p.fixturesDir, "providers", fmt.Sprintf("manga-%s.json", remoteID))
		data, err = os.ReadFile(path)
		if err != nil {
			return nil
		}
	}
	var m mangaFixture
	if err := json.Unmarshal(data, &m); err != nil {
		return nil
	}
	return &m
}

// ---- sdk.Provider ----

func (p *Provider) ID() string   { return ProviderID }
func (p *Provider) Name() string { return ProviderName }

func (p *Provider) Icon() string {
	return `<svg viewBox="0 0 24 24" fill="currentColor"><rect width="24" height="24"/></svg>`
}

func (p *Provider) Capabilities() []string { return []string{"metadata", "content", "tracking"} }

func (p *Provider) ConfigKeys() []sdk.ConfigKeySpec { return nil }

func (p *Provider) RequiresAuth() bool { return false }

func (p *Provider) State() sdk.ProviderState { return sdk.StateActive }

// ---- sdk.Metadata ----

func (p *Provider) Search(ctx context.Context, query string, opts sdk.SearchOptions) ([]sdk.SearchResult, error) {
	var results []sdk.SearchResult
	for _, entry := range p.catalog {
		if query == "" || containsFold(entry.Title, query) {
			results = append(results, sdk.SearchResult{
				RemoteID:     entry.RemoteID,
				Title:        entry.Title,
				Aliases:      entry.Aliases,
				CoverURL:     entry.CoverURL,
				URL:          entry.URL,
				Availability: entry.Availability,
			})
		}
		if opts.Limit > 0 && len(results) >= opts.Limit {
			break
		}
	}
	if opts.Offset > 0 && opts.Offset < len(results) {
		results = results[opts.Offset:]
	}
	if opts.Limit > 0 && len(results) > opts.Limit {
		results = results[:opts.Limit]
	}
	return results, nil
}

func (p *Provider) Details(ctx context.Context, remoteID string) (sdk.MangaMetadata, error) {
	slog.Info("mock: Details called", slog.String("remoteID", remoteID))
	m := p.loadManga(remoteID)
	if m == nil {
		return sdk.MangaMetadata{}, fmt.Errorf("mock: manga not found: %s", remoteID)
	}
	return sdk.MangaMetadata{
		RemoteID:      m.RemoteID,
		Title:         m.Title,
		Aliases:       m.Aliases,
		CoverURL:      m.CoverURL,
		Synopsis:      m.Synopsis,
		Status:        m.Status,
		Author:        m.Author,
		Artist:        m.Artist,
		Genres:        m.Genres,
		TotalChapters: m.TotalChapters,
		Score:         m.Score,
		URL:           m.URL,
		Availability:  m.Availability,
	}, nil
}

func (p *Provider) Cover(ctx context.Context, remoteID string, size sdk.ImageSize) (sdk.ImageRef, error) {
	m := p.loadManga(remoteID)
	if m == nil {
		return sdk.ImageRef{}, fmt.Errorf("mock: manga not found: %s", remoteID)
	}
	return sdk.ImageRef{
		URL: m.CoverURL,
	}, nil
}

func (p *Provider) Aliases(ctx context.Context, remoteID string) ([]string, error) {
	m := p.loadManga(remoteID)
	if m == nil {
		return nil, fmt.Errorf("mock: manga not found: %s", remoteID)
	}
	return m.Aliases, nil
}

// ---- sdk.Content ----

func (p *Provider) HasStableChapterID() bool { return true }

func (p *Provider) FetchChapters(ctx context.Context, mangaRef string) ([]sdk.Chapter, error) {
	m := p.loadManga(mangaRef)
	if m == nil {
		return nil, fmt.Errorf("mock: manga not found: %s", mangaRef)
	}
	chapters := make([]sdk.Chapter, 0, len(m.Chapters))
	for _, ch := range m.Chapters {
		var uploadDate time.Time
		if t, err := time.Parse(time.RFC3339, ch.UploadDate); err == nil {
			uploadDate = t
		}
		chapters = append(chapters, sdk.Chapter{
			ID:          "mock-" + ch.ID,
			Name:        ch.Name,
			URL:         ch.URL,
			Number:      ch.Number,
			UploadDate:  uploadDate,
			SourceOrder: ch.SourceOrder,
		})
	}
	return chapters, nil
}

func (p *Provider) FetchPages(ctx context.Context, mangaRef, chapterRef string) ([]sdk.Page, error) {
	pages := make([]sdk.Page, 0, 8)
	mangaID := mangaRef
	if mangaID == "" {
		mangaID = guessRemoteID(chapterRef)
	}
	for i := 0; i < 8; i++ {
		pages = append(pages, sdk.Page{
			Index: i,
			URL:   fmt.Sprintf("/api/v1/mock/%s/%s/%d", mangaID, chapterRef, i),
		})
	}
	return pages, nil
}

func (p *Provider) FetchPageStream(ctx context.Context, page sdk.Page) (io.ReadCloser, error) {
	return &readCloser{strings.NewReader(string(transparent1x1PNG))}, nil
}

func (p *Provider) RateLimit() sdk.RateLimitHint {
	return sdk.RateLimitHint{RequestsPerSecond: 100}
}

// ---- sdk.Tracking ----

func (p *Provider) Authenticate(ctx context.Context, creds sdk.UserCredentials) (sdk.Session, error) {
	return sdk.Session{AccessToken: "mock-token"}, nil
}

func (p *Provider) PushProgress(ctx context.Context, remoteID string, n int) error {
	return nil
}

func (p *Provider) FetchProgress(ctx context.Context, remoteID string) (sdk.Progress, error) {
	return sdk.Progress{Progress: 0}, nil
}

func (p *Provider) IsAuthenticated() bool {
	return true
}

// ---- helpers ----

func containsFold(s, substr string) bool {
	return len(substr) == 0 || (len(s) >= len(substr) && findSubstr(s, substr))
}

func findSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if equalFold(s[i:i+len(substr)], substr) {
			return true
		}
	}
	return false
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}

func guessRemoteID(chapterRef string) string {
	trimmed := chapterRef
	if len(trimmed) > 5 && trimmed[:5] == "mock-" {
		trimmed = trimmed[5:]
	}
	return trimmed
}

var _ io.ReadCloser = (*readCloser)(nil)

type readCloser struct {
	io.Reader
}

func (rc *readCloser) Close() error { return nil }

var transparent1x1PNG = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
	0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4, 0x89, 0x00, 0x00, 0x00,
	0x0d, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00, 0x00, 0x00, 0x00, 0x49,
	0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
}
