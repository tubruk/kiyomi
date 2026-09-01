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

func TestAtsumaruPlugin_DescribeAndInit(t *testing.T) {
	plug := NewAtsumaruPlugin()
	ctx := context.Background()

	desc, err := plug.Describe(ctx)
	require.NoError(t, err)
	assert.Equal(t, PluginID, desc.PluginID)
	assert.Equal(t, PluginName, desc.PluginName)
	assert.Equal(t, Version, desc.PluginVersion)
	assert.Equal(t, sdk.Version, desc.SDKVersion)
	require.Len(t, desc.Providers, 1)
	assert.Equal(t, "atsumaru", desc.Providers[0].ID)
	assert.Equal(t, []string{"metadata", "content"}, desc.Providers[0].Capabilities)
	assert.Equal(t, int32(2), desc.Providers[0].DefaultRateLimit.RequestsPerSecond)

	err = plug.Init(ctx, sdk.PluginConfig{
		GlobalConfig: map[string]string{
			"adult_mode": "true",
		},
		HTTPConfig: sdk.GlobalHttpConfig{
			UserAgent:      "CustomAtsuUA/1.0",
			TimeoutSeconds: 15,
		},
	})
	require.NoError(t, err)
	assert.True(t, plug.adultMode)
	assert.Equal(t, "CustomAtsuUA/1.0", plug.userAgent)
}

func TestAtsumaruPlugin_SearchAndDetails(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/collections/manga/documents/search":
			_, _ = w.Write([]byte(`{
				"page": 1,
				"found": 1,
				"request_params": {"per_page": 20},
				"hits": [
					{
						"document": {
							"id": "2VgNt",
							"title": "Witch Hat Atelier",
							"largeImage": "posters/witch-large.avif"
						}
					}
				]
			}`))
		case "/api/home2/popular":
			_, _ = w.Write([]byte(`{
				"items": [
					{
						"id": "2VgNt",
						"title": "Witch Hat Atelier",
						"largeImage": "posters/witch-large.avif"
					}
				]
			}`))
		case "/api/home2/recentlyUpdated":
			_, _ = w.Write([]byte(`{
				"items": [
					{
						"id": "2VgNt",
						"title": "Witch Hat Atelier",
						"largeImage": "posters/witch-large.avif"
					}
				]
			}`))
		case "/api/manga/page":
			_, _ = w.Write([]byte(`{
				"mangaPage": {
					"id": "2VgNt",
					"title": "Witch Hat Atelier",
					"englishTitle": "Witch Hat Atelier EN",
					"otherNames": ["Atelier of Witch Hats", "Witch Hat Atelier"],
					"synopsis": "A story about magic.",
					"status": "ongoing",
					"type": "Manga",
					"released": 1451606400000,
					"avgRating": 8.5,
					"totalChapterCount": 98,
					"largeImage": "posters/witch-large.avif",
					"authors": [
						{"id": "a1", "name": "Kamome Shirahama", "type": "Author"},
						{"id": "a2", "name": "Kamome Artist", "type": "Artist"}
					],
					"genres": [{"id": "g1", "name": "Fantasy"}],
					"tags": [{"id": "t1", "name": "Magic"}]
				}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	plug := NewAtsumaruPlugin()
	plug.SetBaseURL(ts.URL)
	ctx := context.Background()

	// 1. Search by Keyword
	results, err := plug.Search(ctx, "Witch Hat", sdk.SearchOptions{Limit: 10})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "2VgNt", results[0].RemoteID)
	assert.Equal(t, "Witch Hat Atelier", results[0].Title)
	assert.Equal(t, ts.URL+"/static/posters/witch-large.avif", results[0].CoverURL)

	// 2. Browse Popular
	popResults, err := plug.Search(ctx, "", sdk.SearchOptions{Mode: "popular"})
	require.NoError(t, err)
	require.Len(t, popResults, 1)
	assert.Equal(t, "2VgNt", popResults[0].RemoteID)

	// 3. Browse Latest
	latResults, err := plug.Search(ctx, "", sdk.SearchOptions{Mode: "latest"})
	require.NoError(t, err)
	require.Len(t, latResults, 1)
	assert.Equal(t, "2VgNt", latResults[0].RemoteID)

	// 4. Details
	details, err := plug.Details(ctx, "2VgNt")
	require.NoError(t, err)
	assert.Equal(t, "2VgNt", details.RemoteID)
	assert.Equal(t, "Witch Hat Atelier", details.Title)
	assert.Equal(t, []string{"Witch Hat Atelier EN", "Atelier of Witch Hats"}, details.Aliases)
	assert.Equal(t, "A story about magic.", details.Synopsis)
	assert.Equal(t, "ongoing", details.Status)
	assert.Equal(t, []string{"Kamome Shirahama"}, details.Authors)
	assert.Equal(t, []string{"Kamome Artist"}, details.Artists)
	assert.Equal(t, []string{"Manga", "Fantasy", "Magic"}, details.Tags)
	assert.Equal(t, sdk.ReadingModeRTL, details.ReadingMode)
	assert.Equal(t, "JP", details.Country)
	assert.Equal(t, 2016, details.ReleaseYear)
	assert.Equal(t, "2016-01-01", details.StartDate)
	assert.Equal(t, float32(8.5), details.Score)
	assert.Equal(t, 98, details.TotalChapters)

	// 5. Cover & Aliases
	cover, err := plug.Cover(ctx, "2VgNt", sdk.ImageSizeLarge)
	require.NoError(t, err)
	assert.Equal(t, ts.URL+"/static/posters/witch-large.avif", cover.URL)

	aliases, err := plug.Aliases(ctx, "2VgNt")
	require.NoError(t, err)
	assert.Equal(t, []string{"Witch Hat Atelier EN", "Atelier of Witch Hats"}, aliases)
}

func TestAtsumaruPlugin_FetchChaptersAndPages(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/manga/page":
			_, _ = w.Write([]byte(`{
				"mangaPage": {
					"id": "2VgNt",
					"title": "Witch Hat Atelier",
					"scanlators": [
						{"id": "sc1", "name": "Thunder"}
					]
				}
			}`))
		case "/api/manga/allChapters":
			_, _ = w.Write([]byte(`{
				"chapters": [
					{
						"id": "ch-2",
						"title": "Chapter 2",
						"number": 2,
						"scanlationMangaId": "sc1",
						"createdAt": 1787322536369
					},
					{
						"id": "ch-1",
						"title": "Chapter 1",
						"number": 1,
						"scanlationMangaId": "sc1",
						"createdAt": 1784516183090
					}
				]
			}`))
		case "/api/read/chapter":
			_, _ = w.Write([]byte(`{
				"readChapter": {
					"id": "ch-1",
					"title": "Chapter 1",
					"pages": [
						{"id": "p-0", "image": "/static/pages/p0.avif", "number": 0},
						{"id": "p-1", "image": "pages/p1.avif", "number": 1}
					]
				}
			}`))
		case "/static/pages/p0.avif":
			w.Header().Set("Content-Type", "image/avif")
			_, _ = w.Write([]byte("avif-image-bytes"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	plug := NewAtsumaruPlugin()
	plug.SetBaseURL(ts.URL)
	ctx := context.Background()

	// 1. FetchChapters (should sort by number asc)
	chapters, err := plug.FetchChapters(ctx, "2VgNt")
	require.NoError(t, err)
	require.Len(t, chapters, 2)
	assert.Equal(t, "ch-1", chapters[0].ID)
	assert.Equal(t, float32(1), chapters[0].Number)
	assert.Equal(t, time.UnixMilli(1784516183090).UTC(), chapters[0].UploadDate)
	assert.Equal(t, "ch-2", chapters[1].ID)
	assert.Equal(t, float32(2), chapters[1].Number)

	// 2. FetchPages
	pages, err := plug.FetchPages(ctx, "2VgNt", "ch-1")
	require.NoError(t, err)
	require.Len(t, pages, 2)
	assert.Equal(t, 1, pages[0].Index)
	assert.Equal(t, ts.URL+"/static/pages/p0.avif", pages[0].URL)
	assert.Equal(t, 2, pages[1].Index)
	assert.Equal(t, ts.URL+"/static/pages/p1.avif", pages[1].URL)

	// 3. FetchPageStream
	rc, err := plug.FetchPageStream(ctx, pages[0])
	require.NoError(t, err)
	data, err := io.ReadAll(rc)
	rc.Close()
	require.NoError(t, err)
	assert.Equal(t, "avif-image-bytes", string(data))

	// 4. Invariants
	assert.True(t, plug.HasStableChapterID())
	assert.Equal(t, 2.0, plug.RateLimit().RequestsPerSecond)
}

func TestAtsumaruPlugin_GRPCIntegration(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/collections/manga/documents/search" {
			_, _ = w.Write([]byte(`{
				"page": 1,
				"found": 1,
				"request_params": {"per_page": 20},
				"hits": [
					{
						"document": {
							"id": "grpc-atsu-1",
							"title": "Atsumaru gRPC Test"
						}
					}
				]
			}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	plug := NewAtsumaruPlugin()
	plug.SetBaseURL(ts.URL)

	lis := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()

	v1.RegisterPluginServiceServer(server, &sdk.GRPCPluginServer{Impl: plug})
	v1.RegisterMetadataProviderServiceServer(server, &sdk.GRPCMetadataProviderServer{
		Providers: map[string]sdk.MetadataProvider{"atsumaru": plug},
	})
	v1.RegisterContentProviderServiceServer(server, &sdk.GRPCContentProviderServer{
		Providers: map[string]sdk.ContentProvider{"atsumaru": plug},
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

	pluginClient := sdk.NewPluginClient(v1.NewPluginServiceClient(conn))
	desc, err := pluginClient.Describe(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "atsumaru", desc.PluginID)

	metaClient := sdk.NewMetadataProviderClient(v1.NewMetadataProviderServiceClient(conn), "atsumaru")
	results, err := metaClient.Search(context.Background(), "test", sdk.SearchOptions{})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "grpc-atsu-1", results[0].RemoteID)
	assert.Equal(t, "Atsumaru gRPC Test", results[0].Title)
}
