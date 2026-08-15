package sdk

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/tubruk/kiyomi/pkg/fingerprint"
)

type failTransport struct {
	attempts int32
	failWith error
	status   int
}

func (f *failTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	count := atomic.AddInt32(&f.attempts, 1)
	if count == 1 && f.failWith != nil {
		return nil, f.failWith
	}
	if count == 1 && f.status != 0 {
		return &http.Response{StatusCode: f.status, Body: http.NoBody}, nil
	}
	return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
}

func TestIsTransientError(t *testing.T) {
	if !IsTransientError(io.EOF, nil) {
		t.Error("expected io.EOF to be transient")
	}
	if !IsTransientError(net.ErrClosed, nil) {
		t.Error("expected net.ErrClosed to be transient")
	}
	if !IsTransientError(errors.New("connection reset by peer"), nil) {
		t.Error("expected connection reset to be transient")
	}

	resp502 := &http.Response{StatusCode: http.StatusBadGateway}
	if !IsTransientError(nil, resp502) {
		t.Error("expected 502 Bad Gateway to be transient")
	}

	resp404 := &http.Response{StatusCode: http.StatusNotFound}
	if IsTransientError(nil, resp404) {
		t.Error("expected 404 Not Found NOT to be transient")
	}
}

func TestRetryTransportRetriesTransientEOF(t *testing.T) {
	ft := &failTransport{failWith: io.EOF}
	rt := NewRetryTransport(ft)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.com", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("expected successful retry after EOF, got error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK after retry, got %d", resp.StatusCode)
	}
	if atomic.LoadInt32(&ft.attempts) != 2 {
		t.Errorf("expected 2 attempts, got %d", ft.attempts)
	}
}

func TestRetryTransportDoesNotRetryNonTransient(t *testing.T) {
	ft := &failTransport{status: http.StatusNotFound}
	rt := NewRetryTransport(ft)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.com", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 status, got %d", resp.StatusCode)
	}
	if atomic.LoadInt32(&ft.attempts) != 1 {
		t.Errorf("expected 1 attempt, got %d", ft.attempts)
	}
}

func TestWithFingerprintStoreHeaderAndCookieInjection(t *testing.T) {
	var capturedUA, capturedSecUA, capturedCookie string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUA = r.Header.Get("User-Agent")
		capturedSecUA = r.Header.Get("Sec-Ch-Ua")
		capturedCookie = r.Header.Get("Cookie")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	store := fingerprint.NewMemoryStore()
	sourceID := "testsource"
	_ = store.Set(sourceID, fingerprint.Profile{
		UserAgent: "CustomUA/2.0",
		ClientHints: &fingerprint.ClientHints{
			UA: `"CustomBrowser";v="1.0"`,
		},
		Cookies: map[string]string{
			server.URL: "session=xyz123",
		},
		TLSProfile: fingerprint.TLSProfileFirefox,
	})

	source, err := NewHttpSource(ProviderConfig{
		ID:        sourceID,
		Name:      "Test Source",
		BaseURL:   server.URL,
		UserAgent: "DefaultUA/1.0",
	})
	if err != nil {
		t.Fatalf("failed to create source: %v", err)
	}

	source.WithFingerprintStore(store)

	req, err := source.NewRequest(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := source.Client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if capturedUA != "CustomUA/2.0" {
		t.Errorf("expected User-Agent CustomUA/2.0, got %q", capturedUA)
	}
	if capturedSecUA != `"CustomBrowser";v="1.0"` {
		t.Errorf("expected Sec-Ch-Ua \"CustomBrowser\";v=\"1.0\", got %q", capturedSecUA)
	}
	if capturedCookie != "session=xyz123" {
		t.Errorf("expected Cookie session=xyz123, got %q", capturedCookie)
	}
}
