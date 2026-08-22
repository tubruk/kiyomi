package dnsresolver

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/miekg/dns"
)

// lookupPlain performs a plain DNS lookup (UDP/TCP) using miekg/dns.
// Client opens UDP socket, sends query, parses response. Try TCP on truncation.
// Default port 53.
func lookupPlain(ctx context.Context, hostname, server string, port int) (net.IP, error) {
	addr := net.JoinHostPort(server, fmt.Sprintf("%d", port))

	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(hostname), dns.TypeA)
	m.RecursionDesired = true

	// Try UDP first.
	cl := &dns.Client{
		Net:     "udp",
		Timeout: 5 * time.Second,
	}
	in, _, err := cl.ExchangeContext(ctx, m, addr)
	if err != nil {
		return nil, fmt.Errorf("dnsresolver: udp exchange: %w", err)
	}
	if in != nil && in.Truncated {
		// Retry over TCP.
		cl.Net = "tcp"
		in, _, err = cl.ExchangeContext(ctx, m, addr)
		if err != nil {
			return nil, fmt.Errorf("dnsresolver: tcp exchange: %w", err)
		}
	}
	if in == nil || len(in.Answer) == 0 {
		return nil, fmt.Errorf("dnsresolver: no answer for %s", hostname)
	}
	for _, a := range in.Answer {
		if a.Header().Rrtype == dns.TypeA {
			if rec, ok := a.(*dns.A); ok {
				return rec.A, nil
			}
		}
	}
	return nil, fmt.Errorf("dnsresolver: no A record for %s", hostname)
}
