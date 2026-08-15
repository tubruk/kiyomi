package sdk

import (
	"time"
)

// ConfigKeySpec describes a single configuration key a provider accepts.
type ConfigKeySpec struct {
	Key         string   // e.g. "client_id", "api_key"
	Type        string   // "string" | "secret" | "url"
	Required    bool
	Description string
	Examples    []string
}

// ProviderConfig holds common metadata and configuration for an extension provider.
type ProviderConfig struct {
	ID          string            // Unique provider ID (e.g. "mangafox")
	Name        string            // Display name (e.g. "MangaFox")
	BaseURL     string            // Base website URL (e.g. "https://fanfox.net")
	Language    string            // ISO language code (e.g. "en")
	UserAgent   string            // Custom User-Agent string (optional)
	ClientHints *ClientHints      // Sec-Ch-Ua-* headers
	Cookies     map[string]string // Map of domain URL -> cookie header value
	RateLimit   time.Duration     // Request rate limit
	HTTPTimeout time.Duration     // Client timeout (defaults to 15s)
	TLSProfile  string            // TLS Client Hello profile ("default", "chrome", "firefox")

	// DNSResolvers, if non-empty, overrides the system DNS resolver.
	// Entries are "host:port" (port defaults to 53 when omitted).
	// Lookups are tried in order; the first answer wins.
	DNSResolvers []string

	// ProxyURL, if non-empty, routes all outbound connections through
	// the supplied proxy. Supports http://, https://, and socks5://
	// schemes. Leave empty for direct connections.
	ProxyURL string
}

type ClientHints struct {
	UA              string
	Platform        string
	Mobile          string
	PlatformVersion string
}

func DefaultClientHints() *ClientHints {
	return &ClientHints{
		UA:              `"Chromium";v="120", "Not_A Brand";v="24"`,
		Platform:        `"macOS"`,
		Mobile:          `?0`,
		PlatformVersion: `""`,
	}
}

const DefaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

// Options defines runtime configuration options for a provider, such as
// auth headers and HTTP referrer.
type Options struct {
	AuthHeaders map[string]string
	Referer     string
}

