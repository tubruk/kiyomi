package http

import (
	"io"
	"net/http"
	"sync"

	"golang.org/x/time/rate"
)

// rateLimitTransport wraps an http.RoundTripper with token-bucket rate limiting and concurrency throttling.
type rateLimitTransport struct {
	base    http.RoundTripper
	limiter *rate.Limiter
	sem     chan struct{}
}

// newRateLimitTransport constructs a rate limiting and concurrency wrapper for the base RoundTripper.
func newRateLimitTransport(base http.RoundTripper, rps float64, burst int, maxConcurrent int) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}

	var limiter *rate.Limiter
	if rps > 0 {
		if burst <= 0 {
			burst = int(rps)
			if burst < 1 {
				burst = 1
			}
		}
		limiter = rate.NewLimiter(rate.Limit(rps), burst)
	}

	var sem chan struct{}
	if maxConcurrent > 0 {
		sem = make(chan struct{}, maxConcurrent)
	}

	if limiter == nil && sem == nil {
		return base
	}

	return &rateLimitTransport{
		base:    base,
		limiter: limiter,
		sem:     sem,
	}
}

func (t *rateLimitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()

	// Acquire concurrency slot if semaphore is configured
	if t.sem != nil {
		select {
		case t.sem <- struct{}{}:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// Wait for rate limiter quota if configured
	if t.limiter != nil {
		if err := t.limiter.Wait(ctx); err != nil {
			if t.sem != nil {
				<-t.sem
			}
			return nil, err
		}
	}

	resp, err := t.base.RoundTrip(req)
	if err != nil {
		if t.sem != nil {
			<-t.sem
		}
		return nil, err
	}

	if t.sem != nil {
		if resp == nil || resp.Body == nil {
			<-t.sem
		} else {
			resp.Body = &bodyWithRelease{
				ReadCloser: resp.Body,
				release: func() {
					<-t.sem
				},
			}
		}
	}

	return resp, nil
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
