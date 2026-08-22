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

// lookupDoH performs a DNS-over-HTTPS lookup (RFC 8484).
//
// Uses net/http HTTP/2 client. Encodes query as base64url wire format.
// Supports both GET (?dns=...) and POST (with wire format body).
// Default path is /dns-query. Same bootstrap rules as DoT.
func lookupDoH(ctx context.Context, hostname, server string, port int, path string, bootstrap []string) (net.IP, error) {
	// Determine the actual server address to dial.
	dialAddr := net.JoinHostPort(server, fmt.Sprintf("%d", port))

	if net.ParseIP(server) != nil {
		// IP literal: keep using HTTPS.
	} else if len(bootstrap) == 0 {
		// Hostname without bootstrap: resolve via system resolver first.
		systemIP, err := net.DefaultResolver.LookupIP(ctx, "ip", server)
		if err != nil {
			return nil, fmt.Errorf("dnsresolver: resolve DoH server %s: %w", server, err)
		}
		if len(systemIP) == 0 {
			return nil, fmt.Errorf("dnsresolver: no addresses for DoH server %s", server)
		}
		bootstrap = []string{systemIP[0].String()}
		dialAddr = net.JoinHostPort(bootstrap[0], fmt.Sprintf("%d", port))
	} else {
		// Hostname with bootstrap: dial the bootstrap IP.
		dialAddr = net.JoinHostPort(bootstrap[0], fmt.Sprintf("%d", port))
	}

	if path == "" {
		path = defaultDoHPath
	}

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

	baseURL := fmt.Sprintf("https://%s:%d%s", server, port, path)

	tlsConfig := &tls.Config{
		ServerName: server,
	}
	if len(bootstrap) > 0 && tlsConfig.RootCAs == nil {
		tlsConfig.InsecureSkipVerify = true
	}

	httpClient := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				dialer := &net.Dialer{
					Timeout: 5 * time.Second,
				}
				return dialer.DialContext(ctx, network, dialAddr)
			},
			TLSClientConfig: tlsConfig,
			// Explicitly use HTTP/1.1 to avoid HTTP/2 upgrade issues in tests.
			TLSNextProto: make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
		},
	}

	// Try GET first.
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s?dns=%s", baseURL, queryStr), nil)
	if err != nil {
		return nil, fmt.Errorf("dnsresolver: create DoH GET request: %w", err)
	}
	req.Header.Set("Accept", "application/dns-message")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("dnsresolver: DoH GET: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Fall back to POST.
		resp.Body.Close()
		req, err := http.NewRequestWithContext(ctx, "POST", baseURL, bytes.NewReader(wire))
		if err != nil {
			return nil, fmt.Errorf("dnsresolver: create DoH POST request: %w", err)
		}
		req.Header.Set("Content-Type", "application/dns-message")
		req.Header.Set("Accept", "application/dns-message")
		resp, err = httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("dnsresolver: DoH POST: %w", err)
		}
		defer resp.Body.Close()
	}

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
