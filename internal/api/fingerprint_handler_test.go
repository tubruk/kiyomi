package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/tubruk/kiyomi/internal/config"
	"github.com/tubruk/kiyomi/internal/library"
	"github.com/tubruk/kiyomi/pkg/fingerprint"
)

func setupTestHandler(t *testing.T) (*Handler, *echo.Echo) {
	t.Helper()
	tmpDir := t.TempDir()
	cfg := &config.Config{
		LibraryDir: tmpDir,
		CacheDir:   filepath.Join(tmpDir, "cache"),
	}
	lib := library.NewLibrary(tmpDir)

	h := NewHandler(cfg, lib)
	e := echo.New()
	h.RegisterRoutes(e)
	return h, e
}

func TestFingerprintHandlers(t *testing.T) {
	_, e := setupTestHandler(t)

	t.Run("GET missing fingerprint returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/mangadex/fingerprint", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404 Not Found, got %d", rec.Code)
		}
	})

	t.Run("PUT fingerprint creates profile", func(t *testing.T) {
		payload := FingerprintRequest{
			UserAgent: "Mozilla/5.0 (CustomUA) KiyomiTest/1.0",
			ClientHints: &ClientHintsDTO{
				UA:       `"CustomUA";v="1"`,
				Platform: `"macOS"`,
				Mobile:   "?0",
			},
			Cookies: map[string]string{
				"https://api.mangadex.org": "session=abc12345",
			},
			TLSProfile: "chrome",
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPut, "/api/v1/providers/mangadex/fingerprint", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp FingerprintResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if resp.ProviderID != "mangadex" {
			t.Errorf("expected providerId 'mangadex', got %q", resp.ProviderID)
		}
		if resp.UserAgent != payload.UserAgent {
			t.Errorf("expected userAgent %q, got %q", payload.UserAgent, resp.UserAgent)
		}
		if resp.TLSProfile != "chrome" {
			t.Errorf("expected tlsProfile 'chrome', got %q", resp.TLSProfile)
		}
		if resp.Cookies["https://api.mangadex.org"] != "session=abc12345" {
			t.Errorf("expected cookie session=abc12345, got %v", resp.Cookies)
		}
		if resp.ClientHints == nil || resp.ClientHints.UA != `"CustomUA";v="1"` {
			t.Errorf("expected client hints UA '\"CustomUA\";v=\"1\"', got %v", resp.ClientHints)
		}
	})

	t.Run("GET fingerprint returns updated profile", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/mangadex/fingerprint", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp FingerprintResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if resp.ProviderID != "mangadex" {
			t.Errorf("expected providerId 'mangadex', got %q", resp.ProviderID)
		}
		if resp.UserAgent != "Mozilla/5.0 (CustomUA) KiyomiTest/1.0" {
			t.Errorf("unexpected userAgent: %q", resp.UserAgent)
		}
	})

	t.Run("PUT fingerprint invalid TLS profile returns 400", func(t *testing.T) {
		payload := FingerprintRequest{
			TLSProfile: "invalid-profile",
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPut, "/api/v1/providers/mangafox/fingerprint", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request for invalid TLS profile, got %d", rec.Code)
		}
	})

	t.Run("DELETE fingerprint removes profile", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/providers/mangadex/fingerprint", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected 204 No Content, got %d", rec.Code)
		}

		// GET should now return 404
		getReq := httptest.NewRequest(http.MethodGet, "/api/v1/providers/mangadex/fingerprint", nil)
		getRec := httptest.NewRecorder()
		e.ServeHTTP(getRec, getReq)

		if getRec.Code != http.StatusNotFound {
			t.Errorf("expected 404 Not Found after DELETE, got %d", getRec.Code)
		}
	})
}

func TestStreamRemoteImageFingerprintWiring(t *testing.T) {
	var receivedUA string
	var receivedCookie string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUA = r.Header.Get("User-Agent")
		receivedCookie = r.Header.Get("Cookie")
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("fake-png-data"))
	}))
	defer ts.Close()

	h, _ := setupTestHandler(t)

	// Set fingerprint override
	err := h.fpStore.Set("mangafox", fingerprint.Profile{
		UserAgent: "TestProxyUA/1.0",
		Cookies: map[string]string{
			ts.URL: "cf_clearance=test_cookie_val",
		},
	})
	if err != nil {
		t.Fatalf("failed to set fingerprint profile: %v", err)
	}

	e := echo.New()
	c := e.NewContext(
		httptest.NewRequest(http.MethodGet, "/proxy/image?url="+ts.URL+"/image.png", nil),
		httptest.NewRecorder(),
	)

	content := &library.ContentSource{
		ProviderID: "mangafox",
	}

	if err := h.streamRemoteImage(c, ts.URL+"/image.png", content); err != nil {
		t.Fatalf("streamRemoteImage error: %v", err)
	}

	if receivedUA != "TestProxyUA/1.0" {
		t.Errorf("expected User-Agent 'TestProxyUA/1.0', got %q", receivedUA)
	}
	if receivedCookie != "cf_clearance=test_cookie_val" {
		t.Errorf("expected Cookie 'cf_clearance=test_cookie_val', got %q", receivedCookie)
	}
}
