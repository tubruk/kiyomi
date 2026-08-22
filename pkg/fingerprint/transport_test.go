package fingerprint

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// recordingConn is a net.Conn that records every byte written to
// it and returns canned bytes on read. Used to capture the
// Client Hello the transport writes during a utls handshake
// without standing up a real TLS server.
type recordingConn struct {
	mu       sync.Mutex
	written  bytes.Buffer
	readBuf  bytes.Buffer
	closed   bool
	remote   net.Addr
}

func newRecordingConn() *recordingConn {
	return &recordingConn{
		remote: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 443},
	}
}

func (c *recordingConn) Read(b []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.readBuf.Len() == 0 {
		return 0, net.ErrClosed
	}
	return c.readBuf.Read(b)
}

func (c *recordingConn) Write(b []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.written.Write(b)
}

func (c *recordingConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	return nil
}

func (c *recordingConn) LocalAddr() net.Addr  { return &net.TCPAddr{IP: net.IPv4zero, Port: 0} }
func (c *recordingConn) RemoteAddr() net.Addr { return c.remote }
func (c *recordingConn) SetDeadline(time.Time) error      { return nil }
func (c *recordingConn) SetReadDeadline(time.Time) error  { return nil }
func (c *recordingConn) SetWriteDeadline(time.Time) error { return nil }

func (c *recordingConn) Written() []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]byte(nil), c.written.Bytes()...)
}

// dialerShim is a minimal dialer for tests. The production code
// uses *net.Dialer directly; the shim exists so the test can
// capture the bytes the client writes during a handshake.
type dialerShim struct {
	dial func(ctx context.Context, network, addr string) (net.Conn, error)
}

func (d *dialerShim) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return d.dial(ctx, network, addr)
}

// minimalServerHello is a 2-byte placeholder. utls's handshake
// reads this first, then proceeds to parse extensions. We don't
// need the handshake to complete — we only need the Client
// Hello bytes to be written. The handshake will fail afterwards
// (the rest of the server response is missing), which is what
// the tests assert.
var minimalServerHello = []byte{0x15, 0x03, 0x03, 0x00, 0x02, 0x02, 0x00}

func runDial(t *testing.T, profile TLSProfile) []byte {
	t.Helper()
	conn := newRecordingConn()
	conn.readBuf.Write(minimalServerHello)
	d := &dialerShim{dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
		return conn, nil
	}}
	var resolve ProfileResolver
	if profile != TLSProfileDefault {
		resolve = func() (TLSProfile, bool) { return profile, true }
	}
	// The handshake is expected to fail because we only
	// supplied a 7-byte stub. We don't care about the error;
	// we only care about the bytes the client wrote.
	_, _ = dialWithProfile(context.Background(), d, "tcp", "example.com:443", resolve)
	return conn.Written()
}

func TestDialChrome120WritesValidClientHello(t *testing.T) {
	hello := runDial(t, TLSProfileChrome)
	if len(hello) < 9 {
		t.Fatalf("Client Hello too short: %d bytes", len(hello))
	}
	// TLS record header: content_type=22 (handshake),
	// version=0x0301 (TLS 1.0 record-layer — the
	// ClientHello body carries the real version).
	if hello[0] != 0x16 {
		t.Errorf("content_type = 0x%02x, want 0x16 (handshake)", hello[0])
	}
	if hello[1] != 0x03 || hello[2] != 0x01 {
		t.Errorf("record version = %02x %02x, want 03 01", hello[1], hello[2])
	}
	// handshake type 0x01 = ClientHello, then 3-byte length.
	if hello[5] != 0x01 {
		t.Errorf("handshake type = 0x%02x, want 0x01 (ClientHello)", hello[5])
	}
	// legacy_version field inside ClientHello: TLS 1.2
	// (0x0303) for a TLS 1.3-capable client.
	if hello[9] != 0x03 || hello[10] != 0x03 {
		t.Errorf("legacy_version = %02x %02x, want 03 03 (TLS 1.2)", hello[9], hello[10])
	}
	// Chrome 120 Client Hello is ~530 bytes; assert a
	// plausible scale.
	if len(hello) < 200 {
		t.Errorf("Client Hello too small: %d bytes; Chrome 120 should be ~530+", len(hello))
	}
}

func TestDialFirefox120WritesClientHello(t *testing.T) {
	hello := runDial(t, TLSProfileFirefox)
	if len(hello) < 9 {
		t.Fatalf("Client Hello too short: %d bytes", len(hello))
	}
	if hello[0] != 0x16 || hello[5] != 0x01 {
		t.Errorf("not a ClientHello: %x %x", hello[0], hello[5])
	}
	if len(hello) < 200 {
		t.Errorf("Firefox Client Hello too small: %d bytes", len(hello))
	}
}

func TestDialChromeAndFirefoxDiffer(t *testing.T) {
	chrome := runDial(t, TLSProfileChrome)
	firefox := runDial(t, TLSProfileFirefox)
	if bytes.Equal(chrome, firefox) {
		t.Fatal("Chrome and Firefox Client Hellos are identical — utls is not differentiating")
	}
}

func TestDialDefaultProfileBypassesUtls(t *testing.T) {
	chromeHello := runDial(t, TLSProfileChrome)

	conn := newRecordingConn()
	d := &dialerShim{dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
		return conn, nil
	}}

	// nil resolver: no profile, returns rawConn directly to allow native TLS in Transport.
	c, err := dialWithProfile(context.Background(), d, "tcp", "example.com:443", nil)
	if err != nil {
		t.Fatalf("dialWithProfile error: %v", err)
	}
	if c != conn {
		t.Error("expected rawConn to be returned unchanged for default profile")
	}
	defaultHello := conn.Written()
	if len(defaultHello) != 0 {
		t.Errorf("expected 0 bytes written by utls for default profile, got %d", len(defaultHello))
	}
	if bytes.Equal(chromeHello, defaultHello) {
		t.Error("default profile wrote Chrome utls fingerprint instead of bypassing utls")
	}
}

func TestDialUnknownProfileClosesConnAndErrors(t *testing.T) {
	conn := newRecordingConn()
	conn.readBuf.Write(minimalServerHello)
	d := &dialerShim{dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
		return conn, nil
	}}

	resolve := func() (TLSProfile, bool) { return TLSProfile("safari"), true }
	dialCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := dialWithProfile(dialCtx, d, "tcp", "example.com:443", resolve)
	if err == nil {
		t.Fatal("expected error for unknown TLS profile, got nil")
	}
	conn.mu.Lock()
	closed := conn.closed
	conn.mu.Unlock()
	if !closed {
		t.Error("conn was not closed on unknown profile")
	}
}

func TestHelloIDForKnownProfiles(t *testing.T) {
	for _, p := range []TLSProfile{TLSProfileChrome, TLSProfileFirefox} {
		if _, err := helloIDFor(p); err != nil {
			t.Errorf("helloIDFor(%q) error: %v", p, err)
		}
	}
}

func TestHelloIDForUnknownErrors(t *testing.T) {
	if _, err := helloIDFor(TLSProfile("safari")); err == nil {
		t.Error("expected error for unknown profile")
	}
	if _, err := helloIDFor(TLSProfileDefault); err == nil {
		t.Error("expected error for default profile reaching utls path")
	}
}

func TestTlsProfileFromResolverNormalisesEmpty(t *testing.T) {
	got, ok := tlsProfileFromResolver(func() (TLSProfile, bool) { return "", true })
	if !ok || got != TLSProfileDefault {
		t.Errorf("empty profile = (%q, %v), want (default, true)", got, ok)
	}
}

func TestTlsProfileFromResolverNilIsFalse(t *testing.T) {
	got, ok := tlsProfileFromResolver(nil)
	if ok || got != TLSProfileDefault {
		t.Errorf("nil resolver = (%q, %v), want (default, false)", got, ok)
	}
}

func TestNewTransportReturnsUsableTransport(t *testing.T) {
	tr := NewTransport(nil)
	if tr == nil {
		t.Fatal("NewTransport returned nil")
	}
	if tr.DialContext == nil {
		t.Error("DialContext is nil")
	}
	if tr.TLSHandshakeTimeout == 0 {
		t.Error("TLSHandshakeTimeout should be set")
	}

	// Round-trip a single request through the transport
	// using a recording conn as the "server". The handshake
	// will fail because we don't speak TLS, but the dial
	// itself should succeed.
	conn := newRecordingConn()
	conn.readBuf.Write([]byte("not a real server hello"))
	tr.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return conn, nil
	}
	client := &http.Client{Transport: tr, Timeout: time.Second}
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "https://example.com/", nil)
	resp, err := client.Do(req)
	if err == nil {
		_ = resp.Body.Close()
		t.Error("expected error from a fake server, got nil")
	}
}

// newTestHTTPProxy starts a local HTTP CONNECT proxy server for testing.
func newTestHTTPProxy(t *testing.T, expectedBasicAuth string) (string, *atomic.Int32, func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start test proxy listener: %v", err)
	}
	connectCount := &atomic.Int32{}
	closed := make(chan struct{})

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				select {
				case <-closed:
					return
				default:
					return
				}
			}
			go func(c net.Conn) {
				defer c.Close()
				br := bufio.NewReader(c)
				req, err := http.ReadRequest(br)
				if err != nil {
					return
				}
				if req.Method == http.MethodConnect {
					if expectedBasicAuth != "" && req.Header.Get("Proxy-Authorization") != expectedBasicAuth {
						resp := "HTTP/1.1 407 Proxy Authentication Required\r\nProxy-Authenticate: Basic realm=\"test\"\r\nContent-Length: 0\r\n\r\n"
						_, _ = c.Write([]byte(resp))
						return
					}
					connectCount.Add(1)
					targetConn, err := net.DialTimeout("tcp", req.Host, 5*time.Second)
					if err != nil {
						resp := "HTTP/1.1 502 Bad Gateway\r\nContent-Length: 0\r\n\r\n"
						_, _ = c.Write([]byte(resp))
						return
					}
					defer targetConn.Close()
					_, _ = c.Write([]byte("HTTP/1.1 200 Connection established\r\n\r\n"))

					var wg sync.WaitGroup
					wg.Add(2)
					go func() {
						defer wg.Done()
						_, _ = io.Copy(targetConn, br)
					}()
					go func() {
						defer wg.Done()
						_, _ = io.Copy(c, targetConn)
					}()
					wg.Wait()
				} else {
					resp := "HTTP/1.1 200 OK\r\nContent-Length: 5\r\n\r\nproxy"
					_, _ = c.Write([]byte(resp))
				}
			}(conn)
		}
	}()

	cleanup := func() {
		close(closed)
		_ = ln.Close()
	}
	return ln.Addr().String(), connectCount, cleanup
}

func TestHTTPSOverProxyWithUTLSChrome(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello over chrome utls proxy"))
	}))
	defer ts.Close()

	proxyAddr, connectCount, cleanupProxy := newTestHTTPProxy(t, "")
	defer cleanupProxy()

	proxyURL, _ := url.Parse("http://" + proxyAddr)
	tr := NewTransport(func() (TLSProfile, bool) {
		return TLSProfileChrome, true
	})
	SetProxy(tr, proxyURL)
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	client := &http.Client{Transport: tr, Timeout: 5 * time.Second}
	resp, err := client.Get(ts.URL)
	if err != nil {
		t.Fatalf("request failed over proxy: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "hello over chrome utls proxy" {
		t.Errorf("unexpected body: %q", string(body))
	}
	if connectCount.Load() != 1 {
		t.Errorf("expected 1 CONNECT request, got %d", connectCount.Load())
	}
}

func TestHTTPSOverProxyWithUTLSFirefox(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello over firefox utls proxy"))
	}))
	defer ts.Close()

	proxyAddr, connectCount, cleanupProxy := newTestHTTPProxy(t, "")
	defer cleanupProxy()

	proxyURL, _ := url.Parse("http://" + proxyAddr)
	tr := NewTransport(func() (TLSProfile, bool) {
		return TLSProfileFirefox, true
	})
	SetProxy(tr, proxyURL)
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	client := &http.Client{Transport: tr, Timeout: 5 * time.Second}
	resp, err := client.Get(ts.URL)
	if err != nil {
		t.Fatalf("request failed over proxy: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "hello over firefox utls proxy" {
		t.Errorf("unexpected body: %q", string(body))
	}
	if connectCount.Load() != 1 {
		t.Errorf("expected 1 CONNECT request, got %d", connectCount.Load())
	}
}

func TestHTTPSOverProxyWithDefaultProfile(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello over native tls proxy"))
	}))
	defer ts.Close()

	proxyAddr, connectCount, cleanupProxy := newTestHTTPProxy(t, "")
	defer cleanupProxy()

	proxyURL, _ := url.Parse("http://" + proxyAddr)
	tr := NewTransport(func() (TLSProfile, bool) {
		return TLSProfileDefault, true
	})
	SetProxy(tr, proxyURL)
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	client := &http.Client{Transport: tr, Timeout: 5 * time.Second}
	resp, err := client.Get(ts.URL)
	if err != nil {
		t.Fatalf("request failed over proxy: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "hello over native tls proxy" {
		t.Errorf("unexpected body: %q", string(body))
	}
	if connectCount.Load() != 1 {
		t.Errorf("expected 1 CONNECT request, got %d", connectCount.Load())
	}
}

func TestHTTPSOverProxyWithAuthentication(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("authenticated proxy ok"))
	}))
	defer ts.Close()

	expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("user:secret123"))
	proxyAddr, connectCount, cleanupProxy := newTestHTTPProxy(t, expectedAuth)
	defer cleanupProxy()

	// 1. Request with correct credentials
	proxyURL, _ := url.Parse("http://user:secret123@" + proxyAddr)
	tr := NewTransport(func() (TLSProfile, bool) {
		return TLSProfileChrome, true
	})
	SetProxy(tr, proxyURL)
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	client := &http.Client{Transport: tr, Timeout: 5 * time.Second}
	resp, err := client.Get(ts.URL)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "authenticated proxy ok" {
		t.Errorf("unexpected body: %q", string(body))
	}
	if connectCount.Load() != 1 {
		t.Errorf("expected 1 CONNECT request, got %d", connectCount.Load())
	}

	// 2. Request with invalid credentials should fail
	badProxyURL, _ := url.Parse("http://user:wrong@" + proxyAddr)
	trBad := NewTransport(func() (TLSProfile, bool) {
		return TLSProfileChrome, true
	})
	SetProxy(trBad, badProxyURL)
	trBad.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	clientBad := &http.Client{Transport: trBad, Timeout: 5 * time.Second}
	_, badErr := clientBad.Get(ts.URL)
	if badErr == nil {
		t.Fatal("expected error due to 407 Proxy Authentication Required, got nil")
	}
	if !strings.Contains(badErr.Error(), "407") {
		t.Errorf("expected error to mention 407, got: %v", badErr)
	}
}

func TestHTTPSOverProxyConnectFailure(t *testing.T) {
	proxyAddr, _, cleanupProxy := newTestHTTPProxy(t, "")
	cleanupProxy() // immediately close proxy listener to cause connection failure

	proxyURL, _ := url.Parse("http://" + proxyAddr)
	tr := NewTransport(func() (TLSProfile, bool) {
		return TLSProfileChrome, true
	})
	SetProxy(tr, proxyURL)
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	client := &http.Client{Transport: tr, Timeout: 2 * time.Second}
	_, err := client.Get("https://example.com:443")
	if err == nil {
		t.Fatal("expected error when proxy is unreachable, got nil")
	}
}

func TestTransportRespectsTLSClientConfigInsecureSkipVerify(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer ts.Close()

	// With InsecureSkipVerify = false, self-signed test server should fail verification
	trSecure := NewTransport(func() (TLSProfile, bool) {
		return TLSProfileChrome, true
	})
	trSecure.TLSClientConfig = &tls.Config{InsecureSkipVerify: false}
	clientSecure := &http.Client{Transport: trSecure, Timeout: 5 * time.Second}
	_, err := clientSecure.Get(ts.URL)
	if err == nil {
		t.Error("expected cert verification error for self-signed cert, got nil")
	}

	// With InsecureSkipVerify = true, it should succeed
	trInsecure := NewTransport(func() (TLSProfile, bool) {
		return TLSProfileChrome, true
	})
	trInsecure.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	clientInsecure := &http.Client{Transport: trInsecure, Timeout: 5 * time.Second}
	resp, err := clientInsecure.Get(ts.URL)
	if err != nil {
		t.Fatalf("expected success with InsecureSkipVerify: true, got %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

func TestTransportRespectsTLSClientConfigCustomRootCAs(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("custom root ca ok"))
	}))
	defer ts.Close()

	certPool := x509.NewCertPool()
	certPool.AddCert(ts.Certificate())

	tr := NewTransport(func() (TLSProfile, bool) {
		return TLSProfileChrome, true
	})
	tr.TLSClientConfig = &tls.Config{
		RootCAs:            certPool,
		InsecureSkipVerify: false,
	}

	client := &http.Client{Transport: tr, Timeout: 5 * time.Second}
	resp, err := client.Get(ts.URL)
	if err != nil {
		t.Fatalf("expected success with custom RootCAs, got: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "custom root ca ok" {
		t.Errorf("unexpected body: %q", string(body))
	}
}

func TestSetProxyFuncAndProxyURL(t *testing.T) {
	u, _ := url.Parse("http://127.0.0.1:8888")
	tr := NewTransport(nil)

	SetProxy(tr, u)
	if tr.Proxy == nil {
		t.Fatal("expected tr.Proxy to be set")
	}

	// HTTPS request should yield nil proxy to route through DialTLSContext
	httpsReq, _ := http.NewRequest(http.MethodGet, "https://example.com/test", nil)
	gotURL, err := tr.Proxy(httpsReq)
	if err != nil || gotURL != nil {
		t.Errorf("expected (nil, nil) for HTTPS request, got (%v, %v)", gotURL, err)
	}

	// HTTP request should yield the configured proxy URL
	httpReq, _ := http.NewRequest(http.MethodGet, "http://example.com/test", nil)
	gotHTTPURL, err := tr.Proxy(httpReq)
	if err != nil || gotHTTPURL == nil || gotHTTPURL.String() != "http://127.0.0.1:8888" {
		t.Errorf("expected proxy URL for HTTP request, got (%v, %v)", gotHTTPURL, err)
	}

	// Clearing proxy with nil
	SetProxy(tr, nil)
	gotCleared, err := tr.Proxy(httpReq)
	if err != nil || gotCleared != nil {
		t.Errorf("expected (nil, nil) after clearing proxy, got (%v, %v)", gotCleared, err)
	}
}
