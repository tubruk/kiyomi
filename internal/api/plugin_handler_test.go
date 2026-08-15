package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/tubruk/kiyomi/internal/config"
	"github.com/tubruk/kiyomi/internal/library"
	"github.com/tubruk/kiyomi/internal/plugin/host"
	sdk "github.com/tubruk/kiyomi/plugin-sdk"
)

type mockPluginService struct {
	desc    sdk.PluginDescriptor
	initErr error
	lastCfg sdk.PluginConfig
}

func (m *mockPluginService) Describe(ctx context.Context) (sdk.PluginDescriptor, error) {
	return m.desc, nil
}

func (m *mockPluginService) Init(ctx context.Context, cfg sdk.PluginConfig) error {
	m.lastCfg = cfg
	return m.initErr
}

func setupPluginTestHandler(t *testing.T) (*Handler, *echo.Echo, *host.PluginManager) {
	t.Helper()
	tmpDir := t.TempDir()
	pluginDir := filepath.Join(tmpDir, "plugins")
	cfg := &config.Config{
		LibraryDir: tmpDir,
		CacheDir:   filepath.Join(tmpDir, "cache"),
		PluginDir:  pluginDir,
	}
	lib := library.NewLibrary(tmpDir)

	h := NewHandler(cfg, lib)
	pm := host.NewPluginManager(host.ManagerOptions{
		PluginDir: pluginDir,
		Registry:  h.registry,
	})
	h.SetPluginManager(pm)

	e := echo.New()
	h.RegisterRoutes(e)
	return h, e, pm
}

func TestListPlugins(t *testing.T) {
	h, e, pm := setupPluginTestHandler(t)

	t.Run("empty plugin manager returns empty array", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/plugins", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}

		var resp []PluginItemDTO
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if len(resp) != 0 {
			t.Errorf("expected 0 plugins, got %d", len(resp))
		}
	})

	t.Run("nil plugin manager returns empty array", func(t *testing.T) {
		h.SetPluginManager(nil)
		defer h.SetPluginManager(pm)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/plugins", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}

		var resp []PluginItemDTO
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if len(resp) != 0 {
			t.Errorf("expected 0 plugins, got %d", len(resp))
		}
	})

	t.Run("returns installed plugins with descriptors and compatibility", func(t *testing.T) {
		mockSvc := &mockPluginService{
			desc: sdk.PluginDescriptor{
				PluginID:      "test-plugin-a",
				PluginName:    "Test Plugin A",
				PluginVersion: "1.0.0",
				SDKVersion:    sdk.Version,
				PluginSettingsSchema: []sdk.SettingSpec{
					{
						Key:          "api_key",
						Label:        "API Key",
						Type:         "secret",
						DefaultValue: "",
					},
				},
				Providers: []sdk.ProviderDescriptor{
					{
						ID:           "test-source",
						Name:         "Test Source",
						Capabilities: []string{"metadata", "content"},
						SettingsSchema: []sdk.SettingSpec{
							{
								Key:          "quality",
								Label:        "Image Quality",
								Type:         "select",
								Options:      []string{"low", "medium", "high"},
								DefaultValue: "high",
							},
						},
					},
				},
			},
		}

		buf := host.NewRingBuffer(100)
		buf.Push(host.PluginLogEntry{
			Timestamp: time.Now().UTC(),
			Level:     "INFO",
			Message:   "plugin started",
		})

		inst := &host.PluginInstance{
			PluginID:       "test-plugin-a",
			Name:           "Test Plugin A",
			Version:        "1.0.0",
			SDKVersion:     sdk.Version,
			ExecutablePath: "/bin/test-plugin-a",
			PID:            4242,
			State:          host.StateRunning,
			LoadedAt:       time.Now().UTC(),
			PluginService:  mockSvc,
			Descriptor:     mockSvc.desc,
			LogBuffer:      buf,
			Adapters:       make(map[string]*host.GRPCProviderAdapter),
		}

		pm.RegisterInstanceForTest(inst)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/plugins", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp []PluginItemDTO
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if len(resp) != 1 {
			t.Fatalf("expected 1 plugin, got %d", len(resp))
		}

		p := resp[0]
		if p.PluginID != "test-plugin-a" {
			t.Errorf("expected pluginId 'test-plugin-a', got %q", p.PluginID)
		}
		if p.PluginName != "Test Plugin A" {
			t.Errorf("expected pluginName 'Test Plugin A', got %q", p.PluginName)
		}
		if !p.SDKCompatible {
			t.Errorf("expected SDKCompatible true")
		}
		if p.PID != 4242 {
			t.Errorf("expected PID 4242, got %d", p.PID)
		}
		if len(p.Providers) != 1 {
			t.Fatalf("expected 1 provider, got %d", len(p.Providers))
		}
		if p.Providers[0].ID != "test-source" {
			t.Errorf("expected provider ID 'test-source', got %q", p.Providers[0].ID)
		}
	})
}

func TestPluginReload(t *testing.T) {
	_, e, pm := setupPluginTestHandler(t)

	mockSvc := &mockPluginService{
		desc: sdk.PluginDescriptor{
			PluginID:      "reload-plugin",
			PluginName:    "Reload Plugin",
			PluginVersion: "1.0.0",
			SDKVersion:    sdk.Version,
		},
	}

	inst := &host.PluginInstance{
		PluginID:       "reload-plugin",
		Name:           "Reload Plugin",
		Version:        "1.0.0",
		SDKVersion:     sdk.Version,
		ExecutablePath: "/bin/reload-plugin",
		PID:            1001,
		State:          host.StateRunning,
		LoadedAt:       time.Now().UTC(),
		PluginService:  mockSvc,
		Descriptor:     mockSvc.desc,
		Adapters:       make(map[string]*host.GRPCProviderAdapter),
	}
	pm.RegisterInstanceForTest(inst)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/plugins/reload", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp ReloadResponseDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("expected status 'ok', got %q", resp.Status)
	}
}

func TestGetPluginLogs(t *testing.T) {
	_, e, pm := setupPluginTestHandler(t)

	buf := host.NewRingBuffer(100)
	testTime := time.Date(2026, 8, 16, 11, 0, 0, 0, time.UTC)
	buf.Push(host.PluginLogEntry{
		Timestamp: testTime,
		Level:     "INFO",
		Message:   "fetching chapter list",
		Fields:    map[string]any{"page": float64(1)},
		Raw:       "fetching chapter list",
	})
	buf.Push(host.PluginLogEntry{
		Timestamp: testTime.Add(time.Second),
		Level:     "WARN",
		Message:   "rate limit backoff triggered",
		Fields:    map[string]any{"retryAfter": float64(2)},
		Raw:       "rate limit backoff triggered",
	})

	inst := &host.PluginInstance{
		PluginID:   "log-plugin",
		Name:       "Log Plugin",
		Version:    "1.0.0",
		SDKVersion: sdk.Version,
		PID:        5555,
		State:      host.StateRunning,
		LoadedAt:   time.Now().UTC(),
		LogBuffer:  buf,
		Adapters:   make(map[string]*host.GRPCProviderAdapter),
	}
	pm.RegisterInstanceForTest(inst)

	t.Run("GET logs for existing plugin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/plugins/log-plugin/logs", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}

		var logs []host.PluginLogEntry
		if err := json.Unmarshal(rec.Body.Bytes(), &logs); err != nil {
			t.Fatalf("failed to unmarshal logs: %v", err)
		}
		if len(logs) != 2 {
			t.Fatalf("expected 2 log entries, got %d", len(logs))
		}
		if logs[0].Message != "fetching chapter list" {
			t.Errorf("expected first log message 'fetching chapter list', got %q", logs[0].Message)
		}
		if logs[1].Level != "WARN" {
			t.Errorf("expected second log level 'WARN', got %q", logs[1].Level)
		}
	})

	t.Run("GET logs for non-existent plugin returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/plugins/nonexistent/logs", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404 Not Found, got %d", rec.Code)
		}
	})
}

func TestUpdatePluginConfig(t *testing.T) {
	_, e, pm := setupPluginTestHandler(t)

	mockSvc := &mockPluginService{
		desc: sdk.PluginDescriptor{
			PluginID:      "config-plugin",
			PluginName:    "Config Plugin",
			PluginVersion: "1.0.0",
			SDKVersion:    sdk.Version,
			Providers: []sdk.ProviderDescriptor{
				{
					ID:   "cfg-provider",
					Name: "Config Provider",
				},
			},
		},
	}

	inst := &host.PluginInstance{
		PluginID:      "config-plugin",
		Name:          "Config Plugin",
		Version:       "1.0.0",
		SDKVersion:    sdk.Version,
		PID:           8888,
		State:         host.StateRunning,
		LoadedAt:      time.Now().UTC(),
		PluginService: mockSvc,
		Descriptor:    mockSvc.desc,
		Adapters:      make(map[string]*host.GRPCProviderAdapter),
	}
	pm.RegisterInstanceForTest(inst)

	t.Run("POST valid config payload updates manager and re-inits plugin", func(t *testing.T) {
		payload := UpdatePluginConfigRequest{
			GlobalConfig: map[string]string{
				"api_token": "secret123",
			},
			ProviderConfigs: map[string]map[string]string{
				"cfg-provider": {
					"rate_limit": "10",
				},
			},
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/plugins/config-plugin/config", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}

		if mockSvc.lastCfg.GlobalConfig["api_token"] != "secret123" {
			t.Errorf("expected api_token 'secret123', got %q", mockSvc.lastCfg.GlobalConfig["api_token"])
		}
		if mockSvc.lastCfg.ProviderConfigs["cfg-provider"]["rate_limit"] != "10" {
			t.Errorf("expected rate_limit '10', got %q", mockSvc.lastCfg.ProviderConfigs["cfg-provider"]["rate_limit"])
		}
	})

	t.Run("POST invalid json body returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/plugins/config-plugin/config", bytes.NewReader([]byte("{invalid-json")))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d", rec.Code)
		}
	})

	t.Run("POST config for non-existent plugin returns 404", func(t *testing.T) {
		body, _ := json.Marshal(UpdatePluginConfigRequest{})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/plugins/unknown/config", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404 Not Found, got %d", rec.Code)
		}
	})
}

func TestCollisionsAndPreferences(t *testing.T) {
	h, e, _ := setupPluginTestHandler(t)

	// Builtin provider is registered under mangadex
	// Register a colliding mock provider with the same ID but from a plugin
	candProvider := &mockProvider{id: "mangadex", name: "MangaDex Plugin"}
	adapter := host.NewGRPCProviderAdapter(
		sdk.ProviderDescriptor{ID: "mangadex", Name: "MangaDex Plugin"},
		"plugin-dex",
		"2.0.0",
		nil, nil, nil, nil,
	)
	h.registry.Register(candProvider)
	h.registry.Register(adapter)

	t.Run("GET /plugins/collisions returns colliding providers", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/plugins/collisions", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}

		var collisions []ProviderCollisionDTO
		if err := json.Unmarshal(rec.Body.Bytes(), &collisions); err != nil {
			t.Fatalf("failed to unmarshal collisions: %v", err)
		}

		foundMD := false
		for _, c := range collisions {
			if c.ProviderID == "mangadex" {
				foundMD = true
				if len(c.Candidates) < 2 {
					t.Errorf("expected at least 2 candidates for mangadex, got %d", len(c.Candidates))
				}
				if c.Selected != "builtin" {
					t.Errorf("expected default winner 'builtin', got %q", c.Selected)
				}
			}
		}
		if !foundMD {
			t.Errorf("expected collision for 'mangadex', but none found")
		}
	})

	t.Run("POST /plugins/preference sets user preferred plugin", func(t *testing.T) {
		prefPayload := SetPreferenceRequest{
			ProviderID: "mangadex",
			Preference: "plugin-dex",
		}
		body, _ := json.Marshal(prefPayload)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/plugins/preference", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}

		// Verify winning provider is now plugin-dex
		activeWinner, ok := h.registry.Get("mangadex")
		if !ok {
			t.Fatalf("expected active provider for mangadex")
		}
		if pi, ok := activeWinner.(interface{ PluginID() string }); !ok || pi.PluginID() != "plugin-dex" {
			t.Errorf("expected active provider plugin ID 'plugin-dex', got %v", activeWinner)
		}

		// Verify GET /plugins/collisions reflects the new preference
		getReq := httptest.NewRequest(http.MethodGet, "/api/v1/plugins/collisions", nil)
		getRec := httptest.NewRecorder()
		e.ServeHTTP(getRec, getReq)

		var collisions []ProviderCollisionDTO
		_ = json.Unmarshal(getRec.Body.Bytes(), &collisions)
		for _, c := range collisions {
			if c.ProviderID == "mangadex" {
				if c.Selected != "plugin-dex" {
					t.Errorf("expected selected 'plugin-dex', got %q", c.Selected)
				}
			}
		}
	})

	t.Run("POST /plugins/preference invalid body returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/plugins/preference", bytes.NewReader([]byte("{}")))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d", rec.Code)
		}
	})
}
