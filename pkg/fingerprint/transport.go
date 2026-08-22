package fingerprint

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/proxy"
)

// ProfileResolver returns the TLSProfile to use for an outgoing
// HTTPS connection. It is called once per TCP dial from the
// transport, so a runtime change to the fingerprint store takes
// effect on the next new connection (already-pooled
// keep-alive connections keep their original Client Hello
// until the idle timeout evicts them). The bool is false when
// no override is registered for the source; the transport
// then uses Go's native crypto/tls.
type ProfileResolver func() (TLSProfile, bool)

// dialer is the minimal interface dialWithProfile needs. Both
// *net.Dialer and the test shim satisfy it.
type dialer interface {
	DialContext(ctx context.Context, network, addr string) (net.Conn, error)
}

type dialerFunc func(ctx context.Context, network, addr string) (net.Conn, error)

func (f dialerFunc) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return f(ctx, network, addr)
}

type transportState struct {
	mu      sync.RWMutex
	proxyFn func(*http.Request) (*url.URL, error)
}

var transportStates sync.Map // map[*http.Transport]*transportState

type bufferedConn struct {
	net.Conn
	r *bufio.Reader
}

func (b *bufferedConn) Read(p []byte) (int, error) {
	return b.r.Read(p)
}

type dialerWrapper struct {
	d   dialer
	ctx context.Context
}

func (w *dialerWrapper) Dial(network, addr string) (net.Conn, error) {
	ctx := w.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	return w.d.DialContext(ctx, network, addr)
}

func (w *dialerWrapper) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return w.d.DialContext(ctx, network, addr)
}

// SetProxy configures the proxy URL on t, ensuring HTTPS requests are routed
// through DialTLSContext for TLS fingerprint spoofing while plain HTTP requests
// use standard proxy forwarding.
func SetProxy(t *http.Transport, u *url.URL) {
	if t == nil {
		return
	}
	if u == nil {
		SetProxyFunc(t, nil)
		return
	}
	SetProxyFunc(t, http.ProxyURL(u))
}

// SetProxyFunc configures the proxy resolver function on t.
func SetProxyFunc(t *http.Transport, fn func(*http.Request) (*url.URL, error)) {
	if t == nil {
		return
	}
	if v, ok := transportStates.Load(t); ok {
		state := v.(*transportState)
		state.mu.Lock()
		state.proxyFn = fn
		state.mu.Unlock()
	}
	t.Proxy = func(req *http.Request) (*url.URL, error) {
		if req.URL != nil && req.URL.Scheme == "https" {
			return nil, nil
		}
		if fn != nil {
			return fn(req)
		}
		return nil, nil
	}
}

// ProxyFunc returns a proxy function suitable for http.Transport that handles
// HTTPS proxying via DialTLSContext (for utls fingerprinting) while delegating
// HTTP requests to fn.
func ProxyFunc(fn func(*http.Request) (*url.URL, error)) func(*http.Request) (*url.URL, error) {
	if fn == nil {
		return nil
	}
	return func(req *http.Request) (*url.URL, error) {
		if req.URL != nil && req.URL.Scheme == "https" {
			return nil, nil
		}
		return fn(req)
	}
}

// ProxyURL returns a proxy function that routes HTTP requests to u, while letting
// HTTPS requests fall through to DialTLSContext for utls fingerprinting.
func ProxyURL(u *url.URL) func(*http.Request) (*url.URL, error) {
	if u == nil {
		return nil
	}
	return ProxyFunc(http.ProxyURL(u))
}

// buildUTLSConfig constructs a *utls.Config inheriting from cfg if present.
func buildUTLSConfig(cfg *tls.Config, defaultServerName string) *utls.Config {
	uCfg := &utls.Config{
		ServerName: defaultServerName,
	}
	if cfg != nil {
		if cfg.ServerName != "" {
			uCfg.ServerName = cfg.ServerName
		}
		uCfg.InsecureSkipVerify = cfg.InsecureSkipVerify
		uCfg.RootCAs = cfg.RootCAs
		uCfg.ClientCAs = cfg.ClientCAs
		uCfg.MinVersion = cfg.MinVersion
		uCfg.MaxVersion = cfg.MaxVersion
		uCfg.KeyLogWriter = cfg.KeyLogWriter
		uCfg.VerifyPeerCertificate = cfg.VerifyPeerCertificate
		if len(cfg.NextProtos) > 0 {
			uCfg.NextProtos = append([]string(nil), cfg.NextProtos...)
		}
		if len(cfg.CipherSuites) > 0 {
			uCfg.CipherSuites = append([]uint16(nil), cfg.CipherSuites...)
		}
	}
	return uCfg
}

// buildTLSConfig returns a cloned *tls.Config (or a new one) configured with defaultServerName and ALPN.
func buildTLSConfig(cfg *tls.Config, defaultServerName string) *tls.Config {
	var out *tls.Config
	if cfg != nil {
		out = cfg.Clone()
	} else {
		out = &tls.Config{}
	}
	if out.ServerName == "" {
		out.ServerName = defaultServerName
	}
	if len(out.NextProtos) == 0 {
		out.NextProtos = []string{"h2", "http/1.1"}
	}
	return out
}

// dialProxy establishes a TCP connection to the proxy and negotiates HTTP CONNECT (or SOCKS5).
func dialProxy(ctx context.Context, d dialer, proxyURL *url.URL, targetAddr string, tlsCfg *tls.Config) (net.Conn, error) {
	switch proxyURL.Scheme {
	case "http", "https":
		proxyHost, proxyPort, err := net.SplitHostPort(proxyURL.Host)
		if err != nil {
			proxyHost = proxyURL.Host
			if proxyURL.Scheme == "https" {
				proxyPort = "443"
			} else {
				proxyPort = "80"
			}
		}
		proxyAddr := net.JoinHostPort(proxyHost, proxyPort)

		conn, err := d.DialContext(ctx, "tcp", proxyAddr)
		if err != nil {
			return nil, fmt.Errorf("fingerprint: dial proxy %s: %w", proxyAddr, err)
		}

		if proxyURL.Scheme == "https" {
			proxyTLSConfig := buildTLSConfig(tlsCfg, proxyHost)
			tlsProxyConn := tls.Client(conn, proxyTLSConfig)
			if hsErr := tlsProxyConn.HandshakeContext(ctx); hsErr != nil {
				_ = conn.Close()
				return nil, fmt.Errorf("fingerprint: tls handshake to proxy %s: %w", proxyAddr, hsErr)
			}
			conn = tlsProxyConn
		}

		connectReq := &http.Request{
			Method: http.MethodConnect,
			URL:    &url.URL{Opaque: targetAddr},
			Host:   targetAddr,
			Header: make(http.Header),
		}
		connectReq.Header.Set("User-Agent", "Go-http-client/1.1")

		if proxyURL.User != nil {
			username := proxyURL.User.Username()
			password, _ := proxyURL.User.Password()
			auth := username + ":" + password
			basicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
			connectReq.Header.Set("Proxy-Authorization", basicAuth)
		}

		if writeErr := connectReq.Write(conn); writeErr != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("fingerprint: write CONNECT to proxy %s: %w", proxyAddr, writeErr)
		}

		br := bufio.NewReader(conn)
		resp, readErr := http.ReadResponse(br, connectReq)
		if readErr != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("fingerprint: read CONNECT response from proxy %s: %w", proxyAddr, readErr)
		}
		_ = resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			_ = conn.Close()
			return nil, fmt.Errorf("fingerprint: proxy %s CONNECT failed with status %d: %s", proxyAddr, resp.StatusCode, resp.Status)
		}

		if br.Buffered() > 0 {
			return &bufferedConn{
				Conn: conn,
				r:    br,
			}, nil
		}
		return conn, nil

	case "socks5", "socks5h":
		forward := &dialerWrapper{d: d, ctx: ctx}
		sDialer, err := proxy.FromURL(proxyURL, forward)
		if err != nil {
			return nil, fmt.Errorf("fingerprint: invalid socks5 proxy url: %w", err)
		}
		if ctxDialer, ok := sDialer.(proxy.ContextDialer); ok {
			return ctxDialer.DialContext(ctx, "tcp", targetAddr)
		}
		return sDialer.Dial("tcp", targetAddr)

	default:
		return nil, fmt.Errorf("fingerprint: unsupported proxy scheme %q", proxyURL.Scheme)
	}
}

// NewTransport returns an *http.Transport whose TLS Client Hello
// is chosen at dial time based on the profile returned by
// resolve. The transport's connection pool, idle timeout, and
// other http.Transport semantics are preserved; DialTLSContext is
// used for TLS connections.
//
// Pass a nil resolver to get behaviour equivalent to
// http.DefaultTransport — same native crypto/tls fingerprint,
// same connection pool, no utls cost.
func NewTransport(resolve ProfileResolver) *http.Transport {
	d := &net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}
	t := &http.Transport{
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
		DialContext:           d.DialContext,
	}

	state := &transportState{
		proxyFn: http.ProxyFromEnvironment,
	}
	transportStates.Store(t, state)

	t.Proxy = func(req *http.Request) (*url.URL, error) {
		if req.URL != nil && req.URL.Scheme == "https" {
			return nil, nil
		}
		state.mu.RLock()
		fn := state.proxyFn
		state.mu.RUnlock()
		if fn != nil {
			return fn(req)
		}
		return nil, nil
	}

	t.DialTLSContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		dialerToUse := dialer(d)
		if t.DialContext != nil {
			dialerToUse = dialerFunc(t.DialContext)
		}

		var proxyURL *url.URL
		state.mu.RLock()
		fn := state.proxyFn
		state.mu.RUnlock()
		if fn != nil {
			dummyReq := &http.Request{
				Method: http.MethodConnect,
				URL: &url.URL{
					Scheme: "https",
					Host:   addr,
				},
				Header: make(http.Header),
			}
			var proxyErr error
			proxyURL, proxyErr = fn(dummyReq)
			if proxyErr != nil {
				return nil, fmt.Errorf("fingerprint: proxy resolution: %w", proxyErr)
			}
		}

		var rawConn net.Conn
		if proxyURL != nil {
			var pErr error
			rawConn, pErr = dialProxy(ctx, dialerToUse, proxyURL, addr, t.TLSClientConfig)
			if pErr != nil {
				return nil, pErr
			}
		}

		var shim dialer
		if rawConn != nil {
			shim = dialerFunc(func(ctx context.Context, network, addr string) (net.Conn, error) {
				return rawConn, nil
			})
		} else {
			shim = dialerToUse
		}

		conn, err := dialWithProfile(ctx, shim, network, addr, resolve, t.TLSClientConfig)
		if err != nil {
			if rawConn != nil {
				_ = rawConn.Close()
			}
			return nil, err
		}
		if _, ok := conn.(*utls.UConn); ok {
			return conn, nil
		}

		host, _, splitErr := net.SplitHostPort(addr)
		if splitErr != nil {
			host = addr
		}
		tlsConfig := buildTLSConfig(t.TLSClientConfig, host)
		tlsConn := tls.Client(conn, tlsConfig)
		if hsErr := tlsConn.HandshakeContext(ctx); hsErr != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("fingerprint: tls handshake: %w", hsErr)
		}
		return tlsConn, nil
	}
	return t
}

// dialWithProfile performs the TCP dial, then wraps the connection in
// a utls UConn using the requested Client Hello for non-default profiles.
// For default or unregistered profiles, it returns rawConn, nil.
func dialWithProfile(ctx context.Context, d dialer, network, addr string, resolve ProfileResolver, tlsConfig ...*tls.Config) (net.Conn, error) {
	rawConn, err := d.DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}

	profile, ok := tlsProfileFromResolver(resolve)
	if !ok || profile == TLSProfileDefault {
		return rawConn, nil
	}

	host, _, splitErr := net.SplitHostPort(addr)
	if splitErr != nil {
		host = addr
	}

	helloID, err := helloIDFor(profile)
	if err != nil {
		_ = rawConn.Close()
		return nil, fmt.Errorf("fingerprint: dial: %w", err)
	}

	var baseTLSConfig *tls.Config
	if len(tlsConfig) > 0 {
		baseTLSConfig = tlsConfig[0]
	}
	utlsConfig := buildUTLSConfig(baseTLSConfig, host)
	uConn := utls.UClient(rawConn, utlsConfig, helloID)
	if hsErr := uConn.HandshakeContext(ctx); hsErr != nil {
		_ = rawConn.Close()
		return nil, fmt.Errorf("fingerprint: utls handshake: %w", hsErr)
	}
	return uConn, nil
}

// tlsProfileFromResolver normalises a resolver call. The
// resolver returns the raw profile plus an "is registered"
// bool; we also normalise the empty string to TLSProfileDefault
// so the transport doesn't have to think about it.
func tlsProfileFromResolver(resolve ProfileResolver) (TLSProfile, bool) {
	if resolve == nil {
		return TLSProfileDefault, false
	}
	profile, ok := resolve()
	if !ok {
		return TLSProfileDefault, false
	}
	return NormalizeTLSProfile(profile), true
}

// helloIDFor maps our TLSProfile enum to the utls ClientHelloID.
// Unknown profiles return an error so the caller closes the
// connection rather than silently falling back. "default" is
// never expected to reach here (the caller short-circuits it)
// but the error message makes that explicit.
func helloIDFor(p TLSProfile) (utls.ClientHelloID, error) {
	switch p {
	case TLSProfileChrome:
		return utls.HelloChrome_120, nil
	case TLSProfileFirefox:
		return utls.HelloFirefox_120, nil
	case TLSProfileDefault:
		return utls.ClientHelloID{}, errors.New("fingerprint: default profile reached utls path; caller bug")
	default:
		return utls.ClientHelloID{}, fmt.Errorf("fingerprint: unknown TLS profile %q", p)
	}
}
