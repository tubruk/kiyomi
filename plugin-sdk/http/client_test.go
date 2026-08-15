package http

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	utls "github.com/refraction-networking/utls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdk "github.com/tubruk/kiyomi/plugin-sdk"
	v1 "github.com/tubruk/kiyomi/plugin-sdk/proto/v1"
)

func TestNewClient_DefaultsAndBasicGetPost(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, DefaultUserAgent, r.Header.Get("User-Agent"))
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok get"))
			return
		}
		if r.Method == http.MethodPost {
			body, _ := io.ReadAll(r.Body)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("echo: " + string(body)))
			return
		}
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer ts.Close()

	client := NewClient()
	require.NotNil(t, client)
	require.NotNil(t, client.StandardClient())

	ctx := context.Background()

	// Test GET
	resp, err := client.Get(ctx, ts.URL)
	require.NoError(t, err)
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "ok get", string(data))

	// Test POST
	postResp, err := client.Post(ctx, ts.URL, "text/plain", io.NopCloser(httptest.NewRecorder().Body))
	require.NoError(t, err)
	defer postResp.Body.Close()

	// Test PostForm
	formResp, err := client.PostForm(ctx, ts.URL, url.Values{"title": []string{"hello"}})
	require.NoError(t, err)
	defer formResp.Body.Close()
	formData, _ := io.ReadAll(formResp.Body)
	assert.Contains(t, string(formData), "title=hello")
}

func TestNewFingerprintedClient_GlobalHttpConfig(t *testing.T) {
	var receivedUA string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	globalCfg := &v1.GlobalHttpConfig{
		UserAgent:      "Kiyomi-Custom-Bot/1.0",
		TimeoutSeconds: 15,
		ProxyUrl:       "http://127.0.0.1:9999",
	}

	// Create with a mock transport to verify proxy was configured
	client := NewFingerprintedClient(globalCfg, WithTimeout(5*time.Second))
	require.NotNil(t, client)
	assert.Equal(t, "Kiyomi-Custom-Bot/1.0", client.config.userAgent)
	assert.Equal(t, "http://127.0.0.1:9999", client.config.proxyURL)
	assert.Equal(t, 5*time.Second, client.config.timeout)

	// Test SDK GlobalHttpConfig variant
	sdkClient := NewClient(WithSDKGlobalHttpConfig(sdk.GlobalHttpConfig{
		UserAgent:      "Kiyomi-SDK-UA/2.0",
		TimeoutSeconds: 12,
		ProxyURL:       "http://127.0.0.1:8888",
	}))
	assert.Equal(t, "Kiyomi-SDK-UA/2.0", sdkClient.config.userAgent)
	assert.Equal(t, 12*time.Second, sdkClient.config.timeout)

	// Send request without proxy
	directClient := NewFingerprintedClient(&v1.GlobalHttpConfig{UserAgent: "Kiyomi-Direct/1.0"})
	resp, err := directClient.Get(context.Background(), ts.URL)
	require.NoError(t, err)
	_ = resp.Body.Close()
	assert.Equal(t, "Kiyomi-Direct/1.0", receivedUA)
}

func TestWithTLSProfile_Options(t *testing.T) {
	c1 := NewClient(WithTLSProfile(TLSProfileChrome))
	assert.Equal(t, TLSProfileChrome, c1.config.tlsProfile)

	c2 := NewClient(WithTLSProfile(TLSProfileFirefox))
	assert.Equal(t, TLSProfileFirefox, c2.config.tlsProfile)

	c3 := NewClient(WithUTLSClientHello(utls.HelloChrome_120))
	assert.True(t, c3.config.hasCustomHelloID)

	assert.True(t, TLSProfileChrome.Valid())
	assert.True(t, TLSProfileFirefox.Valid())
	assert.True(t, TLSProfileDefault.Valid())
	assert.False(t, TLSProfile("unknown").Valid())
}

func TestWithClientHints_AndHeaders(t *testing.T) {
	var receivedUA, receivedPlatform, receivedMobile, customHeader string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUA = r.Header.Get("Sec-Ch-Ua")
		receivedPlatform = r.Header.Get("Sec-Ch-Ua-Platform")
		receivedMobile = r.Header.Get("Sec-Ch-Ua-Mobile")
		customHeader = r.Header.Get("X-Custom-Auth")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client := NewClient(
		WithClientHints(DefaultClientHints()),
		WithHeader("X-Custom-Auth", "secret-token"),
	)

	resp, err := client.Get(context.Background(), ts.URL)
	require.NoError(t, err)
	_ = resp.Body.Close()

	assert.NotEmpty(t, receivedUA)
	assert.Equal(t, `"Windows"`, receivedPlatform)
	assert.Equal(t, "?0", receivedMobile)
	assert.Equal(t, "secret-token", customHeader)
}

func TestRateLimiter(t *testing.T) {
	var requestCount int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// 5 requests per second
	client := NewClient(
		WithRateLimit(5, 1),
	)

	start := time.Now()
	for i := 0; i < 3; i++ {
		resp, err := client.Get(context.Background(), ts.URL)
		require.NoError(t, err)
		_ = resp.Body.Close()
	}
	elapsed := time.Since(start)

	assert.Equal(t, int32(3), atomic.LoadInt32(&requestCount))
	// 3 requests at 5 rps with burst 1 should take around >= 300ms
	assert.GreaterOrEqual(t, elapsed, 300*time.Millisecond)

	// Test with RateLimitSpec options
	specClient := NewClient(WithRateLimitSpec(&v1.RateLimitSpec{
		RequestsPerSecond:     10,
		MaxConcurrentRequests: 2,
	}))
	assert.Equal(t, float64(10), specClient.config.rps)
	assert.Equal(t, 2, specClient.config.maxConcurrent)

	sdkSpecClient := NewClient(WithSDKRateLimitSpec(sdk.RateLimitSpec{
		RequestsPerSecond:     8,
		MaxConcurrentRequests: 3,
	}))
	assert.Equal(t, float64(8), sdkSpecClient.config.rps)
	assert.Equal(t, 3, sdkSpecClient.config.maxConcurrent)
}

func TestConcurrencyLimit(t *testing.T) {
	var currentActive int32
	var maxObservedActive int32
	var mu sync.Mutex

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		active := atomic.AddInt32(&currentActive, 1)
		mu.Lock()
		if active > maxObservedActive {
			maxObservedActive = active
		}
		mu.Unlock()

		time.Sleep(30 * time.Millisecond)
		atomic.AddInt32(&currentActive, -1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("done"))
	}))
	defer ts.Close()

	// Limit to max 2 concurrent requests
	client := NewClient(WithConcurrencyLimit(2))

	var wg sync.WaitGroup
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := client.Get(context.Background(), ts.URL)
			if assert.NoError(t, err) {
				_, _ = io.ReadAll(resp.Body)
				_ = resp.Body.Close()
			}
		}()
	}
	wg.Wait()

	mu.Lock()
	assert.LessOrEqual(t, maxObservedActive, int32(2), "concurrency limit must never be exceeded")
	mu.Unlock()
}

func TestRetryTransport_TransientErrors(t *testing.T) {
	var attempts int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		att := atomic.AddInt32(&attempts, 1)
		if att < 3 {
			w.WriteHeader(http.StatusServiceUnavailable) // 503
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("finally succeeded"))
	}))
	defer ts.Close()

	client := NewClient(
		WithRetry(4, 10*time.Millisecond, 50*time.Millisecond),
	)

	resp, err := client.Get(context.Background(), ts.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "finally succeeded", string(body))
	assert.Equal(t, int32(3), atomic.LoadInt32(&attempts))
}

func TestRetryTransport_NonRetryableStatus(t *testing.T) {
	var attempts int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusNotFound) // 404
	}))
	defer ts.Close()

	client := NewClient(WithRetry(3, 10*time.Millisecond, 20*time.Millisecond))
	resp, err := client.Get(context.Background(), ts.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Equal(t, int32(1), atomic.LoadInt32(&attempts), "404 should not be retried")
}

func TestRetryTransport_RetryAfterHeader(t *testing.T) {
	var attempts int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		att := atomic.AddInt32(&attempts, 1)
		if att == 1 {
			w.Header().Set("Retry-After", "1") // 1 second
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("rate limit cleared"))
	}))
	defer ts.Close()

	client := NewClient(WithRetry(3, 10*time.Millisecond, 2*time.Second))
	resp, err := client.Get(context.Background(), ts.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, int32(2), atomic.LoadInt32(&attempts))
}

func TestGetJSON_AndGetDocument(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/json" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"123","title":"Solo Leveling"}`))
			return
		}
		if r.URL.Path == "/html" {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<html><body><h1 class="title">One Piece</h1></body></html>`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	client := NewClient()

	// Test GetJSON
	var result struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}
	err := client.GetJSON(context.Background(), ts.URL+"/json", &result)
	require.NoError(t, err)
	assert.Equal(t, "123", result.ID)
	assert.Equal(t, "Solo Leveling", result.Title)

	// Test GetDocument
	doc, err := client.GetDocument(context.Background(), ts.URL+"/html")
	require.NoError(t, err)
	assert.Equal(t, "One Piece", doc.Find("h1.title").Text())

	// Test Clone
	cloned := client.Clone(WithUserAgent("Cloned-Agent/1.0"))
	assert.Equal(t, "Cloned-Agent/1.0", cloned.config.userAgent)
}
