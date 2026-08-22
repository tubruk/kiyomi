# Provider Plugin Developer Guide

This guide describes how to author custom provider plugins for Kiyomi. Custom plugins allow you to extend Kiyomi with new metadata sources, content providers (scrapers for online catalogs), or tracking services (syncing reading progress).

---

## Architecture Overview

Kiyomi uses HashiCorp's [go-plugin](https://github.com/hashicorp/go-plugin) system. Plugins run as independent, out-of-process OS subprocesses that communicate with the main Kiyomi host application via gRPC over standard I/O streams.

This design provides:
* **Decoupling**: Write plugins using any Go version or dependencies, completely decoupled from Kiyomi's internal code.
* **Safety**: Crashes or panics in a plugin do not bring down the main Kiyomi process.
* **Hot-Reloading**: Plugins can be reloaded, replaced, or configured at runtime without restarting the main server.

---

## Getting Started

### 1. Import the Plugin SDK

Every plugin must import the Kiyomi Plugin SDK:

```go
import (
	sdk "github.com/tubruk/kiyomi/plugin-sdk"
)
```

The SDK contains all interface definitions, Protobuf service schemas, HTTP utilities, scraper helpers, and error types.

### 2. Plugin Structure

A typical plugin project is structured as follows:

```
my-custom-plugin/
├── go.mod
├── main.go             # Binary entry point (sdk.ServePlugin)
├── plugin.go           # Plugin struct and Init lifecycle
├── metadata_provider.go# MetadataProvider interface implementation
├── content_provider.go # ContentProvider interface implementation
└── client.go           # Internal site API/scraper client
```

#### `main.go`
Instantiates your plugin struct and launches the gRPC server:

```go
package main

import (
	"github.com/tubruk/kiyomi/plugin-sdk"
	"github.com/myusername/my-custom-plugin/internal/plugin"
)

func main() {
	myPlugin := plugin.NewMyPlugin()
	sdk.ServePlugin(myPlugin)
}
```

#### `plugin.go`
Implements the lifecycle methods of the `sdk.Plugin` interface:

```go
type MyPlugin struct {
	// Add config, HTTP clients, cache, etc.
}

func (p *MyPlugin) Describe(ctx context.Context) (sdk.PluginDescriptor, error) {
	return sdk.PluginDescriptor{
		PluginID:      "my-custom-plugin",
		PluginName:    "My Custom Plugin",
		PluginVersion: "0.1.0",
		SDKVersion:    sdk.Version,
		Providers: []sdk.ProviderDescriptor{
			{
				ID:           "my-source",
				Name:         "My Manga Source",
				Description:  "Retrieves manga from My Source website",
				Capabilities: []string{"content", "metadata"},
			},
		},
	}, nil
}

func (p *MyPlugin) Init(ctx context.Context, config sdk.PluginConfig) error {
	// Parse configurations, initialize HTTP clients, etc.
	return nil
}
```

---

## Implementing Capabilities

Your plugin exposes one or more capabilities by implementing the corresponding Go interfaces defined in the SDK.

### Content Provider

If your plugin provides manga chapters and pages, implement `sdk.ContentProvider`:

```go
type ContentProvider interface {
	HasStableChapterID() bool
	FetchChapters(ctx context.Context, mangaRef string) ([]Chapter, error)
	FetchPages(ctx context.Context, mangaRef, chapterRef string) ([]Page, error)
	FetchPageStream(ctx context.Context, page Page) (io.ReadCloser, error)
	RateLimit() RateLimitHint
}
```

### Metadata Provider

If your plugin searches manga or provides rich catalog details, implement `sdk.MetadataProvider`:

```go
type MetadataProvider interface {
	Search(ctx context.Context, query string, opts SearchOptions) ([]SearchResult, error)
	Details(ctx context.Context, remoteID string) (MangaMetadata, error)
	Cover(ctx context.Context, remoteID string, size ImageSize) (ImageRef, error)
	Aliases(ctx context.Context, remoteID string) ([]string, error)
}
```

---

## Best Practices

1. **URL-Safe Identifiers**: Keep all manga, chapter, and page references URL-safe and plain (e.g. plain IDs, slug paths). Avoid base64 encoding inside the core app; sanitize references inside your plugin client instead.
2. **Robust Error Handling**: Return standardized SDK errors (e.g., `sdk.ErrNotFound` or custom typed errors) rather than panic. All panics are intercepted, but proper errors allow the UI to show helpful diagnostic messages.
3. **Structured Logging**: Write logs using the structured logger provided in the SDK. Captured logs are routed back to the host, visible in the Web UI Diagnostics log view.
4. **Evade Anti-Bots**: Use the fingerprinted HTTP client transport helper provided in the SDK (`plugin-sdk/http`) to query upstream servers, maintaining correct TLS fingerprints to avoid Cloudflare/Akamai blockages.

---

## Building and Installing

To compile your plugin, run:

```bash
go build -o my-custom-plugin .
```

To install the plugin:
1. Place the compiled binary file in the plugins folder of your `$KIYOMI_HOME` directory (defaults to `$KIYOMI_HOME/plugins`).
2. Grant executable permissions to the binary: `chmod +x $KIYOMI_HOME/plugins/my-custom-plugin`.
3. Restart Kiyomi or trigger reload from the UI under **Settings → Plugins**.

---

## References & Further Reading

* [Design Spec: Plugin Architecture](../design/provider_plugin_architecture.md)
* [Design Spec: Providers & SDK Contract](../design/providers.md)
* [DNS Overrides](./dns_overrides.md) — opt-in/opt-out DNS resolver override via env var or wire config
* [First-Party Examples](../../plugins)
