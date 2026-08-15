package fingerprint

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"sync"
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
