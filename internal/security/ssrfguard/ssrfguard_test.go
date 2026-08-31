package ssrfguard

import (
	"net"
	"strings"
	"testing"
)

func TestValidateURL(t *testing.T) {
	cases := []struct {
		name            string
		url             string
		allowPrivate    bool
		wantErr         bool
		wantErrContains string
	}{
		// Allowed: public-ish hostnames and IPs.
		{"public hostname", "https://example.com/cover.jpg", false, false, ""},
		{"public ipv4", "https://8.8.8.8/cover.jpg", false, false, ""},
		{"public ipv6", "https://[2606:4700:4700::1111]/cover.jpg", false, false, ""},
		{"allow private when opted in", "http://127.0.0.1/x", true, false, ""},
		{"allow RFC1918 when opted in", "http://10.0.0.1/x", true, false, ""},

		// Rejected: bad scheme.
		{"file scheme", "file:///etc/passwd", false, true, "scheme"},
		{"gopher scheme", "gopher://127.0.0.1/", false, true, "scheme"},
		{"javascript scheme", "javascript:alert(1)", false, true, "scheme"},

		// Rejected: private/loopback/link-local IPv4.
		{"loopback literal", "http://127.0.0.1/x", false, true, "blocked address range"},
		{"RFC1918 10/8", "http://10.0.0.1/x", false, true, "blocked address range"},
		{"RFC1918 172.16/12", "http://172.16.0.1/x", false, true, "blocked address range"},
		{"RFC1918 192.168/16", "http://192.168.1.1/x", false, true, "blocked address range"},
		{"link-local 169.254", "http://169.254.169.254/latest/meta-data/", false, true, "blocked address range"},
		{"unspecified", "http://0.0.0.0/x", false, true, "blocked address range"},
		{"cgnat", "http://100.64.0.1/test", false, true, "blocked address range"},
		{"rfc 1122", "http://0.0.0.1/test", false, true, "blocked address range"},

		// Rejected: IPv6 special-purpose ranges.
		{"ipv6 loopback", "http://[::1]/x", false, true, "blocked address range"},
		{"ipv6 unspecified", "http://[::]/x", false, true, "blocked address range"},
		{"ipv6 link-local", "http://[fe80::1]/x", false, true, "blocked address range"},
		{"ipv4 compatible loopback", "http://[::127.0.0.1]", false, true, "blocked address range"},
		{"ipv4 compatible link-local", "http://[::169.254.169.254]", false, true, "blocked address range"},

		// Rejected: missing host or bad URL.
		{"empty host", "http:///path", false, true, "host"},
		{"unparseable", "://not-a-url", false, true, "invalid URL"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			u, err := ValidateURL(tc.url, tc.allowPrivate)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (url=%s)", tc.url)
				}
				if tc.wantErrContains != "" && !strings.Contains(err.Error(), tc.wantErrContains) {
					t.Errorf("expected error to contain %q, got %q", tc.wantErrContains, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if u == nil {
				t.Fatalf("expected non-nil url on success")
			}
		})
	}
}

func TestIsBlockedIP(t *testing.T) {
	cases := []struct {
		ip   string
		want bool
	}{
		{"127.0.0.1", true},
		{"10.1.2.3", true},
		{"172.16.0.1", true},
		{"192.168.0.1", true},
		{"169.254.1.1", true},
		{"0.0.0.0", true},
		{"0.0.0.1", true},
		{"100.64.0.1", true},
		{"224.0.0.1", true}, // multicast
		{"::1", true},
		{"::127.0.0.1", true},
		{"::169.254.169.254", true},
		{"fe80::1", true},
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		{"2606:4700:4700::1111", false},
	}
	for _, tc := range cases {
		t.Run(tc.ip, func(t *testing.T) {
			ip := net.ParseIP(tc.ip)
			if ip == nil {
				t.Fatalf("invalid IP: %s", tc.ip)
			}
			got := isBlockedIP(ip)
			if got != tc.want {
				t.Errorf("isBlockedIP(%s) = %v, want %v", tc.ip, got, tc.want)
			}
		})
	}
}

func TestIsBlockedIP_NilIsBlocked(t *testing.T) {
	if !isBlockedIP(nil) {
		t.Errorf("expected isBlockedIP(nil) == true")
	}
}
