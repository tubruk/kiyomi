package http

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	utls "github.com/refraction-networking/utls"
)

// TLSProfile selects which browser's TLS Client Hello to emulate for outbound HTTPS connections.
type TLSProfile string

const (
	// TLSProfileDefault uses Go's native net/http transport.
	TLSProfileDefault TLSProfile = "default"
	// TLSProfileChrome emulates a modern Chrome Client Hello via utls.
	TLSProfileChrome TLSProfile = "chrome"
	// TLSProfileFirefox emulates a modern Firefox Client Hello via utls.
	TLSProfileFirefox TLSProfile = "firefox"
)

// Valid reports whether p is one of the recognized TLS profile names.
func (p TLSProfile) Valid() bool {
	switch p {
	case TLSProfileDefault, TLSProfileChrome, TLSProfileFirefox:
		return true
	}
	return false
}

// ClientHints holds Sec-Ch-Ua-* headers to emulate browser client hints.
type ClientHints struct {
	UA              string `json:"ua,omitempty"`
	Platform        string `json:"platform,omitempty"`
	Mobile          string `json:"mobile,omitempty"`
	PlatformVersion string `json:"platformVersion,omitempty"`
}

// DefaultClientHints returns standard Chrome client hints.
func DefaultClientHints() ClientHints {
	return ClientHints{
		UA:       `"Google Chrome";v="120", "Chromium";v="120", "Not?A_Brand";v="24"`,
		Platform: `"Windows"`,
		Mobile:   "?0",
	}
}

// DefaultUserAgent is the standard browser User-Agent fallback.
const DefaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

type dialer interface {
	DialContext(ctx context.Context, network, addr string) (net.Conn, error)
}

type dialerFunc func(ctx context.Context, network, addr string) (net.Conn, error)

func (f dialerFunc) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return f(ctx, network, addr)
}

func helloIDForProfile(p TLSProfile) (utls.ClientHelloID, bool) {
	switch p {
	case TLSProfileChrome:
		return utls.HelloChrome_120, true
	case TLSProfileFirefox:
		return utls.HelloFirefox_120, true
	default:
		return utls.ClientHelloID{}, false
	}
}

func dialWithProfile(ctx context.Context, d dialer, network, addr string, helloID utls.ClientHelloID) (net.Conn, error) {
	rawConn, err := d.DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}

	host, _, splitErr := net.SplitHostPort(addr)
	if splitErr != nil {
		host = addr
	}

	tlsConfig := &utls.Config{
		ServerName: host,
	}
	uConn := utls.UClient(rawConn, tlsConfig, helloID)
	if hsErr := uConn.HandshakeContext(ctx); hsErr != nil {
		_ = rawConn.Close()
		return nil, fmt.Errorf("utls handshake failed: %w", hsErr)
	}
	return uConn, nil
}

// buildBaseTransport constructs the core *http.Transport with proxy, connection pooling, and TLS settings.
func buildBaseTransport(cfg *clientConfig) http.RoundTripper {
	d := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	t := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
		DialContext:           d.DialContext,
	}

	if cfg.proxyURL != "" {
		if u, err := url.Parse(cfg.proxyURL); err == nil {
			t.Proxy = http.ProxyURL(u)
		}
	}

	helloID := cfg.utlsHelloID
	hasCustom := cfg.hasCustomHelloID
	if !hasCustom && cfg.tlsProfile != "" && cfg.tlsProfile != TLSProfileDefault {
		if id, ok := helloIDForProfile(cfg.tlsProfile); ok {
			helloID = id
			hasCustom = true
		}
	}

	if hasCustom {
		t.DialTLSContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			dialerToUse := dialer(d)
			if t.DialContext != nil {
				dialerToUse = dialerFunc(t.DialContext)
			}
			return dialWithProfile(ctx, dialerToUse, network, addr, helloID)
		}
	}

	return t
}

// headerTransport applies default headers, User-Agent, Sec-Ch-Ua client hints, and custom cookies to outgoing requests.
type headerTransport struct {
	base   http.RoundTripper
	config *clientConfig
}

func (t *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.config == nil {
		return t.base.RoundTrip(req)
	}

	// Apply User-Agent
	ua := t.config.userAgent
	if ua == "" {
		ua = DefaultUserAgent
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", ua)
	}

	// Apply Client Hints if set
	if t.config.clientHints != nil {
		hints := t.config.clientHints
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

	// Apply custom headers
	for k, v := range t.config.defaultHeaders {
		if req.Header.Get(k) == "" {
			req.Header.Set(k, v)
		}
	}

	// Apply cookies if domain match
	if len(t.config.cookies) > 0 && req.URL != nil && t.config.jar != nil {
		for domainURL, rawHeader := range t.config.cookies {
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
				t.config.jar.SetCookies(u, jarCookies)
			}
		}
		for _, c := range t.config.jar.Cookies(req.URL) {
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
