package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// PluginWatcher monitors plugins/ for changes, rebuilds binaries into dev-plugins/
// and calls POST http://localhost:8080/api/v1/plugins/reload
type PluginWatcher struct {
	pluginsDir string
	outDir     string
	hostURL    string
}

func NewPluginWatcher(pluginsDir, outDir, hostURL string) *PluginWatcher {
	absPluginsDir, err := filepath.Abs(pluginsDir)
	if err == nil {
		pluginsDir = absPluginsDir
	}
	absOutDir, err := filepath.Abs(outDir)
	if err == nil {
		outDir = absOutDir
	}
	return &PluginWatcher{
		pluginsDir: pluginsDir,
		outDir:     outDir,
		hostURL:    hostURL,
	}
}

func (w *PluginWatcher) Run(ctx context.Context) error {
	return w.RunWithContext(ctx)
}

func (w *PluginWatcher) RunWithContext(ctx context.Context) error {
	if err := os.MkdirAll(w.outDir, 0755); err != nil {
		return fmt.Errorf("failed to create output dir: %w", err)
	}

	log.Printf("[Plugin Watcher] Monitoring %s for changes...", w.pluginsDir)
	log.Printf("[Plugin Watcher] Compiled binaries output: %s", w.outDir)
	log.Printf("[Plugin Watcher] Reload endpoint: %s/api/v1/plugins/reload", w.hostURL)

	// Initial build of all plugins
	w.rebuildAll(ctx)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	lastModTime := make(map[string]time.Time)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			w.checkModifications(ctx, lastModTime)
		}
	}
}

func isPluginModule(dirPath string) bool {
	goModPath := filepath.Join(dirPath, "go.mod")
	info, err := os.Stat(goModPath)
	return err == nil && !info.IsDir()
}

func (w *PluginWatcher) checkModifications(ctx context.Context, lastModTime map[string]time.Time) {
	entries, err := os.ReadDir(w.pluginsDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pluginName := entry.Name()
		pluginPath := filepath.Join(w.pluginsDir, pluginName)
		if !isPluginModule(pluginPath) {
			continue
		}

		hasChanges := false
		_ = filepath.Walk(pluginPath, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			last, exists := lastModTime[path]
			if !exists || info.ModTime().After(last) {
				lastModTime[path] = info.ModTime()
				if exists {
					hasChanges = true
				}
			}
			return nil
		})

		if hasChanges {
			log.Printf("[Plugin Watcher] File change detected in plugin '%s'. Rebuilding...", pluginName)
			w.buildAndReload(ctx, pluginName)
		}
	}
}

func (w *PluginWatcher) rebuildAll(ctx context.Context) {
	entries, err := os.ReadDir(w.pluginsDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pluginPath := filepath.Join(w.pluginsDir, entry.Name())
		if isPluginModule(pluginPath) {
			w.buildPlugin(ctx, entry.Name())
		}
	}
	w.triggerReload(ctx)
}

func (w *PluginWatcher) buildAndReload(ctx context.Context, pluginName string) {
	if w.buildPlugin(ctx, pluginName) {
		w.triggerReload(ctx)
	}
}

func (w *PluginWatcher) buildPlugin(ctx context.Context, pluginName string) bool {
	binName := pluginName + "-plugin"
	if os.Getenv("GOOS") == "windows" {
		binName += ".exe"
	}
	outPath := filepath.Join(w.outDir, binName)
	pluginSrcPath := filepath.Join(w.pluginsDir, pluginName)
	if !filepath.IsAbs(pluginSrcPath) && !strings.HasPrefix(pluginSrcPath, ".") {
		pluginSrcPath = "./" + pluginSrcPath
	}

	cmd := exec.CommandContext(ctx, "go", "build", "-o", outPath, ".")
	cmd.Dir = pluginSrcPath
	cmd.Env = append(os.Environ(), "GOWORK=off")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	start := time.Now()
	if err := cmd.Run(); err != nil {
		log.Printf("[Plugin Watcher] ❌ Build failed for '%s': %v", pluginName, err)
		return false
	}
	log.Printf("[Plugin Watcher] ✅ Built %s in %v", binName, time.Since(start).Round(time.Millisecond))
	return true
}

func (w *PluginWatcher) triggerReload(ctx context.Context) {
	reloadURL := fmt.Sprintf("%s/api/v1/plugins/reload", w.hostURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reloadURL, nil)
	if err != nil {
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("[Plugin Watcher] ⚠️ Reload trigger sent (server offline or starting up): %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		log.Printf("[Plugin Watcher] 🔄 Reload trigger succeeded (200 OK)")
	} else {
		log.Printf("[Plugin Watcher] ⚠️ Reload status: %d", resp.StatusCode)
	}
}

func main() {
	pluginsDir := os.Getenv("PLUGINS_SRC_DIR")
	if pluginsDir == "" {
		pluginsDir = "plugins"
	}
	outDir := os.Getenv("KIYOMI_PLUGIN_DIR")
	if outDir == "" {
		outDir = "dev-plugins"
	}
	hostURL := os.Getenv("KIYOMI_HOST_URL")
	if hostURL == "" {
		hostURL = "http://localhost:8080"
	}

	watcher := NewPluginWatcher(pluginsDir, outDir, hostURL)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := watcher.RunWithContext(ctx); err != nil {
		log.Fatalf("[Plugin Watcher] Error: %v", err)
	}
}
