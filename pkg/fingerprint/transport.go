package fingerprint

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	utls "github.com/refraction-networking/utls"
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
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
		DialContext:           d.DialContext,
	}
	t.DialTLSContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		dialerToUse := dialer(d)
		if t.DialContext != nil {
			dialerToUse = dialerFunc(t.DialContext)
		}
		conn, err := dialWithProfile(ctx, dialerToUse, network, addr, resolve)
		if err != nil {
			return nil, err
		}
		if _, ok := conn.(*utls.UConn); ok {
			return conn, nil
		}
		host, _, splitErr := net.SplitHostPort(addr)
		if splitErr != nil {
			host = addr
		}
		tlsConfig := &tls.Config{
			ServerName: host,
			NextProtos: []string{"h2", "http/1.1"},
		}
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
func dialWithProfile(ctx context.Context, d dialer, network, addr string, resolve ProfileResolver) (net.Conn, error) {
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

	tlsConfig := &utls.Config{ServerName: host}
	uConn := utls.UClient(rawConn, tlsConfig, helloID)
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
