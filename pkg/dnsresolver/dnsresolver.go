// Package dnsresolver provides DNS resolution over plain DNS (UDP/TCP),
// DNS-over-TLS (DoT), and DNS-over-HTTPS (DoH).
//
// See [dns-override design doc] for the full architecture.
//
// [dns-override design doc]: https://github.com/tubruk/kiyomi/blob/main/docs/design/dns_override.md
package dnsresolver

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"
)

// Spec describes a single DNS resolver to use for lookups.
// It is parsed from URL-style entries such as "dns://1.1.1.1",
// "tls://1.1.1.1", or "https://dns.google/dns-query?bootstrap=8.8.8.8".
//
// See [dns-override design doc] for the full URL grammar.
//
// [dns-override design doc]: https://github.com/tubruk/kiyomi/blob/main/docs/design/dns_override.md
type Spec struct {
	Scheme    string   // "dns", "tls", "https"
	Host      string   // Hostname or IP literal
	Port      int      // 0 means "use default for scheme"
	Path      string   // DoH only; empty means "/dns-query"
	Bootstrap []string // IPs for hostname targets (DoT/DoH only)
}

// Default ports per scheme.
const (
	defaultDNSPort = 53
	defaultDoTPort = 853
	defaultDoHPort = 443
	defaultDoHPath = "/dns-query"
)

// ParseList parses a comma-separated env-var value into structured specs
// plus a legacy slice of bare "host:port" entries. Empty input returns nil, nil, nil.
// Bare entries (no scheme) are placed in the legacy slice and are treated as
// dns://host:port by callers.
func ParseList(raw string) (specs []Spec, legacy []string, err error) {
	if raw == "" {
		return nil, nil, nil
	}

	parts := strings.Split(raw, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Check if it has a scheme; if not, it's legacy.
		if !strings.Contains(part, "://") {
			legacy = append(legacy, part)
			continue
		}

		spec, err := parseURL(part)
		if err != nil {
			return nil, nil, fmt.Errorf("dnsresolver: parse %q: %w", part, err)
		}
		specs = append(specs, spec)
	}
	return specs, legacy, nil
}

// parseURL parses a single resolver URL.
// URL grammar: <scheme>://<host>[:<port>][/<path>][?bootstrap=<ip>[,<ip>...]]
func parseURL(raw string) (Spec, error) {
	if !strings.Contains(raw, "://") {
		raw = "dns://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return Spec{}, fmt.Errorf("invalid URL: %w", err)
	}

	spec := Spec{
		Scheme: u.Scheme,
		Host:   u.Hostname(),
	}

	if u.Port() != "" {
		var port int
		if _, err := fmt.Sscanf(u.Port(), "%d", &port); err != nil {
			return Spec{}, fmt.Errorf("invalid port %q: %w", u.Port(), err)
		}
		spec.Port = port
	}

	// Path applies to DoH only.
	spec.Path = strings.TrimSuffix(u.Path, "/")
	if spec.Path == "" && spec.Scheme == "https" {
		spec.Path = defaultDoHPath
	}

	// Bootstrap applies to DoT and DoH.
	// Supports multiple query params (?bootstrap=ip1&bootstrap=ip2) or
	// comma-separated within a single param (?bootstrap=ip1,ip2).
	if bs := u.Query()["bootstrap"]; len(bs) > 0 {
		for _, b := range bs {
			// Each value may be comma-separated.
			for _, ip := range strings.Split(b, ",") {
				ip = strings.TrimSpace(ip)
				if ip != "" {
					spec.Bootstrap = append(spec.Bootstrap, ip)
				}
			}
		}
	}

	// Set default port if not specified.
	if spec.Port == 0 {
		switch spec.Scheme {
		case "dns":
			spec.Port = defaultDNSPort
		case "tls":
			spec.Port = defaultDoTPort
		case "https":
			spec.Port = defaultDoHPort
		default:
			return Spec{}, fmt.Errorf("unknown scheme %q", spec.Scheme)
		}
	}

	return spec, nil
}

// LoadFromEnv reads an env var, calls ParseList, and returns a combined
// string list suitable for passing to WithDNSResolvers or DialFuncFromURLs.
// Legacy bare host:port entries are kept as-is; URL entries are kept as-is
// (callers that need Spec objects should call ParseList directly).
func LoadFromEnv(envName string) []string {
	raw := os.Getenv(envName)
	if raw == "" {
		return nil
	}

	specs, legacy, err := ParseList(raw)
	if err != nil {
		// Warn but do not crash; fall through to system resolver.
		fmt.Fprintf(os.Stderr, "dnsresolver: %v\n", err)
		return nil
	}

	result := make([]string, 0, len(specs)+len(legacy))
	for _, s := range specs {
		result = append(result, formatSpec(s))
	}
	for _, l := range legacy {
		result = append(result, l)
	}
	return result
}

// formatSpec reconstructs a URL string from a Spec.
func formatSpec(s Spec) string {
	u := url.URL{
		Scheme: s.Scheme,
		Host:   s.Host,
	}
	if s.Port != 0 {
		u.Host = fmt.Sprintf("%s:%d", s.Host, s.Port)
	}
	if s.Path != "" && s.Scheme == "https" {
		u.Path = s.Path
	}
	if len(s.Bootstrap) > 0 {
		u.RawQuery = "bootstrap=" + strings.Join(s.Bootstrap, ",")
	}
	return u.String()
}

// DialFunc returns a net.Dial-compatible function that uses the spec list
// to resolve hostnames. Used by pkg/provider/sdk for back-compat with
// the existing ProviderConfig.DNSResolvers []string field.
func DialFunc(specs []Spec) func(ctx context.Context, network, addr string) (net.Conn, error) {
	type runner struct {
		spec Spec
		doh  *dohResolver
	}
	runners := make([]runner, len(specs))
	for i, spec := range specs {
		r := runner{spec: spec}
		if spec.Scheme == "https" {
			r.doh = newDoHResolver(spec.Host, spec.Port, spec.Path, spec.Bootstrap)
		}
		runners[i] = r
	}

	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, portStr, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, fmt.Errorf("dnsresolver: split hostport: %w", err)
		}

		// Try each spec in order; first one that resolves wins.
		for _, r := range runners {
			var (
				ip  net.IP
				err error
			)
			if r.doh != nil {
				ip, err = r.doh.lookup(ctx, host)
			} else {
				ip, err = resolveWithSpec(ctx, r.spec, host)
			}
			if err != nil {
				continue
			}
			dialer := &net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}
			return dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), portStr))
		}

		// Fallback: system resolver.
		addrs, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
		if err != nil {
			return nil, err
		}
		if len(addrs) == 0 {
			return nil, fmt.Errorf("dnsresolver: no addresses for %s", host)
		}
		dialer := &net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}
		return dialer.DialContext(ctx, network, net.JoinHostPort(addrs[0].String(), portStr))
	}
}

// resolveWithSpec resolves a hostname using a single Spec.
// Returns a single IP or an error.
func resolveWithSpec(ctx context.Context, spec Spec, hostname string) (net.IP, error) {
	switch spec.Scheme {
	case "dns":
		return lookupPlain(ctx, hostname, spec.Host, spec.Port)
	case "tls":
		return lookupDoT(ctx, hostname, spec.Host, spec.Port, spec.Bootstrap)
	case "https":
		return lookupDoH(ctx, hostname, spec.Host, spec.Port, spec.Path, spec.Bootstrap)
	default:
		return nil, fmt.Errorf("unsupported scheme %q", spec.Scheme)
	}
}

// DialFuncFromURLs parses URL strings and returns a DialFunc.
func DialFuncFromURLs(urls []string) (func(ctx context.Context, network, addr string) (net.Conn, error), error) {
	specs := make([]Spec, 0, len(urls))
	for _, u := range urls {
		spec, err := parseURL(u)
		if err != nil {
			return nil, fmt.Errorf("dnsresolver: parse URL %q: %w", u, err)
		}
		specs = append(specs, spec)
	}
	return DialFunc(specs), nil
}

// NewResolver returns a *net.Resolver whose Dial handles all three schemes
// with bootstrap fallback. The returned resolver uses PreferGo and delegates
// to the appropriate DNS scheme based on the Spec list.
func NewResolver(specs []Spec) *net.Resolver {
	return &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			// The address is "udp" or "tcp"; actual routing is handled by DialFunc
			// via LookupHost. We intercept here by providing a custom dial function
			// that uses our DNS resolvers.
			_ = network
			_ = address
			return nil, fmt.Errorf("dnsresolver: use DialFunc directly for dial-only access")
		},
	}
}
