# Kiyomi Plugin Architecture & Independent Lifecycle

> **Status**: Approved Design Specification (Grill-Me Outcome - Complete)

## 1. Overview & Architectural Goals

This specification defines the architecture for extending Kiyomi with plugins that can be developed, versioned, released, and hot-reloaded independently of the main Kiyomi backend application.

### Key Goals
1. **Module & Boundary Isolation**: Clear separation between main app (`cmd/kiyomi`, `internal/*`), Plugin SDK (`plugin-sdk`), and plugins.
2. **Monorepo & Multi-Repo Flexibility**: Main app and built-in plugins reside in the primary repo. 1st-party standalone plugins can also live in the main monorepo but release independently. 3rd-party developers can build in external repos.
3. **Multi-Provider & Multi-Feature Bundling**: A single plugin binary can contain and serve multiple providers (e.g. `MangaDex`, `MangaFox`, `MangaKakalot`) or future plugin capabilities (image processors, storage engines).
4. **Hot-Reloadability**: Plugins can be reloaded dynamically at runtime without restarting the main Kiyomi backend process.
5. **Dependency & Compiler Decoupling**: Plugins and main app can use different Go dependency versions or third-party packages as long as they target compatible Plugin SDK versions.
6. **Multi-Tier SDK**: High-level declarative helpers (DOM/JSON scrapers) for rapid development alongside low-level Go code interfaces for full programmatic control.

---

## 2. Standardized Terminology

| Term | Definition |
|---|---|
| **Plugin** | A loadable executable package (binary) created by developers containing one or more providers or capabilities. |
| **Provider** | A specific source capability implementation inside a plugin (e.g. metadata for AniList, content for MangaDex, or tracking for MyAnimeList). |
| **Plugin SDK** | Standardized Go contracts (`plugin-sdk`), Protobuf schemas, gRPC boilerplate, HTTP client engine, and helper utilities. |

---

## 3. Execution & Transport Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           Kiyomi Main App                               │
│                                                                         │
│  ┌───────────────────────────────────────────────────────────────────┐  │
│  │                         Plugin Registry                           │  │
│  └───────────────────────────────────────────────────────────────────┘  │
│          │                                         │                    │
│          ▼ (Direct Go Calls)                       ▼ (gRPC over Stdio)  │
│  ┌───────────────────────┐             ┌─────────────────────────────┐  │
│  │    Built-in Plugins   │             │  HashiCorp `go-plugin` Host │  │
│  │ (In-Process Direct)   │             └─────────────────────────────┘  │
│  └───────────────────────┘                            │                 │
└───────────────────────────────────────────────────────┼─────────────────┘
                                                        │ Stdio / IPC
                                                        ▼
                                         ┌─────────────────────────────┐
                                         │   External Plugin Binary    │
                                         │                             │
                                         │  ┌───────────┐ ┌─────────┐  │
                                         │  │ ProviderA │ │ProviderB│  │
                                         │  └───────────┘ └─────────┘  │
                                         └─────────────────────────────┘
```

---

## 4. Plugin SDK Modular Subpackage Structure

To keep the core SDK interface lean and decoupled, all helper utilities are organized into **modular SDK subpackages** under `/plugin-sdk/`:

```
plugin-sdk/
├── go.mod             # Independent lightweight SDK module (github.com/tubruk/kiyomi/plugin-sdk)
├── version.go         # const Version = "0.1.0"
├── provider.go        # Core MetadataProvider, ContentProvider, Tracker contracts
├── proto/             # gRPC Protobuf schemas
├── scraper/           # Declarative HTMLSource & JSONSource DOM scrapers
├── http/              # uTLS fingerprinted HTTP client with host proxy auto-wiring
├── errors/            # Standardized domain error types (ErrNotFound, ErrRateLimited, etc.)
├── cache/             # Memory TTL cache helper subpackage for session/auth tokens
├── paginator/         # Offset & cursor pagination helper subpackage
├── logger/            # Structured slog transport helper subpackage
└── utils/             # Date parsing & URL resolution helpers
```

### 3rd-Party Developer Import Experience
Third-party developers import explicitly and cleanly:
```go
import (
    "github.com/tubruk/kiyomi/plugin-sdk"
    "github.com/tubruk/kiyomi/plugin-sdk/http"
    "github.com/tubruk/kiyomi/plugin-sdk/scraper"
)
```

### Go Module Major Versioning (`v0` → `v1` → `v2+`)
- **`v0.x.x` & `v1.x.x`**: `module github.com/tubruk/kiyomi/plugin-sdk` (Git tags `plugin-sdk/v0.1.0`, `plugin-sdk/v1.0.0`).
- **`v2.x.x+`**: `module github.com/tubruk/kiyomi/plugin-sdk/v2` (Git tag `plugin-sdk/v2.0.0`, import `github.com/tubruk/kiyomi/plugin-sdk/v2`).

---

## 5. Subprocess Log & Stdio Capture Architecture

Because plugins run as out-of-process binaries, HashiCorp `go-plugin` handles `stdout` and `stderr` interception automatically:

```
┌─────────────────────────┐               ┌─────────────────────────┐
│     Plugin Subprocess   │               │     Kiyomi Host App     │
│                         │  stderr / IPC │                         │
│  slog.Info(...)         ├──────────────►│ PluginManager Interceptor│
│  fmt.Println(...)       │               └────────────┬────────────┘
└─────────────────────────┘                            │
                                                       ├──► Host Terminal Logs (slog)
                                                       │    Tagged with [plugin_id=xxx]
                                                       │
                                                       └──► In-Memory Ring Buffer (200 lines)
                                                            Exposed via GET /api/v1/plugins/{id}/logs
```

1. **Automatic Stdio Interception**: HashiCorp `go-plugin` captures `stderr` and `stdout` log output from the plugin process via stdio pipes.
2. **Nested Structured Logging (`slog.Group`)**:
   - The SDK logger handler serializes structured `slog.Record` instances (including nested `slog.Group` attributes and maps) as structured JSON events across stdio pipes.
   - The host's `PluginManager` deserializes records preserving all nested groups and key-values without flattening or truncation, injecting host attributes (`plugin_id=<id>`).
3. **In-Memory Ring Buffer**: The host maintains a per-plugin ring buffer (last 200 lines). The Web UI **Diagnostics & Logs Modal** fetches these logs directly via `GET /api/v1/plugins/{id}/logs`.

---

## 6. Repository & Module Boundaries

### Repository Directory Structure
```
kiyomi/
├── go.mod                      # Main app go.mod (cmd/kiyomi, internal/*)
├── go.work                     # Go workspace linking main app, SDK, & standalone plugins
├── cmd/
│   └── kiyomi/                 # Main server binary entry point
├── internal/
│   ├── api/                    # REST API handlers
│   ├── library/                # Filesystem storage & domain logic
│   └── provider/               # Built-in in-process providers
├── plugin-sdk/                 # Dedicated Go module (plugin-sdk/go.mod)
│   ├── go.mod                  # module github.com/tubruk/kiyomi/plugin-sdk
│   └── ...
└── plugins/                    # Standalone 1st-party plugins monorepo folder
    ├── mangadex/
    │   ├── go.mod              # Standalone plugin go.mod
    │   ├── main.go             # Lightweight entry point calling sdk.ServePlugin
    │   ├── plugin.go           # Plugin struct, Describe(), and Init() config parsing
    │   ├── metadata_provider.go# sdk.MetadataProvider implementation (Search, Details, Cover, Aliases)
    │   ├── content_provider.go # sdk.ContentProvider implementation (FetchChapters, FetchPages, RateLimit)
    │   ├── client.go           # Upstream API client and HTTP request helpers
    │   ├── types.go            # Upstream API DTOs and JSON payload structs
    │   └── mangadex_test.go    # Unit and gRPC integration tests
    └── mangafox/
        ├── go.mod
        ├── main.go
        ├── plugin.go
        ├── metadata_provider.go
        ├── content_provider.go
        ├── client.go
        ├── types.go
        └── mangafox_test.go
```

### Recommended Plugin File Structure
For maintainability, consistency, and separation of concerns, plugins (both 1st-party and 3rd-party) should separate their responsibilities into distinct modular files under the plugin package:

| File | Purpose / Responsibilities |
|---|---|
| `main.go` | Lightweight binary entry point. Instantiates the plugin struct and calls `sdk.ServePlugin(...)`. |
| `plugin.go` | Plugin lifecycle definitions: plugin struct, constants, constructor (`New...Plugin`), `Describe(ctx)`, and `Init(ctx, config)` settings parser. |
| `metadata_provider.go` | Implements `sdk.MetadataProvider` methods (`Search`, `Details`, `Cover`, `Aliases`). |
| `content_provider.go` | Implements `sdk.ContentProvider` methods (`HasStableChapterID`, `RateLimit`, `FetchChapters`, `FetchPages`, `FetchPageStream`). |
| `client.go` | Upstream network interaction: HTTP clients, scraper engines, DOM queries, URL resolvers, request builders, and data extractors. |
| `types.go` | Internal upstream API request/response JSON schemas, parser DTOs, and regular expression patterns. |
| `<name>_test.go` | Comprehensive unit tests for metadata extraction, chapter/page resolution, settings handling, and `bufconn` gRPC integration. |

---

## 7. Self-Describing Plugins & Scoped Config Protobuf Schema

### Protobuf Schema (`proto/plugin.proto`)
```protobuf
syntax = "proto3";
package kiyomi.plugin.v1;

service PluginService {
  rpc Describe (DescribeRequest) returns (DescribeResponse);
  rpc Init (InitRequest) returns (InitResponse);
}

message DescribeResponse {
  string sdk_version = 1;          // Automatically set by SDK const Version
  string plugin_id = 2;            // e.g. "community-manga-pack"
  string plugin_name = 3;          // e.g. "Community Manga Pack"
  string plugin_version = 4;       // e.g. "1.2.0"
  repeated SettingSpec plugin_settings_schema = 5; // Global plugin settings
  repeated ProviderDescriptor providers = 6;       // Contained providers
}

message ProviderDescriptor {
  string id = 1;                    // e.g. "mangadex"
  string name = 2;                  // e.g. "MangaDex"
  string description = 3;
  repeated string capabilities = 4; // ["metadata", "content", "tracking"]
  repeated SettingSpec settings_schema = 5; // Scoped to this specific provider ID
  RateLimitSpec default_rate_limit = 6;
}

message RateLimitSpec {
  int32 requests_per_second = 1;
  int32 max_concurrent_requests = 2;
}

message SettingSpec {
  string key = 1;
  string label = 2;
  string description = 3;
  string type = 4; // "string", "number", "boolean", "secret", "select"
  string default_value = 5;
  repeated string options = 6;
}

message InitRequest {
  map<string, string> global_config = 1;                  // Shared plugin-wide settings
  map<string, ProviderConfigMap> provider_configs = 2;    // Map of provider_id -> { key -> value }
  GlobalHttpConfig http_config = 3;                       // System-wide proxy & User-Agent
}

message ProviderConfigMap {
  map<string, string> settings = 1;                       // Key-value settings for this provider ID
}

message GlobalHttpConfig {
  string proxy_url = 1;
  string user_agent = 2;
  int32 timeout_seconds = 3;
}
```

---

## 8. Rate Limiting & Concurrency Management Hierarchy

1. **Provider Defaults (Declared by Plugin/SDK)**: Each provider declares safe upstream defaults via `RateLimitSpec` in `Describe()`.
2. **SDK Transport Enforcement**: The Plugin SDK's HTTP engine enforces rate limiters and semaphores automatically.
3. **Host & User Overrides**: Users override rate limits per provider in the Web UI settings modal (managed by the host and passed via `Init()`).

---

## 9. Version Updates, Incompatibility & Conflict Handling

### A. Version Update Hot-Swapping
1. **Zero Downtime**: Existing instance serves active requests while the updated binary completes `go-plugin` handshake & `Init()`.
2. **Graceful Drain**: In-flight gRPC calls drain cleanly.
3. **Atomic Swap**: Host `ProviderRegistry` swaps routing tables atomically. Old subprocess is terminated.

### B. Conflict & Compatibility Strategies
1. **Provider ID Collisions**: Precedence: In-Process Built-in > User Explicit Preference > Highest SemVer Version. Non-selected registered as namespaced alias (`mangadex@plugin-b`).
2. **SDK Version Compatibility & Pre-v1 (`0.x.x`) SemVer Rules**: Exact minor version match for `0.x.x` releases (`v0.1.x` host accepts `v0.1.y`, rejects `v0.2.y`).

---

## 10. Web UI Plugin Management Specification

Located under **Settings → Plugins** (`/settings/plugins`):

### UI Components
1. **Plugin Cards List**: Displays installed plugin binaries, author, SDK compatibility badge, and contained providers.
2. **Scoped Settings Modal**: Toggles between **Plugin-Wide Settings** and **Provider Settings**.
3. **Conflict Resolution UI**: Warning banner for Provider ID collisions with preference dropdown.
4. **Hot-Reload Action**: `[ ↻ Reload Plugins ]` button calling `POST /api/v1/plugins/reload`.
5. **Diagnostics & Log Modal**: Displays process PID, memory usage, and real-time log output stream fetched from `GET /api/v1/plugins/{id}/logs`.

---

## 11. Phased Implementation Roadmap

- **Phase 1: Plugin SDK Module & Protobuf Contracts (`plugin-sdk/`)**
  - Create dedicated Go sub-module at `plugin-sdk/go.mod` (`module github.com/tubruk/kiyomi/plugin-sdk`).
  - Define `const Version = "0.1.0"` in `plugin-sdk/version.go`.
  - Create modular subpackages: `plugin-sdk/cache`, `plugin-sdk/paginator`, `plugin-sdk/errors`, `plugin-sdk/logger`.
  - Define Protobuf specs for `PluginService`, `MetadataProvider`, `ContentProvider`, and `Tracker`.
  - Implement gRPC client/server boilerplate & `sdk.ServePlugin(...)`.

- **Phase 2: Host Plugin Manager & Log Interceptor (`internal/plugin/host`)**
  - Integrate HashiCorp `go-plugin` host client.
  - Build `PluginManager` with stdio log interceptor & in-memory log ring buffer (preserving nested `slog.Group` attributes).
  - Implement dual-mode `ProviderRegistry` with version update hot-swapping & collision handling.

- **Phase 3: Multi-Tier Declarative Scraper SDK & Fingerprinted Client (`plugin-sdk/scraper`, `plugin-sdk/http`)**
  - Implement `plugin-sdk/scraper` declarative helpers.
  - Implement `plugin-sdk/http` with uTLS TLS fingerprinting, proxy-aware HTTP transport, and rate-limiting wrappers.

- **Phase 4: Hot-Reloading API & Web UI Integration**
  - Implement `POST /api/v1/plugins/reload` and `GET /api/v1/plugins/{id}/logs` endpoints.
  - Create `/settings/plugins` Web UI page with dynamic settings modals, provider collision resolution UI, reload action, and diagnostic log viewers.

- **Phase 5: Provider Migration & Monorepo Setup**
  - Update `go.work` to tie main app, `plugin-sdk/`, and `plugins/*`.
  - Move standalone 1st-party providers (e.g. MangaDex/MangaFox) into `plugins/` as separate Go modules.
