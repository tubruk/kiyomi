// Package fingerprint provides per-source overrides for HTTP request
// fingerprinting and session cookies. It is the foundation for the
// anti-bot strategy sketched in IDEA.md: Layer 0 (per-source
// User-Agent + cookie injection) and Layer 1 (TLS Client Hello
// fingerprint spoofing via utls).
//
// The name is intentionally neutral. A request "fingerprint" is the
// combination of User-Agent string, Accept-* headers, TLS Client
// Hello, and other signals that anti-abuse services use to decide
// whether a request looks like a real browser. Letting the operator
// override that fingerprint for a given source is what this package
// does — no judgement on the legitimacy of doing so.
//
// The store is in-memory and process-local; cookies and overrides
// are lost on restart. That is honest for a self-hosted single-
// process tool — a user re-pasting a cf_clearance cookie is cheaper
// than the migration cost of a disk-backed store today. The Store
// interface exists so a SQLite-backed implementation can be swapped
// in later without changing callers.
package fingerprint

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
)

// TLSProfile selects which browser's TLS Client Hello to emulate for
// outbound HTTPS connections. "default" keeps Go's native
// crypto/tls fingerprint (lowest overhead, highest detection risk).
// Chrome and Firefox are the v1 set; more profiles can be added
// without changing the wire format.
type TLSProfile string

const (
	// TLSProfileDefault uses Go's native net/http transport. No
	// fingerprint spoofing, no extra cost. Use this for sources
	// that don't gate on TLS fingerprint.
	TLSProfileDefault TLSProfile = "default"

	// TLSProfileChrome emulates a modern Chrome Client Hello via
	// utls. Covers the majority of scraping-friendly Cloudflare
	// configurations.
	TLSProfileChrome TLSProfile = "chrome"

	// TLSProfileFirefox emulates a modern Firefox Client Hello.
	// Useful as a fallback when Cloudflare has flagged the Chrome
	// fingerprint for a given source.
	TLSProfileFirefox TLSProfile = "firefox"
)

// Valid reports whether p is one of the recognised TLS profile names.
func (p TLSProfile) Valid() bool {
	switch p {
	case TLSProfileDefault, TLSProfileChrome, TLSProfileFirefox:
		return true
	}
	return false
}

// ErrUnknownSource is returned by Store.Get / Store.Set when the
// source ID has no profile. Callers should not interpret this as a
// hard error; "no override" is the common case.
var ErrUnknownSource = errors.New("fingerprint: no profile for source")

// Profile is the per-source fingerprint override. Empty / zero fields
// fall through to the source's compiled-in defaults — the override
// is additive, not a replacement.
type Profile struct {
	// UserAgent overrides the source's compiled-in User-Agent when
	// non-empty.
	UserAgent string

	// ClientHints overrides the Sec-Ch-Ua-* headers. When nil,
	// the source's compiled-in ClientHints (or the SDK default)
	// are used. A non-nil empty struct clears every hint.
	ClientHints *ClientHints

	// Cookies maps a domain URL (e.g. "https://fanfox.net") to a
	// raw cookie header value ("cf_clearance=abc; __cf_bm=def").
	// The domain URL is used as the CookieJar key, not the
	// individual request URL.
	Cookies map[string]string

	// TLSProfile selects the Client Hello fingerprint. Empty is
	// treated as "default" (Go native).
	TLSProfile TLSProfile
}

// ClientHints is the Sec-Ch-Ua-* payload. It mirrors the shape
// used by sdk.ClientHints but lives in this package to avoid an
// import cycle (fingerprint is consumed by sdk, not the other way
// around). The SDK converts between the two at the request
// boundary.
type ClientHints struct {
	UA              string
	Platform        string
	Mobile          string
	PlatformVersion string
}

// IsZero reports whether the profile would defer entirely to the
// source's compiled-in defaults. Useful for the API layer: a zero
// profile returned from GET means "no override set", which the
// caller can render as 404 or as an empty payload. A non-nil but
// empty ClientHints is NOT zero — it is the explicit "clear all
// hints" signal.
func (p Profile) IsZero() bool {
	return p.UserAgent == "" && p.ClientHints == nil && len(p.Cookies) == 0 && p.TLSProfile == ""
}

// Store holds per-source fingerprint profiles. The interface keeps
// the in-memory implementation swappable for a future persistent one
// (SQLite, encrypted file) without changing callers.
type Store interface {
	// Get returns the profile for sourceID, or ErrUnknownSource if
	// no override has been registered.
	Get(sourceID string) (Profile, error)

	// Set replaces the profile for sourceID. A zero profile
	// deletes the override.
	Set(sourceID string, p Profile) error

	// Delete removes the override for sourceID. Missing entries
	// are not an error.
	Delete(sourceID string) error

	// IDs returns the IDs of all sources with a registered
	// override. Order is unspecified.
	IDs() []string
}

// MemoryStore is the default in-memory Store. Safe for concurrent
// use. Zero value is ready to use.
type MemoryStore struct {
	mu       sync.RWMutex
	profiles map[string]Profile
}

// NewMemoryStore returns an empty MemoryStore. Provided for
// readability at call sites; the zero value is equally usable.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{profiles: make(map[string]Profile)}
}

// Get implements Store. The returned Profile is a deep copy —
// mutating it does not affect the store.
func (m *MemoryStore) Get(sourceID string) (Profile, error) {
	sourceID = strings.TrimSpace(sourceID)
	if sourceID == "" {
		return Profile{}, errors.New("fingerprint: empty source id")
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.profiles[sourceID]
	if !ok {
		return Profile{}, ErrUnknownSource
	}
	return p.Clone(), nil
}

// Set implements Store. A zero profile deletes the entry so the
// source falls back to compiled-in defaults.
func (m *MemoryStore) Set(sourceID string, p Profile) error {
	sourceID = strings.TrimSpace(sourceID)
	if sourceID == "" {
		return errors.New("fingerprint: empty source id")
	}
	if err := p.Validate(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.profiles == nil {
		m.profiles = make(map[string]Profile)
	}
	if p.IsZero() {
		delete(m.profiles, sourceID)
		return nil
	}
	m.profiles[sourceID] = p.Clone()
	return nil
}

// Delete implements Store.
func (m *MemoryStore) Delete(sourceID string) error {
	sourceID = strings.TrimSpace(sourceID)
	if sourceID == "" {
		return errors.New("fingerprint: empty source id")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.profiles, sourceID)
	return nil
}

// IDs implements Store.
func (m *MemoryStore) IDs() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]string, 0, len(m.profiles))
	for id := range m.profiles {
		out = append(out, id)
	}
	return out
}

// Validate checks the profile for obviously-broken values: bad
// cookie domain URLs and an unrecognised TLS profile name.
func (p Profile) Validate() error {
	if p.TLSProfile != "" && !p.TLSProfile.Valid() {
		return fmt.Errorf("fingerprint: unknown tls profile %q", p.TLSProfile)
	}
	for domain := range p.Cookies {
		u, err := url.Parse(domain)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
			return fmt.Errorf("fingerprint: cookie domain %q must be an absolute http(s) URL", domain)
		}
	}
	return nil
}

// Clone returns a deep copy. Defensive copy so callers can mutate
// the returned profile without poisoning the store. ClientHints is
// itself a value type (struct of strings), so a shallow copy is
// sufficient.
func (p Profile) Clone() Profile {
	out := Profile{
		UserAgent:  p.UserAgent,
		TLSProfile: p.TLSProfile,
	}
	if p.ClientHints != nil {
		hints := *p.ClientHints
		out.ClientHints = &hints
	}
	if len(p.Cookies) > 0 {
		out.Cookies = make(map[string]string, len(p.Cookies))
		for k, v := range p.Cookies {
			out.Cookies[k] = v
		}
	}
	return out
}

// NormalizeTLSProfile maps "" to "default" so downstream code never
// has to handle the empty case. Unknown values pass through; the
// transport layer is responsible for rejecting them.
func NormalizeTLSProfile(p TLSProfile) TLSProfile {
	if p == "" {
		return TLSProfileDefault
	}
	return p
}
