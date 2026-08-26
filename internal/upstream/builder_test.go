package upstream_test

import (
	"context"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/tubruk/kiyomi/internal/upstream"
	"github.com/tubruk/kiyomi/pkg/fingerprint"
	"github.com/tubruk/kiyomi/pkg/provider"
	"github.com/tubruk/kiyomi/pkg/provider/sdk"
)

type mockProviderWithConfig struct {
	id      string
	name    string
	baseURL string
}

func (m *mockProviderWithConfig) ID() string                   { return m.id }
func (m *mockProviderWithConfig) Name() string                 { return m.name }
func (m *mockProviderWithConfig) Icon() string                 { return "" }
func (m *mockProviderWithConfig) Capabilities() []string       { return []string{"content"} }
func (m *mockProviderWithConfig) ConfigKeys() []sdk.ConfigKeySpec { return nil }
func (m *mockProviderWithConfig) RequiresAuth() bool           { return false }
func (m *mockProviderWithConfig) State() sdk.ProviderState     { return sdk.StateActive }
func (m *mockProviderWithConfig) HasStableChapterID() bool     { return true }
func (m *mockProviderWithConfig) RateLimit() sdk.RateLimitHint { return sdk.RateLimitHint{} }
func (m *mockProviderWithConfig) FetchChapters(_ context.Context, _ string) ([]sdk.Chapter, error) {
	return nil, nil
}
func (m *mockProviderWithConfig) FetchPages(_ context.Context, _, _ string) ([]sdk.Page, error) {
	return nil, nil
}
func (m *mockProviderWithConfig) FetchPageStream(_ context.Context, _ sdk.Page) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}
func (m *mockProviderWithConfig) GetConfig() sdk.ProviderConfig {
	return sdk.ProviderConfig{
		ID:      m.id,
		Name:    m.name,
		BaseURL: m.baseURL,
	}
}

type mockMetadataProviderWithConfig struct {
	id      string
	name    string
	baseURL string
}

func (m *mockMetadataProviderWithConfig) ID() string                   { return m.id }
func (m *mockMetadataProviderWithConfig) Name() string                 { return m.name }
func (m *mockMetadataProviderWithConfig) Icon() string                 { return "" }
func (m *mockMetadataProviderWithConfig) Capabilities() []string       { return []string{"metadata"} }
func (m *mockMetadataProviderWithConfig) ConfigKeys() []sdk.ConfigKeySpec { return nil }
func (m *mockMetadataProviderWithConfig) RequiresAuth() bool           { return false }
func (m *mockMetadataProviderWithConfig) State() sdk.ProviderState     { return sdk.StateActive }
func (m *mockMetadataProviderWithConfig) Search(_ context.Context, _ string, _ sdk.SearchOptions) ([]sdk.SearchResult, error) {
	return nil, nil
}
func (m *mockMetadataProviderWithConfig) Details(_ context.Context, _ string) (sdk.MangaMetadata, error) {
	return sdk.MangaMetadata{}, nil
}
func (m *mockMetadataProviderWithConfig) GetConfig() sdk.ProviderConfig {
	return sdk.ProviderConfig{
		ID:      m.id,
		Name:    m.name,
		BaseURL: m.baseURL,
	}
}

type mockProviderWithBaseURLMethod struct {
	id      string
	baseURL string
}

func (m *mockProviderWithBaseURLMethod) ID() string                   { return m.id }
func (m *mockProviderWithBaseURLMethod) Name() string                 { return m.id }
func (m *mockProviderWithBaseURLMethod) Icon() string                 { return "" }
func (m *mockProviderWithBaseURLMethod) Capabilities() []string       { return []string{"custom"} }
func (m *mockProviderWithBaseURLMethod) ConfigKeys() []sdk.ConfigKeySpec { return nil }
func (m *mockProviderWithBaseURLMethod) RequiresAuth() bool           { return false }
func (m *mockProviderWithBaseURLMethod) State() sdk.ProviderState     { return sdk.StateActive }
func (m *mockProviderWithBaseURLMethod) BaseURL() string              { return m.baseURL }

func TestRequestBuilder_DefaultHeaders(t *testing.T) {
	b := upstream.NewRequestBuilder(nil, nil, nil)
	req, err := b.BuildRequest(context.Background(), "https://example.com/image.jpg", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := req.Header.Get("User-Agent"); got != sdk.DefaultUserAgent {
		t.Errorf("expected User-Agent %q, got %q", sdk.DefaultUserAgent, got)
	}
	if got := req.Header.Get("Accept"); got != upstream.DefaultAcceptHeader {
		t.Errorf("expected Accept %q, got %q", upstream.DefaultAcceptHeader, got)
	}
	if got := req.Header.Get("Referer"); got != "" {
		t.Errorf("expected no Referer, got %q", got)
	}
}

func TestRequestBuilder_UserAgentAndClientHintsFromStore(t *testing.T) {
	fpStore := fingerprint.NewMemoryStore()
	customUA := "Mozilla/5.0 (Custom Browser 1.0)"
	_ = fpStore.Set("prov-1", fingerprint.Profile{
		UserAgent: customUA,
		ClientHints: &fingerprint.ClientHints{
			UA:       `"Custom";v="1"`,
			Platform: `"Linux"`,
			Mobile:   `?0`,
		},
	})

	b := upstream.NewRequestBuilder(fpStore, nil, nil)
	req, err := b.BuildRequest(context.Background(), "https://example.com/image.jpg", "prov-1", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := req.Header.Get("User-Agent"); got != customUA {
		t.Errorf("expected User-Agent %q, got %q", customUA, got)
	}
	if got := req.Header.Get("Sec-Ch-Ua"); got != `"Custom";v="1"` {
		t.Errorf("expected Sec-Ch-Ua %q, got %q", `"Custom";v="1"`, got)
	}
	if got := req.Header.Get("Sec-Ch-Ua-Platform"); got != `"Linux"` {
		t.Errorf("expected Sec-Ch-Ua-Platform %q, got %q", `"Linux"`, got)
	}
	if got := req.Header.Get("Sec-Ch-Ua-Mobile"); got != `?0` {
		t.Errorf("expected Sec-Ch-Ua-Mobile %q, got %q", `?0`, got)
	}
}

func TestRequestBuilder_CookieInjection(t *testing.T) {
	fpStore := fingerprint.NewMemoryStore()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar, Timeout: 5 * time.Second}

	_ = fpStore.Set("prov-cookie", fingerprint.Profile{
		Cookies: map[string]string{
			"https://example.com": "session_id=abc123xyz; theme=dark",
		},
	})

	b := upstream.NewRequestBuilder(fpStore, nil, client)
	_, err := b.BuildRequest(context.Background(), "https://example.com/page/1.jpg", "prov-cookie", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	u, _ := url.Parse("https://example.com")
	cookies := client.Jar.Cookies(u)
	if len(cookies) != 2 {
		t.Fatalf("expected 2 cookies in jar, got %d", len(cookies))
	}

	cookieMap := make(map[string]string)
	for _, c := range cookies {
		cookieMap[c.Name] = c.Value
	}

	if cookieMap["session_id"] != "abc123xyz" {
		t.Errorf("expected session_id=abc123xyz, got %q", cookieMap["session_id"])
	}
	if cookieMap["theme"] != "dark" {
		t.Errorf("expected theme=dark, got %q", cookieMap["theme"])
	}

	// Cross-domain CDN request receives cookie header directly on request
	reqCross, err := b.BuildRequest(context.Background(), "https://cdn.example.org/page/1.jpg", "prov-cookie", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := reqCross.Header.Get("Cookie"); got != "session_id=abc123xyz; theme=dark" {
		t.Errorf("expected cross-domain Cookie header %q, got %q", "session_id=abc123xyz; theme=dark", got)
	}
}

func TestRequestBuilder_MangaFoxDefaultCookiesAndReferer(t *testing.T) {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar, Timeout: 5 * time.Second}
	b := upstream.NewRequestBuilder(nil, nil, client)

	// Explicit providerID "mangafox" to cross-domain CDN url
	req1, err := b.BuildRequest(context.Background(), "https://s1.zjcdn.net/store/manga/123/cover.jpg", "mangafox", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := req1.Header.Get("Referer"); got != "https://fanfox.net/" {
		t.Errorf("expected Referer https://fanfox.net/, got %q", got)
	}
	if got := req1.Header.Get("Cookie"); got != "isAdult=1; readway=2" {
		t.Errorf("expected Cookie header %q, got %q", "isAdult=1; readway=2", got)
	}

	// Inferred providerID from zjcdn URL
	req2, err := b.BuildRequest(context.Background(), "https://s1.zjcdn.net/store/manga/123/cover.jpg", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := req2.Header.Get("Referer"); got != "https://fanfox.net/" {
		t.Errorf("expected Referer https://fanfox.net/, got %q", got)
	}
	if got := req2.Header.Get("Cookie"); got != "isAdult=1; readway=2" {
		t.Errorf("expected Cookie header %q, got %q", "isAdult=1; readway=2", got)
	}

	// Check client cookie jar was populated with default cookies
	u, _ := url.Parse("https://fanfox.net")
	jarCookies := client.Jar.Cookies(u)
	jarMap := make(map[string]string)
	for _, c := range jarCookies {
		jarMap[c.Name] = c.Value
	}
	if jarMap["isAdult"] != "1" {
		t.Errorf("expected isAdult=1 in jar, got %q", jarMap["isAdult"])
	}
	if jarMap["readway"] != "2" {
		t.Errorf("expected readway=2 in jar, got %q", jarMap["readway"])
	}
}

func TestRequestBuilder_FingerprintStoreCookiesPrecedence(t *testing.T) {
	fpStore := fingerprint.NewMemoryStore()
	_ = fpStore.Set("mangafox", fingerprint.Profile{
		Cookies: map[string]string{
			"https://fanfox.net": "session_id=custom123; user=alice",
		},
	})

	b := upstream.NewRequestBuilder(fpStore, nil, nil)
	req, err := b.BuildRequest(context.Background(), "https://s1.zjcdn.net/store/manga/123/cover.jpg", "mangafox", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := req.Header.Get("Referer"); got != "https://fanfox.net/" {
		t.Errorf("expected Referer https://fanfox.net/, got %q", got)
	}
	if got := req.Header.Get("Cookie"); got != "session_id=custom123; user=alice" {
		t.Errorf("expected explicit Cookie header %q, got %q", "session_id=custom123; user=alice", got)
	}
}

func TestRequestBuilder_RefererResolution(t *testing.T) {
	reg := provider.NewRegistry()
	mockP1 := &mockProviderWithConfig{id: "mangafox", name: "MangaFox", baseURL: "https://fanfox.net"}
	mockP2 := &mockProviderWithConfig{id: "customsite", name: "CustomSite", baseURL: "https://custom.org/manga/"}
	reg.Register(mockP1)
	reg.Register(mockP2)

	b := upstream.NewRequestBuilder(nil, reg, nil)

	tests := []struct {
		name            string
		urlStr          string
		providerID      string
		explicitReferer string
		expectedReferer string
	}{
		{
			name:            "Explicit referer overrides all",
			urlStr:          "https://fanfox.net/img.jpg",
			providerID:      "mangafox",
			explicitReferer: "https://explicit.example.com/",
			expectedReferer: "https://explicit.example.com/",
		},
		{
			name:            "Derived from provider BaseURL without trailing slash",
			urlStr:          "https://fanfox.net/img.jpg",
			providerID:      "mangafox",
			explicitReferer: "",
			expectedReferer: "https://fanfox.net/",
		},
		{
			name:            "Derived from provider BaseURL with existing trailing slash",
			urlStr:          "https://custom.org/manga/img.jpg",
			providerID:      "customsite",
			explicitReferer: "",
			expectedReferer: "https://custom.org/manga/",
		},
		{
			name:            "Fallback heuristic fanfox.net",
			urlStr:          "https://cdn.fanfox.net/cover.jpg",
			providerID:      "",
			explicitReferer: "",
			expectedReferer: "https://fanfox.net/",
		},
		{
			name:            "Fallback heuristic mfcdn.net",
			urlStr:          "https://img.mfcdn.net/1.jpg",
			providerID:      "",
			explicitReferer: "",
			expectedReferer: "https://fanfox.net/",
		},
		{
			name:            "Fallback heuristic mangafox.me",
			urlStr:          "https://mangafox.me/img.png",
			providerID:      "",
			explicitReferer: "",
			expectedReferer: "https://fanfox.net/",
		},
		{
			name:            "Fallback heuristic zjcdn",
			urlStr:          "https://zjcdn.net/page.jpg",
			providerID:      "",
			explicitReferer: "",
			expectedReferer: "https://fanfox.net/",
		},
		{
			name:            "Fallback heuristic mangadex.org",
			urlStr:          "https://uploads.mangadex.org/covers/1.jpg",
			providerID:      "",
			explicitReferer: "",
			expectedReferer: "https://mangadex.org/",
		},
		{
			name:            "Fallback heuristic mangadex.network",
			urlStr:          "https://s2.mangadex.network/data/1.jpg",
			providerID:      "",
			explicitReferer: "",
			expectedReferer: "https://mangadex.org/",
		},
		{
			name:            "Unknown domain without provider",
			urlStr:          "https://unknown-domain.com/image.jpg",
			providerID:      "",
			explicitReferer: "",
			expectedReferer: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := b.BuildRequest(context.Background(), tt.urlStr, tt.providerID, tt.explicitReferer)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := req.Header.Get("Referer"); got != tt.expectedReferer {
				t.Errorf("expected Referer %q, got %q", tt.expectedReferer, got)
			}
		})
	}
}

func TestRequestBuilder_NilBuilderAndNewRequest(t *testing.T) {
	req, err := upstream.NewRequest(context.Background(), "https://fanfox.net/img.png", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := req.Header.Get("Referer"); got != "https://fanfox.net/" {
		t.Errorf("expected Referer https://fanfox.net/, got %q", got)
	}
	if got := req.Header.Get("User-Agent"); got != sdk.DefaultUserAgent {
		t.Errorf("expected default User-Agent, got %q", got)
	}

	var nilBuilder *upstream.RequestBuilder
	if nilBuilder.FingerprintStore() != nil {
		t.Error("expected nil FingerprintStore on nil builder")
	}
	if nilBuilder.Registry() != nil {
		t.Error("expected nil Registry on nil builder")
	}
	if nilBuilder.HTTPClient() != nil {
		t.Error("expected nil HTTPClient on nil builder")
	}
}

func TestRequestBuilder_InferredProviderFingerprint(t *testing.T) {
	fpStore := fingerprint.NewMemoryStore()
	_ = fpStore.Set("mangafox", fingerprint.Profile{
		UserAgent: "Mozilla/5.0 (MangaFox Inferred UA)",
	})
	_ = fpStore.Set("mangadex", fingerprint.Profile{
		UserAgent: "Mozilla/5.0 (MangaDex Inferred UA)",
	})

	b := upstream.NewRequestBuilder(fpStore, nil, nil)

	// Inferred mangafox from fanfox URL
	req1, err := b.BuildRequest(context.Background(), "https://cdn.fanfox.net/cover.jpg", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := req1.Header.Get("User-Agent"); got != "Mozilla/5.0 (MangaFox Inferred UA)" {
		t.Errorf("expected inferred MangaFox UA, got %q", got)
	}

	// Inferred mangadex from uploads.mangadex.org URL
	req2, err := b.BuildRequest(context.Background(), "https://uploads.mangadex.org/covers/123.jpg", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := req2.Header.Get("User-Agent"); got != "Mozilla/5.0 (MangaDex Inferred UA)" {
		t.Errorf("expected inferred MangaDex UA, got %q", got)
	}
}

func TestRequestBuilder_RefererFromMetadataAndGenericProvider(t *testing.T) {
	reg := provider.NewRegistry()
	metaProv := &mockMetadataProviderWithConfig{id: "metaprov", name: "MetaProv", baseURL: "https://meta-source.org"}
	genProv := &mockProviderWithBaseURLMethod{id: "genprov", baseURL: "https://gen-source.org"}
	reg.Register(metaProv)
	reg.Register(genProv)

	b := upstream.NewRequestBuilder(nil, reg, nil)

	// Referer resolved via GetMetadata
	req1, err := b.BuildRequest(context.Background(), "https://meta-source.org/cover.png", "metaprov", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := req1.Header.Get("Referer"); got != "https://meta-source.org/" {
		t.Errorf("expected Referer https://meta-source.org/, got %q", got)
	}

	// Referer resolved via Get + BaseURL()
	req2, err := b.BuildRequest(context.Background(), "https://gen-source.org/cover.png", "genprov", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := req2.Header.Get("Referer"); got != "https://gen-source.org/" {
		t.Errorf("expected Referer https://gen-source.org/, got %q", got)
	}

	// Referer resolved via namespaced ID
	req3, err := b.BuildRequest(context.Background(), "https://meta-source.org/cover.png", "metaprov@builtin", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := req3.Header.Get("Referer"); got != "https://meta-source.org/" {
		t.Errorf("expected Referer https://meta-source.org/, got %q", got)
	}
}
