package fingerprint

import (
	"errors"
	"strings"
	"sync"
	"testing"
)

func TestTLSProfileValid(t *testing.T) {
	for _, p := range []TLSProfile{TLSProfileDefault, TLSProfileChrome, TLSProfileFirefox} {
		if !p.Valid() {
			t.Errorf("expected %q to be valid", p)
		}
	}
	for _, p := range []TLSProfile{"", "edge", "safari", "CHROME"} {
		if p.Valid() {
			t.Errorf("expected %q to be invalid", p)
		}
	}
}

func TestNormalizeTLSProfile(t *testing.T) {
	if got := NormalizeTLSProfile(""); got != TLSProfileDefault {
		t.Errorf("normalize empty = %q, want %q", got, TLSProfileDefault)
	}
	if got := NormalizeTLSProfile(TLSProfileChrome); got != TLSProfileChrome {
		t.Errorf("normalize chrome = %q, want %q", got, TLSProfileChrome)
	}
}

func TestProfileIsZero(t *testing.T) {
	if !(Profile{}).IsZero() {
		t.Error("zero profile must be zero")
	}
	if (Profile{UserAgent: "x"}).IsZero() {
		t.Error("profile with UA must not be zero")
	}
	if (Profile{TLSProfile: TLSProfileChrome}).IsZero() {
		t.Error("profile with TLS profile must not be zero")
	}
	if (Profile{Cookies: map[string]string{"x": "y"}}).IsZero() {
		t.Error("profile with cookies must not be zero")
	}
}

func TestProfileValidate(t *testing.T) {
	if err := (Profile{}).Validate(); err != nil {
		t.Errorf("empty profile should validate: %v", err)
	}
	if err := (Profile{TLSProfile: "safari"}).Validate(); err == nil {
		t.Error("unknown TLS profile must fail validation")
	}
	if err := (Profile{Cookies: map[string]string{"not-a-url": "x=1"}}).Validate(); err == nil {
		t.Error("invalid cookie domain must fail validation")
	}
	if err := (Profile{Cookies: map[string]string{"ftp://foo": "x=1"}}).Validate(); err == nil {
		t.Error("non-http cookie domain must fail validation")
	}
	if err := (Profile{Cookies: map[string]string{"https://fanfox.net": "cf_clearance=abc"}}).Validate(); err != nil {
		t.Errorf("valid cookie domain should pass: %v", err)
	}
}

func TestProfileCloneIsIndependent(t *testing.T) {
	p := Profile{
		UserAgent:  "x",
		Cookies:    map[string]string{"https://a": "k=v"},
		TLSProfile: TLSProfileChrome,
	}
	c := p.Clone()
	c.Cookies["https://a"] = "mutated"
	c.UserAgent = "y"
	if p.UserAgent != "x" {
		t.Errorf("clone leaked UA back to original: %q", p.UserAgent)
	}
	if p.Cookies["https://a"] != "k=v" {
		t.Errorf("clone leaked cookies back to original: %q", p.Cookies["https://a"])
	}
}

func TestMemoryStoreGetSetDelete(t *testing.T) {
	s := NewMemoryStore()

	if _, err := s.Get("missing"); !errors.Is(err, ErrUnknownSource) {
		t.Errorf("Get missing = %v, want ErrUnknownSource", err)
	}

	want := Profile{UserAgent: "ua", TLSProfile: TLSProfileChrome}
	if err := s.Set("src", want); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get("src")
	if err != nil {
		t.Fatal(err)
	}
	if got.UserAgent != "ua" || got.TLSProfile != TLSProfileChrome {
		t.Errorf("Get = %+v, want %+v", got, want)
	}

	ids := s.IDs()
	if len(ids) != 1 || ids[0] != "src" {
		t.Errorf("IDs = %v, want [src]", ids)
	}

	if err := s.Delete("src"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get("src"); !errors.Is(err, ErrUnknownSource) {
		t.Errorf("Get after Delete = %v, want ErrUnknownSource", err)
	}
}

func TestMemoryStoreSetZeroProfileDeletes(t *testing.T) {
	s := NewMemoryStore()
	if err := s.Set("src", Profile{UserAgent: "ua"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Set("src", Profile{}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get("src"); !errors.Is(err, ErrUnknownSource) {
		t.Errorf("Get after zero-Set = %v, want ErrUnknownSource", err)
	}
	if len(s.IDs()) != 0 {
		t.Errorf("IDs after zero-Set = %v, want empty", s.IDs())
	}
}

func TestMemoryStoreValidationRejectsBadInput(t *testing.T) {
	s := NewMemoryStore()
	if err := s.Set("", Profile{UserAgent: "x"}); err == nil {
		t.Error("empty source id should be rejected")
	}
	if err := s.Set("src", Profile{TLSProfile: "edge"}); err == nil {
		t.Error("unknown TLS profile should be rejected")
	}
}

func TestMemoryStoreGetReturnsClone(t *testing.T) {
	s := NewMemoryStore()
	if err := s.Set("src", Profile{Cookies: map[string]string{"https://a": "k=v"}}); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get("src")
	if err != nil {
		t.Fatal(err)
	}
	got.Cookies["https://a"] = "mutated"
	got2, _ := s.Get("src")
	if got2.Cookies["https://a"] != "k=v" {
		t.Errorf("Get returned a shared map; mutation leaked: %q", got2.Cookies["https://a"])
	}
}

func TestMemoryStoreConcurrentSafe(t *testing.T) {
	s := NewMemoryStore()
	var wg sync.WaitGroup
	const n = 50
	for i := range n {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			_ = s.Set("src", Profile{UserAgent: "u"})
		}(i)
		go func() {
			defer wg.Done()
			_, _ = s.Get("src")
			_ = s.IDs()
		}()
	}
	wg.Wait()
}

func TestMemoryStoreDeleteUnknownIsNoop(t *testing.T) {
	s := NewMemoryStore()
	if err := s.Delete("never-set"); err != nil {
		t.Errorf("Delete unknown = %v, want nil", err)
	}
}

func TestMemoryStoreEmptySourceIDRejected(t *testing.T) {
	s := NewMemoryStore()
	for name, fn := range map[string]func() error{
		"Get":    func() error { _, err := s.Get(""); return err },
		"Set":    func() error { return s.Set("", Profile{}) },
		"Delete": func() error { return s.Delete("") },
	} {
		t.Run(name, func(t *testing.T) {
			err := fn()
			if err == nil {
				t.Fatalf("%s with empty id should error", name)
			}
			if !strings.Contains(err.Error(), "empty") {
				t.Errorf("%s error = %v, want 'empty' in message", name, err)
			}
		})
	}
}

func TestProfileIsZeroWithClientHints(t *testing.T) {
	// A non-nil but empty ClientHints is NOT zero — it's the
	// explicit "clear all hints" signal.
	p := Profile{ClientHints: &ClientHints{}}
	if p.IsZero() {
		t.Error("non-nil empty ClientHints must not be zero")
	}
}

func TestProfileClonePreservesClientHints(t *testing.T) {
	p := Profile{
		UserAgent: "ua",
		ClientHints: &ClientHints{
			UA:       `"X";v="1"`,
			Platform: `"Linux"`,
			Mobile:   `?0`,
		},
		Cookies: map[string]string{"https://a": "k=v"},
	}
	c := p.Clone()
	// Mutate the clone's ClientHints; original must not change.
	c.ClientHints.UA = "mutated"
	if p.ClientHints.UA != `"X";v="1"` {
		t.Errorf("ClientHints.UA leaked to original: %q", p.ClientHints.UA)
	}
	// Mutate clone's cookies; original must not change.
	c.Cookies["https://a"] = "mutated"
	if p.Cookies["https://a"] != "k=v" {
		t.Errorf("Cookies leaked to original: %q", p.Cookies["https://a"])
	}
}

func TestMemoryStoreRoundTripsClientHints(t *testing.T) {
	s := NewMemoryStore()
	if err := s.Set("src", Profile{
		ClientHints: &ClientHints{UA: `"RoundTrip";v="1"`},
	}); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get("src")
	if err != nil {
		t.Fatal(err)
	}
	if got.ClientHints == nil || got.ClientHints.UA != `"RoundTrip";v="1"` {
		t.Errorf("ClientHints did not round-trip: %+v", got.ClientHints)
	}
}

func TestMemoryStoreValidationAcceptsClientHints(t *testing.T) {
	s := NewMemoryStore()
	// ClientHints has no validation today — all string fields are
	// valid as-is. The store must not reject a profile that sets
	// only ClientHints.
	if err := s.Set("src", Profile{ClientHints: &ClientHints{UA: "x"}}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
