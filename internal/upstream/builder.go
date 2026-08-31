package upstream

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sort"
	"strings"

	"github.com/tubruk/kiyomi/pkg/fingerprint"
	"github.com/tubruk/kiyomi/pkg/provider"
	"github.com/tubruk/kiyomi/pkg/provider/sdk"
)

// DefaultAcceptHeader defines the default Accept header for image and content requests.
const DefaultAcceptHeader = "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8"

// RequestBuilder builds outbound HTTP requests to upstream providers configured
// with client profiles, browser identity attributes, and appropriate headers.
type RequestBuilder struct {
	fpStore    fingerprint.Store
	registry   *provider.Registry
	httpClient *http.Client
}

// NewRequestBuilder creates a new RequestBuilder with the supplied fingerprint store,
// provider registry, and HTTP client.
func NewRequestBuilder(fpStore fingerprint.Store, registry *provider.Registry, httpClient *http.Client) *RequestBuilder {
	if httpClient != nil && httpClient.Jar == nil {
		jar, _ := cookiejar.New(nil)
		httpClient.Jar = jar
	}
	return &RequestBuilder{
		fpStore:    fpStore,
		registry:   registry,
		httpClient: httpClient,
	}
}

// NewRequest creates a standalone request using default headers and heuristics.
func NewRequest(ctx context.Context, rawURL, providerID, explicitReferer string) (*http.Request, error) {
	return (*RequestBuilder)(nil).BuildRequest(ctx, rawURL, providerID, explicitReferer)
}

// FingerprintStore returns the configured fingerprint store.
func (b *RequestBuilder) FingerprintStore() fingerprint.Store {
	if b == nil {
		return nil
	}
	return b.fpStore
}

// Registry returns the configured provider registry.
func (b *RequestBuilder) Registry() *provider.Registry {
	if b == nil {
		return nil
	}
	return b.registry
}

// HTTPClient returns the configured HTTP client.
func (b *RequestBuilder) HTTPClient() *http.Client {
	if b == nil {
		return nil
	}
	return b.httpClient
}

// BuildRequest builds an HTTP GET request for the given upstream URL and provider ID,
// applying client profiles (User-Agent, Client Hints, session cookies) and resolving Referer headers.
func (b *RequestBuilder) BuildRequest(ctx context.Context, rawURL string, providerID string, explicitReferer string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}

	if providerID == "" {
		if strings.Contains(rawURL, "fanfox.net") || strings.Contains(rawURL, "mfcdn.net") || strings.Contains(rawURL, "mangafox.me") || strings.Contains(rawURL, "zjcdn") {
			providerID = "mangafox"
		} else if strings.Contains(rawURL, "mangadex.org") || strings.Contains(rawURL, "mangadex.network") {
			providerID = "mangadex"
		}
	}

	baseProviderID := providerID
	if idx := strings.Index(baseProviderID, "@"); idx >= 0 {
		baseProviderID = baseProviderID[:idx]
	}
	isMangaFox := baseProviderID == "mangafox" || strings.Contains(rawURL, "fanfox.net") || strings.Contains(rawURL, "mfcdn.net") || strings.Contains(rawURL, "mangafox.me") || strings.Contains(rawURL, "zjcdn")
	isMangaDex := baseProviderID == "mangadex" || strings.Contains(rawURL, "mangadex.org") || strings.Contains(rawURL, "mangadex.network")

	userAgent := sdk.DefaultUserAgent
	var clientHints *sdk.ClientHints
	var cookies map[string]string

	if b != nil && b.fpStore != nil && providerID != "" {
		prof, err := b.fpStore.Get(providerID)
		if err != nil && strings.Contains(providerID, "@") {
			prof, err = b.fpStore.Get(baseProviderID)
		} else if err != nil && !strings.Contains(providerID, "@") {
			prof, err = b.fpStore.Get(providerID + "@builtin")
		}
		if err == nil {
			if prof.UserAgent != "" {
				userAgent = prof.UserAgent
			}
			if prof.ClientHints != nil {
				clientHints = &sdk.ClientHints{
					UA:              prof.ClientHints.UA,
					Platform:        prof.ClientHints.Platform,
					Mobile:          prof.ClientHints.Mobile,
					PlatformVersion: prof.ClientHints.PlatformVersion,
				}
			}
			if len(prof.Cookies) > 0 {
				cookies = prof.Cookies
			}
		}
	}

	if isMangaFox && len(cookies) == 0 {
		cookies = map[string]string{
			"https://fanfox.net": "isAdult=1; readway=2",
		}
	}

	if len(cookies) > 0 {
		jarNames := make(map[string]bool)
		if b != nil && b.httpClient != nil && b.httpClient.Jar != nil {
			for domainURL, rawHeader := range cookies {
				u, parseErr := url.Parse(domainURL)
				if parseErr != nil || u.Host == "" {
					continue
				}
				parts := strings.Split(rawHeader, ";")
				var jarCookies []*http.Cookie
				for _, part := range parts {
					part = strings.TrimSpace(part)
					if part == "" {
						continue
					}
					kv := strings.SplitN(part, "=", 2)
					if len(kv) == 2 {
						jarCookies = append(jarCookies, &http.Cookie{
							Name:  strings.TrimSpace(kv[0]),
							Value: strings.TrimSpace(kv[1]),
							Path:  "/",
						})
					}
				}
				if len(jarCookies) > 0 {
					b.httpClient.Jar.SetCookies(u, jarCookies)
				}
			}

			if req.URL != nil {
				for _, jc := range b.httpClient.Jar.Cookies(req.URL) {
					jarNames[jc.Name] = true
				}
			}
		}

		// Also set req.Header.Set("Cookie", ...) so cross-domain CDN image servers
		// (like s1.zjcdn.net, mfcdn.net) receive cookies regardless of CookieJar domain matching.
		var cookieHeaderParts []string
		seenNames := make(map[string]bool)
		keys := make([]string, 0, len(cookies))
		for k := range cookies {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			rawHeader := cookies[k]
			parts := strings.Split(rawHeader, ";")
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if part == "" {
					continue
				}
				kv := strings.SplitN(part, "=", 2)
				name := strings.TrimSpace(kv[0])
				if !seenNames[name] && !jarNames[name] {
					seenNames[name] = true
					cookieHeaderParts = append(cookieHeaderParts, part)
				}
			}
		}
		if len(cookieHeaderParts) > 0 {
			req.Header.Set("Cookie", strings.Join(cookieHeaderParts, "; "))
		}
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", DefaultAcceptHeader)

	if clientHints != nil {
		if clientHints.UA != "" {
			req.Header.Set("Sec-Ch-Ua", clientHints.UA)
		}
		if clientHints.Platform != "" {
			req.Header.Set("Sec-Ch-Ua-Platform", clientHints.Platform)
		}
		if clientHints.Mobile != "" {
			req.Header.Set("Sec-Ch-Ua-Mobile", clientHints.Mobile)
		}
	}

	referer := explicitReferer
	if referer == "" && providerID != "" && b != nil && b.registry != nil {
		var prov sdk.Provider
		if cp, ok := b.registry.GetContent(providerID); ok {
			prov = cp
		} else if mp, ok := b.registry.GetMetadata(providerID); ok {
			prov = mp
		} else if gp, ok := b.registry.Get(providerID); ok {
			prov = gp
		} else if strings.Contains(providerID, "@") {
			baseID := strings.SplitN(providerID, "@", 2)[0]
			if cp, ok := b.registry.GetContent(baseID); ok {
				prov = cp
			} else if mp, ok := b.registry.GetMetadata(baseID); ok {
				prov = mp
			} else if gp, ok := b.registry.Get(baseID); ok {
				prov = gp
			}
		} else {
			builtinID := providerID + "@builtin"
			if cp, ok := b.registry.GetContent(builtinID); ok {
				prov = cp
			} else if mp, ok := b.registry.GetMetadata(builtinID); ok {
				prov = mp
			} else if gp, ok := b.registry.Get(builtinID); ok {
				prov = gp
			}
		}
		if prov != nil {
			if hsGetter, ok := prov.(interface{ GetConfig() sdk.ProviderConfig }); ok {
				cfg := hsGetter.GetConfig()
				if cfg.BaseURL != "" {
					referer = cfg.BaseURL
					if !strings.HasSuffix(referer, "/") {
						referer += "/"
					}
				}
			} else if buGetter, ok := prov.(interface{ BaseURL() string }); ok {
				bu := buGetter.BaseURL()
				if bu != "" {
					referer = bu
					if !strings.HasSuffix(referer, "/") {
						referer += "/"
					}
				}
			}
		}
	}
	if referer == "" {
		if isMangaFox {
			referer = "https://fanfox.net/"
		} else if isMangaDex {
			referer = "https://mangadex.org/"
		}
	}
	if referer != "" {
		req.Header.Set("Referer", referer)
	}

	return req, nil
}

