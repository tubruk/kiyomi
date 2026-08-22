package dnsresolver

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/miekg/dns"
)

// dohResolver executes DNS-over-HTTPS queries (RFC 8484) using a reusable http.Client.
type dohResolver struct {
	baseURL string
	client  *http.Client
}

// newDoHResolver creates a dohResolver with a configured http.Client and Transport
// to reuse TCP/TLS connections across lookups.
func newDoHResolver(server string, port int, path string, bootstrap []string) *dohResolver {
	if path == "" {
		path = defaultDoHPath
	}

	dialAddr := net.JoinHostPort(server, fmt.Sprintf("%d", port))
	if len(bootstrap) > 0 {
		dialAddr = net.JoinHostPort(bootstrap[0], fmt.Sprintf("%d", port))
	}

	tlsConfig := &tls.Config{
		ServerName: server,
	}
	if len(bootstrap) > 0 && tlsConfig.RootCAs == nil {
		tlsConfig.InsecureSkipVerify = true
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				dialer := &net.Dialer{
					Timeout: 5 * time.Second,
				}
				targetAddr := dialAddr
				if net.ParseIP(server) == nil && len(bootstrap) == 0 {
					systemIP, err := net.DefaultResolver.LookupIP(ctx, "ip", server)
					if err != nil {
						return nil, fmt.Errorf("dnsresolver: resolve DoH server %s: %w", server, err)
					}
					if len(systemIP) == 0 {
						return nil, fmt.Errorf("dnsresolver: no addresses for DoH server %s", server)
					}
					targetAddr = net.JoinHostPort(systemIP[0].String(), fmt.Sprintf("%d", port))
				}
				return dialer.DialContext(ctx, network, targetAddr)
			},
			TLSClientConfig: tlsConfig,
			// Explicitly use HTTP/1.1 to avoid HTTP/2 upgrade issues in tests.
			TLSNextProto: make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
		},
	}

	return &dohResolver{
		baseURL: fmt.Sprintf("https://%s:%d%s", server, port, path),
		client:  client,
	}
}

// lookup performs a DNS-over-HTTPS lookup (RFC 8484).
// Supports both GET (?dns=...) and POST (with wire format body).
func (r *dohResolver) lookup(ctx context.Context, hostname string) (net.IP, error) {
	// Build the DNS query message.
	m := &dns.Msg{}
	m.SetQuestion(dns.Fqdn(hostname), dns.TypeA)
	m.RecursionDesired = true
	wire, err := m.Pack()
	if err != nil {
		return nil, fmt.Errorf("dnsresolver: pack DoH query: %w", err)
	}

	// Encode wire format as base64url without padding.
	enc := make([]byte, base64.RawURLEncoding.EncodedLen(len(wire)))
	base64.RawURLEncoding.Encode(enc, wire)
	queryStr := string(enc)

	// Try GET first.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s?dns=%s", r.baseURL, queryStr), nil)
	if err != nil {
		return nil, fmt.Errorf("dnsresolver: create DoH GET request: %w", err)
	}
	req.Header.Set("Accept", "application/dns-message")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("dnsresolver: DoH GET: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Drain and close before retrying with POST.
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.baseURL, bytes.NewReader(wire))
		if err != nil {
			return nil, fmt.Errorf("dnsresolver: create DoH POST request: %w", err)
		}
		req.Header.Set("Content-Type", "application/dns-message")
		req.Header.Set("Accept", "application/dns-message")
		resp, err = r.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("dnsresolver: DoH POST: %w", err)
		}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("dnsresolver: read DoH response: %w", err)
	}

	dnsResp := &dns.Msg{}
	if err := dnsResp.Unpack(body); err != nil {
		return nil, fmt.Errorf("dnsresolver: unpack DoH response: %w", err)
	}

	for _, a := range dnsResp.Answer {
		if a.Header().Rrtype == dns.TypeA {
			if rec, ok := a.(*dns.A); ok {
				return rec.A, nil
			}
		}
	}
	return nil, fmt.Errorf("dnsresolver: no A record for %s", hostname)
}

// lookupDoH performs a DNS-over-HTTPS lookup (RFC 8484).
//
// Uses net/http client. Encodes query as base64url wire format.
// Supports both GET (?dns=...) and POST (with wire format body).
// Default path is /dns-query. Same bootstrap rules as DoT.
func lookupDoH(ctx context.Context, hostname, server string, port int, path string, bootstrap []string) (net.IP, error) {
	r := newDoHResolver(server, port, path, bootstrap)
	return r.lookup(ctx, hostname)
}
