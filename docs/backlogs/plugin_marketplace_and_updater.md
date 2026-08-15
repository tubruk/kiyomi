# Backlog: Plugin Marketplace / Repository & Auto-Updater System

> **Status**: Proposed / Backlog  
> **Target Components**: `internal/plugin/marketplace`, `internal/api`, `internal/config`, `web/src`  
> **Related Design Documents**: [`docs/design/provider_plugin_architecture.md`](file:///Users/akhyar.amarullah/.gemini/antigravity-cli/brain/c1ba8040-3aa9-4b1c-90e2-dc2a9c891a2e/.system_generated/worktrees/subagent-Backlog-Specification-Author-self-e332dc30/docs/design/provider_plugin_architecture.md)

---

## 1. Overview & Architectural Vision

Kiyomi's modular provider architecture enables out-of-process gRPC plugins that can be compiled, versioned, and executed independently of the main server process. However, manual plugin discovery, downloading, executable permissions setting, and updating require terminal access or direct filesystem interaction.

This specification outlines the architecture for an end-to-end **Plugin Marketplace, Multi-Repository Distribution, and Auto-Updater System**. The system allows users to browse, install, configure, update, and remove plugin binaries directly from the Kiyomi Web UI with 1-click workflows, backed by strict cryptographic integrity verification and zero-downtime hot-reloading.

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                          Remote Marketplace Indices                             │
│                                                                                 │
│   ┌────────────────────────────────┐       ┌────────────────────────────────┐   │
│   │   Official Kiyomi Repository   │       │   Community 3rd-Party Repos    │   │
│   │  (https://plugins.kiyomi.app)  │       │     (Custom JSON Index URLs)   │   │
│   └────────────────┬───────────────┘       └────────────────┬───────────────┘   │
└────────────────────┼────────────────────────────────────────┼───────────────────┘
                     │                                        │
                     ▼                                        ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                       Kiyomi Backend (internal/plugin/marketplace)               │
│                                                                                 │
│  ┌───────────────────────────────────────────────────────────────────────────┐  │
│  │                    Repository Manager & Cache Engine                      │  │
│  │     - Multi-repo index fetching      - In-memory & disk caching (TTL)     │  │
│  └─────────────────────────────────────┬─────────────────────────────────────┘  │
│                                        │                                        │
│          ┌─────────────────────────────┴─────────────────────────────┐          │
│          ▼                                                           ▼          │
│  ┌───────────────────────────────┐                   ┌───────────────────────┐  │
│  │       Installer Engine        │                   │  Auto-Update Engine   │  │
│  │ - OS/Arch target resolution   │                   │ - Background cron     │  │
│  │ - Staged stream download      │                   │ - SemVer comparison   │  │
│  │ - Permissions (chmod 0755)    │                   │ - Update policies:    │  │
│  │ - Atomic file swap (Rename)   │                   │   (Off/Notify/Auto)   │  │
│  └───────────────┬───────────────┘                   └───────────┬───────────┘  │
│                  │                                               │              │
│                  ▼                                               │              │
│  ┌───────────────────────────────┐                               │              │
│  │ Security & Integrity Verifier │                               │              │
│  │ - SHA-256 checksum digest     │                               │              │
│  │ - Ed25519 signature check     │                               │              │
│  └───────────────┬───────────────┘                               │              │
│                  │                                               │              │
│                  ▼                                               ▼              │
│  ┌───────────────────────────────────────────────────────────────────────────┐  │
│  │                   PluginManager.Reload() / Hot-Swap                       │  │
│  │                Zero-downtime gRPC subprocess recreation                   │  │
│  └───────────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### Key Architectural Invariants & Security Guarantees
1. **Zero Unverified Executions**: Remote binaries MUST pass both **SHA-256 digest calculation** and **Ed25519 digital signature verification** before any executable permissions (`chmod 0755`) are applied or subprocesses spawned.
2. **Atomic Installation & Safe Rollback**: Downloads write to temporary staging files (`<plugin_id>.tmp.<uuid>`). If verification fails or download is interrupted, the staging file is purged and the running plugin remains untouched. Upon successful verification, the file is atomically renamed (`os.Rename`) into the plugin directory.
3. **Decentralized Multi-Repository Support**: Users can subscribe to the Official Kiyomi repository as well as arbitrary community repository index URLs.
4. **Zero-Downtime Hot-Swapping**: Updates trigger `PluginManager.ReloadPlugin(ctx, pluginID)` without requiring a full server restart.
5. **Configurable Background Auto-Updates**: Supports three distinct user policies: `disabled`, `notify` (badge/toast), and `auto-apply`.

---

## 2. Plugin Index & Manifest Schemas

### 2.1 Repository Index Schema (`index.json`)

A plugin repository serves a statically hosted `index.json` file over HTTPS (e.g., via GitHub Pages, Cloudflare R2, or AWS S3):

```json
{
  "$schema": "https://kiyomi.app/schemas/v1/plugin-index.json",
  "version": 1,
  "repository_name": "Official Kiyomi Plugin Repository",
  "repository_url": "https://plugins.kiyomi.app/index.json",
  "generated_at": "2026-08-16T12:00:00Z",
  "public_key": "MCowBQYDK2VwAyEANkP8w6W+62kG/F2b13wVfQ0qZ74h5xL8kY10eJ9M3qw=",
  "plugins": [
    {
      "id": "mangadex-plugin",
      "name": "MangaDex Provider",
      "description": "High-speed provider for MangaDex catalog, chapters, and high-res scanlations.",
      "author": "Kiyomi Core Team",
      "repository_url": "https://github.com/tubruk/kiyomi-plugin-mangadex",
      "version": "1.2.0",
      "sdk_version": "0.1.0",
      "min_app_version": "0.1.0",
      "tags": ["manga", "manhwa", "official", "multilingual"],
      "icon_url": "https://plugins.kiyomi.app/icons/mangadex.svg",
      "binaries": {
        "darwin-arm64": {
          "url": "https://github.com/tubruk/kiyomi-plugin-mangadex/releases/download/v1.2.0/mangadex-plugin-darwin-arm64",
          "sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
          "signature": "R1hVd21hZ...base64-ed25519-signature...",
          "size_bytes": 14680064
        },
        "darwin-amd64": {
          "url": "https://github.com/tubruk/kiyomi-plugin-mangadex/releases/download/v1.2.0/mangadex-plugin-darwin-amd64",
          "sha256": "4b227777d4dd1fc61c6f884f48641d02b4d121d3fd328cb08b5531fcacdabf8a",
          "signature": "V2hsYW...base64-ed25519-signature...",
          "size_bytes": 15204352
        },
        "linux-amd64": {
          "url": "https://github.com/tubruk/kiyomi-plugin-mangadex/releases/download/v1.2.0/mangadex-plugin-linux-amd64",
          "sha256": "ef2d127de37b942baad06145e54b0c619a1f22327b2ebbcfbec78f5564afe39d",
          "signature": "SGVsbG8...base64-ed25519-signature...",
          "size_bytes": 14942208
        },
        "linux-arm64": {
          "url": "https://github.com/tubruk/kiyomi-plugin-mangadex/releases/download/v1.2.0/mangadex-plugin-linux-arm64",
          "sha256": "5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8",
          "signature": "Qnl0ZXM...base64-ed25519-signature...",
          "size_bytes": 14417920
        },
        "windows-amd64": {
          "url": "https://github.com/tubruk/kiyomi-plugin-mangadex/releases/download/v1.2.0/mangadex-plugin-windows-amd64.exe",
          "sha256": "8f434346648f6b96df89dda901c5176b10a6d83961dd3c1ac88b59b2dc327aa4",
          "signature": "V2luZG9...base64-ed25519-signature...",
          "size_bytes": 15728640
        }
      }
    }
  ]
}
```

### 2.2 Go Model Definitions (`internal/plugin/marketplace/models.go`)

```go
package marketplace

import (
	"time"
)

// RepositoryIndex represents a remote plugin catalog index.
type RepositoryIndex struct {
	Version        int              `json:"version"`
	RepositoryName string           `json:"repository_name"`
	RepositoryURL  string           `json:"repository_url"`
	GeneratedAt    time.Time        `json:"generated_at"`
	PublicKey      string           `json:"public_key"` // Base64 Ed25519 public key
	Plugins        []PluginManifest `json:"plugins"`
}

// PluginManifest defines metadata, compatibility, and binaries for a single marketplace plugin.
type PluginManifest struct {
	ID            string                       `json:"id"`
	Name          string                       `json:"name"`
	Description   string                       `json:"description"`
	Author        string                       `json:"author"`
	RepositoryURL string                       `json:"repository_url"`
	Version       string                       `json:"version"`
	SDKVersion    string                       `json:"sdk_version"`
	MinAppVersion string                       `json:"min_app_version,omitempty"`
	Tags          []string                     `json:"tags,omitempty"`
	IconURL       string                       `json:"icon_url,omitempty"`
	Binaries      map[string]BinaryReleaseSpec `json:"binaries"` // key format: "{os}-{arch}"
}

// BinaryReleaseSpec defines downloadable binary artifacts and integrity checksums.
type BinaryReleaseSpec struct {
	URL       string `json:"url"`
	SHA256    string `json:"sha256"`
	Signature string `json:"signature"` // Base64 Ed25519 signature
	SizeBytes int64  `json:"size_bytes"`
}

// RepositoryConfig stores configured repository URLs in host configuration.
type RepositoryConfig struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	Enabled   bool      `json:"enabled"`
	IsDefault bool      `json:"is_default"`
	Trusted   bool      `json:"trusted"`
	PublicKey string    `json:"public_key,omitempty"`
	AddedAt   time.Time `json:"added_at"`
}
```

---

## 3. Core Architecture & Components

### 3.1 Repository Manager (`internal/plugin/marketplace/repository.go`)

The `RepositoryManager` manages repository sources, downloads index files over HTTPS, applies in-memory and disk caching, and aggregates results.

```go
package marketplace

import (
	"context"
	"sync"
	"time"
)

type RepositoryManager interface {
	// ListRepositories returns all active configured repository sources.
	ListRepositories(ctx context.Context) ([]RepositoryConfig, error)
	// AddRepository adds a custom community repository URL.
	AddRepository(ctx context.Context, repo RepositoryConfig) error
	// RemoveRepository removes a custom repository URL.
	RemoveRepository(ctx context.Context, id string) error
	// FetchMergedIndex fetches and combines plugins from all enabled repositories with caching.
	FetchMergedIndex(ctx context.Context, forceRefresh bool) ([]MarketplacePluginSummary, error)
}
```

#### Responsibilities:
1. **Multi-Source Aggregation**: Iterates over all enabled repository endpoints, fetching indices in parallel with `errgroup`.
2. **Caching Strategy**:
   - Stores indices in memory with a default TTL of 1 hour.
   - Persists a local backup cache on disk (`cache/marketplace_index.json`) for offline startup resilience.
   - Supports `forceRefresh=true` query param to bypass cache on manual user request.
3. **Resilience**: If a 3rd-party community repository times out or returns 5xx, errors are logged via `slog.Warn` and healthy repositories continue to populate the catalog.

---

### 3.2 Security & Cryptographic Verifier (`internal/plugin/marketplace/verifier.go`)

The `Verifier` performs rigorous cryptographic validation before a binary is saved to the filesystem or executed.

```go
package marketplace

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

var (
	ErrChecksumMismatch  = errors.New("binary SHA256 checksum mismatch")
	ErrSignatureInvalid  = errors.New("cryptographic signature verification failed")
	ErrUntrustedKey      = errors.New("repository public key is untrusted")
)

type Verifier struct {
	trustedPublicKeys map[string]ed25519.PublicKey
}

func (v *Verifier) VerifyChecksum(computedHash string, expectedHash string) error {
	if computedHash != expectedHash {
		return fmt.Errorf("%w: expected %s, got %s", ErrChecksumMismatch, expectedHash, computedHash)
	}
	return nil
}

func (v *Verifier) VerifySignature(rawPayload []byte, signatureBase64 string, publicKeyBase64 string) error {
	pubBytes, err := base64.StdEncoding.DecodeString(publicKeyBase64)
	if err != nil || len(pubBytes) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid ed25519 public key format: %w", err)
	}

	sigBytes, err := base64.StdEncoding.DecodeString(signatureBase64)
	if err != nil || len(sigBytes) != ed25519.SignatureSize {
		return fmt.Errorf("invalid ed25519 signature format: %w", err)
	}

	if !ed25519.Verify(pubBytes, rawPayload, sigBytes) {
		return ErrSignatureInvalid
	}
	return nil
}
```

#### Verification Pipeline:
1. **Streaming Digest**: During HTTP download, content is piped through `io.TeeReader` and `sha256.New()`, calculating the SHA256 digest on the fly without loading the full binary into memory.
2. **Signature Verification**: Verifies the binary digest using the repository's registered Ed25519 public key.
3. **Official vs Community Policy**:
   - **Official Kiyomi Repo**: Shipped with hardcoded trusted public key.
   - **Community Repos**: The public key declared in `index.json` is displayed to the user for one-time confirmation ("Trust this repository").

---

### 3.3 Installer Engine (`internal/plugin/marketplace/installer.go`)

The `Installer` orchestrates the target binary selection, streaming download, verification, atomic file placement, and hot-reloading.

```go
package marketplace

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"github.com/google/uuid"
	"github.com/tubruk/kiyomi/internal/plugin/host"
)

type Installer struct {
	pluginsDir    string
	verifier      *Verifier
	pluginManager *host.PluginManager
	httpClient    *http.Client
}
```

#### Step-by-Step Installation Algorithm:
1. **Platform Resolution**:
   - Resolves key `<runtime.GOOS>-<runtime.GOARCH>` (e.g. `darwin-arm64`, `linux-amd64`, `windows-amd64`).
   - If not found in manifest `binaries`, returns `ErrUnsupportedPlatform` with clear diagnostic context.
2. **Staging Directory**:
   - Creates a temporary file in `<pluginsDir>/.staging/<plugin_id>-<uuid>.tmp`.
3. **HTTP Streaming Download**:
   - Downloads binary chunk-by-chunk with context deadline and cancellation support.
   - Pipes to `sha256.New()` hasher.
4. **Integrity & Security Check**:
   - Compares computed SHA-256 with manifest `sha256`.
   - Verifies Ed25519 signature with `verifier.VerifySignature`.
5. **Permissions & Atomic Replacement**:
   - Sets executable bit: `os.Chmod(stagingPath, 0755)` on POSIX systems.
   - On Windows, ensures binary filename ends with `.exe`.
   - Performs atomic move: `os.Rename(stagingPath, targetExecutablePath)`.
6. **Hot-Reload Trigger**:
   - Calls `pluginManager.ReloadPlugin(ctx, pluginID)` to immediately boot the new gRPC subprocess and update the host's `ProviderRegistry`.

---

### 3.4 Auto-Updater Engine (`internal/plugin/marketplace/updater.go`)

The `Updater` runs background checks for new plugin versions, checks compatibility, and applies updates based on configured user policy.

```go
package marketplace

import (
	"context"
	"log/slog"
	"time"
)

type UpdatePolicy string

const (
	UpdatePolicyDisabled UpdatePolicy = "disabled"
	UpdatePolicyNotify   UpdatePolicy = "notify"
	UpdatePolicyAuto     UpdatePolicy = "auto_apply"
)

type UpdaterConfig struct {
	Policy        UpdatePolicy  `json:"policy"`         // "disabled" | "notify" | "auto_apply"
	CheckInterval time.Duration `json:"check_interval"` // e.g. 6h, 12h, 24h
}

type AutoUpdater struct {
	config        UpdaterConfig
	repoManager   RepositoryManager
	installer     *Installer
	notifications chan UpdateNotification
}
```

#### SemVer & SDK Compatibility Rules:
- **Version Check**: Evaluates if `manifest.Version > installed.Version` using standard Semantic Versioning 2.0.0 rules.
- **SDK Compatibility Gate**: Before auto-applying an update, checks `host.CheckSDKCompatibility(sdk.Version, manifest.SDKVersion)`. If the update targets an incompatible SDK, auto-update is skipped and a warning notification is surfaced.
- **Update Workflow**:
  - `UpdatePolicyDisabled`: Periodic timer is idle.
  - `UpdatePolicyNotify`: Emits an `UpdateAvailable` event surfaced in the UI navbar badge and plugins page.
  - `UpdatePolicyAuto`: Automatically initiates download, verification, atomic swap, and triggers `PluginManager.Reload()`. Logs success via `slog.Info`.

---

## 4. REST API Specifications (`internal/api/marketplace_handler.go`)

All marketplace REST APIs live under `/api/v1/marketplace/` and follow Kiyomi's standard JSON response envelope and error handling standards.

### 4.1 Summary of API Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/marketplace/plugins` | List catalog plugins with install/update status |
| `POST` | `/api/v1/marketplace/install` | Download, verify, install/update a plugin |
| `POST` | `/api/v1/marketplace/uninstall` | Terminate subprocess and remove plugin executable |
| `GET` | `/api/v1/marketplace/repositories` | List configured repository sources |
| `POST` | `/api/v1/marketplace/repositories` | Add or update a repository source |
| `DELETE` | `/api/v1/marketplace/repositories/:id` | Remove a custom repository source |
| `GET` | `/api/v1/marketplace/updates/check` | Force an immediate update check |
| `GET` | `/api/v1/marketplace/settings` | Get auto-updater configuration & policy |
| `POST` | `/api/v1/marketplace/settings` | Update auto-updater configuration & policy |

---

### 4.2 Endpoint Details & Contracts

#### `GET /api/v1/marketplace/plugins`
Query Parameters:
- `force_refresh` (bool, optional): Bypasses index cache if `true`.
- `query` (string, optional): Text search query filtering name, description, tags, author.
- `tag` (string, optional): Filter by tag.

**Response (`200 OK`)**:
```json
[
  {
    "id": "mangadex-plugin",
    "name": "MangaDex Provider",
    "description": "High-speed provider for MangaDex catalog and scanlations.",
    "author": "Kiyomi Core Team",
    "repositoryUrl": "https://github.com/tubruk/kiyomi-plugin-mangadex",
    "repositorySource": "Official Kiyomi Repository",
    "latestVersion": "1.2.0",
    "sdkVersion": "0.1.0",
    "sdkCompatible": true,
    "isInstalled": true,
    "installedVersion": "1.1.0",
    "updateAvailable": true,
    "tags": ["manga", "official"],
    "iconUrl": "https://plugins.kiyomi.app/icons/mangadex.svg",
    "downloadSizeBytes": 14680064
  }
]
```

---

#### `POST /api/v1/marketplace/install`
**Request Body**:
```json
{
  "pluginId": "mangadex-plugin",
  "version": "1.2.0",
  "repositoryUrl": "https://plugins.kiyomi.app/index.json"
}
```

**Response (`200 OK`)**:
```json
{
  "status": "ok",
  "message": "Plugin 'mangadex-plugin' v1.2.0 installed and reloaded successfully",
  "pluginId": "mangadex-plugin",
  "installedVersion": "1.2.0",
  "activeProviders": ["mangadex"]
}
```

**Error Responses**:
- `400 Bad Request`: Incompatible SDK version or unsupported OS/Arch.
- `422 Unprocessable Entity`: Checksum mismatch or invalid Ed25519 signature.
- `500 Internal Server Error`: Staging write or file rename failure.

---

#### `POST /api/v1/marketplace/uninstall`
**Request Body**:
```json
{
  "pluginId": "mangadex-plugin"
}
```

**Response (`200 OK`)**:
```json
{
  "status": "ok",
  "message": "Plugin 'mangadex-plugin' uninstalled successfully",
  "pluginId": "mangadex-plugin"
}
```

---

#### `GET /api/v1/marketplace/repositories` & `POST /api/v1/marketplace/repositories`
**Request Body (`POST`)**:
```json
{
  "name": "Community Manga Hub",
  "url": "https://raw.githubusercontent.com/example/kiyomi-plugins/main/index.json",
  "trusted": true
}
```

**Response (`200 OK`)**:
```json
{
  "status": "ok",
  "repository": {
    "id": "repo-550e8400-e29b-41d4-a716-446655440000",
    "name": "Community Manga Hub",
    "url": "https://raw.githubusercontent.com/example/kiyomi-plugins/main/index.json",
    "enabled": true,
    "isDefault": false,
    "trusted": true,
    "addedAt": "2026-08-16T12:00:00Z"
  }
}
```

---

## 5. Web UI Design & User Experience (`web/src/routes/PluginsSettingsPage.tsx`)

The Plugins Settings page (`/settings/plugins`) is updated to feature a cohesive tabbed interface:

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│ 🧩 Plugins & Marketplace                                                        │
│ Manage installed provider binaries, browse marketplace extensions, and updates  │
├─────────────────────────────────────────────────────────────────────────────────┤
│ [ Installed Plugins (3) ]  [ Marketplace Catalog (12) ]  [ Repositories & Auto-Update ]│
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                 │
│  🔍 Search plugins, authors, or tags...           [ Category: All ▼ ] [ ↻ Refresh ] │
│                                                                                 │
│  ┌─────────────────────────────────┐   ┌─────────────────────────────────┐      │
│  │ 📘 MangaDex Provider   [Official]│   │ 🦊 MangaFox Provider  [Community]│     │
│  │ By Kiyomi Core Team • v1.2.0    │   │ By FanCommunity • v0.9.4        │      │
│  │ MangaDex catalog & scanlations  │   │ Classic manga aggregator source │      │
│  │                                 │   │                                 │      │
│  │ Status: Update Available (1.1.0)│   │ Status: Not Installed           │      │
│  │ [ ⬆ Update to v1.2.0 (14 MB) ]  │   │ [ ⬇ 1-Click Install (12 MB) ]   │      │
│  └─────────────────────────────────┘   └─────────────────────────────────┘      │
│                                                                                 │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### 5.1 Tab Breakdown
1. **Installed Plugins Tab**:
   - Retains current functionality: Active gRPC binaries, PID, memory diagnostics, scoped config editor, collision resolver, live stdout/stderr ring buffer viewer.
   - Adds **Update Available Alert Badge** on cards with an inline **[Update]** action.
2. **Marketplace Catalog Tab**:
   - Filterable grid with search bar, tag pills (`manga`, `webtoons`, `metadata`, `official`).
   - Cards display: Plugin name, description, author, repository badge, size, SDK compatibility indicator.
   - Primary Action Button:
     - **[Install]**: If not yet installed.
     - **[Update to vX.Y.Z]**: If installed version < latest version.
     - **[Installed ✓]**: If up-to-date (with dropdown for [Reinstall] or [Uninstall]).
3. **Repositories & Auto-Update Tab**:
   - **Repository Manager**: List of index URLs, status, trust toggles, and **[ + Add Repository ]** modal.
   - **Auto-Update Policy Selector**: Radio buttons for `Disabled`, `Notify Only`, and `Auto-Apply`.
   - **Check Interval**: Dropdown for `Every 6 Hours`, `Every 12 Hours`, `Every 24 Hours`.
   - **Immediate Check Action**: `[ Check for Updates Now ]` button.

### 5.2 Interactive States & Toast Feedback
- **Installation Progress**: Displays inline spinner and button state (`"Downloading (45%)..." -> "Verifying Signature..." -> "Reloading..." -> "Installed"`).
- **Error Diagnostics**: If checksum or signature verification fails, uses Kiyomi's standard error pattern: concise toast notification + **Details** action opening the `ErrorDetailsModal` with full cryptographic error trace.

---

## 6. Implementation Phasing & Acceptance Criteria

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          Phased Implementation Roadmap                      │
│                                                                             │
│  Phase 1: Schemas, Index Parser, Verifier & Repository Engine              │
│  ├─ Manifest & Index JSON structs                                           │
│  ├─ Ed25519 & SHA256 Verifier                                               │
│  └─ Multi-Repository Index Fetcher & Cache                                  │
│                                                                             │
│  Phase 2: Installer Engine, Auto-Updater & REST Handlers                    │
│  ├─ Platform-aware HTTP downloader & atomic swap                            │
│  ├─ SemVer updater cron & policy runner                                     │
│  └─ Echo REST Handlers (/api/v1/marketplace/*)                              │
│                                                                             │
│  Phase 3: Web UI Marketplace Catalog, Settings & E2E Verification           │
│  ├─ TanStack Query marketplace hooks & API client                           │
│  ├─ Marketplace Catalog & Repositories Tab in PluginsSettingsPage           │
│  └─ Unit, integration, and E2E verification tests                           │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Phase 1: Schemas, Index Parser, Verifier & Repository Engine
- [ ] Create `internal/plugin/marketplace/models.go` with `RepositoryIndex`, `PluginManifest`, `BinaryReleaseSpec`, and `RepositoryConfig`.
- [ ] Implement `internal/plugin/marketplace/verifier.go` with streaming SHA-256 digest validation and Ed25519 signature checks.
- [ ] Implement `internal/plugin/marketplace/repository.go` with multi-repository index fetching, disk caching, and aggregation.
- [ ] **Verification**: Unit tests covering valid signatures, tampered payloads, corrupted checksums, and unreachable repository fallback.

### Phase 2: Installer Engine, Auto-Updater & REST Handlers
- [ ] Implement `internal/plugin/marketplace/installer.go` with platform detection (`runtime.GOOS`/`GOARCH`), staging download, `chmod 0755`, atomic `os.Rename`, and `PluginManager.ReloadPlugin` integration.
- [ ] Implement `internal/plugin/marketplace/updater.go` with background ticker, SemVer 2.0 comparison, SDK version compatibility checks, and policy dispatching (`disabled`, `notify`, `auto_apply`).
- [ ] Implement `internal/api/marketplace_handler.go` with all routes (`/api/v1/marketplace/*`) registered in `internal/api/handler.go`.
- [ ] **Verification**: Integration tests with mock HTTP index server verifying end-to-end install, uninstall, and update workflows. Backend tests pass: `go test -v ./...`.

### Phase 3: Web UI Marketplace Catalog, Settings & Verification
- [ ] Add marketplace API methods in `web/src/api/client.ts` and TanStack Query options in `web/src/lib/queryOptions.ts`.
- [ ] Build `MarketplaceCatalogTab` and `RepositorySettingsTab` in `web/src/routes/PluginsSettingsPage.tsx`.
- [ ] Implement 1-click Install / Update / Uninstall buttons with loading indicators and error modal integration.
- [ ] **Verification**: Frontend build verification with `bun run build` inside `web/` and full test suite passing.

---

## 7. Verification & Testing Strategy

1. **Unit Testing**:
   - `internal/plugin/marketplace/verifier_test.go`: Test Ed25519 signature generation and verification across valid and tampered binary fixtures.
   - `internal/plugin/marketplace/repository_test.go`: Test multi-repository index merging, caching expiry, and error handling for unreachable endpoints.
   - `internal/plugin/marketplace/updater_test.go`: Test SemVer diffing and policy execution logic.
2. **API Integration Testing**:
   - `internal/api/marketplace_handler_test.go`: Mock repository HTTP responses, verify JSON response contracts, status codes, and error payloads.
3. **End-to-End Verification**:
   - Run complete Go test suite: `go test -v ./...`.
   - Run Web UI build: `bun run build` in `web/`.
