// Package ssrfguard validates outbound URLs to prevent server-side request
// forgery (SSRF). It rejects URLs that target private, loopback, link-local,
// multicast, or unspecified address ranges unless the caller explicitly opts
// in via allowPrivateNetworks.
//
// ValidateURL resolves the host once via the standard library resolver and
// inspects every returned IP, so a literal private address (e.g.
// "http://127.0.0.1/x") is rejected just like a hostname that resolves to a
// private address.
package ssrfguard

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
)

// resolverTimeout caps the DNS lookup used by ValidateURL. The check runs
// before we hand the URL to the HTTP client, so a slow/unreachable resolver
// should fail fast rather than block a job worker.
const resolverTimeout = 5 * time.Second

var (
	_, cgnatNet, _   = net.ParseCIDR("100.64.0.0/10")
	_, rfc1122Net, _ = net.ParseCIDR("0.0.0.0/8")
)

// ValidateURL parses rawURL, enforces an http or https scheme, and confirms
// the host does not resolve into a blocked address range. When
// allowPrivateNetworks is true (intended for local development), the
// address-range checks are skipped but the URL must still parse and use
// http/https. The parsed URL is returned so callers don't re-parse.
//
// DNS rebinding is mitigated by resolving the host up front and rejecting any
// returned IP that is private or loopback; the caller then dials that
// hostname. Callers that need stricter rebinding guarantees can wrap their
// http.Transport with a custom DialContext that pins the resolved IP, but the
// upfront check blocks the common cases.
func ValidateURL(rawURL string, allowPrivateNetworks bool) (*url.URL, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return nil, fmt.Errorf("scheme %q not allowed (must be http or https)", scheme)
	}
	host := u.Hostname()
	if host == "" {
		return nil, fmt.Errorf("missing host")
	}

	if allowPrivateNetworks {
		return u, nil
	}

	// Hostname may already be an IP literal.
	if ip := net.ParseIP(host); ip != nil {
		if isBlockedIP(ip) {
			return nil, fmt.Errorf("host %s is in a blocked address range", host)
		}
		return u, nil
	}

	// Resolve the hostname and check every returned address.
	ctx, cancel := context.WithTimeout(context.Background(), resolverTimeout)
	defer cancel()
	ips, err := (&net.Resolver{}).LookupIP(ctx, "ip", host)
	if err != nil {
		return nil, fmt.Errorf("resolve host %s: %w", host, err)
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("host %s has no addresses", host)
	}
	for _, ip := range ips {
		if isBlockedIP(ip) {
			return nil, fmt.Errorf("host %s resolves to %s which is in a blocked address range", host, ip)
		}
	}
	return u, nil
}

// isBlockedIP reports whether ip falls into any address range that should
// not be reachable from this server's outbound HTTP client: RFC1918 private
// space, loopback, link-local unicast/multicast, multicast, interface-local
// multicast, unspecified, CGNAT (RFC 6598), or RFC 1122 ("this network").
func isBlockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if len(ip) == 16 {
		isCompat := true
		for i := 0; i < 12; i++ {
			if ip[i] != 0 {
				isCompat = false
				break
			}
		}
		if isCompat {
			ip = ip[12:16]
		}
	}
	if ip.IsUnspecified() {
		return true
	}
	if ip.IsLoopback() {
		return true
	}
	if ip.IsPrivate() {
		return true
	}
	if ip.IsLinkLocalUnicast() {
		return true
	}
	if ip.IsLinkLocalMulticast() {
		return true
	}
	if ip.IsMulticast() {
		return true
	}
	if ip.IsInterfaceLocalMulticast() {
		return true
	}
	if cgnatNet != nil && cgnatNet.Contains(ip) {
		return true
	}
	if rfc1122Net != nil && rfc1122Net.Contains(ip) {
		return true
	}
	return false
}
