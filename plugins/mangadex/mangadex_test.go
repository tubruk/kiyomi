package main

import (
	"context"
	"io"
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

func TestMangaDexPlugin_DescribeAndInit(t *testing.T) {
	plug := NewMangaDexPlugin()
	ctx := context.Background()

	desc, err := plug.Describe(ctx)
	require.NoError(t, err)
	assert.Equal(t, PluginID, desc.PluginID)
	assert.Equal(t, PluginName, desc.PluginName)
	assert.Equal(t, Version, desc.PluginVersion)
	assert.Equal(t, sdk.Version, desc.SDKVersion)
	require.Len(t, desc.Providers, 1)
	assert.Equal(t, "mangadex", desc.Providers[0].ID)
	assert.Equal(t, []string{"metadata", "content"}, desc.Providers[0].Capabilities)

	err = plug.Init(ctx, sdk.PluginConfig{
		GlobalConfig: map[string]string{
			"data_saver": "true",
		},
		HTTPConfig: sdk.GlobalHttpConfig{
			UserAgent:      "CustomUA/1.0",
			TimeoutSeconds: 15,
		},
	})
	require.NoError(t, err)
	assert.True(t, plug.dataSaver)
	assert.Equal(t, "CustomUA/1.0", plug.userAgent)
}

func TestMangaDexPlugin_SearchAndDetailsAvailability(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/manga":
			_, _ = w.Write([]byte(`{
				"data": [
					{
						"id": "avail-1",
						"attributes": {
							"title": {"en": "Available Manga"},
							"availableTranslatedLanguages": ["en"],
							"latestUploadedChapter": "ch-1"
						},
						"relationships": [
							{"type": "cover_art", "attributes": {"fileName": "cover.jpg"}}
						]
					},
					{
						"id": "unavail-1",
						"attributes": {
							"title": {"ja": "Japanese Only"},
							"availableTranslatedLanguages": ["ja"],
							"latestUploadedChapter": "ch-2"
						},
						"relationships": []
					}
				]
			}`))
		case "/manga/avail-1":
			_, _ = w.Write([]byte(`{
				"data": {
					"id": "avail-1",
					"attributes": {
						"title": {"en": "Available Manga"},
						"description": {"en": "Great series"},
						"status": "ongoing",
						"originalLanguage": "ja",
						"availableTranslatedLanguages": ["en"],
						"latestUploadedChapter": "ch-1",
						"tags": []
					},
					"relationships": [
						{"type": "author", "attributes": {"name": "Oda"}},
						{"type": "cover_art", "attributes": {"fileName": "cover.jpg"}}
					]
				}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	plug := NewMangaDexPlugin()
	plug.SetBaseURL(ts.URL)

	// Search
	results, err := plug.Search(context.Background(), "One Piece", sdk.SearchOptions{Limit: 5})
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, "avail-1", results[0].RemoteID)
	assert.Equal(t, "Available Manga", results[0].Title)
	assert.Equal(t, "https://uploads.mangadex.org/covers/avail-1/cover.jpg", results[0].CoverURL)
	assert.Equal(t, sdk.AvailabilityAvailable, results[0].Availability)
	assert.Equal(t, sdk.AvailabilityUnavailable, results[1].Availability)

	// Details
	details, err := plug.Details(context.Background(), "avail-1")
	require.NoError(t, err)
	assert.Equal(t, "avail-1", details.RemoteID)
	assert.Equal(t, "Available Manga", details.Title)
	assert.Equal(t, "Great series", details.Synopsis)
	assert.Equal(t, "Oda", details.Author)
	assert.Equal(t, sdk.ReadingModeRTL, details.ReadingMode)
	assert.Equal(t, sdk.AvailabilityAvailable, details.Availability)

	// Cover & Aliases
	cover, err := plug.Cover(context.Background(), "avail-1", sdk.ImageSizeLarge)
	require.NoError(t, err)
	assert.Equal(t, "https://uploads.mangadex.org/covers/avail-1/cover.jpg", cover.URL)

	aliases, err := plug.Aliases(context.Background(), "avail-1")
	require.NoError(t, err)
	assert.Nil(t, aliases)
}

func TestMangaDexPlugin_ReadingMode(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/manga/manga-ja":
			_, _ = w.Write([]byte(`{"data":{"id":"manga-ja","attributes":{"title":{"en":"JA Manga"},"originalLanguage":"ja","tags":[]}}}`))
		case "/manga/manga-ko":
			_, _ = w.Write([]byte(`{"data":{"id":"manga-ko","attributes":{"title":{"en":"KO Manhwa"},"originalLanguage":"ko","tags":[]}}}`))
		case "/manga/manga-zh":
			_, _ = w.Write([]byte(`{"data":{"id":"manga-zh","attributes":{"title":{"en":"ZH Manhua"},"originalLanguage":"zh","tags":[]}}}`))
		case "/manga/manga-longstrip":
			_, _ = w.Write([]byte(`{"data":{"id":"manga-longstrip","attributes":{"title":{"en":"Webtoon"},"originalLanguage":"ja","tags":[{"attributes":{"name":{"en":"Long Strip"}}}]}}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	plug := NewMangaDexPlugin()
	plug.SetBaseURL(ts.URL)

	ctx := context.Background()

	dJa, err := plug.Details(ctx, "manga-ja")
	require.NoError(t, err)
	assert.Equal(t, sdk.ReadingModeRTL, dJa.ReadingMode)

	dKo, err := plug.Details(ctx, "manga-ko")
	require.NoError(t, err)
	assert.Equal(t, sdk.ReadingModeLongstrip, dKo.ReadingMode)

	dZh, err := plug.Details(ctx, "manga-zh")
	require.NoError(t, err)
	assert.Equal(t, sdk.ReadingModeLongstrip, dZh.ReadingMode)

	dLong, err := plug.Details(ctx, "manga-longstrip")
	require.NoError(t, err)
	assert.Equal(t, sdk.ReadingModeLongstrip, dLong.ReadingMode)
}

func TestMangaDexPlugin_FetchChaptersAndPages(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/manga/manga-uuid/feed":
			_, _ = w.Write([]byte(`{
				"data": [
					{
						"id": "ch-101",
						"attributes": {
							"chapter": "1",
							"title": "Chapter 1",
							"publishAt": "2023-11-14T22:13:20Z"
						}
					}
				]
			}`))
		case "/at-home/server/ch-101":
			_, _ = w.Write([]byte(`{
				"baseUrl": "https://uploads.mangadex.org",
				"chapter": {
					"hash": "hash123",
					"data": ["p1.jpg", "p2.jpg"],
					"dataSaver": ["p1-low.jpg", "p2-low.jpg"]
				}
			}`))
		case "/test-img.jpg":
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = w.Write([]byte("jpeg-stream-data"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	plug := NewMangaDexPlugin()
	plug.SetBaseURL(ts.URL)
	ctx := context.Background()

	// Chapters
	chapters, err := plug.FetchChapters(ctx, "manga-uuid")
	require.NoError(t, err)
	require.Len(t, chapters, 1)
	assert.Equal(t, "ch-101", chapters[0].ID)
	assert.Equal(t, float32(1.0), chapters[0].Number)
	assert.Equal(t, "Chapter 1", chapters[0].Name)
	expectedTime, _ := time.Parse(time.RFC3339, "2023-11-14T22:13:20Z")
	assert.Equal(t, expectedTime, chapters[0].UploadDate)

	// Pages (regular quality)
	pages, err := plug.FetchPages(ctx, "manga-uuid", "ch-101")
	require.NoError(t, err)
	require.Len(t, pages, 2)
	assert.Equal(t, "https://uploads.mangadex.org/data/hash123/p1.jpg", pages[0].URL)

	// Pages (data saver)
	plug.dataSaver = true
	pagesSaver, err := plug.FetchPages(ctx, "manga-uuid", "ch-101")
	require.NoError(t, err)
	require.Len(t, pagesSaver, 2)
	assert.Equal(t, "https://uploads.mangadex.org/data-saver/hash123/p1-low.jpg", pagesSaver[0].URL)

	// Page stream
	rc, err := plug.FetchPageStream(ctx, sdk.Page{URL: ts.URL + "/test-img.jpg"})
	require.NoError(t, err)
	data, err := io.ReadAll(rc)
	rc.Close()
	require.NoError(t, err)
	assert.Equal(t, "jpeg-stream-data", string(data))

	// Capabilities and metadata
	assert.True(t, plug.HasStableChapterID())
	assert.Equal(t, 5.0, plug.RateLimit().RequestsPerSecond)
}

func TestMangaDexPlugin_GRPCIntegration(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/manga" {
			_, _ = w.Write([]byte(`{
				"data": [
					{
						"id": "grpc-manga-1",
						"attributes": {
							"title": {"en": "gRPC Manga"}
						},
						"relationships": []
					}
				]
			}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	plug := NewMangaDexPlugin()
	plug.SetBaseURL(ts.URL)

	// Setup gRPC test server using bufconn
	lis := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()

	v1.RegisterPluginServiceServer(server, &sdk.GRPCPluginServer{Impl: plug})
	v1.RegisterMetadataProviderServiceServer(server, &sdk.GRPCMetadataProviderServer{
		Providers: map[string]sdk.MetadataProvider{"mangadex": plug},
	})
	v1.RegisterContentProviderServiceServer(server, &sdk.GRPCContentProviderServer{
		Providers: map[string]sdk.ContentProvider{"mangadex": plug},
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
	assert.Equal(t, "mangadex", desc.PluginID)

	// Verify metadata provider client over gRPC
	metaClient := sdk.NewMetadataProviderClient(v1.NewMetadataProviderServiceClient(conn), "mangadex")
	results, err := metaClient.Search(context.Background(), "test", sdk.SearchOptions{})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "grpc-manga-1", results[0].RemoteID)
	assert.Equal(t, "gRPC Manga", results[0].Title)
}
