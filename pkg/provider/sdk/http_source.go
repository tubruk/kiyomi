package sdk

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/tubruk/kiyomi/pkg/dnsresolver"
	"github.com/tubruk/kiyomi/pkg/fingerprint"
)

var (
	globalDNSResolvers   []string
	globalDNSResolversMu sync.RWMutex
)

// SetGlobalDNSResolvers sets the process-wide fallback DNS resolver list.
// Call once at startup from cmd/kiyomi/main.go after config.Load().
// The list is consumed by HttpSource.BuildTransport() when the per-provider
// ProviderConfig.DNSResolvers is empty.
func SetGlobalDNSResolvers(urls []string) {
	globalDNSResolversMu.Lock()
	defer globalDNSResolversMu.Unlock()
	globalDNSResolvers = append([]string(nil), urls...)
}

// GetGlobalDNSResolvers returns a snapshot of the global resolver list.
func GetGlobalDNSResolvers() []string {
	globalDNSResolversMu.RLock()
	defer globalDNSResolversMu.RUnlock()
	return append([]string(nil), globalDNSResolvers...)
}

// HttpSource provides HTTP communication, header defaults, cookie management, and URL resolution for providers.
type HttpSource struct {
	Config ProviderConfig
	Client *http.Client

	Fingerprint fingerprint.Store
}

func (h *HttpSource) WithFingerprintStore(s fingerprint.Store) *HttpSource {
	h.Fingerprint = s
	if h.Client != nil {
		h.Client.Transport = h.BuildTransport()
	}
	return h
}

// GetConfig returns the provider config.
func (h *HttpSource) GetConfig() ProviderConfig {
	if h == nil {
		return ProviderConfig{}
	}
	return h.Config
}

func NewHttpSource(cfg ProviderConfig) (*HttpSource, error) {
	if cfg.UserAgent == "" {
		cfg.UserAgent = DefaultUserAgent
	}
	if cfg.HTTPTimeout == 0 {
		cfg.HTTPTimeout = 15 * time.Second
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	for domainURL, cookieStr := range cfg.Cookies {
		u, parseErr := url.Parse(domainURL)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid cookie domain URL %q: %w", domainURL, parseErr)
		}
		rawCookies := strings.Split(cookieStr, ";")
		var cookies []*http.Cookie
		for _, raw := range rawCookies {
			raw = strings.TrimSpace(raw)
			if raw == "" {
				continue
			}
			parts := strings.SplitN(raw, "=", 2)
			if len(parts) == 2 {
				cookies = append(cookies, &http.Cookie{
					Name:  strings.TrimSpace(parts[0]),
					Value: strings.TrimSpace(parts[1]),
				})
			}
		}
		if len(cookies) > 0 {
			jar.SetCookies(u, cookies)
		}
	}

	client := &http.Client{
		Jar:     jar,
		Timeout: cfg.HTTPTimeout,
	}

	source := &HttpSource{
		Config: cfg,
		Client: client,
	}
	client.Transport = source.buildTransport()

	return source, nil
}

func (h *HttpSource) buildTransport() http.RoundTripper {
	return h.BuildTransport()
}

// BuildTransport returns the SDK's outbound transport configured
// with the current ProviderConfig (DNS resolver, proxy URL,
// fingerprint store). Exported so callers can rebuild the
// transport after mutating Config at startup.
func (h *HttpSource) BuildTransport() http.RoundTripper {
	resolver := func() (fingerprint.TLSProfile, bool) {
		if h != nil && h.Fingerprint != nil && h.Config.ID != "" {
			p, err := h.Fingerprint.Get(h.Config.ID)
			if err == nil && p.TLSProfile != "" {
				return p.TLSProfile, true
			}
		}
		return fingerprint.TLSProfileDefault, false
	}

	base := buildOutboundTransport(h.Config, resolver)
	return NewRetryTransport(&headerTransport{
		base:   base,
		source: h,
	})
}

// RetryTransport wraps an http.RoundTripper and retries requests that encounter transient network errors.
type RetryTransport struct {
	Base        http.RoundTripper
	MaxAttempts int
}

// NewRetryTransport returns a RetryTransport wrapping base. Default max attempts is 3.
func NewRetryTransport(base http.RoundTripper) *RetryTransport {
	if base == nil {
		base = http.DefaultTransport
	}
	return &RetryTransport{
		Base:        base,
		MaxAttempts: 3,
	}
}

func (r *RetryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	maxAttempts := r.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 3
	}

	var lastResp *http.Response
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if ctxErr := req.Context().Err(); ctxErr != nil {
			if lastResp != nil && lastResp.Body != nil {
				_ = lastResp.Body.Close()
			}
			return nil, ctxErr
		}

		if attempt > 1 && req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				if lastResp != nil && lastResp.Body != nil {
					_ = lastResp.Body.Close()
				}
				return nil, err
			}
			req.Body = body
		}

		resp, err := r.Base.RoundTrip(req)
		if !IsTransientError(err, resp) {
			return resp, err
		}

		lastResp = resp
		lastErr = err

		if attempt == maxAttempts {
			break
		}

		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}

		delay := time.Duration(100*(1<<(attempt-1))) * time.Millisecond
		select {
		case <-req.Context().Done():
			return nil, req.Context().Err()
		case <-time.After(delay):
		}
	}

	return lastResp, lastErr
}

// SetTransport replaces the HttpSource's outbound transport. Pass
// nil to rebuild from current Config. The new transport takes
// effect on the next request.
func (h *HttpSource) SetTransport(tr http.RoundTripper) {
	if h.Client == nil {
		return
	}
	if tr == nil {
		tr = h.BuildTransport()
	}
	h.Client.Transport = tr
}

// buildOutboundTransport returns a fresh *http.Transport configured
// with fingerprint TLS profile resolution, provider DNS resolvers, and proxy URL.
func buildOutboundTransport(cfg ProviderConfig, resolve fingerprint.ProfileResolver) *http.Transport {
	t := fingerprint.NewTransport(resolve)

	// Merge global + per-provider resolvers. Per-provider wins if non-empty.
	merged := cfg.DNSResolvers
	if len(merged) == 0 {
		merged = GetGlobalDNSResolvers()
	}
	if len(merged) > 0 {

		dialFn, err := dnsresolver.DialFuncFromURLs(merged)
		if err != nil {
			slog.Warn("sdk: invalid dns resolver urls, falling back to system resolver",
				slog.String("error", err.Error()),
			)
		} else if dialFn != nil {
			t.DialContext = dialFn
		}
	}
	if cfg.ProxyURL != "" {
		if u, err := url.Parse(cfg.ProxyURL); err == nil {
			fingerprint.SetProxy(t, u)
		}
	}
	return t
}

// dialUpstream opens a TCP connection to the first reachable DNS
// upstream. Always TCP regardless of the requested network — DNS-
// over-TCP is more reliable across firewalls and NATs than raw UDP,
// and Go's net.Resolver speaks the same wire protocol on TCP.
func dialUpstream(ctx context.Context, resolvers []string) (net.Conn, error) {
	var lastErr error
	for _, r := range resolvers {
		d := &net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}
		conn, err := d.DialContext(ctx, "tcp", r)
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = errors.New("no resolvers configured")
	}
	return nil, fmt.Errorf("sdk: all DNS upstreams unreachable: %w", lastErr)
}

type headerTransport struct {
	base   http.RoundTripper
	source *HttpSource
}

func (t *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.source == nil {
		return t.base.RoundTrip(req)
	}
	var prof fingerprint.Profile
	if t.source.Fingerprint != nil && t.source.Config.ID != "" {
		p, err := t.source.Fingerprint.Get(t.source.Config.ID)
		if err == nil {
			prof = p
		} else if !errors.Is(err, fingerprint.ErrUnknownSource) {
			slog.Warn("sdk: fingerprint lookup error",
				slog.String("sourceId", t.source.Config.ID),
				slog.String("error", err.Error()),
			)
		}
	}

	ua := t.source.Config.UserAgent
	if prof.UserAgent != "" {
		ua = prof.UserAgent
	}
	if ua == "" {
		ua = DefaultUserAgent
	}

	if req.Header.Get("User-Agent") == "" || prof.UserAgent != "" {
		req.Header.Set("User-Agent", ua)
	}

	hints := t.source.Config.ClientHints
	if prof.ClientHints != nil {
		hints = &ClientHints{
			UA:              prof.ClientHints.UA,
			Platform:        prof.ClientHints.Platform,
			Mobile:          prof.ClientHints.Mobile,
			PlatformVersion: prof.ClientHints.PlatformVersion,
		}
	}
	if hints == nil {
		hints = DefaultClientHints()
	}

	if prof.ClientHints != nil {
		if hints.UA != "" {
			req.Header.Set("Sec-Ch-Ua", hints.UA)
		} else {
			req.Header.Del("Sec-Ch-Ua")
		}
		if hints.Platform != "" {
			req.Header.Set("Sec-Ch-Ua-Platform", hints.Platform)
		} else {
			req.Header.Del("Sec-Ch-Ua-Platform")
		}
		if hints.Mobile != "" {
			req.Header.Set("Sec-Ch-Ua-Mobile", hints.Mobile)
		} else {
			req.Header.Del("Sec-Ch-Ua-Mobile")
		}
	} else {
		if hints.UA != "" && req.Header.Get("Sec-Ch-Ua") == "" {
			req.Header.Set("Sec-Ch-Ua", hints.UA)
		}
		if hints.Platform != "" && req.Header.Get("Sec-Ch-Ua-Platform") == "" {
			req.Header.Set("Sec-Ch-Ua-Platform", hints.Platform)
		}
		if hints.Mobile != "" && req.Header.Get("Sec-Ch-Ua-Mobile") == "" {
			req.Header.Set("Sec-Ch-Ua-Mobile", hints.Mobile)
		}
	}

	if len(prof.Cookies) > 0 && req.URL != nil && t.source.Client != nil && t.source.Client.Jar != nil {
		for domainURL, rawHeader := range prof.Cookies {
			u, err := url.Parse(domainURL)
			if err != nil || u.Host == "" {
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
				t.source.Client.Jar.SetCookies(u, jarCookies)
			}
		}
		for _, c := range t.source.Client.Jar.Cookies(req.URL) {
			if !hasCookie(req, c.Name) {
				req.AddCookie(c)
			}
		}
	}

	return t.base.RoundTrip(req)
}

func hasCookie(req *http.Request, name string) bool {
	for _, c := range req.Cookies() {
		if c.Name == name {
			return true
		}
	}
	return false
}

func (h *HttpSource) NewRequest(ctx context.Context, targetURL string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", h.Config.UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	if h.Config.BaseURL != "" {
		req.Header.Set("Referer", h.Config.BaseURL)
	}

	return req, nil
}

func (h *HttpSource) GetDocument(ctx context.Context, targetURL string) (*goquery.Document, error) {
	req, err := h.NewRequest(ctx, targetURL)
	if err != nil {
		return nil, err
	}

	resp, err := h.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed for %s: %w", targetURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http %d for %s", resp.StatusCode, targetURL)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse html document: %w", err)
	}

	return doc, nil
}

func (h *HttpSource) ResolveURL(relativePath string) string {
	if strings.HasPrefix(relativePath, "http://") || strings.HasPrefix(relativePath, "https://") {
		return relativePath
	}
	base := strings.TrimSuffix(h.Config.BaseURL, "/")
	path := strings.TrimPrefix(relativePath, "/")
	return fmt.Sprintf("%s/%s", base, path)
}

func (h *HttpSource) FetchAsset(ctx context.Context, assetURL string) (*http.Response, error) {
	req, err := h.NewRequest(ctx, assetURL)
	if err != nil {
		return nil, err
	}
	return h.Client.Do(req)
}

func FetchAssetStream(ctx context.Context, client *http.Client, assetURL, referer string) (io.ReadCloser, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, assetURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create asset request: %w", err)
	}

	req.Header.Set("User-Agent", DefaultUserAgent)
	if referer != "" {
		req.Header.Set("Referer", referer)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch asset: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close() //nolint:errcheck
		return nil, "", fmt.Errorf("asset request returned status %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	return resp.Body, contentType, nil
}
