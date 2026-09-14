package http

import (
	"net/http"
	"unsafe"

	"github.com/chickenzord/go-brisk"
	utls "github.com/refraction-networking/utls"
	"github.com/tubruk/kiyomi/plugin-sdk/internal/dnsresolver"
)

// TLSProfile selects which browser's TLS Client Hello to emulate for outbound HTTPS connections.
type TLSProfile string

const (
	// TLSProfileDefault uses Go's native net/http transport.
	TLSProfileDefault TLSProfile = "default"
	// TLSProfileChrome emulates a modern Chrome Client Hello via utls.
	TLSProfileChrome TLSProfile = "chrome"
	// TLSProfileFirefox emulates a modern Firefox Client Hello via utls.
	TLSProfileFirefox TLSProfile = "firefox"
	// TLSProfileSafari emulates a modern Safari Client Hello via utls.
	TLSProfileSafari TLSProfile = "safari"
	// TLSProfileEdge emulates a modern Edge Client Hello via utls.
	TLSProfileEdge TLSProfile = "edge"
)

// Valid reports whether p is one of the recognized TLS profile names.
func (p TLSProfile) Valid() bool {
	switch p {
	case TLSProfileDefault, TLSProfileChrome, TLSProfileFirefox, TLSProfileSafari, TLSProfileEdge:
		return true
	}
	return false
}

// ClientHints holds Sec-Ch-Ua-* headers to emulate browser client hints.
type ClientHints struct {
	UA              string `json:"ua,omitempty"`
	Platform        string `json:"platform,omitempty"`
	Mobile          string `json:"mobile,omitempty"`
	PlatformVersion string `json:"platformVersion,omitempty"`
}

// DefaultClientHints returns standard Chrome client hints.
func DefaultClientHints() ClientHints {
	return ClientHints{
		UA:       `"Google Chrome";v="120", "Chromium";v="120", "Not?A_Brand";v="24"`,
		Platform: `"Windows"`,
		Mobile:   "?0",
	}
}

// DefaultUserAgent is the standard browser User-Agent fallback.
const DefaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

func helloIDForProfile(p TLSProfile) (utls.ClientHelloID, bool) {
	switch p {
	case TLSProfileChrome:
		return utls.HelloChrome_120, true
	case TLSProfileFirefox:
		return utls.HelloFirefox_120, true
	case TLSProfileSafari:
		return utls.HelloSafari_16_0, true
	case TLSProfileEdge:
		return utls.HelloEdge_106, true
	default:
		return utls.ClientHelloID{}, false
	}
}

// briskProfileForProfile maps an SDK TLSProfile enum to the corresponding brisk.TLSProfile.
func briskProfileForProfile(p TLSProfile) brisk.TLSProfile {
	switch p {
	case TLSProfileChrome:
		return brisk.TLSProfileChrome120
	case TLSProfileFirefox:
		return brisk.TLSProfileFirefox120
	case TLSProfileSafari:
		return brisk.TLSProfileSafari16
	case TLSProfileEdge:
		return brisk.TLSProfileEdge106
	default:
		return brisk.DefaultTLSProfile
	}
}

// toBriskProfile converts a utls.ClientHelloID to a brisk.TLSProfile.
func toBriskProfile(id utls.ClientHelloID) brisk.TLSProfile {
	switch id.Client {
	case utls.HelloChrome_120.Client:
		return brisk.TLSProfileChrome120
	case utls.HelloChrome_102.Client:
		return brisk.TLSProfileChrome102
	case utls.HelloFirefox_120.Client:
		return brisk.TLSProfileFirefox120
	case utls.HelloFirefox_105.Client:
		return brisk.TLSProfileFirefox105
	case utls.HelloSafari_16_0.Client:
		return brisk.TLSProfileSafari16
	case utls.HelloEdge_106.Client:
		return brisk.TLSProfileEdge106
	default:
		return *(*brisk.TLSProfile)(unsafe.Pointer(&id))
	}
}

// buildBaseTransport constructs the core transport powered by brisk.
func buildBaseTransport(cfg *clientConfig) http.RoundTripper {
	b := brisk.NewBuilder()
	if cfg.proxyURL != "" {
		b.WithProxy(cfg.proxyURL)
	}

	if cfg.customDialContext != nil {
		b.WithDialContext(cfg.customDialContext)
	} else if len(cfg.dnsResolvers) > 0 {
		dialFn, err := dnsresolver.DialFuncFromURLs(cfg.dnsResolvers)
		if err == nil && dialFn != nil {
			b.WithDialContext(dialFn)
		}
	}

	if cfg.hasCustomHelloID {
		b.WithTLSProfile(toBriskProfile(cfg.utlsHelloID))
	} else if cfg.randomTLSProfile {
		b.WithRandomTLSProfile()
	} else if cfg.tlsProfile != "" && cfg.tlsProfile != TLSProfileDefault {
		b.WithTLSProfile(briskProfileForProfile(cfg.tlsProfile))
	}

	tr, err := b.BuildTransport()
	if err != nil {
		return http.DefaultTransport
	}
	return tr
}

// headerTransport applies default headers, User-Agent, and Sec-Ch-Ua client hints to outgoing requests.
type headerTransport struct {
	base    http.RoundTripper
	headers map[string]string
}

func (t *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clonedReq := req.Clone(req.Context())
	for k, v := range t.headers {
		if clonedReq.Header.Get(k) == "" {
			clonedReq.Header.Set(k, v)
		}
	}
	return t.base.RoundTrip(clonedReq)
}

func hasCookie(req *http.Request, name string) bool {
	for _, c := range req.Cookies() {
		if c.Name == name {
			return true
		}
	}
	return false
}
