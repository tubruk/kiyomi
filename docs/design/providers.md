# Providers

> **Status**: Foundational design, scaffolding.

## Overview

Providers are external integrations that supply metadata, content, or tracking for manga. Each provider is an isolated module that exposes a defined capability set. Kiyomi does not ship with hardcoded assumptions about any specific provider; the contract is uniform and provider implementations are interchangeable within their capability class.

### Core Architecture Principle

The primary purpose of the **Provider SDK** (`pkg/provider/sdk/`) is to **abstract away each provider's internal implementation details and ever-changing upstream behavior** (such as HTML scraping quirks, custom date string formats, Cloudflare/anti-bot protection, or API payload variations).

* **Encapsulation of Upstream Quirks**: All upstream peculiarities, site layout changes, custom date parsing (e.g. converting `"Today"`, `"2 hours ago"`, or `"Oct 12,2023"` into standard `time.Time`), and raw site identifiers MUST be handled and normalized internally within the specific provider package.
* **Strict, Standardized Contracts**: The Provider SDK exposes uniform, strictly typed Go interfaces (`Metadata`, `Content`, `Tracking`) and standard Go types (such as `time.Time` for `UploadDate`, clean `sdk.Chapter`, and `sdk.Page`). No raw upstream structures or unparsed strings leak past the provider boundary into API handlers or storage layers.


## Capability Model

A provider declares which capabilities it implements. A provider may implement one or more.

| Capability | Provides | Example Providers |
|---|---|---|
| **Metadata** | Search, series details, cover art, synopsis, aliases | AniList, MyAnimeList, Kitsu |
| **Content** | Chapter lists, page lists, page image streams | MangaDex, MangaFox, local file import |
| **Tracking** | Push reading progress to external account | MyAnimeList, AniList |

A provider does not need to implement all three. For example, a metadata-only provider feeds the library, while a content-only provider feeds the reader. A library entry binds to one Content provider and zero or more Tracking providers.

## Conceptual Model

```
┌─────────────────────────────────────────────────────────┐
│                   Provider Registry                      │
│                                                          │
│   ┌────────────┐  ┌────────────┐  ┌────────────┐         │
│   │  Provider A │  │  Provider B │  │  Provider C │        │
│   │  (metadata) │  │  (content)  │  │  (tracking) │        │
│   └────────────┘  └────────────┘  └────────────┘         │
│         │                │                │               │
│         ▼                ▼                ▼               │
│   Capability:       Capability:       Capability:         │
│   Metadata          Content           Tracking            │
└─────────────────────────────────────────────────────────┘
```

## Capability Contracts

### Metadata Capability

```
MetadataProvider:
  search(query, options)         → SearchResult
  details(remote_id)            → MangaMetadata
  cover(remote_id, size?)       → ImageRef
  aliases(remote_id)            → [string]
```

`SearchResult` contains ranked manga entries with `remote_id` (provider-specific opaque handle).

`MangaMetadata` is a structured payload Kiyomi maps into local library fields. Provider-specific quirks (e.g. MAL's English title, AniList's romaji title) are normalized at the provider adapter boundary, not in the consumer.

### Content Capability

```
ContentProvider:
  has_stable_chapter_id()       → bool
  fetch_chapters(manga_ref)     → [Chapter]
  fetch_pages(manga_ref, chapter_ref)      → [Page]
  fetch_page_stream(page_ref)   → Stream
  rate_limit()                  → RateLimitHint
```

`has_stable_chapter_id()` declares whether chapter references survive renumbering. Drives refresh correlation strategy (see `docs/design/library.md`).

`Chapter` and `Page` are provider-shaped: the library design normalizes them on ingest.

### Tracking Capability

```
TrackingProvider:
  authenticate(user_creds)      → Session
  push_progress(remote_id, n)   → Ack
  fetch_progress(remote_id)     → Progress
  is_authenticated()            → bool
```

Tracking is stateless from the library's perspective. Authentication credentials are stored in user config (or system credential store), not in the library database.

## Identifier Conventions

Each capability uses an opaque `remote_id` for resource handles:

| Capability | Identifier Stability |
|---|---|
| Metadata | Stable (manga rarely renumbered on metadata sites) |
| Content (chapter) | Stable if `has_stable_chapter_id()=true`, fragile otherwise |
| Tracking | Stable (per-user external account) |

Providers expose `remote_id` as opaque, **URL-safe** strings. Provider implementations guarantee URL-safety for their identifiers. The main application (frontend UI and backend service handlers) MUST treat `remote_id` as a raw string and never perform base64 encoding, decoding, or format parsing. Cross-provider matching is handled at the library level. Chapter IDs (`chapter_ref`) MUST be clean, plain, URL-safe strings without Base64 encoding (e.g., plain UUIDs like `ch-1` or transformed path strings like `v01~c001~1.html`).

## Provider Authoring Guidelines

When implementing a content, metadata, or tracking provider plugin, authors MUST follow these core guidelines:

1. **URL-Safe Identifiers (`remote_id` & `chapter_ref`)**:
   - All resource identifiers (`remote_id` handles for manga, chapters, or pages, as well as `chapter_ref`) returned by a provider MUST be clean, plain, URL-safe strings without Base64 encoding.
   - Any raw upstream identifier containing slashes, query parameters, spaces, or non-ASCII characters MUST be sanitized, encoded, or converted into a URL-safe format inside the provider implementation itself (e.g., `v01~c001~1.html`).
   - **No Leakage**: ID transformation (encoding/decoding) is strictly an internal provider concern and MUST NEVER leak into the main application, REST API handlers, or frontend UI callers.

2. **Strict SDK Boundary (Built-in Plugins)**:
   - For built-in provider plugins compiled within the same binary/module, code MUST ONLY import `pkg/provider/sdk`. Direct dependencies on `internal/*` packages are forbidden to ensure easy extraction later into standalone WASM plugins or external modules.

3. **No Unhandled Panics**:
   - Providers MUST catch internal runtime errors and return typed `sdk.ProviderError` values rather than raising panics across the SDK boundary.

4. **Fingerprinted HTTP Client**:
   - Upstream HTTP requests MUST utilize the TLS browser fingerprinting engine (`pkg/fingerprint`) to avoid being flagged or blocked by anti-bot protections (Cloudflare, Akamai, DDOS-GUARD).

## Provider Lifecycle

```
┌─────────────────┐
│   Registered    │  ← code registers provider at startup
└─────────────────┘
        │
        │ user selects
        ▼
┌─────────────────┐
│     Active      │  ← provider responds to capability calls
└─────────────────┘
        │
        │ provider fails / unavailable
        ▼
┌─────────────────┐
│    Disabled     │  ← auto-disabled on repeated failure, user can re-enable
└─────────────────┘
```

Failure policy:
- Transient errors (timeout, 5xx) → retry with backoff
- Auth errors → surface to user, disable until re-authenticated
- Permanent errors (404, takedown) → mark affected resource, do not retry

## Configuration

Each provider may require configuration:

```
Provider:
  id                stable identifier (kitsu, mangadex, ...)
  display_name      user-facing name
  capabilities      [Metadata, Content, Tracking]
  requires_auth     bool
  config_keys       which keys expected (api_key, client_id, ...)
  rate_limit        max concurrent requests (per provider)
```

User config is per-provider, not global. One user may have MAL credentials without AniList credentials. Missing config = capability disabled, not error.

## Failure Isolation

A misbehaving provider must not crash the application. Each provider call runs in a recover boundary. Errors propagate as typed failures:

```
ProviderError:
  kind:        transient | auth | permanent | rate_limit
  provider_id: which provider
  message:     diagnostic detail
  retry_after: hint for rate_limit kind
```

Consumers (library, workers, API) handle each kind distinctly. No provider ever returns a Go panic across the boundary.

## Execution Model (Phase 1 vs Future)

In Phase 1, providers are **built-in Go packages** compiled directly into the same binary (no WASM module yet). All HTTP requests made by content providers utilize the TLS fingerprinting engine (`pkg/fingerprint`) to mirror browser signatures and evade anti-bot blocking.

*Future*: Out-of-process / WASM sandbox isolation for dynamic third-party plugins.

## Dependency Boundary (Critical)

**Built-in providers must not depend on any Kiyomi internal package other than the SDK.** Only the SDK is the public contract for providers.

### Allowed Imports

```
provider/<name>/
  ├── main file (provider impl)
  └── imports:
      └── pkg/provider/sdk       # public contract only
```

### Forbidden Imports

Any of the following in a built-in provider triggers immediate refactor:

- `internal/library/*` — database, schema, filesystem paths
- `internal/storage/*` — disk operations, cache
- `internal/api/*` — HTTP handlers
- `internal/model/*` — Kiyomi-internal data shapes (only SDK types allowed)
- `internal/worker/*` — River jobs, queue dispatch
- `cmd/kiyomi/*` — server bootstrap, config wiring

### Why This Rule Exists

1. **Future extraction to separate modules.** Built-in providers should be lift-and-shiftable into their own Go modules (or external repos) with minimal effort. A clean dependency cut makes that mechanical.
2. **Third-party providers cannot import Kiyomi internals.** If we want to allow external providers (out-of-process, marketplace), they don't have access to Kiyomi internals anyway. Built-in providers must obey the same constraint to be testable and fair.
3. **Forces correct abstraction.** If a provider needs something from internals, the SDK is missing a capability — fix the SDK, don't bypass it.
4. **Testability.** Provider tests should not pull in database connections, HTTP servers, or filesystem setup. Pure SDK mocks suffice.

### What To Do When SDK Is Missing

Provider author needs X from internals:
1. Add X to SDK as a capability or contract type.
2. SDK exposes minimal interface or data type.
3. Host (Kiyomi core) wires real implementation; provider receives SDK-level view.

Example: if a provider needs to know the local chapter path after download, SDK exposes a `DownloadTargetResolver` interface. Host injects the real resolver; provider tests inject a fake.

### Enforcement

- CI step: `go list -deps ./pkg/provider/<name>/...` must not include any `github.com/tubruk/kiyomi/internal/*` path.
- Code review checklist: "does this provider import only `pkg/provider/sdk` from Kiyomi?"
- Lint rule (golangci-lint): forbid `internal/` imports in `pkg/provider/**` package tree.

## Provider Examples

### Local File Provider

```
Capabilities:
  - Content (chapters from local folder structure)
  - Metadata (synthesized from folder names + ComicInfo.xml if present)

Special behavior:
  - has_stable_chapter_id() = true (paths don't change)
  - rate_limit() = very high (local)
  - requires_auth = false
```

### MangaDex

```
Capabilities:
  - Content (chapter/page API)

Special behavior:
  - has_stable_chapter_id() = true (UUIDs)
  - rate_limit() = 5 req/s
  - requires_auth = optional (higher rate with auth)
```

### AniList

```
Capabilities:
  - Metadata (search, details)
  - Tracking (push progress)

Special behavior:
  - rate_limit() = configurable, default 30 req/min
  - requires_auth = false for read, true for tracking
```

## Migration / Multi-Provider

A library entry binds to one Content provider at a time (per library design). The user may migrate to a different Content provider via the library UI. During migration:

- Old provider reference marked stale, not deleted immediately
- New provider reference becomes primary
- Refresh logic only queries primary
- History not stored in filesystem (single source of truth = current)

Tracking providers are independent of content. A library entry may bind to multiple tracking providers concurrently.

## Open Questions

1. Should providers declare their own preferred identifier format (e.g. UUID vs slug)? Host validates or trusts?
2. Should we support provider-provided UI customization (cover styling, custom fields)? Adds complexity, defer.
3. Multi-language providers — does Kitsu Japanese version differ from English? Provider-specific, exposed as separate provider ID or single provider with locale param?
4. Provider marketplace — third-party WASM providers downloadable by user? Future, defer.

## References

- [Library Storage Design](./library.md) — library binding, refresh correlation, `has_stable_chapter_id`
- [Plugin Developer Guide](../plugin_developer/README.md) — instructions for building custom provider plugins
- `docs/design/workers.md` — workers run provider calls with retry/rate-limit
- `docs/developer/architecture.md` — WASM plugin runtime
- `docs/developer/sources_sdk.md` — current SDK (will be superseded by this doc once stable)
