package provider_test

import (
	"context"
	"io"
	"sync"
	"testing"

	"github.com/tubruk/kiyomi/pkg/provider"
	"github.com/tubruk/kiyomi/pkg/provider/sdk"
)

// mockBaseProvider implements sdk.Provider.
type mockBaseProvider struct {
	id   string
	name string
	caps []string
}

func (m *mockBaseProvider) ID() string                    { return m.id }
func (m *mockBaseProvider) Name() string                  { return m.name }
func (m *mockBaseProvider) Icon() string                  { return "icon.png" }
func (m *mockBaseProvider) Capabilities() []string        { return m.caps }
func (m *mockBaseProvider) ConfigKeys() []sdk.ConfigKeySpec { return nil }
func (m *mockBaseProvider) RequiresAuth() bool           { return false }
func (m *mockBaseProvider) State() sdk.ProviderState      { return sdk.StateActive }

// mockMetadataProvider implements sdk.Metadata.
type mockMetadataProvider struct {
	mockBaseProvider
}

func (m *mockMetadataProvider) Search(ctx context.Context, query string, opts sdk.SearchOptions) ([]sdk.SearchResult, error) {
	return nil, nil
}
func (m *mockMetadataProvider) Details(ctx context.Context, remoteID string) (sdk.MangaMetadata, error) {
	return sdk.MangaMetadata{}, nil
}
func (m *mockMetadataProvider) Cover(ctx context.Context, remoteID string, size sdk.ImageSize) (sdk.ImageRef, error) {
	return sdk.ImageRef{}, nil
}
func (m *mockMetadataProvider) Aliases(ctx context.Context, remoteID string) ([]string, error) {
	return nil, nil
}

// mockContentProvider implements sdk.Content.
type mockContentProvider struct {
	mockBaseProvider
}

func (c *mockContentProvider) HasStableChapterID() bool { return true }
func (c *mockContentProvider) FetchChapters(ctx context.Context, mangaRef string) ([]sdk.Chapter, error) {
	return nil, nil
}
func (c *mockContentProvider) FetchPages(ctx context.Context, mangaRef, chapterRef string) ([]sdk.Page, error) {
	return nil, nil
}
func (c *mockContentProvider) FetchPageStream(ctx context.Context, page sdk.Page) (io.ReadCloser, error) {
	return nil, nil
}
func (c *mockContentProvider) RateLimit() sdk.RateLimitHint { return sdk.RateLimitHint{} }

// mockTrackingProvider implements sdk.Tracking.
type mockTrackingProvider struct {
	mockBaseProvider
}

func (t *mockTrackingProvider) Authenticate(ctx context.Context, creds sdk.UserCredentials) (sdk.Session, error) {
	return sdk.Session{}, nil
}
func (t *mockTrackingProvider) PushProgress(ctx context.Context, remoteID string, n int) error {
	return nil
}
func (t *mockTrackingProvider) FetchProgress(ctx context.Context, remoteID string) (sdk.Progress, error) {
	return sdk.Progress{}, nil
}
func (t *mockTrackingProvider) IsAuthenticated() bool { return true }

// mockFullProvider implements sdk.Metadata, sdk.Content, and sdk.Tracking.
type mockFullProvider struct {
	mockBaseProvider
}

func (f *mockFullProvider) Search(ctx context.Context, query string, opts sdk.SearchOptions) ([]sdk.SearchResult, error) {
	return nil, nil
}
func (f *mockFullProvider) Details(ctx context.Context, remoteID string) (sdk.MangaMetadata, error) {
	return sdk.MangaMetadata{}, nil
}
func (f *mockFullProvider) Cover(ctx context.Context, remoteID string, size sdk.ImageSize) (sdk.ImageRef, error) {
	return sdk.ImageRef{}, nil
}
func (f *mockFullProvider) Aliases(ctx context.Context, remoteID string) ([]string, error) {
	return nil, nil
}
func (f *mockFullProvider) HasStableChapterID() bool { return true }
func (f *mockFullProvider) FetchChapters(ctx context.Context, mangaRef string) ([]sdk.Chapter, error) {
	return nil, nil
}
func (f *mockFullProvider) FetchPages(ctx context.Context, mangaRef, chapterRef string) ([]sdk.Page, error) {
	return nil, nil
}
func (f *mockFullProvider) FetchPageStream(ctx context.Context, page sdk.Page) (io.ReadCloser, error) {
	return nil, nil
}
func (f *mockFullProvider) RateLimit() sdk.RateLimitHint { return sdk.RateLimitHint{} }
func (f *mockFullProvider) Authenticate(ctx context.Context, creds sdk.UserCredentials) (sdk.Session, error) {
	return sdk.Session{}, nil
}
func (f *mockFullProvider) PushProgress(ctx context.Context, remoteID string, n int) error {
	return nil
}
func (f *mockFullProvider) FetchProgress(ctx context.Context, remoteID string) (sdk.Progress, error) {
	return sdk.Progress{}, nil
}
func (f *mockFullProvider) IsAuthenticated() bool { return true }

func TestRegistry_RegisterAndGet(t *testing.T) {
	reg := provider.NewRegistry()

	metaProv := &mockMetadataProvider{mockBaseProvider: mockBaseProvider{id: "meta1", name: "Meta One", caps: []string{"metadata"}}}
	contentProv := &mockContentProvider{mockBaseProvider: mockBaseProvider{id: "content1", name: "Content One", caps: []string{"content"}}}
	trackProv := &mockTrackingProvider{mockBaseProvider: mockBaseProvider{id: "track1", name: "Track One", caps: []string{"tracking"}}}
	fullProv := &mockFullProvider{mockBaseProvider: mockBaseProvider{id: "full1", name: "Full One", caps: []string{"metadata", "content", "tracking"}}}

	reg.Register(metaProv)
	reg.Register(contentProv)
	reg.Register(trackProv)
	reg.Register(fullProv)

	// Non-provider object registration test
	reg.Register("invalid_provider")

	// Get tests
	if p, ok := reg.Get("meta1"); !ok || p.ID() != "meta1" {
		t.Errorf("expected to find provider meta1")
	}
	if _, ok := reg.Get("nonexistent"); ok {
		t.Errorf("expected nonexistent to return false")
	}

	// GetMetadata tests
	if m, ok := reg.GetMetadata("meta1"); !ok || m.ID() != "meta1" {
		t.Errorf("expected to find metadata provider meta1")
	}
	if m, ok := reg.GetMetadata("full1"); !ok || m.ID() != "full1" {
		t.Errorf("expected to find metadata provider full1")
	}
	if _, ok := reg.GetMetadata("content1"); ok {
		t.Errorf("content1 should not be indexed as metadata")
	}

	// GetContent tests
	if c, ok := reg.GetContent("content1"); !ok || c.ID() != "content1" {
		t.Errorf("expected to find content provider content1")
	}
	if c, ok := reg.GetContent("full1"); !ok || c.ID() != "full1" {
		t.Errorf("expected to find content provider full1")
	}
	if _, ok := reg.GetContent("meta1"); ok {
		t.Errorf("meta1 should not be indexed as content")
	}

	// GetTracking tests
	if tr, ok := reg.GetTracking("track1"); !ok || tr.ID() != "track1" {
		t.Errorf("expected to find tracking provider track1")
	}
	if tr, ok := reg.GetTracking("full1"); !ok || tr.ID() != "full1" {
		t.Errorf("expected to find tracking provider full1")
	}
	if _, ok := reg.GetTracking("content1"); ok {
		t.Errorf("content1 should not be indexed as tracking")
	}
}

func TestRegistry_Listing(t *testing.T) {
	reg := provider.NewRegistry()

	metaProv := &mockMetadataProvider{mockBaseProvider: mockBaseProvider{id: "meta1", name: "Meta One", caps: []string{"metadata"}}}
	contentProv := &mockContentProvider{mockBaseProvider: mockBaseProvider{id: "content1", name: "Content One", caps: []string{"content"}}}
	fullProv := &mockFullProvider{mockBaseProvider: mockBaseProvider{id: "full1", name: "Full One", caps: []string{"metadata", "content", "tracking"}}}

	reg.Register(metaProv)
	reg.Register(contentProv)
	reg.Register(fullProv)

	metaList := reg.ListMetadata()
	if len(metaList) != 2 {
		t.Errorf("expected 2 metadata providers, got %d", len(metaList))
	}

	contentList := reg.ListContent()
	if len(contentList) != 2 {
		t.Errorf("expected 2 content providers, got %d", len(contentList))
	}

	infoList := reg.ListInfo()
	if len(infoList) != 3 {
		t.Errorf("expected 3 provider info entries, got %d", len(infoList))
	}

	infoMap := make(map[string]sdk.ProviderInfo)
	for _, info := range infoList {
		infoMap[info.ID] = info
	}

	if info, ok := infoMap["full1"]; !ok || info.Name != "Full One" || len(info.Capabilities) != 3 {
		t.Errorf("unexpected info for full1: %+v", info)
	}
}

func TestRegistry_ThreadSafety(t *testing.T) {
	reg := provider.NewRegistry()
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(3)

		// Concurrent Register
		go func(idx int) {
			defer wg.Done()
			p := &mockMetadataProvider{mockBaseProvider: mockBaseProvider{id: "meta_concurrent", name: "Meta Concurrent"}}
			reg.Register(p)
		}(i)

		// Concurrent Read
		go func() {
			defer wg.Done()
			reg.Get("meta_concurrent")
			reg.GetMetadata("meta_concurrent")
			reg.ListMetadata()
		}()

		// Concurrent ListInfo
		go func() {
			defer wg.Done()
			reg.ListInfo()
		}()
	}

	wg.Wait()
}
