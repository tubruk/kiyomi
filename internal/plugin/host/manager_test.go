package host_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tubruk/kiyomi/internal/plugin/host"
	"github.com/tubruk/kiyomi/pkg/provider"
	provsdk "github.com/tubruk/kiyomi/pkg/provider/sdk"
	sdk "github.com/tubruk/kiyomi/plugin-sdk"
)

var testBinaryPath string

func TestMain(m *testing.M) {
	// Build the test plugin executable binary once for integration tests
	tmpDir, err := os.MkdirTemp("", "kiyomi-test-plugin-*")
	if err != nil {
		fmt.Printf("failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	binName := "dummy-plugin"
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	testBinaryPath = filepath.Join(tmpDir, binName)

	// Write dummy plugin source code
	srcCode := `package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"time"

	sdk "github.com/tubruk/kiyomi/plugin-sdk"
	sdklogger "github.com/tubruk/kiyomi/plugin-sdk/logger"
)

type dummyPlugin struct{}

func (p *dummyPlugin) Describe(ctx context.Context) (sdk.PluginDescriptor, error) {
	return sdk.PluginDescriptor{
		PluginID:      "dummy-test-plugin",
		PluginName:    "Dummy Test Plugin",
		PluginVersion: "1.0.0",
		SDKVersion:    sdk.Version,
		Providers: []sdk.ProviderDescriptor{
			{
				ID:           "dummy-source",
				Name:         "Dummy Source",
				Description:  "Source inside dummy plugin",
				Capabilities: []string{"metadata", "content"},
				DefaultRateLimit: sdk.RateLimitSpec{
					RequestsPerSecond: 10,
				},
			},
		},
	}, nil
}

func (p *dummyPlugin) Init(ctx context.Context, config sdk.PluginConfig) error {
	logger := sdklogger.New(os.Stderr, &sdklogger.Options{Level: slog.LevelDebug})
	logger.Info("dummy plugin initialized", slog.Group("meta", slog.String("env", "test"), slog.Int("timeout", config.HTTPConfig.TimeoutSeconds)))
	return nil
}

type dummySource struct{}

func (s *dummySource) Search(ctx context.Context, query string, opts sdk.SearchOptions) ([]sdk.SearchResult, error) {
	return []sdk.SearchResult{
		{
			RemoteID:     "manga-dummy-1",
			Title:        "Dummy Manga for " + query,
			URL:          "https://example.com/dummy/1",
			Availability: sdk.AvailabilityAvailable,
		},
	}, nil
}

func (s *dummySource) Details(ctx context.Context, remoteID string) (sdk.MangaMetadata, error) {
	return sdk.MangaMetadata{
		RemoteID:     remoteID,
		Title:        "Dummy Manga Details",
		Status:       "ongoing",
		Score:        8.8,
		Availability: sdk.AvailabilityAvailable,
	}, nil
}

func (s *dummySource) Cover(ctx context.Context, remoteID string, size sdk.ImageSize) (sdk.ImageRef, error) {
	return sdk.ImageRef{URL: "https://example.com/dummy.jpg"}, nil
}

func (s *dummySource) Aliases(ctx context.Context, remoteID string) ([]string, error) {
	return []string{"Dummy Alias"}, nil
}

func (s *dummySource) HasStableChapterID() bool { return true }

func (s *dummySource) FetchChapters(ctx context.Context, mangaRef string) ([]sdk.Chapter, error) {
	return []sdk.Chapter{
		{
			ID:          "ch-dummy-1",
			Name:        "Chapter 1",
			Number:      1.0,
			UploadDate:  time.Unix(1700000000, 0),
			SourceOrder: 1,
		},
	}, nil
}

func (s *dummySource) FetchPages(ctx context.Context, mangaRef, chapterRef string) ([]sdk.Page, error) {
	return []sdk.Page{
		{Index: 0, URL: "https://example.com/p1.jpg"},
	}, nil
}

func (s *dummySource) FetchPageStream(ctx context.Context, page sdk.Page) (io.ReadCloser, error) {
	return nil, nil
}

func (s *dummySource) RateLimit() sdk.RateLimitHint {
	return sdk.RateLimitHint{RequestsPerSecond: 10}
}

func main() {
	plug := &dummyPlugin{}
	src := &dummySource{}

	sdk.ServePlugin(sdk.ServeOptions{
		Plugin: plug,
		MetadataProviders: map[string]sdk.MetadataProvider{
			"dummy-source": src,
		},
		ContentProviders: map[string]sdk.ContentProvider{
			"dummy-source": src,
		},
	})
}
`
	srcPath := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(srcPath, []byte(srcCode), 0644); err != nil {
		fmt.Printf("failed to write test plugin source: %v\n", err)
		os.Exit(1)
	}

	cmd := exec.Command("go", "build", "-o", testBinaryPath, srcPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("failed to build test plugin binary: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func TestPluginManager_Discovery(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kiyomi-discovery-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create dummy executable files and non-executable files
	bin1 := filepath.Join(tmpDir, "plugin-a")
	require.NoError(t, os.WriteFile(bin1, []byte("#!/bin/sh\nexit 0\n"), 0755))

	bin2 := filepath.Join(tmpDir, "plugin-b")
	require.NoError(t, os.WriteFile(bin2, []byte("#!/bin/sh\nexit 0\n"), 0755))

	textDoc := filepath.Join(tmpDir, "readme.txt")
	require.NoError(t, os.WriteFile(textDoc, []byte("ignore me"), 0644))

	hidden := filepath.Join(tmpDir, ".DS_Store")
	require.NoError(t, os.WriteFile(hidden, []byte("ignore me"), 0755))

	mgr := host.NewPluginManager(host.ManagerOptions{
		PluginDir: tmpDir,
	})

	discovered, err := mgr.DiscoverPlugins()
	require.NoError(t, err)
	assert.Len(t, discovered, 2)
}

func TestPluginManager_LifecycleAndHotReloading(t *testing.T) {
	if testBinaryPath == "" {
		t.Skip("test plugin binary was not compiled")
	}

	reg := provider.NewRegistry()
	mgr := host.NewPluginManager(host.ManagerOptions{
		GlobalConfig: map[string]string{"theme": "dark"},
		HTTPConfig: sdk.GlobalHttpConfig{
			UserAgent:      "KiyomiHost/1.0",
			TimeoutSeconds: 20,
		},
		Registry: reg,
	})
	defer mgr.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 1. Start Plugin
	inst, err := mgr.Start(ctx, testBinaryPath)
	require.NoError(t, err)
	require.NotNil(t, inst)

	assert.Equal(t, "dummy-test-plugin", inst.PluginID)
	assert.Equal(t, "Dummy Test Plugin", inst.Name)
	assert.Equal(t, "1.0.0", inst.Version)
	assert.Equal(t, host.StateRunning, inst.State)
	assert.Greater(t, inst.PID, 0)
	assert.False(t, inst.LoadedAt.IsZero())

	// 2. Check Plugin Status
	status, ok := mgr.GetPluginStatus("dummy-test-plugin")
	assert.True(t, ok)
	assert.Equal(t, "dummy-test-plugin", status.PluginID)
	assert.Equal(t, host.StateRunning, status.State)
	require.Len(t, status.Providers, 1)
	assert.Equal(t, "dummy-source", status.Providers[0].ID)

	// 3. Verify Log Interceptor Captured Subprocess Init Log
	time.Sleep(100 * time.Millisecond) // brief pause for stdio drain
	logs := mgr.GetPluginLogs("dummy-test-plugin")
	require.NotEmpty(t, logs, "expected logs from plugin initialization")

	foundInitLog := false
	for _, l := range logs {
		if l.Message == "dummy plugin initialized" {
			foundInitLog = true
			meta, ok := l.Fields["meta"].(map[string]any)
			require.True(t, ok, "expected meta group in log fields")
			assert.Equal(t, "test", meta["env"])
			assert.Equal(t, float64(20), meta["timeout"])
			break
		}
	}
	assert.True(t, foundInitLog, "expected 'dummy plugin initialized' structured log entry")

	// 4. Verify Provider Registry Dispatch
	metaProv, ok := reg.GetMetadata("dummy-source")
	require.True(t, ok, "expected dummy-source to be registered in metadata")

	results, err := metaProv.Search(ctx, "one piece", provsdk.SearchOptions{Limit: 5})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "Dummy Manga for one piece", results[0].Title)

	contentProv, ok := reg.GetContent("dummy-source")
	require.True(t, ok, "expected dummy-source to be registered in content")
	chapters, err := contentProv.FetchChapters(ctx, "manga-dummy-1")
	require.NoError(t, err)
	require.Len(t, chapters, 1)
	assert.Equal(t, "Chapter 1", chapters[0].Name)

	// 5. Zero-Downtime Hot-Reloading
	err = mgr.Reload(ctx, "dummy-test-plugin")
	require.NoError(t, err)

	reloadedStatus, ok := mgr.GetPluginStatus("dummy-test-plugin")
	require.True(t, ok)
	assert.Equal(t, host.StateRunning, reloadedStatus.State)

	// Verify provider registry immediately continues working after hot reload
	metaProvAfter, ok := reg.GetMetadata("dummy-source")
	require.True(t, ok)
	resultsAfter, err := metaProvAfter.Search(ctx, "bleach", provsdk.SearchOptions{})
	require.NoError(t, err)
	require.Len(t, resultsAfter, 1)
	assert.Equal(t, "Dummy Manga for bleach", resultsAfter[0].Title)

	// 6. Stop Plugin
	err = mgr.Stop(ctx, "dummy-test-plugin")
	require.NoError(t, err)

	_, ok = reg.Get("dummy-source")
	assert.False(t, ok, "dummy-source should be unregistered from registry after stop")
}
