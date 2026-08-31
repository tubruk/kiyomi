# Filesystem-First Library Architecture

## Overview

Kiyomi implements a **filesystem-first, library-centric model**. The local filesystem is the authoritative source of truth for all manga metadata, reading progress, and content. External metadata and content providers serve as optional enrichment and acquisition services.

### Core Principle

> **The filesystem is the sole source of truth.**

This architecture provides:
- **Offline operation**: The entire library and downloaded content remain fully accessible without internet access or active provider connections.
- **Native backup and migration**: Backing up, transferring, or synchronizing the library requires only standard filesystem operations (e.g. `cp -r library/`).
- **Resilience and self-healing**: In-memory and SQLite cache indexes can be rebuilt entirely by scanning manifests on disk.
- **Multi-provider co-existence**: Chapters from multiple content providers live side-by-side without naming collisions or fragile cross-provider correlations.

---

## Directory Layout

The library directory organizes manga and chapters hierarchically by manga ID and content provider ID:

```
<library_root>/
└── <manga_id>/
    ├── meta.json                     # Manga manifest
    ├── cover.<ext>                   # Cover image (e.g. cover.jpg, cover.webp)
    ├── banner.<ext>                  # Banner image (optional)
    └── <provider_id>/                # Grouped by content provider (e.g. mangadex, mangafox, local)
        └── <chapter_id>/             # Local chapter ID
            ├── meta.json             # Chapter manifest
            ├── pages.json            # Page list manifest
            ├── 1.jpg                 # Page images (1-based index)
            ├── 2.jpg
            └── ...
```

### Identifier Conventions

Kiyomi maintains four distinct identifier spaces:

| Identifier | Generation / Format | Storage Location | Scope |
|---|---|---|---|
| **Local Manga ID** (`manga_id`) | ULID or URL-safe slug | Folder name (`<library_root>/<manga_id>/`) | Local filesystem |
| **Provider ID** (`provider_id`) | Alphanumeric identifier (`mangadex`, `mangafox`, `local`) | Folder name (`<library_root>/<manga_id>/<provider_id>/`) | System-wide |
| **Local Chapter ID** (`chapter_id`) | ULID or provider chapter reference | Folder name (`<provider_id>/<chapter_id>/`) | Local provider namespace |
| **Provider Remote ID** (`provider_manga_id` / `chapter_ref`) | Opaque upstream string | `meta.json` (`providers[]`, `content`) | Upstream provider |

#### Rules:
1. **Implicit Manga ID**: The folder name `<manga_id>` is authoritative; no redundant `manga_id` field is stored at the top level of `meta.json`.
2. **Provider Isolation**: Chapters are partitioned under `<provider_id>/` subdirectories. Different providers for the same manga never collide or overwrite each other's chapter files.
3. **The `local` Provider**: The special provider ID `local` is reserved for chapters imported manually (e.g. CBZ/ZIP extractions, local scans, or provider-less chapters).
4. **URL Safety**: Provider identifiers and remote references must be URL-safe strings without base64 wrapper encoding.

### Page File Naming and Formats

- Page image files are named by 1-based page index with their native extension: `1.jpg`, `2.png`, `3.webp`, etc. (or zero-padded equivalents such as `001.jpg`).
- Supported image extensions: `.jpg`, `.jpeg`, `.png`, `.webp`, `.gif`, `.avif`.
- `pages.json` stores the ordered list of page URLs and resolution sources.
- Partial downloads are detected by counting existing on-disk page images against `meta.json` `page_count`.

---

## Metadata Schemas

### Manga Manifest (`<library_root>/<manga_id>/meta.json`)

The manga manifest defines series metadata, provider bindings, and user tracking information.

```json
{
  "title": "Sample Manga",
  "aliases": ["Sample", "サンプル"],
  "description": "A detailed description of the manga series.",
  "authors": ["Yamada, Kanehito"],
  "artists": ["Abe, Tsukasa"],
  "tags": ["type:manga", "demographic:shounen", "Fantasy", "Magic"],
  "collections": ["Favorites"],
  "content_rating": "safe",
  "publisher": "Shonen Jump",
  "release_year": 2020,
  "start_date": "2020-04-06",
  "end_date": "",
  "country": "JP",
  "cover_url": "https://uploads.mangadex.org/covers/abc/cover.jpg",
  "external_links": [
    {
      "provider": "mangadex",
      "label": "MangaDex",
      "url": "https://mangadex.org/title/abc123"
    }
  ],
  "content": {
    "provider_id": "mangadex",
    "provider_manga_id": "abc123",
    "reading_mode": "longstrip",
    "last_synced_at": "2026-08-06T10:00:00Z"
  },
  "providers": [
    {
      "provider_id": "mangadex",
      "provider_manga_id": "abc123",
      "manga_title": "Sample Manga (MangaDex)"
    },
    {
      "provider_id": "kitsu",
      "provider_manga_id": "xyz789",
      "manga_title": "Sample Manga (Kitsu)"
    }
  ],
  "user_status": "plan_to_read",
  "user_rating": 8.5,
  "user_favorite": true,
  "user_notes": "Great world building and story pacing.",
  "last_read_chapter_id": "ch-001",
  "last_read_at": "2026-08-06T10:15:00Z",
  "added_at": "2026-07-01T00:00:00Z",
  "updated_at": "2026-08-06T10:00:00Z"
}
```

#### Field Specifications

| Field | Type | Description |
|---|---|---|
| `title` | string | Canonical display title of the series |
| `aliases` | string[] | Alternative titles, transliterations, or localized names |
| `description` | string | Synopsis or plot summary |
| `authors` | string[] | Story writers / authors |
| `artists` | string[] | Illustrators / artists |
| `tags` | string[] | Normalized taxonomy tags (e.g. `type:manga`, `Fantasy`) |
| `collections` | string[] | User-assigned collections / shelves (e.g. `Favorites`) |
| `content_rating` | string | Content rating: `safe`, `suggestive`, `erotica`, `pornographic` |
| `publisher` | string | Publishing imprint or magazine |
| `release_year` | integer | Publication release year |
| `start_date` | string | Publication start date (`YYYY-MM-DD`) |
| `end_date` | string | Publication end date (`YYYY-MM-DD`) or empty if ongoing |
| `country` | string | ISO country code of origin (`JP`, `KR`, `CN`, etc.) |
| `cover_url` | string | Upstream remote URL for cover art fallback / refresh |
| `external_links` | object[] | External links associated with the manga |
| `external_links[].provider` | string | Provider or service identifier |
| `external_links[].label` | string | Display label for the link |
| `external_links[].url` | string | Target web URL |
| `content` | object | Active content provider source configuration (optional) |
| `content.provider_id` | string | Active content provider identifier |
| `content.provider_manga_id` | string | Active provider series remote identifier |
| `content.reading_mode` | string | Series layout direction: `ltr`, `rtl`, `vertical`, `longstrip` |
| `content.last_synced_at` | timestamp | ISO 8601 timestamp of last upstream chapter reconciliation |
| `providers` | object[] | Array of all bound provider references |
| `providers[].provider_id` | string | Bound provider identifier |
| `providers[].provider_manga_id` | string | Remote series identifier on that provider |
| `providers[].manga_title` | string | Canonical series title reported by provider |
| `user_status` | string | Reading status: `unread`, `reading`, `completed`, `on_hold`, `dropped`, `plan_to_read` |
| `user_rating` | number | User score (0.0 to 10.0; 0 indicates unrated) |
| `user_favorite` | boolean | Favorite / starred series indicator |
| `user_notes` | string | User's private freeform notes |
| `last_read_chapter_id` | string | Local chapter ID of the most recently read chapter |
| `last_read_at` | timestamp | ISO 8601 timestamp of last reading activity |
| `added_at` | timestamp | ISO 8601 timestamp when manga was added to library |
| `updated_at` | timestamp | ISO 8601 timestamp of last metadata modification |

---

### Chapter Manifest (`<library_root>/<manga_id>/<provider_id>/<chapter_id>/meta.json`)

The chapter manifest records release metadata, provider synchronization coordinates, download state, and reading progress.

```json
{
  "title": "Chapter 1: Beginning",
  "number": 1.0,
  "volume": 1,
  "language": "en",
  "upload_date": "2026-07-15T00:00:00Z",
  "source_order": 1,
  "content": {
    "provider_id": "mangadex",
    "chapter_ref": "ch-abc-001",
    "last_synced_at": "2026-08-06T10:00:00Z"
  },
  "page_count": 24,
  "page_format": "jpg",
  "downloaded_at": "2026-08-06T10:05:00Z",
  "downloaded_pages": 24,
  "is_downloaded": true,
  "orphaned": false,
  "is_read": false,
  "last_read_page": 0,
  "last_read_at": ""
}
```

#### Field Specifications

| Field | Type | Description |
|---|---|---|
| `title` | string | Chapter title or name |
| `number` | float | Normalized chapter number (e.g. `1.0`, `10.5`) |
| `volume` | integer | Volume number (0 if unassigned) |
| `language` | string | ISO language code (e.g. `en`) |
| `upload_date` | timestamp | Original upload/release timestamp from upstream |
| `source_order` | integer | Ordinal index in provider's chapter list |
| `content` | object | Provider source coordinates |
| `content.provider_id` | string | Content provider identifier |
| `content.chapter_ref` | string | Provider's opaque chapter identifier |
| `content.last_synced_at` | timestamp | ISO 8601 timestamp of last chapter metadata sync |
| `page_count` | integer | Total page count for this chapter |
| `page_format` | string | File extension of downloaded pages (e.g. `jpg`, `png`, `webp`) |
| `downloaded_at` | timestamp | ISO 8601 timestamp when pages were pulled |
| `downloaded_pages` | integer | On-disk count of downloaded page image files (computed) |
| `is_downloaded` | boolean | `true` when all pages are stored on disk (computed) |
| `orphaned` | boolean | `true` if chapter is from an inactive provider or no longer listed upstream |
| `is_read` | boolean | User read status for this chapter |
| `last_read_page` | integer | Last read page index (1-based) |
| `last_read_at` | timestamp | ISO 8601 timestamp when chapter was last read |

---

### Chapter Page List (`<library_root>/<manga_id>/<provider_id>/<chapter_id>/pages.json`)

The page list manifest maps each page index to its resolved upstream URL and resolution source.

```json
[
  {
    "index": 1,
    "url": "https://uploads.mangadex.org/data/hash/1-abc.jpg",
    "source": "provider"
  },
  {
    "index": 2,
    "url": "https://uploads.mangadex.org/data/hash/2-xyz.jpg",
    "source": "provider"
  }
]
```

#### Field Specifications

| Field | Type | Description |
|---|---|---|
| `index` | integer | 1-based page order index |
| `url` | string | Upstream image URL for live streaming or pull acquisition |
| `source` | string | Page source state: `"provider"` (resolved upstream) or `"library"` (stored on disk) |

---

## Page Source Resolution

When reading a chapter, page bytes are resolved through a three-tier fallback hierarchy:

```
Reader Page Source Resolution Order:
  1. Disk (Library)    → <library_root>/<manga_id>/<provider_id>/<chapter_id>/<index>.<ext>
  2. Cache (Ephemeral) → cache/images/<sha256(url)>.<ext>
  3. Live Stream Proxy → HTTP reverse proxy with TLS fingerprinting & SSRF guard
```

1. **Disk Check**: The server inspects `<library_root>/<manga_id>/<provider_id>/<chapter_id>/` for matching image extensions (`.jpg`, `.jpeg`, `.png`, `.webp`, `.gif`, `.avif`). If present, the file is served directly with static cache headers.
2. **Cache Check**: If not on disk, the server checks the ephemeral disk cache for previously fetched and cached page images.
3. **Live Proxy Stream**: If not cached, the server fetches the page from the upstream provider URL via the request builder (applying appropriate Referer headers, anti-bot TLS fingerprinting, and SSRF address validation), optionally populating the cache.

---

## Background Pull Queue Architecture

Content acquisition in Kiyomi is managed by the asynchronous Background Job Queue (see [background_jobs.md](./background_jobs.md) and [content_pull.md](./content_pull.md)).

```
┌─────────────────────────────────────────────────────────────┐
│                       pull_manga                            │
│  - Fetches chapter list from upstream provider              │
│  - Creates chapter manifests for missing chapters           │
│  - Enqueues pull_cover (if cover not on disk)               │
│  - Enqueues pull_chapter for each chapter                   │
└──────────────────────────────┬──────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────┐
│                      pull_chapter                           │
│  - Fetches page list from upstream provider                 │
│  - Writes pages.json and updates meta.json page_count       │
│  - Enqueues pull_page for each page index                   │
└──────────────────────────────┬──────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────┐
│                       pull_page                             │
│  - Fetches single page image with SSRF validation           │
│  - Writes atomically to <chapter_dir>/<index>.<ext>         │
│  - Updates chapter downloaded_at in chapter meta.json       │
└─────────────────────────────────────────────────────────────┘
```

### Pull Job Types

1. **`pull_manga`**:
   - Fetches the chapter list from the provider API.
   - Saves new chapter manifests under `<library_root>/<manga_id>/<provider_id>/<chapter_id>/meta.json`.
   - Enqueues a `pull_chapter` job for each new chapter.
   - Enqueues a `pull_cover` job if the manga has no cover on disk, using a sentinel check to avoid duplicate acquisitions.
   - Stamps the manga manifest's `content.last_synced_at` timestamp.

2. **`pull_chapter`**:
   - Queries the provider API for the full page list.
   - Writes `pages.json` to `<library_root>/<manga_id>/<provider_id>/<chapter_id>/pages.json` and updates `page_count` in `meta.json`.
   - Enqueues a `pull_page` job for every page in the chapter.

3. **`pull_page`**:
   - Validates the page URL with SSRF protection.
   - Fetches the image from the provider and writes it atomically using a temporary file.
   - Saves the final image at `<library_root>/<manga_id>/<provider_id>/<chapter_id>/<index>.<ext>`.
   - Updates `downloaded_at` in the chapter's `meta.json`.

4. **`pull_cover`**:
   - Validates the cover URL with SSRF protection.
   - Fetches the cover image and writes it atomically to `<library_root>/<manga_id>/cover.<ext>`.

### Concurrency and Rate Limiting

- **Provider Concurrency Group**: All pull jobs targeting an upstream provider share the concurrency key `pull:<provider_id>`. The scheduler guarantees that at most `N` concurrent connections are made per provider.
- **Cover Concurrency Group**: Cover fetches use the dedicated concurrency group `pull:cover`.
- **Throttling**: Handlers apply delays based on the provider's rate limiting specifications to prevent upstream IP blocks.

---

## Refresh and Reconciliation Flow

Refreshing a manga reconciles local chapter manifests with the latest release list from the upstream provider:

```
1. Client triggers POST /api/v1/library/manga/:mangaId/refresh.
2. Read manga-level meta.json to identify active content provider (content.provider_id).
3. Fetch latest chapter list from the upstream provider API.
4. Read existing chapter manifests from <library_root>/<manga_id>/<provider_id>/.
5. Compare upstream list with local manifests:
   a. For new upstream chapters: create <chapter_id>/meta.json with initial metadata.
   b. For existing chapters: update metadata (title, number, upload_date) if changed.
   c. For local chapters not returned upstream: flag as orphaned (orphaned: true).
6. If the provider returns 0 chapters (potential error/takedown):
   - Abort reconciliation and surface error to client.
   - Preserve all local files and manifests without modification.
7. Update content.last_synced_at in manga-level meta.json.
```

---

## Multi-Content Provider Switching

Kiyomi supports binding multiple content providers to a single manga entry without cross-correlating chapter IDs:

1. **Add Provider Binding**: Appends a new `{ provider_id, provider_manga_id, manga_title }` entry to `providers[]` in manga `meta.json`.
2. **Switch Active Provider**: Updates `content.provider_id` and `content.provider_manga_id` in manga `meta.json`.
3. **Isolation**: Chapter manifests and downloaded images for the previous provider remain untouched on disk under `<library_root>/<manga_id>/<old_provider_id>/`.
4. **No Deletions**: Switching providers never deletes downloaded files or reading progress from other providers.

---

## Bulk Chapter Operations and Maintenance

### Reading Progress Updates
- Updates `is_read`, `last_read_page`, and `last_read_at` in chapter `meta.json`.
- Simultaneously updates `last_read_chapter_id` and `last_read_at` in the manga `meta.json`.

### Delete Chapter Files
- Removes downloaded page images (`*.jpg`, `*.png`, etc.) and `pages.json` from the chapter directory.
- Preserves `meta.json` and reading progress. The chapter remains in the UI and can be streamed live or re-pulled.

### Delete Chapters
- Completely deletes the chapter directory `<library_root>/<manga_id>/<provider_id>/<chapter_id>/`.

---

## References

- [Multi-Content Provider Library](./multi_content_provider_library.md) — Multi-provider architecture and UI interaction model
- [Background Jobs](./background_jobs.md) — Generic background job scheduler and worker engine
- [Content Pull System](./content_pull.md) — High-level content pull workflow and naming rationale
- [Providers](./providers.md) — Built-in provider contracts and capability model
- [Reader](./reader.md) — Web reader architecture and reading mode specifications
- [REST API Reference](./api.md) — Complete library and provider HTTP API endpoints
