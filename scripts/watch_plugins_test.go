package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPluginWatcher_BuildAndReload(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "watcher-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	pluginsSrcDir := filepath.Join(tempDir, "plugins")
	devPluginsDir := filepath.Join(tempDir, "dev-plugins")

	testPluginDir := filepath.Join(pluginsSrcDir, "testplugin")
	if err := os.MkdirAll(testPluginDir, 0755); err != nil {
		t.Fatalf("failed to create test plugin dir: %v", err)
	}

	mainGoContent := `package main
import "fmt"
func main() { fmt.Println("test plugin") }
`
	if err := os.WriteFile(filepath.Join(testPluginDir, "main.go"), []byte(mainGoContent), 0644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testPluginDir, "go.mod"), []byte("module testplugin\n\ngo 1.26.4\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	reloaded := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/plugins/reload" && r.Method == http.MethodPost {
			reloaded = true
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	watcher := NewPluginWatcher(pluginsSrcDir, devPluginsDir, server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Perform single build and reload iteration
	watcher.rebuildAll(ctx)

	if !reloaded {
		t.Errorf("expected reload trigger to be sent to mock server")
	}

	// Verify compiled binary output exists in devPluginsDir
	expectedBin := filepath.Join(devPluginsDir, "testplugin-plugin")
	if os.Getenv("GOOS") == "windows" {
		expectedBin += ".exe"
	}
	if _, err := os.Stat(expectedBin); os.IsNotExist(err) {
		t.Errorf("expected compiled binary at %s, but it does not exist", expectedBin)
	}
}
