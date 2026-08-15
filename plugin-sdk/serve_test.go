package sdk

import (
	"bytes"
	"context"
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	v1 "github.com/tubruk/kiyomi/plugin-sdk/proto/v1"
)

type mockPluginImpl struct {
	desc PluginDescriptor
	init PluginConfig
}

func (m *mockPluginImpl) Describe(ctx context.Context) (PluginDescriptor, error) {
	return m.desc, nil
}

func (m *mockPluginImpl) Init(ctx context.Context, config PluginConfig) error {
	m.init = config
	return nil
}

type mockProviderImpl struct {
	id string
}

func (m *mockProviderImpl) ID() string { return m.id }

func (m *mockProviderImpl) Search(ctx context.Context, query string, opts SearchOptions) ([]SearchResult, error) {
	return []SearchResult{
		{
			RemoteID:     "manga-1",
			Title:        "Mock Manga",
			Aliases:      []string{"Alt 1"},
			CoverURL:     "https://example.com/cover.jpg",
			URL:          "https://example.com/manga/1",
			Availability: AvailabilityAvailable,
		},
	}, nil
}

func (m *mockProviderImpl) Details(ctx context.Context, remoteID string) (MangaMetadata, error) {
	return MangaMetadata{
		RemoteID:      remoteID,
		Title:         "Mock Manga Details",
		Aliases:       []string{"Mock Alias"},
		CoverURL:      "https://example.com/cover.jpg",
		Synopsis:      "A great story",
		Status:        "ongoing",
		Author:        "Author A",
		Artist:        "Artist B",
		Genres:        []string{"Action", "Fantasy"},
		TotalChapters: 10,
		ReadingMode:   ReadingModeLTR,
		Score:         8.5,
		URL:           "https://example.com/manga/" + remoteID,
		Availability:  AvailabilityAvailable,
	}, nil
}

func (m *mockProviderImpl) Cover(ctx context.Context, remoteID string, size ImageSize) (ImageRef, error) {
	return ImageRef{
		URL:      "https://example.com/cover.jpg",
		Width:    300,
		Height:   450,
		MIMEType: "image/jpeg",
		Headers:  map[string]string{"Referer": "https://example.com"},
	}, nil
}

func (m *mockProviderImpl) Aliases(ctx context.Context, remoteID string) ([]string, error) {
	return []string{"Alias A", "Alias B"}, nil
}

func (m *mockProviderImpl) HasStableChapterID() bool {
	return true
}

func (m *mockProviderImpl) FetchChapters(ctx context.Context, mangaRef string) ([]Chapter, error) {
	return []Chapter{
		{
			ID:          "ch-1",
			Name:        "Chapter 1",
			URL:         "https://example.com/ch/1",
			Number:      1.0,
			UploadDate:  time.Unix(1700000000, 0),
			SourceOrder: 1,
		},
	}, nil
}

func (m *mockProviderImpl) FetchPages(ctx context.Context, mangaRef, chapterRef string) ([]Page, error) {
	return []Page{
		{
			Index:   0,
			URL:     "https://example.com/pages/1.jpg",
			Headers: map[string]string{"Referer": "https://example.com"},
		},
	}, nil
}

func (m *mockProviderImpl) FetchPageStream(ctx context.Context, page Page) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader([]byte("fake-image-bytes"))), nil
}

func (m *mockProviderImpl) RateLimit() RateLimitHint {
	return RateLimitHint{
		RequestsPerSecond: 5.0,
		RequestsPerMinute: 300.0,
	}
}

func (m *mockProviderImpl) Authenticate(ctx context.Context, creds UserCredentials) (Session, error) {
	return Session{
		AccessToken:  "token-123",
		RefreshToken: "refresh-456",
		ExpiresAt:    time.Unix(1800000000, 0),
		UserID:       "user-789",
	}, nil
}

func (m *mockProviderImpl) PushProgress(ctx context.Context, remoteID string, n int) error {
	return nil
}

func (m *mockProviderImpl) FetchProgress(ctx context.Context, remoteID string) (Progress, error) {
	return Progress{
		Status:    "reading",
		Score:     9,
		Progress:  5,
		UpdatedAt: 1700000000,
	}, nil
}

func (m *mockProviderImpl) IsAuthenticated(ctx context.Context) bool {
	return true
}

func setupTestGRPCServer(t *testing.T) (*grpc.ClientConn, func()) {
	lis := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()

	mockPlug := &mockPluginImpl{
		desc: PluginDescriptor{
			PluginID:      "test-plugin",
			PluginName:    "Test Plugin",
			PluginVersion: "1.0.0",
			SDKVersion:    Version,
			Providers: []ProviderDescriptor{
				{
					ID:           "mock",
					Name:         "Mock Provider",
					Capabilities: []string{"metadata", "content", "tracking"},
				},
			},
		},
	}
	mockProv := &mockProviderImpl{id: "mock"}

	v1.RegisterPluginServiceServer(server, &GRPCPluginServer{Impl: mockPlug})
	v1.RegisterMetadataProviderServiceServer(server, &GRPCMetadataProviderServer{
		Providers: map[string]MetadataProvider{"mock": mockProv},
	})
	v1.RegisterContentProviderServiceServer(server, &GRPCContentProviderServer{
		Providers: map[string]ContentProvider{"mock": mockProv},
	})
	v1.RegisterTrackerServiceServer(server, &GRPCTrackerServer{
		Providers: map[string]Tracker{"mock": mockProv},
	})

	go func() {
		if err := server.Serve(lis); err != nil && err != grpc.ErrServerStopped {
			t.Logf("server error: %v", err)
		}
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

func TestGRPCPluginService(t *testing.T) {
	conn, cleanup := setupTestGRPCServer(t)
	defer cleanup()

	client := &GRPCPluginClient{client: v1.NewPluginServiceClient(conn)}
	ctx := context.Background()

	desc, err := client.Describe(ctx)
	require.NoError(t, err)
	assert.Equal(t, "test-plugin", desc.PluginID)
	assert.Equal(t, "Test Plugin", desc.PluginName)
	assert.Equal(t, "1.0.0", desc.PluginVersion)
	assert.Equal(t, Version, desc.SDKVersion)
	assert.Len(t, desc.Providers, 1)
	assert.Equal(t, "mock", desc.Providers[0].ID)

	err = client.Init(ctx, PluginConfig{
		GlobalConfig: map[string]string{"theme": "dark"},
		ProviderConfigs: map[string]map[string]string{
			"mock": {"token": "secret"},
		},
		HTTPConfig: GlobalHttpConfig{
			UserAgent:      "Custom/1.0",
			TimeoutSeconds: 30,
		},
	})
	require.NoError(t, err)
}

func TestGRPCMetadataProvider(t *testing.T) {
	conn, cleanup := setupTestGRPCServer(t)
	defer cleanup()

	client := NewMetadataProviderClient(v1.NewMetadataProviderServiceClient(conn), "mock")
	ctx := context.Background()

	results, err := client.Search(ctx, "solo", SearchOptions{Limit: 10})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "manga-1", results[0].RemoteID)
	assert.Equal(t, "Mock Manga", results[0].Title)

	details, err := client.Details(ctx, "manga-1")
	require.NoError(t, err)
	assert.Equal(t, "Mock Manga Details", details.Title)
	assert.Equal(t, float32(8.5), details.Score)
	assert.Equal(t, ReadingModeLTR, details.ReadingMode)

	cover, err := client.Cover(ctx, "manga-1", ImageSizeLarge)
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/cover.jpg", cover.URL)
	assert.Equal(t, 300, cover.Width)
	assert.Equal(t, "https://example.com", cover.Headers["Referer"])

	aliases, err := client.Aliases(ctx, "manga-1")
	require.NoError(t, err)
	assert.Equal(t, []string{"Alias A", "Alias B"}, aliases)
}

func TestGRPCContentProvider(t *testing.T) {
	conn, cleanup := setupTestGRPCServer(t)
	defer cleanup()

	client := NewContentProviderClient(v1.NewContentProviderServiceClient(conn), "mock")
	ctx := context.Background()

	assert.True(t, client.HasStableChapterID())

	chapters, err := client.FetchChapters(ctx, "manga-1")
	require.NoError(t, err)
	require.Len(t, chapters, 1)
	assert.Equal(t, "ch-1", chapters[0].ID)
	assert.Equal(t, float32(1.0), chapters[0].Number)

	pages, err := client.FetchPages(ctx, "manga-1", "ch-1")
	require.NoError(t, err)
	require.Len(t, pages, 1)
	assert.Equal(t, "https://example.com/pages/1.jpg", pages[0].URL)
	assert.Equal(t, "https://example.com", pages[0].Headers["Referer"])

	rc, err := client.FetchPageStream(ctx, pages[0])
	require.NoError(t, err)
	defer rc.Close()

	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, "fake-image-bytes", string(data))

	rateLimit := client.RateLimit()
	assert.Equal(t, 5.0, rateLimit.RequestsPerSecond)
}

func TestGRPCTracker(t *testing.T) {
	conn, cleanup := setupTestGRPCServer(t)
	defer cleanup()

	client := NewTrackerClient(v1.NewTrackerServiceClient(conn), "mock")
	ctx := context.Background()

	sess, err := client.Authenticate(ctx, UserCredentials{AccessToken: "abc"})
	require.NoError(t, err)
	assert.Equal(t, "token-123", sess.AccessToken)
	assert.Equal(t, "user-789", sess.UserID)

	err = client.PushProgress(ctx, "manga-1", 5)
	require.NoError(t, err)

	prog, err := client.FetchProgress(ctx, "manga-1")
	require.NoError(t, err)
	assert.Equal(t, "reading", prog.Status)
	assert.Equal(t, 5, prog.Progress)
	assert.Equal(t, 9, prog.Score)

	assert.True(t, client.IsAuthenticated(ctx))
}
