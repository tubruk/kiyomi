---
name: kiyomi-provider
description: Guidelines, architectures, interfaces, and testing practices for developing metadata, content, and tracking providers (plugins) in Kiyomi.
---

# Kiyomi Provider / Plugin Development Skill

This skill defines the operational standards, interface specifications, utility helper usage, and testing methodologies for building metadata, content, and tracking provider plugins in the Kiyomi project.

---

## 1. Architecture Overview

Kiyomi providers are decoupled, out-of-process Go plugin binaries built with HashiCorp `go-plugin` and gRPC over standard I/O:
- **Location**: Standalone plugins live under `plugins/<plugin_id>/` as separate Go modules (e.g. `plugins/mangadex`, `plugins/mangafox`, `plugins/myanimelist`).
- **SDK**: All plugins import and build against `github.com/tubruk/kiyomi/plugin-sdk`.
- **Workspace Integration**: Every plugin module MUST be declared in the repository root `go.work` file.
- **Host Discovery**: The Kiyomi server discovers and manages plugin subprocesses from `dev-plugins/` (or configured `KIYOMI_PLUGIN_DIR`) via its internal `PluginManager`.

---

## 2. Plugin Interfaces & Capabilities

Each plugin is an executable that implements the base `sdk.Plugin` lifecycle interface, along with one or more provider capability interfaces defined in [`plugin-sdk/provider.go`](./plugin-sdk/provider.go).

### 2.1 Base `sdk.Plugin` Lifecycle
```go
type Plugin interface {
	Describe(ctx context.Context) (PluginDescriptor, error)
	Init(ctx context.Context, config PluginConfig) error
}
```
* **`Describe`**: Returns self-describing metadata: `PluginID`, `PluginName`, `PluginVersion`, `SDKVersion`, configuration schemas (`SettingSpec`), and advertised `Providers` with their capabilities (`"metadata"`, `"content"`, `"tracking"`).
* **`Init`**: Receives host-provided configurations, HTTP parameters (proxy, DNS resolvers, user agent, timeout), and provider-specific settings.

### 2.2 `sdk.MetadataProvider` Capability
Supplies series discovery, details, cover art, and title aliases:
```go
type MetadataProvider interface {
	Search(ctx context.Context, query string, opts SearchOptions) ([]SearchResult, error)
	Details(ctx context.Context, remoteID string) (MangaMetadata, error)
	Cover(ctx context.Context, remoteID string, size ImageSize) (ImageRef, error)
	Aliases(ctx context.Context, remoteID string) ([]string, error)
}
```

### 2.3 `sdk.ContentProvider` Capability
Supplies chapter lists, page lists, and page image streams:
```go
type ContentProvider interface {
	HasStableChapterID() bool
	FetchChapters(ctx context.Context, mangaRef string) ([]Chapter, error)
	FetchPages(ctx context.Context, mangaRef, chapterRef string) ([]Page, error)
	FetchPageStream(ctx context.Context, page Page) (io.ReadCloser, error)
	RateLimit() RateLimitHint
}
```

### 2.4 `sdk.Tracker` Capability
Synchronizes reading progress with external accounts (e.g. AniList, MyAnimeList, Kitsu):
```go
type Tracker interface {
	Authenticate(ctx context.Context, creds UserCredentials) (Session, error)
	PushProgress(ctx context.Context, remoteID string, n int) error
	FetchProgress(ctx context.Context, remoteID string) (Progress, error)
	IsAuthenticated(ctx context.Context) bool
}
```

---

## 3. Plugin Boilerplate & Entry Point

Every plugin in `plugins/<plugin_id>/` defines a `main.go` using `sdk.ServePlugin`:

```go
package main

import (
	sdk "github.com/tubruk/kiyomi/plugin-sdk"
)

func main() {
	plug := NewMyPlugin()

	sdk.ServePlugin(sdk.ServeOptions{
		Plugin: plug,
		MetadataProviders: map[string]sdk.MetadataProvider{
			PluginID: plug,
		},
		ContentProviders: map[string]sdk.ContentProvider{
			PluginID: plug,
		},
	})
}
```

---

## 4. HTTP Client & Scraper Utilities

Plugins should use the utilities provided in `plugin-sdk`:

### 4.1 Configured HTTP Client (`plugin-sdk/http`)
Initialize HTTP clients during `Init` using `sdkhttp.NewClient`:
```go
httpClient := sdkhttp.NewClient(
	sdkhttp.WithSDKGlobalHttpConfig(config.HTTPConfig),
	sdkhttp.WithTimeout(timeout),
).StandardClient()
```

### 4.2 Logging (`plugin-sdk/logger`)
Logs written to stderr via `sdklogger` are automatically captured, formatted, and forwarded through the host's log ring buffer and dashboard:
```go
logger := sdklogger.New(os.Stderr, &sdklogger.Options{Level: slog.LevelInfo})
```

### 4.3 HTML & JSON Scrapers (`plugin-sdk/scraper`)
- `scraper.NewHTMLSource(cfg)`: Provides document fetching, CSS selectors, cookie jars, and URL resolution for HTML-based scrapers.
- `scraper.NewJSONSource(cfg)`: Helper for structured REST API consumption.

---

## 5. Provider Authoring Guidelines

### 5.1 Abstracting Upstream Behavior
- **Core Principle**: The primary purpose of the Provider SDK is to abstract away each provider's internal quirks and upstream behaviors (HTML scraping differences, site layout variations, custom date formats, or anti-bot protections).
- **Internal Normalization**: All upstream site peculiarities (such as parsing relative dates like `"Today"` or `"2 hours ago"`, custom date formats into `time.Time`, or comic reading modes) MUST be normalized internally within the plugin.
- **Strict Native Types**: Provider methods MUST only return standard SDK types (`time.Time` for `UploadDate`, `sdk.Chapter`, `sdk.Page`, `sdk.MangaMetadata`). Raw upstream structures or unparsed strings MUST NEVER leak past the provider boundary.

### 5.2 URL-Safe Clean Chapter IDs (`chapter_ref`)
- Chapter IDs (`chapter_ref` / `remote_id`) MUST be clean, plain, URL-safe strings without Base64 wrappers (e.g. plain UUIDs or sanitized slugs like `ch-1` or `v01-c001`).
- Base64 encoding/decoding (`sdk.EncodeID` / `sdk.DecodeID`) MUST NOT be used for chapter IDs.
- Any raw upstream identifier containing slashes, query parameters, or spaces MUST be sanitized inside the provider implementation.

---

## 6. Testing & Verification

1. **Unit Tests**: Place tests in `<plugin>_test.go` using `httptest.NewServer` to mock upstream REST APIs or HTML pages.
2. **gRPC Integration Testing**: Test the plugin service and provider bindings over in-memory gRPC buffers (`google.golang.org/grpc/test/bufconn`):
   ```go
   lis := bufconn.Listen(1024 * 1024)
   server := grpc.NewServer()
   v1.RegisterPluginServiceServer(server, &sdk.GRPCPluginServer{Impl: plug})
   v1.RegisterMetadataProviderServiceServer(server, &sdk.GRPCMetadataProviderServer{
       Providers: map[string]sdk.MetadataProvider{PluginID: plug},
   })
   go func() { _ = server.Serve(lis) }()
   defer server.Stop()
   ```
3. **Verification Commands**:
   - In plugin module: `go test -v ./...`
   - Whole repository: `go test -v ./...`
   - Frontend build: `cd web && bun run build`
   - Plugin compilation: `go build -o ../../dev-plugins/<plugin_id>-plugin .`
