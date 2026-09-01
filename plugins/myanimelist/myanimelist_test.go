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
				"start_date": "1999-09-21",
				"end_date": "2014-11-10",
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
				],
				"serialization": [
					{
						"node": {
							"id": 1,
							"name": "Shounen Jump (Weekly)"
						},
						"role": "Serialization"
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
				"start_date": "2018-03-04",
				"end_date": "2021-12-29",
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
				],
				"serialization": [
					{
						"node": {
							"id": 2,
							"name": "KakaoPage"
						},
						"role": "Serialization"
					}
				]
			}`))
		case "/manga/200":
			_, _ = w.Write([]byte(`{
				"id": 200,
				"title": "Tales of Demons and Gods",
				"main_picture": {
					"medium": "https://cdn.myanimelist.net/images/manga/1/200m.jpg",
					"large": "https://cdn.myanimelist.net/images/manga/1/200l.jpg"
				},
				"alternative_titles": {
					"synonyms": ["Yao Shen Ji"],
					"en": "Tales of Demons and Gods",
					"ja": "妖神记"
				},
				"start_date": "2015-08-25",
				"end_date": "2023-12-31",
				"synopsis": "Nie Li experienced death...",
				"mean": 7.85,
				"status": "finished",
				"media_type": "manhua",
				"num_chapters": 450,
				"genres": [
					{"id": 1, "name": "Action"},
					{"id": 2, "name": "Fantasy"},
					{"id": 3, "name": "Martial Arts"}
				],
				"authors": [
					{
						"node": {
							"id": 201,
							"first_name": "Mad Snail",
							"last_name": ""
						},
						"role": "Story"
					},
					{
						"node": {
							"id": 202,
							"first_name": "Jiang",
							"last_name": "Ruo"
						},
						"role": "Art"
					}
				],
				"serialization": [
					{
						"node": {
							"id": 10,
							"name": "AC.QQ"
						},
						"role": "Serialization"
					},
					{
						"node": {
							"id": 11,
							"name": "KuaiKan Manhua"
						},
						"role": "Serialization"
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
	assert.Equal(t, []string{"Masashi Kishimoto"}, details.Authors)
	assert.Equal(t, []string{"Masashi Kishimoto"}, details.Artists)
	assert.Equal(t, []string{"Action", "Adventure"}, details.Tags)
	assert.Equal(t, []string{"Shounen Jump (Weekly)"}, details.Publishers)
	assert.Equal(t, 1999, details.ReleaseYear)
	assert.Equal(t, "1999-09-21", details.StartDate)
	assert.Equal(t, "2014-11-10", details.EndDate)
	assert.Equal(t, "JP", details.Country)
	assert.Equal(t, 700, details.TotalChapters)
	assert.Equal(t, sdk.ReadingModeRTL, details.ReadingMode)
	assert.Equal(t, float32(8.06), details.Score)

	// Details - Manhwa (Longstrip)
	detailsManhwa, err := plug.Details(context.Background(), "99")
	require.NoError(t, err)
	assert.Equal(t, "99", detailsManhwa.RemoteID)
	assert.Equal(t, "Solo Leveling", detailsManhwa.Title)
	assert.Equal(t, sdk.ReadingModeLongstrip, detailsManhwa.ReadingMode)
	assert.Equal(t, []string{"Chugong"}, detailsManhwa.Authors)
	assert.Equal(t, []string{"DUBU (REDICE STUDIO)"}, detailsManhwa.Artists)
	assert.Equal(t, []string{"Action"}, detailsManhwa.Tags)
	assert.Equal(t, []string{"KakaoPage"}, detailsManhwa.Publishers)
	assert.Equal(t, 2018, detailsManhwa.ReleaseYear)
	assert.Equal(t, "2018-03-04", detailsManhwa.StartDate)
	assert.Equal(t, "2021-12-29", detailsManhwa.EndDate)
	assert.Equal(t, "KR", detailsManhwa.Country)

	// Details - Manhua (Chinese, Longstrip)
	detailsManhua, err := plug.Details(context.Background(), "200")
	require.NoError(t, err)
	assert.Equal(t, "200", detailsManhua.RemoteID)
	assert.Equal(t, "Tales of Demons and Gods", detailsManhua.Title)
	assert.Equal(t, sdk.ReadingModeLongstrip, detailsManhua.ReadingMode)
	assert.Equal(t, []string{"Mad Snail"}, detailsManhua.Authors)
	assert.Equal(t, []string{"Jiang Ruo"}, detailsManhua.Artists)
	assert.Equal(t, []string{"Action", "Fantasy", "Martial Arts"}, detailsManhua.Tags)
	assert.Equal(t, []string{"AC.QQ", "KuaiKan Manhua"}, detailsManhua.Publishers)
	assert.Equal(t, 2015, detailsManhua.ReleaseYear)
	assert.Equal(t, "2015-08-25", detailsManhua.StartDate)
	assert.Equal(t, "2023-12-31", detailsManhua.EndDate)
	assert.Equal(t, "CN", detailsManhua.Country)
	assert.Equal(t, float32(7.85), detailsManhua.Score)

	// Cover & Aliases
	cover, err := plug.Cover(context.Background(), "11", sdk.ImageSizeLarge)
	require.NoError(t, err)
	assert.Equal(t, "https://cdn.myanimelist.net/images/manga/3/117764l.jpg", cover.URL)

	aliases, err := plug.Aliases(context.Background(), "11")
	require.NoError(t, err)
	assert.Equal(t, []string{"NARUTO -ナルト-", "NARUTO"}, aliases)
}

func TestMyAnimeListPlugin_Details_MetadataExpansion(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/manga/301":
			// Manga: JP country, year extraction, publishers list, discrete author/artist
			_, _ = w.Write([]byte(`{
				"id": 301,
				"title": "Fullmetal Alchemist",
				"main_picture": {
					"large": "https://cdn.myanimelist.net/images/manga/3/fma.jpg"
				},
				"start_date": "2001-07-12",
				"end_date": "2010-09-11",
				"media_type": "manga",
				"genres": [
					{"name": "Action"},
					{"name": "Adventure"},
					{"name": "Drama"}
				],
				"authors": [
					{"node": {"first_name": "Hiromu", "last_name": "Arakawa"}, "role": "Story & Art"}
				],
				"serialization": [
					{"node": {"name": "Shounen Gangan"}},
					{"node": {"name": "Square Enix"}}
				]
			}`))
		case "/manga/302":
			// Manhwa: KR country, longstrip
			_, _ = w.Write([]byte(`{
				"id": 302,
				"title": "Tower of God",
				"main_picture": {
					"large": "https://cdn.myanimelist.net/images/manga/1/tog.jpg"
				},
				"start_date": "2010-06-30",
				"media_type": "manhwa",
				"genres": [{"name": "Fantasy"}],
				"authors": [
					{"node": {"first_name": "SIU", "last_name": ""}, "role": "Story & Art"}
				],
				"serialization": [
					{"node": {"name": "Naver Webtoon"}}
				]
			}`))
		case "/manga/303":
			// Manhua: CN country, longstrip, discrete Story & Art authors
			_, _ = w.Write([]byte(`{
				"id": 303,
				"title": "Soul Land",
				"main_picture": {
					"large": "https://cdn.myanimelist.net/images/manga/2/sl.jpg"
				},
				"start_date": "2011-05-01",
				"end_date": "2018-01-01",
				"media_type": "manhua",
				"genres": [{"name": "Action"}, {"name": "Fantasy"}],
				"authors": [
					{"node": {"first_name": "Tang Jia San", "last_name": "Shao"}, "role": "Story"},
					{"node": {"first_name": "Mu", "last_name": "Feng Chun"}, "role": "Art"}
				],
				"serialization": [
					{"node": {"name": "Zhiyin Manke"}}
				]
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	plug := NewMyAnimeListPlugin()
	plug.SetBaseURL(ts.URL)
	plug.SetClientID("expansion-client-id")
	ctx := context.Background()

	// Verify Manga (JP)
	fma, err := plug.Details(ctx, "301")
	require.NoError(t, err)
	assert.Equal(t, "JP", fma.Country)
	assert.Equal(t, 2001, fma.ReleaseYear)
	assert.Equal(t, "2001-07-12", fma.StartDate)
	assert.Equal(t, "2010-09-11", fma.EndDate)
	assert.Equal(t, sdk.ReadingModeRTL, fma.ReadingMode)
	assert.Equal(t, []string{"Hiromu Arakawa"}, fma.Authors)
	assert.Equal(t, []string{"Hiromu Arakawa"}, fma.Artists)
	assert.Equal(t, []string{"Action", "Adventure", "Drama"}, fma.Tags)
	assert.Equal(t, []string{"Shounen Gangan", "Square Enix"}, fma.Publishers)

	// Verify Manhwa (KR)
	tog, err := plug.Details(ctx, "302")
	require.NoError(t, err)
	assert.Equal(t, "KR", tog.Country)
	assert.Equal(t, 2010, tog.ReleaseYear)
	assert.Equal(t, "2010-06-30", tog.StartDate)
	assert.Equal(t, "", tog.EndDate)
	assert.Equal(t, sdk.ReadingModeLongstrip, tog.ReadingMode)
	assert.Equal(t, []string{"SIU"}, tog.Authors)
	assert.Equal(t, []string{"SIU"}, tog.Artists)
	assert.Equal(t, []string{"Naver Webtoon"}, tog.Publishers)

	// Verify Manhua (CN) with discrete author and artist
	sl, err := plug.Details(ctx, "303")
	require.NoError(t, err)
	assert.Equal(t, "CN", sl.Country)
	assert.Equal(t, 2011, sl.ReleaseYear)
	assert.Equal(t, "2011-05-01", sl.StartDate)
	assert.Equal(t, "2018-01-01", sl.EndDate)
	assert.Equal(t, sdk.ReadingModeLongstrip, sl.ReadingMode)
	assert.Equal(t, []string{"Tang Jia San Shao"}, sl.Authors)
	assert.Equal(t, []string{"Mu Feng Chun"}, sl.Artists)
	assert.Equal(t, []string{"Action", "Fantasy"}, sl.Tags)
	assert.Equal(t, []string{"Zhiyin Manke"}, sl.Publishers)
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
