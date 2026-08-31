package main

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	sdk "github.com/tubruk/kiyomi/plugin-sdk"
	v1 "github.com/tubruk/kiyomi/plugin-sdk/proto/v1"
)

func TestMyAnimeListPlugin_DescribeAndInit(t *testing.T) {
	plug := NewMyAnimeListPlugin()
	ctx := context.Background()

	desc, err := plug.Describe(ctx)
	require.NoError(t, err)
	assert.Equal(t, PluginID, desc.PluginID)
	assert.Equal(t, PluginName, desc.PluginName)
	assert.Equal(t, Version, desc.PluginVersion)
	assert.Equal(t, sdk.Version, desc.SDKVersion)
	require.Len(t, desc.Providers, 1)
	assert.Equal(t, "myanimelist", desc.Providers[0].ID)
	assert.Equal(t, []string{"metadata"}, desc.Providers[0].Capabilities)

	err = plug.Init(ctx, sdk.PluginConfig{
		ProviderConfigs: map[string]map[string]string{
			"myanimelist": {
				"client_id": "test-client-id-123",
			},
		},
		HTTPConfig: sdk.GlobalHttpConfig{
			UserAgent:      "CustomUA/1.0",
			TimeoutSeconds: 15,
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "test-client-id-123", plug.clientID)
	assert.Equal(t, "CustomUA/1.0", plug.userAgent)
}

func TestMyAnimeListPlugin_SearchAndDetails(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "test-client-id", r.Header.Get("X-MAL-CLIENT-ID"))
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/manga":
			_, _ = w.Write([]byte(`{
				"data": [
					{
						"node": {
							"id": 11,
							"title": "Naruto",
							"main_picture": {
								"medium": "https://cdn.myanimelist.net/images/manga/3/117764.jpg",
								"large": "https://cdn.myanimelist.net/images/manga/3/117764l.jpg"
							},
							"alternative_titles": {
								"synonyms": ["NARUTO"],
								"en": "Naruto",
								"ja": "NARUTO -ナルト-"
							},
							"mean": 8.06,
							"status": "finished"
						}
					}
				]
			}`))
		case "/manga/ranking":
			_, _ = w.Write([]byte(`{
				"data": [
					{
						"node": {
							"id": 2,
							"title": "Berserk",
							"main_picture": {
								"medium": "https://cdn.myanimelist.net/images/manga/1/157897.jpg",
								"large": "https://cdn.myanimelist.net/images/manga/1/157897l.jpg"
							},
							"alternative_titles": {
								"synonyms": ["Berserk: The Prototype"],
								"en": "Berserk",
								"ja": "ベルセルク"
							},
							"mean": 9.47,
							"status": "currently_publishing"
						}
					}
				]
			}`))
		case "/manga/11":
			_, _ = w.Write([]byte(`{
				"id": 11,
				"title": "Naruto",
				"main_picture": {
					"medium": "https://cdn.myanimelist.net/images/manga/3/117764.jpg",
					"large": "https://cdn.myanimelist.net/images/manga/3/117764l.jpg"
				},
				"alternative_titles": {
					"synonyms": ["NARUTO"],
					"en": "Naruto",
					"ja": "NARUTO -ナルト-"
				},
				"synopsis": "Moments prior to Naruto Uzumaki's birth...",
				"mean": 8.06,
				"status": "finished",
				"media_type": "manga",
				"num_chapters": 700,
				"genres": [
					{"id": 1, "name": "Action"},
					{"id": 2, "name": "Adventure"}
				],
				"authors": [
					{
						"node": {
							"id": 1879,
							"first_name": "Masashi",
							"last_name": "Kishimoto"
						},
						"role": "Story & Art"
					}
				]
			}`))
		case "/manga/99":
			_, _ = w.Write([]byte(`{
				"id": 99,
				"title": "Solo Leveling",
				"main_picture": {
					"medium": "https://cdn.myanimelist.net/images/manga/3/2222.jpg",
					"large": "https://cdn.myanimelist.net/images/manga/3/2222l.jpg"
				},
				"alternative_titles": {
					"synonyms": ["Na Honjaman Level Up"],
					"en": "Solo Leveling",
					"ja": "俺だけレベルアップな件"
				},
				"synopsis": "10 years ago...",
				"mean": 8.67,
				"status": "finished",
				"media_type": "manhwa",
				"num_chapters": 201,
				"genres": [
					{"id": 1, "name": "Action"}
				],
				"authors": [
					{
						"node": {
							"id": 100,
							"first_name": "Chugong",
							"last_name": ""
						},
						"role": "Story"
					},
					{
						"node": {
							"id": 101,
							"first_name": "DUBU",
							"last_name": "(REDICE STUDIO)"
						},
						"role": "Art"
					}
				]
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	plug := NewMyAnimeListPlugin()
	plug.SetBaseURL(ts.URL)
	plug.SetClientID("test-client-id")

	// Search with query
	results, err := plug.Search(context.Background(), "Naruto", sdk.SearchOptions{Limit: 5})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "11", results[0].RemoteID)
	assert.Equal(t, "Naruto", results[0].Title)
	assert.Equal(t, "https://cdn.myanimelist.net/images/manga/3/117764l.jpg", results[0].CoverURL)
	assert.Equal(t, []string{"NARUTO -ナルト-", "NARUTO"}, results[0].Aliases)
	assert.Equal(t, "https://myanimelist.net/manga/11", results[0].URL)

	// Search popular / ranking
	popular, err := plug.Search(context.Background(), "", sdk.SearchOptions{Limit: 5, Mode: "popular"})
	require.NoError(t, err)
	require.Len(t, popular, 1)
	assert.Equal(t, "2", popular[0].RemoteID)
	assert.Equal(t, "Berserk", popular[0].Title)

	// Search latest ranking
	latest, err := plug.Search(context.Background(), "", sdk.SearchOptions{Limit: 5, Mode: "latest"})
	require.NoError(t, err)
	require.Len(t, latest, 1)
	assert.Equal(t, "2", latest[0].RemoteID)

	// Details - Manga
	details, err := plug.Details(context.Background(), "11")
	require.NoError(t, err)
	assert.Equal(t, "11", details.RemoteID)
	assert.Equal(t, "Naruto", details.Title)
	assert.Equal(t, "Moments prior to Naruto Uzumaki's birth...", details.Synopsis)
	assert.Equal(t, "Masashi Kishimoto", details.Author)
	assert.Equal(t, "Masashi Kishimoto", details.Artist)
	assert.Equal(t, []string{"Action", "Adventure"}, details.Genres)
	assert.Equal(t, 700, details.TotalChapters)
	assert.Equal(t, sdk.ReadingModeRTL, details.ReadingMode)
	assert.Equal(t, float32(8.06), details.Score)

	// Details - Manhwa (Longstrip)
	detailsManhwa, err := plug.Details(context.Background(), "99")
	require.NoError(t, err)
	assert.Equal(t, "99", detailsManhwa.RemoteID)
	assert.Equal(t, "Solo Leveling", detailsManhwa.Title)
	assert.Equal(t, sdk.ReadingModeLongstrip, detailsManhwa.ReadingMode)
	assert.Equal(t, "Chugong", detailsManhwa.Author)
	assert.Equal(t, "DUBU (REDICE STUDIO)", detailsManhwa.Artist)

	// Cover & Aliases
	cover, err := plug.Cover(context.Background(), "11", sdk.ImageSizeLarge)
	require.NoError(t, err)
	assert.Equal(t, "https://cdn.myanimelist.net/images/manga/3/117764l.jpg", cover.URL)

	aliases, err := plug.Aliases(context.Background(), "11")
	require.NoError(t, err)
	assert.Equal(t, []string{"NARUTO -ナルト-", "NARUTO"}, aliases)
}

func TestMyAnimeListPlugin_MissingClientID(t *testing.T) {
	plug := NewMyAnimeListPlugin()
	plug.SetClientID("")

	_, err := plug.Search(context.Background(), "Naruto", sdk.SearchOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "client_id is not configured")

	_, err = plug.Details(context.Background(), "11")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "client_id is not configured")
}

func TestMyAnimeListPlugin_GRPCIntegration(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/manga" {
			_, _ = w.Write([]byte(`{
				"data": [
					{
						"node": {
							"id": 100,
							"title": "gRPC MAL Manga",
							"main_picture": {
								"large": "https://cdn.myanimelist.net/images/manga/1/100l.jpg"
							},
							"alternative_titles": {
								"synonyms": ["gRPC Synonym"]
							}
						}
					}
				]
			}`))
			return
		} else if r.URL.Path == "/manga/100" {
			_, _ = w.Write([]byte(`{
				"id": 100,
				"title": "gRPC MAL Manga",
				"main_picture": {
					"large": "https://cdn.myanimelist.net/images/manga/1/100l.jpg"
				},
				"alternative_titles": {
					"synonyms": ["gRPC Synonym"]
				},
				"synopsis": "gRPC Synopsis",
				"status": "finished",
				"media_type": "manga"
			}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	plug := NewMyAnimeListPlugin()
	plug.SetBaseURL(ts.URL)
	plug.SetClientID("grpc-test-client-id")

	// Setup gRPC test server using bufconn
	lis := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()

	v1.RegisterPluginServiceServer(server, &sdk.GRPCPluginServer{Impl: plug})
	v1.RegisterMetadataProviderServiceServer(server, &sdk.GRPCMetadataProviderServer{
		Providers: map[string]sdk.MetadataProvider{"myanimelist": plug},
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

	// Verify plugin service Describe
	pluginClient := sdk.NewPluginClient(v1.NewPluginServiceClient(conn))
	desc, err := pluginClient.Describe(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "myanimelist", desc.PluginID)

	// Verify metadata provider client over gRPC
	metaClient := sdk.NewMetadataProviderClient(v1.NewMetadataProviderServiceClient(conn), "myanimelist")
	results, err := metaClient.Search(context.Background(), "test", sdk.SearchOptions{})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "100", results[0].RemoteID)
	assert.Equal(t, "gRPC MAL Manga", results[0].Title)

	details, err := metaClient.Details(context.Background(), "100")
	require.NoError(t, err)
	assert.Equal(t, "100", details.RemoteID)
	assert.Equal(t, "gRPC MAL Manga", details.Title)
	assert.Equal(t, "gRPC Synopsis", details.Synopsis)

	cover, err := metaClient.Cover(context.Background(), "100", sdk.ImageSizeLarge)
	require.NoError(t, err)
	assert.Equal(t, "https://cdn.myanimelist.net/images/manga/1/100l.jpg", cover.URL)

	aliases, err := metaClient.Aliases(context.Background(), "100")
	require.NoError(t, err)
	assert.Equal(t, []string{"gRPC Synonym"}, aliases)
}
