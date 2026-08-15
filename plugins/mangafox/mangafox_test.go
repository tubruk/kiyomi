package main

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	sdk "github.com/tubruk/kiyomi/plugin-sdk"
	v1 "github.com/tubruk/kiyomi/plugin-sdk/proto/v1"
)

func TestMangaFoxPlugin_DescribeAndInit(t *testing.T) {
	plug := NewMangaFoxPlugin()
	ctx := context.Background()

	desc, err := plug.Describe(ctx)
	require.NoError(t, err)
	assert.Equal(t, PluginID, desc.PluginID)
	assert.Equal(t, PluginName, desc.PluginName)
	assert.Equal(t, Version, desc.PluginVersion)
	assert.Equal(t, sdk.Version, desc.SDKVersion)
	require.Len(t, desc.Providers, 1)
	assert.Equal(t, "mangafox", desc.Providers[0].ID)
	assert.Equal(t, []string{"metadata", "content"}, desc.Providers[0].Capabilities)
	assert.Equal(t, int32(1), desc.Providers[0].DefaultRateLimit.RequestsPerSecond)

	err = plug.Init(ctx, sdk.PluginConfig{
		HTTPConfig: sdk.GlobalHttpConfig{
			UserAgent:      "CustomMangaFoxUA/1.0",
			TimeoutSeconds: 25,
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "CustomMangaFoxUA/1.0", plug.userAgent)
}

func TestMangaFoxPlugin_SearchParsing(t *testing.T) {
	mockHTML := `
	<ul class="manga-list-4-list">
		<li>
			<p class="manga-list-4-item-title"><a href="/manga/one_piece/">One Piece</a></p>
			<img class="manga-list-4-cover" src="https://fmcdn.mfcdn.net/store/manga/106/cover.jpg" />
		</li>
	</ul>
	`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockHTML))
	}))
	defer ts.Close()

	plug := NewMangaFoxPlugin()
	plug.SetBaseURL(ts.URL)

	results, err := plug.Search(context.Background(), "One Piece", sdk.SearchOptions{})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "one_piece", results[0].RemoteID)
	assert.Equal(t, "One Piece", results[0].Title)
	assert.Equal(t, "https://fmcdn.mfcdn.net/store/manga/106/cover.jpg", results[0].CoverURL)
	assert.Equal(t, sdk.AvailabilityAvailable, results[0].Availability)
}

func TestMangaFoxPlugin_DetailsNormalization(t *testing.T) {
	mockHTML := `
	<div class="detail-info-right-title-font">One Piece</div>
	<img class="detail-info-cover-img" src="https://fmcdn.mfcdn.net/store/manga/106/cover.jpg" />
	<p class="detail-info-right-say"><a title="Author">Eiichiro Oda</a></p>
	<span class="detail-info-right-title-tip">Ongoing</span>
	<p class="detail-info-right-tag"><a>Action</a><a>Adventure</a></p>
	<div class="fullcontent">Luffy sets out on a journey...</div>
	`

	var requestedURL string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedURL = r.URL.String()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockHTML))
	}))
	defer ts.Close()

	plug := NewMangaFoxPlugin()
	plug.SetBaseURL(ts.URL)

	tests := []string{
		"one_piece",
		"/manga/one_piece/",
		"manga/one_piece",
		"https://fanfox.net/manga/one_piece/",
	}

	for _, inputID := range tests {
		meta, err := plug.Details(context.Background(), inputID)
		require.NoError(t, err)
		assert.Equal(t, "one_piece", meta.RemoteID)
		assert.Equal(t, "One Piece", meta.Title)
		assert.Equal(t, "Eiichiro Oda", meta.Author)
		assert.Equal(t, "Ongoing", meta.Status)
		assert.Equal(t, sdk.ReadingModeRTL, meta.ReadingMode)
		assert.Equal(t, "Luffy sets out on a journey...", meta.Synopsis)
		assert.Equal(t, "/manga/one_piece/", requestedURL)
	}

	// Test Cover and Aliases
	cover, err := plug.Cover(context.Background(), "one_piece", sdk.ImageSizeLarge)
	require.NoError(t, err)
	assert.Equal(t, "https://fmcdn.mfcdn.net/store/manga/106/cover.jpg", cover.URL)

	aliases, err := plug.Aliases(context.Background(), "one_piece")
	require.NoError(t, err)
	assert.Nil(t, aliases)
}

func TestMangaFoxPlugin_ReadingMode(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		switch r.URL.Path {
		case "/manga/standard_manga/":
			_, _ = w.Write([]byte(`
				<div class="detail-info-right-title-font">Standard Manga</div>
				<p class="detail-info-right-tag"><a>Action</a><a>Adventure</a></p>
			`))
		case "/manga/webtoon_manga/":
			_, _ = w.Write([]byte(`
				<div class="detail-info-right-title-font">Webtoon Manga</div>
				<p class="detail-info-right-tag"><a>Action</a><a>Webtoons</a></p>
			`))
		case "/manga/manhwa_manga/":
			_, _ = w.Write([]byte(`
				<div class="detail-info-right-title-font">Manhwa Manga</div>
				<p class="detail-info-right-tag"><a>Fantasy</a><a>Manhwa</a></p>
			`))
		case "/manga/longstrip_manga/":
			_, _ = w.Write([]byte(`
				<div class="detail-info-right-title-font">Long Strip Manga</div>
				<p class="detail-info-right-tag"><a>Drama</a><a>Long Strip</a></p>
			`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	plug := NewMangaFoxPlugin()
	plug.SetBaseURL(ts.URL)

	tests := []struct {
		slug     string
		expected sdk.ReadingMode
	}{
		{"standard_manga", sdk.ReadingModeRTL},
		{"webtoon_manga", sdk.ReadingModeLongstrip},
		{"manhwa_manga", sdk.ReadingModeLongstrip},
		{"longstrip_manga", sdk.ReadingModeLongstrip},
	}

	for _, tt := range tests {
		meta, err := plug.Details(context.Background(), tt.slug)
		require.NoError(t, err)
		assert.Equal(t, tt.expected, meta.ReadingMode)
	}
}

func TestMangaFoxPlugin_FetchChaptersCleanIDs(t *testing.T) {
	mockHTML := `
	<ul class="detail-main-list">
		<li>
			<a href="/manga/one_piece/v01/c001/1.html">
				<span class="title3">Vol.01 Ch.001 Romance Dawn</span>
				<span class="title2">Dec 1, 2020</span>
			</a>
		</li>
		<li>
			<a href="/manga/one_piece/c002/1.html">
				<span class="title3">Ch.002 They Call Him Straw Hat Luffy</span>
				<span class="title2">Dec 2, 2020</span>
			</a>
		</li>
	</ul>
	`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockHTML))
	}))
	defer ts.Close()

	plug := NewMangaFoxPlugin()
	plug.SetBaseURL(ts.URL)

	chapters, err := plug.FetchChapters(context.Background(), "one_piece")
	require.NoError(t, err)
	require.Len(t, chapters, 2)

	assert.Equal(t, "v01-c001", chapters[0].ID)
	assert.Equal(t, "Vol.01 Ch.001 Romance Dawn", chapters[0].Name)
	assert.False(t, chapters[0].UploadDate.IsZero())

	assert.Equal(t, "c002", chapters[1].ID)
	assert.Equal(t, "Ch.002 They Call Him Straw Hat Luffy", chapters[1].Name)
}

func TestMangaFoxPlugin_FetchPagesCleanIDs(t *testing.T) {
	var requestedPaths []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPaths = append(requestedPaths, r.URL.Path)
		w.WriteHeader(http.StatusOK)
		mockMobileHTML := `<html><body><img data-original="https://fmcdn.mfcdn.net/store/manga/106/001.jpg" /></body></html>`
		_, _ = w.Write([]byte(mockMobileHTML))
	}))
	defer ts.Close()

	plug := NewMangaFoxPlugin()
	plug.SetBaseURL(ts.URL)

	testRefs := []struct {
		mangaRef     string
		chapterRef   string
		expectedPath string
	}{
		{
			mangaRef:     "one_piece",
			chapterRef:   "v01-c001",
			expectedPath: "/manga/one_piece/v01/c001/1.html",
		},
		{
			mangaRef:     "one_piece",
			chapterRef:   "c002",
			expectedPath: "/manga/one_piece/c002/1.html",
		},
		{
			mangaRef:     "one_piece",
			chapterRef:   "one_piece~v01~c001~1.html",
			expectedPath: "/manga/one_piece/v01/c001/1.html",
		},
	}

	for _, tt := range testRefs {
		requestedPaths = nil
		pages, err := plug.FetchPages(context.Background(), tt.mangaRef, tt.chapterRef)
		require.NoError(t, err)
		require.Len(t, pages, 1)
		assert.Equal(t, "https://fmcdn.mfcdn.net/store/manga/106/001.jpg", pages[0].URL)
		require.NotEmpty(t, requestedPaths)
		assert.Equal(t, tt.expectedPath, requestedPaths[0])
	}
}

func TestMangaFoxPlugin_DateParsing(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Time
	}{
		{"", time.Time{}},
		{"Jan 02, 2006", time.Date(2006, 1, 2, 0, 0, 0, 0, time.UTC)},
		{"Dec 1, 2020", time.Date(2020, 12, 1, 0, 0, 0, 0, time.UTC)},
		{"2023-05-15", time.Date(2023, 5, 15, 0, 0, 0, 0, time.UTC)},
	}

	for _, tt := range tests {
		actual := parseDate(tt.input)
		assert.Equal(t, tt.expected, actual)
	}

	// Relative dates
	today := parseDate("Today")
	assert.False(t, today.IsZero())
	yesterday := parseDate("Yesterday")
	assert.False(t, yesterday.IsZero())
	hoursAgo := parseDate("3 hours ago")
	assert.False(t, hoursAgo.IsZero())
}

func TestMangaFoxPlugin_GRPCIntegration(t *testing.T) {
	mockHTML := `
	<ul class="manga-list-4-list">
		<li>
			<p class="manga-list-4-item-title"><a href="/manga/bleach/">Bleach</a></p>
			<img class="manga-list-4-cover" src="https://fmcdn.mfcdn.net/store/manga/9/cover.jpg" />
		</li>
	</ul>
	`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockHTML))
	}))
	defer ts.Close()

	plug := NewMangaFoxPlugin()
	plug.SetBaseURL(ts.URL)

	// Setup gRPC server with bufconn
	lis := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()

	v1.RegisterPluginServiceServer(server, &sdk.GRPCPluginServer{Impl: plug})
	v1.RegisterMetadataProviderServiceServer(server, &sdk.GRPCMetadataProviderServer{
		Providers: map[string]sdk.MetadataProvider{"mangafox": plug},
	})
	v1.RegisterContentProviderServiceServer(server, &sdk.GRPCContentProviderServer{
		Providers: map[string]sdk.ContentProvider{"mangafox": plug},
	})

	go func() {
		_ = server.Serve(lis)
	}()
	defer server.Stop()

	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer conn.Close()

	// Verify plugin service
	pluginClient := sdk.NewPluginClient(v1.NewPluginServiceClient(conn))
	desc, err := pluginClient.Describe(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "mangafox", desc.PluginID)

	// Verify metadata provider client
	metaClient := sdk.NewMetadataProviderClient(v1.NewMetadataProviderServiceClient(conn), "mangafox")
	results, err := metaClient.Search(context.Background(), "Bleach", sdk.SearchOptions{})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "bleach", results[0].RemoteID)
	assert.Equal(t, "Bleach", results[0].Title)

	// Verify content provider client
	contentClient := sdk.NewContentProviderClient(v1.NewContentProviderServiceClient(conn), "mangafox")
	assert.True(t, contentClient.HasStableChapterID())
	assert.Equal(t, 1.0, contentClient.RateLimit().RequestsPerSecond)
}
