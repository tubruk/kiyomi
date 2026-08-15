// Package webui owns the static asset bundle for the Kiyomi web client.
//
// The bundle is embedded into the Go binary at compile time via //go:embed
// so a release build is a single self-contained executable with no
// runtime dependency on the filesystem. For development, callers can
// pass a non-empty DevDir to FS() to instead serve the most recent
// build from disk (typically pointed at <home>/web/dist) so a `bun run
// build` in web/ takes effect without recompiling the Go server.
//
// Build pipeline
//
// The web SPA lives in web/ and is built with `bun run build` which
// produces web/dist. To embed that build into the Go binary, run
// `go generate ./pkg/webui/...` after the web build. The generator
// copies web/dist/* into pkg/webui/dist/, replacing the placeholder
// index.html. Then `go build` embeds the result.
//
// The placeholder dist/index.html is committed so a fresh clone
// builds cleanly even before the web bundle has been produced.
package webui

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
)

// Source string used in the FS() return value so callers can log which
// mode is active. Values are stable and safe to compare with ==.
const (
	SourceEmbedded  = "embedded"
	SourceDevLocal  = "dev-local"
)

// distFS is the embedded web UI bundle. The path is relative to this
// file; `go:embed` requires the directory to exist at build time, so
// web/dist must always be present (even if empty) for the build to
// succeed. CI / first-time builds can `mkdir -p web/dist` to satisfy
// this requirement.
//
//go:embed all:dist
var distFS embed.FS

// distSub is the fs.FS view rooted at webui/dist (the embedded
// directory itself, not its parent package directory). It is
// computed once at package init.
var distSub fs.FS

func init() {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		// fs.Sub on an embed.FS with a known-good root cannot fail at
		// runtime in practice, but if it ever does we want to know.
		panic(fmt.Sprintf("webui: failed to sub embed root: %v", err))
	}
	distSub = sub
}

// FS returns an http.FileSystem for the web UI bundle and a short
// string describing where it was loaded from. The devDir argument
// controls the source:
//
//   - devDir == "": serve the embedded bundle. This is the default for
//     release binaries.
//   - devDir != "": serve the directory at devDir from the
//     filesystem. Used during development so a freshly built
//     web/dist is picked up without recompiling the Go server.
//
// The returned fs is always non-nil. If devDir is non-empty but does
// not point at a usable directory, FS falls back to the embedded
// bundle and returns SourceEmbedded together with an error so the
// caller can log the override failure.
func FS(devDir string) (http.FileSystem, string, error) {
	devDir = strings.TrimSpace(devDir)
	if devDir == "" {
		return http.FS(distSub), SourceEmbedded, nil
	}

	info, err := os.Stat(devDir)
	if err != nil {
		return http.FS(distSub), SourceEmbedded, fmt.Errorf("webui: dev dir %q unavailable, falling back to embedded: %w", devDir, err)
	}
	if !info.IsDir() {
		return http.FS(distSub), SourceEmbedded, fmt.Errorf("webui: dev dir %q is not a directory, falling back to embedded", devDir)
	}

	abs, err := filepath.Abs(devDir)
	if err != nil {
		abs = devDir
	}
	slog.Info("webui: serving assets from filesystem (dev mode)", slog.String("path", abs))
	return http.Dir(abs), SourceDevLocal, nil
}

// ErrNoIndex is returned by IndexFile when the bundle has no top-level
// index.html. Callers should treat it as a configuration error.
var ErrNoIndex = errors.New("webui: bundle has no index.html")

// IndexBytes returns the bytes of the bundle's index.html. It is
// useful for SPA handlers that need to synthesize a response when
// no static asset matches the request. The devDir argument matches
// FS(): empty = embedded, non-empty = filesystem directory.
func IndexBytes(devDir string) ([]byte, error) {
	devDir = strings.TrimSpace(devDir)
	if devDir == "" {
		return fs.ReadFile(distSub, "index.html")
	}
	return os.ReadFile(filepath.Join(devDir, "index.html"))
}

// Register registers static file serving and SPA fallback routing on the Echo instance.
func Register(e *echo.Echo) {
	devDir := os.Getenv("KIYOMI_WEB_DIR")
	fsys, _, err := FS(devDir)
	if err != nil {
		slog.Warn("webui: failed to initialize web UI filesystem, falling back to embedded", slog.String("error", err.Error()))
	}

	fileServer := http.FileServer(fsys)

	e.GET("/*", func(c echo.Context) error {
		path := c.Request().URL.Path

		// Let the router handle API routes; this is a catch-all route,
		// but as a sanity check we do not intercept anything under /api/
		if strings.HasPrefix(path, "/api/") {
			return echo.ErrNotFound
		}

		// Try to open the file in the filesystem to see if it exists
		f, err := fsys.Open(strings.TrimPrefix(path, "/"))
		if err == nil {
			f.Close()
			fileServer.ServeHTTP(c.Response(), c.Request())
			return nil
		}

		// File not found; serve index.html for SPA routing
		indexBytes, err := IndexBytes(devDir)
		if err != nil {
			return c.String(http.StatusInternalServerError, "index.html not found")
		}
		return c.HTML(http.StatusOK, string(indexBytes))
	})
}
