package host_test

import (
	"bytes"
	"context"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"github.com/tubruk/kiyomi/internal/plugin/host"
	provsdk "github.com/tubruk/kiyomi/pkg/provider/sdk"
	sdk "github.com/tubruk/kiyomi/plugin-sdk"
	v1 "github.com/tubruk/kiyomi/plugin-sdk/proto/v1"
)

type mockBackendProvider struct {
	id string
}

func (m *mockBackendProvider) ID() string { return m.id }

func (m *mockBackendProvider) Search(ctx context.Context, query string, opts sdk.SearchOptions) ([]sdk.SearchResult, error) {
	return []sdk.SearchResult{
		{
			RemoteID:     "remote-100",
			Title:        "Mocked Search Title",
			Aliases:      []string{"Alias 1", "Alias 2"},
			CoverURL:     "https://cdn.example.com/cover.jpg",
			URL:          "https://example.com/series/100",
			Availability: sdk.AvailabilityAvailable,
		},
	}, nil
}

func (m *mockBackendProvider) Details(ctx context.Context, remoteID string) (sdk.MangaMetadata, error) {
	return sdk.MangaMetadata{
		RemoteID:      remoteID,
		Title:         "Mocked Manga Details",
		Aliases:       []string{"Alias X"},
		CoverURL:      "https://cdn.example.com/cover.jpg",
		Synopsis:      "Detailed synopsis",
		Status:        "completed",
		Author:        "Author John",
		Artist:        "Artist Jane",
		Genres:        []string{"Action", "Shounen"},
		TotalChapters: 50,
		ReadingMode:   sdk.ReadingModeRTL,
		Score:         9.2,
		URL:           "https://example.com/series/" + remoteID,
		Availability:  sdk.AvailabilityAvailable,
	}, nil
}

func (m *mockBackendProvider) Cover(ctx context.Context, remoteID string, size sdk.ImageSize) (sdk.ImageRef, error) {
	return sdk.ImageRef{
		URL:      "https://cdn.example.com/cover-lg.jpg",
		Width:    600,
		Height:   900,
		MIMEType: "image/jpeg",
	}, nil
}

func (m *mockBackendProvider) Aliases(ctx context.Context, remoteID string) ([]string, error) {
	return []string{"Alt Name 1", "Alt Name 2"}, nil
}

func (m *mockBackendProvider) HasStableChapterID() bool {
	return true
}

func (m *mockBackendProvider) FetchChapters(ctx context.Context, mangaRef string) ([]sdk.Chapter, error) {
	return []sdk.Chapter{
		{
			ID:          "ch-42",
			Name:        "Chapter 42",
			URL:         "https://example.com/ch/42",
			Number:      42.0,
			UploadDate:  time.Unix(1700000000, 0),
			SourceOrder: 42,
		},
	}, nil
}

func (m *mockBackendProvider) FetchPages(ctx context.Context, mangaRef, chapterRef string) ([]sdk.Page, error) {
	return []sdk.Page{
		{
			Index: 0,
			URL:   "https://example.com/ch42/0.jpg",
		},
		{
			Index: 1,
			URL:   "https://example.com/ch42/1.jpg",
		},
	}, nil
}

func (m *mockBackendProvider) FetchPageStream(ctx context.Context, page sdk.Page) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader([]byte("simulated-page-data-bytes"))), nil
}

func (m *mockBackendProvider) RateLimit() sdk.RateLimitHint {
	return sdk.RateLimitHint{
		RequestsPerSecond: 8.0,
		RequestsPerMinute: 480.0,
	}
}

func (m *mockBackendProvider) Authenticate(ctx context.Context, creds sdk.UserCredentials) (sdk.Session, error) {
	return sdk.Session{
		AccessToken:  "token-xyz",
		RefreshToken: "refresh-abc",
		ExpiresAt:    time.Unix(1900000000, 0),
		UserID:       "user-456",
	}, nil
}

func (m *mockBackendProvider) PushProgress(ctx context.Context, remoteID string, n int) error {
	return nil
}

func (m *mockBackendProvider) FetchProgress(ctx context.Context, remoteID string) (sdk.Progress, error) {
	return sdk.Progress{
		Status:    "reading",
		Score:     10,
		Progress:  42,
		UpdatedAt: 1700000000,
	}, nil
}

func (m *mockBackendProvider) IsAuthenticated(ctx context.Context) bool {
	return true
}

func setupBufconnClients(t *testing.T) (*grpc.ClientConn, func()) {
	lis := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()

	mockProv := &mockBackendProvider{id: "test-prov"}

	v1.RegisterMetadataProviderServiceServer(server, &sdk.GRPCMetadataProviderServer{
		Providers: map[string]sdk.MetadataProvider{"test-prov": mockProv},
	})
	v1.RegisterContentProviderServiceServer(server, &sdk.GRPCContentProviderServer{
		Providers: map[string]sdk.ContentProvider{"test-prov": mockProv},
	})
	v1.RegisterTrackerServiceServer(server, &sdk.GRPCTrackerServer{
		Providers: map[string]sdk.Tracker{"test-prov": mockProv},
	})

	go func() {
		_ = server.Serve(lis)
	}()

	dialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}

	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	cleanup := func() {
		conn.Close()
		server.Stop()
		lis.Close()
	}

	return conn, cleanup
}

func TestGRPCProviderAdapter_MetadataMethods(t *testing.T) {
	conn, cleanup := setupBufconnClients(t)
	defer cleanup()

	metaClient := v1.NewMetadataProviderServiceClient(conn)
	contentClient := v1.NewContentProviderServiceClient(conn)
	trackClient := v1.NewTrackerServiceClient(conn)

	desc := sdk.ProviderDescriptor{
		ID:           "test-prov",
		Name:         "Test Provider",
		Capabilities: []string{"metadata", "content", "tracking"},
		SettingsSchema: []sdk.SettingSpec{
			{Key: "token", Label: "Token", Type: "secret"},
		},
		DefaultRateLimit: sdk.RateLimitSpec{RequestsPerSecond: 8},
	}

	var activeCalls int64
	adapter := host.NewGRPCProviderAdapter(desc, "test-plugin", "1.0.0", metaClient, contentClient, trackClient, &activeCalls)

	assert.Equal(t, "test-prov", adapter.ID())
	assert.Equal(t, "Test Provider", adapter.Name())
	assert.Equal(t, "icon.png", adapter.Icon())
	assert.Equal(t, []string{"metadata", "content", "tracking"}, adapter.Capabilities())
	assert.Equal(t, provsdk.StateActive, adapter.State())
	assert.True(t, adapter.RequiresAuth())
	assert.Equal(t, "test-plugin", adapter.PluginID())
	assert.Equal(t, "1.0.0", adapter.Version())
	assert.False(t, adapter.IsBuiltIn())
	require.Len(t, adapter.ConfigKeys(), 1)
	assert.Equal(t, "token", adapter.ConfigKeys()[0].Key)

	ctx := context.Background()

	// Search
	results, err := adapter.Search(ctx, "mock", provsdk.SearchOptions{Limit: 5})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "remote-100", results[0].RemoteID)
	assert.Equal(t, "Mocked Search Title", results[0].Title)

	// Details
	details, err := adapter.Details(ctx, "remote-100")
	require.NoError(t, err)
	assert.Equal(t, "Mocked Manga Details", details.Title)
	assert.Equal(t, float32(9.2), details.Score)
	assert.Equal(t, provsdk.ReadingModeRTL, details.ReadingMode)

	// Cover
	cover, err := adapter.Cover(ctx, "remote-100", provsdk.ImageSizeLarge)
	require.NoError(t, err)
	assert.Equal(t, "https://cdn.example.com/cover-lg.jpg", cover.URL)
	assert.Equal(t, 600, cover.Width)

	// Aliases
	aliases, err := adapter.Aliases(ctx, "remote-100")
	require.NoError(t, err)
	assert.Equal(t, []string{"Alt Name 1", "Alt Name 2"}, aliases)
}

func TestGRPCProviderAdapter_ContentMethods(t *testing.T) {
	conn, cleanup := setupBufconnClients(t)
	defer cleanup()

	metaClient := v1.NewMetadataProviderServiceClient(conn)
	contentClient := v1.NewContentProviderServiceClient(conn)
	trackClient := v1.NewTrackerServiceClient(conn)

	desc := sdk.ProviderDescriptor{
		ID:           "test-prov",
		Name:         "Test Provider",
		Capabilities: []string{"content"},
	}

	var activeCalls int64
	adapter := host.NewGRPCProviderAdapter(desc, "test-plugin", "1.0.0", metaClient, contentClient, trackClient, &activeCalls)
	ctx := context.Background()

	assert.True(t, adapter.HasStableChapterID())

	// Chapters
	chapters, err := adapter.FetchChapters(ctx, "remote-100")
	require.NoError(t, err)
	require.Len(t, chapters, 1)
	assert.Equal(t, "ch-42", chapters[0].ID)
	assert.Equal(t, float32(42.0), chapters[0].Number)

	// Pages
	pages, err := adapter.FetchPages(ctx, "remote-100", "ch-42")
	require.NoError(t, err)
	require.Len(t, pages, 2)
	assert.Equal(t, "https://example.com/ch42/0.jpg", pages[0].URL)

	// Stream
	rc, err := adapter.FetchPageStream(ctx, pages[0])
	require.NoError(t, err)
	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, "simulated-page-data-bytes", string(data))
	err = rc.Close()
	require.NoError(t, err)

	// Rate Limit
	rateLimit := adapter.RateLimit()
	assert.Equal(t, 8.0, rateLimit.RequestsPerSecond)
}

func TestGRPCProviderAdapter_TrackingMethods(t *testing.T) {
	conn, cleanup := setupBufconnClients(t)
	defer cleanup()

	metaClient := v1.NewMetadataProviderServiceClient(conn)
	contentClient := v1.NewContentProviderServiceClient(conn)
	trackClient := v1.NewTrackerServiceClient(conn)

	desc := sdk.ProviderDescriptor{
		ID:           "test-prov",
		Name:         "Test Provider",
		Capabilities: []string{"tracking"},
	}

	var activeCalls int64
	adapter := host.NewGRPCProviderAdapter(desc, "test-plugin", "1.0.0", metaClient, contentClient, trackClient, &activeCalls)
	ctx := context.Background()

	sess, err := adapter.Authenticate(ctx, provsdk.UserCredentials{AccessToken: "abc"})
	require.NoError(t, err)
	assert.Equal(t, "token-xyz", sess.AccessToken)
	assert.Equal(t, "user-456", sess.UserID)

	err = adapter.PushProgress(ctx, "remote-100", 42)
	require.NoError(t, err)

	prog, err := adapter.FetchProgress(ctx, "remote-100")
	require.NoError(t, err)
	assert.Equal(t, "reading", prog.Status)
	assert.Equal(t, 42, prog.Progress)
	assert.Equal(t, 10, prog.Score)

	assert.True(t, adapter.IsAuthenticated())
}

func TestGRPCProviderAdapter_StreamReaderConcurrentClose(t *testing.T) {
	conn, cleanup := setupBufconnClients(t)
	defer cleanup()

	metaClient := v1.NewMetadataProviderServiceClient(conn)
	contentClient := v1.NewContentProviderServiceClient(conn)
	trackClient := v1.NewTrackerServiceClient(conn)

	desc := sdk.ProviderDescriptor{
		ID:           "test-prov",
		Name:         "Test Provider",
		Capabilities: []string{"content"},
	}

	var activeCalls int64
	adapter := host.NewGRPCProviderAdapter(desc, "test-plugin", "1.0.0", metaClient, contentClient, trackClient, &activeCalls)
	ctx := context.Background()

	rc, err := adapter.FetchPageStream(ctx, provsdk.Page{URL: "https://example.com/p1.jpg"})
	require.NoError(t, err)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = rc.Close()
		}()
	}
	wg.Wait()

	assert.Equal(t, int64(0), activeCalls)
}
