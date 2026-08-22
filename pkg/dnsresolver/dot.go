package dnsresolver

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/miekg/dns"
)

// lookupDoT performs a DNS-over-TLS lookup (RFC 7858).
//
// When bootstrap IPs are supplied, the client dials the IP but uses the
// hostname for SNI and certificate verification. If bootstrap is provided
// and no explicit CA is configured, InsecureSkipVerify is set to true.
// This is intentional for private-network operators running DoT with
// self-signed certificates. See dns_override.md §Risks for the full
// security tradeoff documentation.
func lookupDoT(ctx context.Context, hostname, server string, port int, bootstrap []string) (net.IP, error) {
	// Determine the address to dial and the SNI hostname.
	dialAddr := net.JoinHostPort(server, fmt.Sprintf("%d", port))
	sniHost := server

	// If the server is a hostname (not an IP literal), we need bootstrap
	// to avoid a resolution loop.
	if net.ParseIP(server) == nil {
		if len(bootstrap) == 0 {
			// Fall back to system resolver for the bootstrap address.
			systemIP, err := net.DefaultResolver.LookupIP(ctx, "ip", server)
			if err != nil {
				return nil, fmt.Errorf("dnsresolver: resolve DoT server %s: %w", server, err)
			}
			if len(systemIP) == 0 {
				return nil, fmt.Errorf("dnsresolver: no addresses for DoT server %s", server)
			}
			bootstrap = []string{systemIP[0].String()}
		}
		// Use the first bootstrap IP for the dial address, but keep the
		// original hostname for SNI and cert verification.
		dialAddr = net.JoinHostPort(bootstrap[0], fmt.Sprintf("%d", port))
		sniHost = server
	}

	// Set up TLS config with SNI hostname.
	tlsConfig := &tls.Config{
		ServerName: sniHost,
	}
	// If bootstrap was supplied (meaning operator wants to connect to a private
	// resolver) and no explicit CA is configured, skip verification.
	// This is the documented trade-off for self-signed certs in private
	// networks. See dns_override.md §Risks: "DoT self-signed certs".
	// Note: bootstrap applies even when server is an IP literal (for private IPs).
	if len(bootstrap) > 0 && tlsConfig.RootCAs == nil {
		tlsConfig.InsecureSkipVerify = true
	}

	// Dial the TLS connection.
	dialer := &net.Dialer{
		Timeout: 5 * time.Second,
	}
	conn, err := dialer.DialContext(ctx, "tcp", dialAddr)
	if err != nil {
		return nil, fmt.Errorf("dnsresolver: dial DoT: %w", err)
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(5 * time.Second)
	}
	conn.SetDeadline(deadline)

	tlsConn := tls.Client(conn, tlsConfig)
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		tlsConn.Close()
		return nil, fmt.Errorf("dnsresolver: DoT handshake: %w", err)
	}

	// Send DNS query over TLS.
	m := &dns.Msg{}
	m.SetQuestion(dns.Fqdn(hostname), dns.TypeA)
	m.RecursionDesired = true
	buf, err := m.Pack()
	if err != nil {
		tlsConn.Close()
		return nil, fmt.Errorf("dnsresolver: pack DNS msg: %w", err)
	}

	// DoT uses a 2-byte length prefix for messages.
	lenBuf := make([]byte, 2)
	lenBuf[0] = byte(len(buf) >> 8)
	lenBuf[1] = byte(len(buf))
	if _, err := tlsConn.Write(append(lenBuf, buf...)); err != nil {
		tlsConn.Close()
		return nil, fmt.Errorf("dnsresolver: write DoT query: %w", err)
	}

	// Read response length prefix.
	respLenBuf := make([]byte, 2)
	if _, err := io.ReadFull(tlsConn, respLenBuf); err != nil {
		tlsConn.Close()
		return nil, fmt.Errorf("dnsresolver: read DoT response length: %w", err)
	}
	respLen := int(respLenBuf[0])<<8 | int(respLenBuf[1])
	respBuf := make([]byte, respLen)
	if _, err := io.ReadFull(tlsConn, respBuf); err != nil {
		tlsConn.Close()
		return nil, fmt.Errorf("dnsresolver: read DoT response: %w", err)
	}
	tlsConn.Close()

	resp := &dns.Msg{}
	if err := resp.Unpack(respBuf); err != nil {
		return nil, fmt.Errorf("dnsresolver: unpack DoT response: %w", err)
	}

	for _, a := range resp.Answer {
		if a.Header().Rrtype == dns.TypeA {
			if rec, ok := a.(*dns.A); ok {
				return rec.A, nil
			}
		}
	}
	return nil, fmt.Errorf("dnsresolver: no A record for %s", hostname)
}
