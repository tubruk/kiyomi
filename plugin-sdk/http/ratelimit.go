package http

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/chickenzord/go-brisk"
)

// roundTripperFunc provides an adapter to allow the use of ordinary functions as http.RoundTrippers.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// concurrencyMiddleware wraps an http.RoundTripper with concurrency throttling and body-close release semantics.
func concurrencyMiddleware(maxConcurrent int) brisk.Middleware {
	if maxConcurrent <= 0 {
		return nil
	}
	sem := make(chan struct{}, maxConcurrent)
	return func(next http.RoundTripper) http.RoundTripper {
		return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			ctx := req.Context()

			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return nil, ctx.Err()
			}

			resp, err := next.RoundTrip(req)
			if err != nil {
				<-sem
				return nil, err
			}

			if resp == nil || resp.Body == nil {
				<-sem
				return resp, nil
			}

			resp.Body = &bodyWithRelease{
				ReadCloser: resp.Body,
				release: func() {
					<-sem
				},
			}
			return resp, nil
		})
	}
}

// cookieMiddleware ensures configured cookies are mapped to the jar and present on outgoing requests.
func cookieMiddleware(jar http.CookieJar, cookies map[string]string) brisk.Middleware {
	if len(cookies) == 0 || jar == nil {
		return nil
	}
	return func(next http.RoundTripper) http.RoundTripper {
		return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL != nil {
				for domainURL, rawHeader := range cookies {
					u, err := url.Parse(domainURL)
					if err != nil || u.Host == "" {
						continue
					}
					parts := strings.Split(rawHeader, ";")
					var jarCookies []*http.Cookie
					for _, part := range parts {
						part = strings.TrimSpace(part)
						if part == "" {
							continue
						}
						kv := strings.SplitN(part, "=", 2)
						if len(kv) == 2 {
							jarCookies = append(jarCookies, &http.Cookie{
								Name:  strings.TrimSpace(kv[0]),
								Value: strings.TrimSpace(kv[1]),
								Path:  "/",
							})
						}
					}
					if len(jarCookies) > 0 {
						jar.SetCookies(u, jarCookies)
					}
				}
				for _, c := range jar.Cookies(req.URL) {
					if !hasCookie(req, c.Name) {
						req.AddCookie(c)
					}
				}
			}
			return next.RoundTrip(req)
		})
	}
}

// bodyWithRelease wraps an io.ReadCloser and executes a release function exactly once when closed.
type bodyWithRelease struct {
	io.ReadCloser
	release func()
	once    sync.Once
}

func (b *bodyWithRelease) Close() error {
	var err error
	if b.ReadCloser != nil {
		err = b.ReadCloser.Close()
	}
	b.once.Do(b.release)
	return err
}

// newRateLimitTransport constructs a rate limiting and concurrency wrapper for the base RoundTripper.
// Retained for backward compatibility with existing callers.
func newRateLimitTransport(base http.RoundTripper, rps float64, burst int, maxConcurrent int) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}

	rt := base
	if rps > 0 {
		if burst <= 0 {
			burst = int(rps)
			if burst < 1 {
				burst = 1
			}
		}
		tb := brisk.NewTokenBucket(rps, burst)
		rt = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if err := tb.Wait(req.Context()); err != nil {
				return nil, err
			}
			return rt.RoundTrip(req)
		})
	}

	if maxConcurrent > 0 {
		mw := concurrencyMiddleware(maxConcurrent)
		rt = mw(rt)
	}

	return rt
}
