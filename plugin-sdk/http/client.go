package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chickenzord/go-brisk"
	utls "github.com/refraction-networking/utls"
	sdk "github.com/tubruk/kiyomi/plugin-sdk"
	"github.com/tubruk/kiyomi/plugin-sdk/internal/dnsresolver"
	v1 "github.com/tubruk/kiyomi/plugin-sdk/proto/v1"
)

// clientConfig holds the internal options for assembling the HTTP client.
type clientConfig struct {
	timeout            time.Duration
	userAgent          string
	proxyURL           string
	tlsProfile         TLSProfile
	utlsHelloID        utls.ClientHelloID
	hasCustomHelloID   bool
	rps                float64
	burst              int
	maxConcurrent      int
	maxRetries         int
	minBackoff         time.Duration
	maxBackoff         time.Duration
	jar                http.CookieJar
	cookies            map[string]string
	defaultHeaders     map[string]string
	clientHints        *ClientHints
	customTransport    http.RoundTripper
	dnsResolvers       []string
	customDialContext  func(ctx context.Context, network, addr string) (net.Conn, error)
	delayMin           time.Duration
	delayMax           time.Duration
	hostRateLimitRPS   float64
	hostRateLimitBurst int
	singleflight       bool
	randomTLSProfile   bool
}

// Option configures a Client during creation.
type Option func(*clientConfig)

// WithTimeout sets the request timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *clientConfig) {
		c.timeout = d
	}
}

// WithUserAgent sets a default User-Agent string.
func WithUserAgent(ua string) Option {
	return func(c *clientConfig) {
		c.userAgent = ua
	}
}

// WithProxy configures an outbound HTTP/HTTPS/SOCKS5 proxy URL.
func WithProxy(proxyURL string) Option {
	return func(c *clientConfig) {
		c.proxyURL = proxyURL
	}
}

// WithTLSProfile sets the browser TLS fingerprint profile.
func WithTLSProfile(profile TLSProfile) Option {
	return func(c *clientConfig) {
		c.tlsProfile = profile
	}
}

// WithUTLSClientHello specifies a custom utls ClientHelloID.
func WithUTLSClientHello(id utls.ClientHelloID) Option {
	return func(c *clientConfig) {
		c.utlsHelloID = id
		c.hasCustomHelloID = true
	}
}

// WithRateLimit sets a token bucket rate limiter.
func WithRateLimit(rps float64, burst int) Option {
	return func(c *clientConfig) {
		c.rps = rps
		c.burst = burst
	}
}

// WithRateLimitSpec configures rate limiting and concurrency from protobuf RateLimitSpec.
func WithRateLimitSpec(spec *v1.RateLimitSpec) Option {
	return func(c *clientConfig) {
		if spec == nil {
			return
		}
		if spec.RequestsPerSecond > 0 {
			c.rps = float64(spec.RequestsPerSecond)
			c.burst = int(spec.RequestsPerSecond)
		}
		if spec.MaxConcurrentRequests > 0 {
			c.maxConcurrent = int(spec.MaxConcurrentRequests)
		}
	}
}

// WithSDKRateLimitSpec configures rate limiting and concurrency from Go SDK RateLimitSpec.
func WithSDKRateLimitSpec(spec sdk.RateLimitSpec) Option {
	return func(c *clientConfig) {
		if spec.RequestsPerSecond > 0 {
			c.rps = float64(spec.RequestsPerSecond)
			c.burst = int(spec.RequestsPerSecond)
		}
		if spec.MaxConcurrentRequests > 0 {
			c.maxConcurrent = int(spec.MaxConcurrentRequests)
		}
	}
}

// WithConcurrencyLimit sets a maximum cap on simultaneous active requests.
func WithConcurrencyLimit(maxConcurrent int) Option {
	return func(c *clientConfig) {
		c.maxConcurrent = maxConcurrent
	}
}

// WithRetry configures the retry transport parameters.
func WithRetry(maxAttempts int, minBackoff, maxBackoff time.Duration) Option {
	return func(c *clientConfig) {
		c.maxRetries = maxAttempts
		c.minBackoff = minBackoff
		c.maxBackoff = maxBackoff
	}
}

// WithMaxRetries configures the maximum attempts for transient errors.
func WithMaxRetries(maxAttempts int) Option {
	return func(c *clientConfig) {
		c.maxRetries = maxAttempts
	}
}

// WithCookieJar sets a custom cookie jar.
func WithCookieJar(jar http.CookieJar) Option {
	return func(c *clientConfig) {
		c.jar = jar
	}
}

// WithCookies supplies domain-scoped cookie strings.
func WithCookies(cookies map[string]string) Option {
	return func(c *clientConfig) {
		if c.cookies == nil {
			c.cookies = make(map[string]string)
		}
		for k, v := range cookies {
			c.cookies[k] = v
		}
	}
}

// WithHeaders sets default headers for all requests.
func WithHeaders(headers map[string]string) Option {
	return func(c *clientConfig) {
		if c.defaultHeaders == nil {
			c.defaultHeaders = make(map[string]string)
		}
		for k, v := range headers {
			c.defaultHeaders[k] = v
		}
	}
}

// WithHeader adds a single default header.
func WithHeader(key, value string) Option {
	return func(c *clientConfig) {
		if c.defaultHeaders == nil {
			c.defaultHeaders = make(map[string]string)
		}
		c.defaultHeaders[key] = value
	}
}

// WithClientHints configures Sec-Ch-Ua-* headers.
func WithClientHints(hints ClientHints) Option {
	return func(c *clientConfig) {
		c.clientHints = &hints
	}
}

// WithTransport overrides the base transport.
func WithTransport(rt http.RoundTripper) Option {
	return func(c *clientConfig) {
		c.customTransport = rt
	}
}

// WithDNSResolvers sets the URL list used by the SDK's default DNS loader.
func WithDNSResolvers(urls []string) Option {
	return func(c *clientConfig) {
		c.dnsResolvers = urls
	}
}

// WithDNSResolver sets a custom DialContext. Wins over WithDNSResolvers.
func WithDNSResolver(fn func(ctx context.Context, network, addr string) (net.Conn, error)) Option {
	return func(c *clientConfig) {
		c.customDialContext = fn
	}
}

// WithGlobalHttpConfig auto-wires proxy, User-Agent, and timeout from protobuf GlobalHttpConfig.
func WithGlobalHttpConfig(cfg *v1.GlobalHttpConfig) Option {
	return func(c *clientConfig) {
		if cfg == nil {
			return
		}
		if cfg.ProxyUrl != "" {
			c.proxyURL = cfg.ProxyUrl
		}
		if cfg.UserAgent != "" {
			c.userAgent = cfg.UserAgent
		}
		if cfg.TimeoutSeconds > 0 {
			c.timeout = time.Duration(cfg.TimeoutSeconds) * time.Second
		}
		if len(cfg.DnsResolvers) > 0 {
			c.dnsResolvers = cfg.DnsResolvers
		}
	}
}

// WithSDKGlobalHttpConfig auto-wires proxy, User-Agent, and timeout from SDK GlobalHttpConfig.
func WithSDKGlobalHttpConfig(cfg sdk.GlobalHttpConfig) Option {
	return func(c *clientConfig) {
		if cfg.ProxyURL != "" {
			c.proxyURL = cfg.ProxyURL
		}
		if cfg.UserAgent != "" {
			c.userAgent = cfg.UserAgent
		}
		if cfg.TimeoutSeconds > 0 {
			c.timeout = time.Duration(cfg.TimeoutSeconds) * time.Second
		}
		if len(cfg.DNSResolvers) > 0 {
			c.dnsResolvers = cfg.DNSResolvers
		}
	}
}

// WithDelayJitter introduces an artificial randomized sleep between requests within [min, max] duration.
func WithDelayJitter(min, max time.Duration) Option {
	return func(c *clientConfig) {
		c.delayMin = min
		c.delayMax = max
	}
}

// WithHostRateLimit throttles requests on a per-destination-host basis.
func WithHostRateLimit(rps float64, burst int) Option {
	return func(c *clientConfig) {
		c.hostRateLimitRPS = rps
		c.hostRateLimitBurst = burst
	}
}

// WithSingleflight deduplicates concurrent in-flight requests for identical GET/HEAD requests.
func WithSingleflight() Option {
	return func(c *clientConfig) {
		c.singleflight = true
	}
}

// WithRandomTLSProfile enables randomized ClientHello rotation among standard browser profiles.
func WithRandomTLSProfile() Option {
	return func(c *clientConfig) {
		c.randomTLSProfile = true
	}
}

// Client wraps http.Client with fingerprinting, rate limiting, and convenience methods.
type Client struct {
	httpClient *http.Client
	config     clientConfig
}

// buildHTTPClient constructs a standard library *http.Client configured via go-brisk.
func buildHTTPClient(cfg *clientConfig) *http.Client {
	if cfg.customTransport != nil {
		var rt http.RoundTripper = cfg.customTransport

		// 1. Headers & Client Hints
		headers := make(map[string]string, len(cfg.defaultHeaders)+4)
		ua := cfg.userAgent
		if ua == "" {
			ua = DefaultUserAgent
		}
		headers["User-Agent"] = ua
		for k, v := range cfg.defaultHeaders {
			headers[k] = v
		}
		if cfg.clientHints != nil {
			if cfg.clientHints.UA != "" {
				headers["Sec-Ch-Ua"] = cfg.clientHints.UA
			}
			if cfg.clientHints.Platform != "" {
				headers["Sec-Ch-Ua-Platform"] = cfg.clientHints.Platform
			}
			if cfg.clientHints.Mobile != "" {
				headers["Sec-Ch-Ua-Mobile"] = cfg.clientHints.Mobile
			}
		}
		rt = &headerTransport{
			base:    rt,
			headers: headers,
		}

		// 2. Rate limiting
		if cfg.rps > 0 {
			burst := cfg.burst
			if burst <= 0 {
				burst = int(cfg.rps)
				if burst < 1 {
					burst = 1
				}
			}
			limiter := brisk.NewTokenBucket(cfg.rps, burst)
			rt = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				if err := limiter.Wait(req.Context()); err != nil {
					return nil, err
				}
				return rt.RoundTrip(req)
			})
		}
		if cfg.hostRateLimitRPS > 0 {
			burst := cfg.hostRateLimitBurst
			if burst <= 0 {
				burst = int(cfg.hostRateLimitRPS)
				if burst < 1 {
					burst = 1
				}
			}
			hostLimiter := brisk.NewHostRateLimiter(cfg.hostRateLimitRPS, burst)
			rt = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				if err := hostLimiter.WaitRequest(req); err != nil {
					return nil, err
				}
				return rt.RoundTrip(req)
			})
		}
		if cfg.delayMin > 0 || cfg.delayMax > 0 {
			jitter := brisk.NewDelayJitter(cfg.delayMin, cfg.delayMax)
			rt = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				if err := jitter.Wait(req.Context()); err != nil {
					return nil, err
				}
				return rt.RoundTrip(req)
			})
		}

		// 3. Singleflight
		if cfg.singleflight {
			rt = brisk.NewSingleflightRoundTripper(rt, brisk.SingleflightConfig{})
		}

		// 4. Retry
		if cfg.maxRetries > 0 {
			maxRetries := cfg.maxRetries - 1
			if maxRetries < 0 {
				maxRetries = 0
			}
			minBackoff := cfg.minBackoff
			if minBackoff <= 0 {
				minBackoff = 100 * time.Millisecond
			}
			maxBackoff := cfg.maxBackoff
			if maxBackoff <= 0 {
				maxBackoff = 5 * time.Second
			}
			rb := brisk.NewRetryBuilder().
				MaxRetries(maxRetries).
				InitialBackoff(minBackoff).
				MaxBackoff(maxBackoff).
				BackoffFactor(2.0).
				Jitter(true).
				RespectRetryAfter(true).
				When(func(resp *http.Response, err error) bool {
					return IsTransientError(err, resp)
				})
			rt = brisk.NewRetryRoundTripper(rt, rb.Config())
		}

		// 5. Cookies
		if len(cfg.cookies) > 0 && cfg.jar != nil {
			cookieMW := cookieMiddleware(cfg.jar, cfg.cookies)
			rt = cookieMW(rt)
		}

		// 6. Concurrency semaphore
		if cfg.maxConcurrent > 0 {
			mw := concurrencyMiddleware(cfg.maxConcurrent)
			rt = mw(rt)
		}

		return &http.Client{
			Transport: rt,
			Jar:       cfg.jar,
			Timeout:   cfg.timeout,
		}
	}

	builder := brisk.NewBuilder()
	builder.WithTimeout(cfg.timeout)
	builder.WithCookieJar(cfg.jar)

	if cfg.proxyURL != "" {
		builder.WithProxy(cfg.proxyURL)
	}

	ua := cfg.userAgent
	if ua == "" {
		ua = DefaultUserAgent
	}
	builder.WithUserAgent(ua)

	if len(cfg.defaultHeaders) > 0 {
		builder.WithHeaders(cfg.defaultHeaders)
	}

	if cfg.clientHints != nil {
		if cfg.clientHints.UA != "" {
			builder.WithHeader("Sec-Ch-Ua", cfg.clientHints.UA)
		}
		if cfg.clientHints.Platform != "" {
			builder.WithHeader("Sec-Ch-Ua-Platform", cfg.clientHints.Platform)
		}
		if cfg.clientHints.Mobile != "" {
			builder.WithHeader("Sec-Ch-Ua-Mobile", cfg.clientHints.Mobile)
		}
	}

	// TLS fingerprinting
	if cfg.hasCustomHelloID {
		builder.WithTLSProfile(toBriskProfile(cfg.utlsHelloID))
	} else if cfg.randomTLSProfile {
		builder.WithRandomTLSProfile()
	} else if cfg.tlsProfile != "" {
		builder.WithTLSProfile(briskProfileForProfile(cfg.tlsProfile))
	}

	// DNS Resolver / DialContext
	if cfg.customDialContext != nil {
		builder.WithDialContext(cfg.customDialContext)
	} else if len(cfg.dnsResolvers) > 0 {
		dialFn, err := dnsresolver.DialFuncFromURLs(cfg.dnsResolvers)
		if err == nil && dialFn != nil {
			builder.WithDialContext(dialFn)
		}
	}

	// Rate limiting
	if cfg.rps > 0 {
		burst := cfg.burst
		if burst <= 0 {
			burst = int(cfg.rps)
			if burst < 1 {
				burst = 1
			}
		}
		builder.WithRateLimit(cfg.rps, burst)
	}
	if cfg.hostRateLimitRPS > 0 {
		burst := cfg.hostRateLimitBurst
		if burst <= 0 {
			burst = int(cfg.hostRateLimitRPS)
			if burst < 1 {
				burst = 1
			}
		}
		builder.WithHostRateLimit(cfg.hostRateLimitRPS, burst)
	}
	if cfg.delayMin > 0 || cfg.delayMax > 0 {
		builder.WithRequestDelay(cfg.delayMin, cfg.delayMax)
	}

	// Singleflight
	if cfg.singleflight {
		builder.WithSingleflight()
	}

	// Concurrency semaphore via brisk.WithMiddleware
	if cfg.maxConcurrent > 0 {
		builder.WithMiddleware(concurrencyMiddleware(cfg.maxConcurrent))
	}

	// Cookies middleware
	if len(cfg.cookies) > 0 && cfg.jar != nil {
		builder.WithMiddleware(cookieMiddleware(cfg.jar, cfg.cookies))
	}

	// Retry via brisk.RetryBuilder
	if cfg.maxRetries > 0 {
		maxRetries := cfg.maxRetries - 1
		if maxRetries < 0 {
			maxRetries = 0
		}
		minBackoff := cfg.minBackoff
		if minBackoff <= 0 {
			minBackoff = 100 * time.Millisecond
		}
		maxBackoff := cfg.maxBackoff
		if maxBackoff <= 0 {
			maxBackoff = 5 * time.Second
		}
		rb := brisk.NewRetryBuilder().
			MaxRetries(maxRetries).
			InitialBackoff(minBackoff).
			MaxBackoff(maxBackoff).
			BackoffFactor(2.0).
			Jitter(true).
			RespectRetryAfter(true).
			When(func(resp *http.Response, err error) bool {
				return IsTransientError(err, resp)
			})
		builder.WithRetryConfig(rb.Config())
	}

	client, err := builder.Build()
	if err != nil {
		return http.DefaultClient
	}
	return client
}

// NewClient creates a new HTTP client with the provided options.
func NewClient(opts ...Option) *Client {
	cfg := clientConfig{
		timeout:        30 * time.Second,
		userAgent:      DefaultUserAgent,
		maxRetries:     3,
		minBackoff:     100 * time.Millisecond,
		maxBackoff:     5 * time.Second,
		defaultHeaders: make(map[string]string),
		cookies:        make(map[string]string),
	}

	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	// Auto-load from env if no DNS source was set explicitly.
	if len(cfg.dnsResolvers) == 0 && cfg.customDialContext == nil {
		if envList := dnsresolver.LoadFromEnv("KIYOMI_DNS_RESOLVERS"); len(envList) > 0 {
			cfg.dnsResolvers = envList
		}
	}

	if cfg.jar == nil {
		jar, _ := cookiejar.New(nil)
		cfg.jar = jar
	}

	httpClient := buildHTTPClient(&cfg)

	return &Client{
		httpClient: httpClient,
		config:     cfg,
	}
}

// NewFingerprintedClient creates a client configured with global host HTTP settings and fingerprinting options.
func NewFingerprintedClient(httpConfig *v1.GlobalHttpConfig, opts ...Option) *Client {
	allOpts := make([]Option, 0, len(opts)+1)
	if httpConfig != nil {
		allOpts = append(allOpts, WithGlobalHttpConfig(httpConfig))
	}
	allOpts = append(allOpts, opts...)
	return NewClient(allOpts...)
}

// StandardClient returns the underlying *http.Client.
func (c *Client) StandardClient() *http.Client {
	if c == nil {
		return http.DefaultClient
	}
	return c.httpClient
}

// Do sends an HTTP request and returns an HTTP response.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	return c.httpClient.Do(req)
}

// NewRequest returns a new http.Request with the given context, method, URL, and body.
func (c *Client) NewRequest(ctx context.Context, method, targetURL string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, targetURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	return req, nil
}

// Get issues a GET request to the specified URL.
func (c *Client) Get(ctx context.Context, targetURL string) (*http.Response, error) {
	req, err := c.NewRequest(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

// Post issues a POST request with the given contentType and body.
func (c *Client) Post(ctx context.Context, targetURL string, contentType string, body io.Reader) (*http.Response, error) {
	req, err := c.NewRequest(ctx, http.MethodPost, targetURL, body)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return c.Do(req)
}

// PostForm issues a POST request with form-encoded data.
func (c *Client) PostForm(ctx context.Context, targetURL string, data url.Values) (*http.Response, error) {
	return c.Post(ctx, targetURL, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
}

// GetJSON issues a GET request and unmarshals the JSON response into target.
func (c *Client) GetJSON(ctx context.Context, targetURL string, target any) error {
	req, err := c.NewRequest(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed for %s: %w", targetURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodySnippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("http %d for %s: %s", resp.StatusCode, targetURL, strings.TrimSpace(string(bodySnippet)))
	}

	if target == nil {
		return nil
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("failed to decode JSON from %s: %w", targetURL, err)
	}
	return nil
}

// GetDocument issues a GET request and parses the HTML response into a goquery.Document.
func (c *Client) GetDocument(ctx context.Context, targetURL string) (*goquery.Document, error) {
	req, err := c.NewRequest(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed for %s: %w", targetURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http %d for %s", resp.StatusCode, targetURL)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML document from %s: %w", targetURL, err)
	}
	return doc, nil
}

// Clone creates a shallow clone of the client config and returns a new Client.
func (c *Client) Clone(opts ...Option) *Client {
	newCfg := c.config
	// Clone maps
	if c.config.defaultHeaders != nil {
		newCfg.defaultHeaders = make(map[string]string, len(c.config.defaultHeaders))
		for k, v := range c.config.defaultHeaders {
			newCfg.defaultHeaders[k] = v
		}
	}
	if c.config.cookies != nil {
		newCfg.cookies = make(map[string]string, len(c.config.cookies))
		for k, v := range c.config.cookies {
			newCfg.cookies[k] = v
		}
	}

	for _, opt := range opts {
		if opt != nil {
			opt(&newCfg)
		}
	}

	httpClient := buildHTTPClient(&newCfg)

	return &Client{
		httpClient: httpClient,
		config:     newCfg,
	}
}
