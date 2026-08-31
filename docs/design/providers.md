# Providers

## Overview

Providers are external integrations that supply metadata, content, or tracking for manga in Kiyomi. Each provider is an isolated module exposing a well-defined capability contract. Kiyomi does not ship with hardcoded assumptions about any specific provider; the contract is uniform and implementations are interchangeable within their capability class.

---

## Core Architectural Principles

1. **Encapsulation of Upstream Quirks**: All upstream peculiarities, HTML scraping quirks, custom date string parsing (e.g. converting `"Today"`, `"2 hours ago"`, or `"Oct 12, 2023"` into standardized UTC timestamps), and raw site identifiers are handled and normalized internally within the specific provider module.
2. **Strict, Standardized Contracts**: The Provider SDK exposes uniform capability interfaces (`Metadata`, `Content`, `Tracking`) and normalized data types (`SearchResult`, `MangaMetadata`, `Chapter`, `Page`). No raw upstream payloads or unparsed strings leak past the provider boundary into API handlers or storage layers.
3. **URL-Safe Opaque Identifiers**: All resource identifiers (`remote_id` for manga and `chapter_ref` for chapters) returned by a provider MUST be clean, plain, URL-safe strings without Base64 encoding. Upstream identifiers with special characters or slashes are sanitized internally within the provider adapter.
4. **Anti-Bot & TLS Fingerprinting**: Upstream HTTP requests utilize browser TLS fingerprinting to replicate browser TLS Client Hello signatures and headers, bypassing Cloudflare and anti-bot obstacles.

---

## Capability Model

A provider declares which capabilities it implements. A single provider may implement one, two, or all three capabilities:

| Capability | Provided Functions | Example Providers |
|---|---|---|
| **Metadata** | Search, series details, cover art URLs, synopsis, genres, authors, aliases | AniList, MyAnimeList, Kitsu |
| **Content** | Chapter listings, page lists, page image streams, stable chapter IDs | MangaDex, MangaFox, Local File Provider |
| **Tracking** | Read progress synchronization, status push/pull to user accounts | MyAnimeList, AniList |

```
┌─────────────────────────────────────────────────────────┐
│                   Provider Registry                     │
│                                                         │
│   ┌────────────┐  ┌────────────┐  ┌────────────┐        │
│   │ Provider A │  │ Provider B │  │ Provider C │        │
│   │ (metadata) │  │ (content)  │  │ (tracking) │        │
│   └─────┬──────┘  └─────┬──────┘  └─────┬──────┘        │
│         │               │               │               │
│         ▼               ▼               ▼               │
│   Capability:      Capability:     Capability:          │
│     Metadata         Content         Tracking           │
└─────────────────────────────────────────────────────────┘
```

A manga entry in the library binds to zero or more providers concurrently via `providers[]`, with one active content provider namespace designated in `content`.

---

## Capability Contracts & Specifications

### 1. Metadata Capability
Enables discovering manga, querying series descriptions, fetching canonical covers, and resolving alternative titles:

```yaml
MetadataProvider:
  Search(query: string, options: SearchOptions) -> list of SearchResult
  Details(remote_id: string) -> MangaMetadata
```

#### Search Options & Modes:
- `query`: Text search term (empty string when browsing feeds).
- `mode`: Browsing mode (`popular` for top-ranked series, `latest` for recently updated releases).
- `limit` / `offset`: Pagination parameters.

#### Schema Specifications:

```yaml
SearchResult:
  remote_id: string
  title: string
  cover_url: string
  availability: string  # optional status indicator

MangaMetadata:
  remote_id: string
  title: string
  aliases: list of string
  synopsis: string
  author: string
  artist: string
  genres: list of string
  cover_url: string
  status: string        # ongoing, completed, hiatus, cancelled
  reading_mode: string  # rtl, ltr, vertical, longstrip
  total_chapters: integer
```

---

### 2. Content Capability
Enables querying chapter releases, resolving page image URLs, and streaming image bytes:

```yaml
ContentProvider:
  HasStableChapterID() -> boolean
  FetchChapters(manga_remote_id: string) -> list of Chapter
  FetchPages(manga_remote_id: string, chapter_remote_id: string) -> list of Page
```

#### Key Elements:
- `HasStableChapterID()`: Declares whether chapter references survive upstream renumbering. Drives refresh and merge correlation strategies.
- Chapter and page identifiers are opaque, URL-safe strings (e.g. `ch-101` or `v01~c001~1.html`).

#### Schema Specifications:

```yaml
Chapter:
  id: string            # URL-safe chapter reference
  name: string          # Display title (e.g., "Chapter 1: The Beginning")
  number: float         # Normalized chapter number (e.g., 1.0, 10.5)
  volume: integer       # Volume number (0 if unassigned)
  upload_date: timestamp # UTC upload timestamp
  url: string           # Upstream web link
  source_order: integer # Upstream index order

Page:
  index: integer        # 1-based page index
  url: string           # Direct or proxied page image URL
  headers: map[string, string] # Required request headers (Referer, User-Agent)
```

---

### 3. Tracking Capability
Enables bidirectional synchronization of reading progress with external tracking platforms:

```yaml
TrackingProvider:
  Authenticate(credentials: AuthCredentials) -> Session
  PushProgress(manga_remote_id: string, progress: ChapterProgress) -> Acknowledgement
  FetchProgress(manga_remote_id: string) -> RemoteProgress
  IsAuthenticated() -> boolean
```

Tracking operates statelessly relative to the core library storage. User authentication credentials and tokens are stored in user configuration rather than the library database.

---

## Concurrency & Rate Limiting

Content providers enforce rate-limiting to prevent IP bans and respect upstream API quotas:

1. **Concurrency Group Keys**: All background pull jobs targeting a provider are scoped to a concurrency group key: `pull:<provider_id>` (e.g. `pull:mangadex`).
2. **Scheduler Throttling**: The background job queue scheduler ensures at most `N` concurrent tasks run per provider key across the worker pool.
3. **Adaptive Backoff**: When a rate-limit error (`429 Too Many Requests`) is encountered, the provider returns a typed error with a `retry_after` duration, allowing the scheduler to delay subsequent attempts.

---

## Error Handling & Classification

Providers encapsulate all upstream HTTP failures, timeouts, and challenge pages into structured, typed error classifications:

| Error Kind | Cause | Handling Strategy |
|---|---|---|
| `transient` | Network timeout, socket disconnect, HTTP 500/502/503 | Automatic retry with exponential backoff in background queue |
| `auth` | Expired credentials, invalid API token, missing login | Surface notification to user; pause tracking sync |
| `rate_limit` | HTTP 429, upstream throttle quota reached | Back off according to `retry_after` header |
| `permanent` | HTTP 404 Not Found, DMCA takedown, invalid resource ID | Fail immediately without retry; flag resource |

```yaml
ProviderError:
  kind: string        # transient | auth | rate_limit | permanent
  provider_id: string # Identifier of the failing provider
  message: string     # Human-readable diagnostic description
  retry_after: integer # Optional backoff hint in seconds
```

---

## Execution & Plugin Architecture

Kiyomi supports dual execution models for providers:

1. **Built-in Providers**: Compiled directly into the main application binary for zero-dependency execution.
   - **Strict SDK Boundary**: Built-in providers adhere strictly to the public SDK capability contracts and standard libraries without coupling to internal server subsystems, ensuring seamless extraction into standalone plugins.
2. **Dynamic Host Plugins**: Loaded at runtime by the plugin host manager from WebAssembly modules or standalone binaries.
   - Discovered and hot-reloaded dynamically from the configured `plugins/` directory.
   - Collisions between built-in and plugin providers are resolved through user preferences configured via `/api/v1/plugins/preference`.

---

## Multi-Content Provider Storage

When multiple content providers are bound to a manga, each provider maintains an isolated chapter folder namespace under the local library:

```
<library_root>/<manga_id>/<provider_id>/<chapter_id>/
```

- Switching the active content provider updates `manga.content.provider_id` without deleting or modifying existing chapters downloaded from other providers.
- Orphaned chapters (chapters downloaded under a previously active provider) remain intact on disk for offline reading.

---

## References

- [Multi-Content Provider Library](./multi_content_provider_library.md) — Multi-provider filesystem layout and chapter directory structure.
- [Filesystem-First Library](./library.md) — Manga and chapter manifest schemas and refresh correlation.
- [Background Job Queue](./background_jobs.md) — Generic background task runner, concurrency groups, and queue schema.
- [Content Pull System](./content_pull.md) — Granular chapter and page acquisition pipeline.
- [Reader Design](./reader.md) — 3-tier page resolution and reading modes.
- [Anti-Bot Strategy](./antibot.md) — TLS fingerprint matching and cookie injection.
- [REST API](./api.md) — Provider catalog, fingerprint, and binding endpoints.
- [Plugin Developer Guide](../plugin_developer/_index.md) — Instructions for developing standalone provider plugins.

