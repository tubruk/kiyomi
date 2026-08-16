# Filesystem-First Library Architecture

> [!NOTE]
> Currently, the repository implements a **purely filesystem-based metadata library** (Phase 1). There is **no active SQLite database index** (`kiyomi.db`) compiled or running. All library metadata, chapter records, page lists, and reading progress details are written directly to and read directly from local `meta.json` and `pages.json` files on the filesystem. The SQLite schema described below remains an approved design specification for future implementation.

## Overview

Kiyomi is shifting from a provider-centric model (Tachiyomi/Mihon-style, where external sources are authoritative) to a **filesystem-first, library-centric model**. The local manga library is the product; providers become optional enrichment services.

### Core Principle

> **Filesystem is the source of truth. The database is a disposable index that can be wiped and rebuilt from disk at any time.**

This enables:
- True offline-first operation (library works without providers)
- Native backup, sync, and migration via `cp -r library/`
- Import/export between Kiyomi instances
- Recovery from DB corruption without data loss
- Adoption of existing manga collections (later, lower priority)

---

## Directory Layout

```
$KIYOMI_HOME/
├── library/                          # canonical manga storage (was: downloads/)
│   └── <manga_id>/
│       ├── meta.json                 # manga manifest
│       ├── cover.<ext>               # cover image
│       ├── banner.<ext>              # banner image (optional)
│       └── <chapter_id>/
│           ├── meta.json             # chapter manifest
│           ├── 001.jpg               # page images (zero-padded, 3+ digits)
│           ├── 002.jpg
│           └── ...
├── kiyomi.db                        # SQLite index (disposable cache)
├── cache/                           # transient: thumbnails, provider images
└── ...
```

### ID Conventions

- `<manga_id>` — ULID/KSUID, locally generated, **stable across renames**
- `<chapter_id>` — ULID, locally generated, **stable across renumbering**
- `<manga_id>` is NOT the provider slug; slug can change, ID stays

### Page File Naming

- Zero-padded to 3 digits minimum: `001.jpg`, `002.jpg`, ... `024.jpg`
- Single format per chapter (enforced via `meta.json` `page_format` field)
- Gap in numbering = incomplete download, flagged during scan

### Edge Cases (during scan)

| State | Detection | Result |
|---|---|---|
| Missing `meta.json` | folder exists, no manifest | Orphan, flagged |
| Missing page files | chapter in DB, no images | `is_downloaded=0` |
| Page gap (001, 003, no 002) | numbered scan | Incomplete, flagged |
| Extra files (`.DS_Store`, thumbs) | filename not matching `NNN.ext` | Ignored, logged |
| Chapter folder, manga missing | parent dir gone | Stale, deleted |

---

## Metadata Schemas

### `library/<manga_id>/meta.json`

```json
{
  "title": "Sample Manga",
  "aliases": ["Sample", "サンプル"],
  "description": "...",
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

  "content": {
    "provider_id": "kitsu",
    "manga_id": "abc123",
    "reading_mode": "longstrip",
    "last_synced_at": "2026-08-06T10:00:00Z"
  },

  "user_status": "plan_to_read",
  "user_rating": 8.5,
  "user_favorite": true,
  "user_notes": "Great world building and story pacing.",

  "added_at": "2026-07-01T00:00:00Z",
  "updated_at": "2026-08-06T10:00:00Z"
}
```

**Notes:**
- `content` is optional (user can add manga without provider binding)
- `user_status` enum: `unread` | `reading` | `completed` | `on_hold` | `dropped` | `plan_to_read` (backend lowercase, UI renders as title case labels)
- `user_favorite` boolean: marks entry as favorite / starred series
- `user_rating` float (0.0–10.0): user score evaluation (0 represents unrated)
- `user_notes` string: freeform private notes stored locally on filesystem
- `user_*` fields are pure user data, never overwritten by provider sync
- `tags` follows existing taxonomy (structural prefix + flat descriptors)
- `collections` is user-owned, untouched by syncs

### `library/<manga_id>/<chapter_id>/meta.json`

```json
{
  "title": "Chapter 1: Beginning",
  "number": 1.0,
  "volume": 1,
  "language": "en",
  "upload_date": "2026-07-15T00:00:00Z",

  "content": {
    "provider_id": "kitsu",
    "chapter_ref": "ch-abc-001",
    "last_synced_at": "2026-08-06T10:00:00Z"
  },

  "page_count": 24,
  "page_format": "jpg",
  "downloaded_at": "2026-08-06T10:05:00Z"
}
```

**Notes:**
- `content` is single (no array, no history)
- `number_mapping` is optional, populated only when LLM-assisted normalization runs
- Refresh reads `content.provider_id` only, never walks history
- Migration overwrites `content`, no audit trail in filesystem

---

## Provider Contract Changes

### New Capability Flag: `has_stable_chapter_id`

Add `has_stable_chapter_id bool` to the provider `Content` capability interface. This declares whether the provider's chapter reference (API ID) survives renumbering.

| Provider `has_stable_chapter_id` | Refresh correlation strategy |
|---|---|
| `true` | Match by `provider_chapter_ref`. Renumbering auto-detected via ID continuity. |
| `false` | Fallback to normalized title/number pattern matching, then ordinal position with low-confidence warning. |

```go
type ContentProvider interface {
    HasStableChapterID() bool
    FetchChapters(ctx, mangaRemoteID) ([]Chapter, error)
    FetchPages(ctx, mangaRemoteID, chapterRemoteID) ([]Page, error)
    FetchPageStream(ctx, page) (io.ReadCloser, error)
}
```

**Why this matters:** manga-level `content.manga_id` is the manga identity used to fetch the chapter list. Chapter-level `content.chapter_ref` is what's used to fetch page URLs from the provider API. When the provider's chapter IDs are stable across renumbering, the same `content.chapter_ref` refers to the same chapter over time; when not, renumbering breaks correlation and fallback heuristics apply. No separate stable-id field is stored in `meta.json` — the provider capability flag tells us whether `content.chapter_ref` itself is durable.

### `number_mapping` (Open Extension Point)

The chapter-level `meta.json` does NOT include any number mapping field by default. The reasoning: the value `number` already holds the normalized kiyomi number that the UI displays; if heuristic correlation (title/number pattern) succeeds, no extra metadata is needed. If it fails or produces low-confidence results, an extension point for richer metadata may be added later (e.g. LLM-assisted normalization, manual overrides, provenance). Schema kept minimal until a concrete use case demands it.

---

## Database Schema (Disposable Index)

> [!NOTE]
> As mentioned, the database index is not yet implemented. In the current filesystem-only model, all user reading progress, ratings, status, notes, and metadata are persisted directly in `meta.json` files on disk.

### Planned Schema (For Future DB Caching Phases)

- `manga` — list, search, filters
- `chapter_progress` — user reading state (cached from filesystem)
- `reading_progress` — derived aggregate, cache only

### What Gets Added

```sql
CREATE TABLE IF NOT EXISTS chapters (
    id TEXT PRIMARY KEY,
    manga_id TEXT NOT NULL REFERENCES manga(id) ON DELETE CASCADE,
    number REAL NOT NULL DEFAULT 0.0,
    title TEXT NOT NULL DEFAULT '',
    content_provider_id TEXT NOT NULL DEFAULT '',
    content_chapter_ref TEXT NOT NULL DEFAULT '',
    is_orphan BOOLEAN NOT NULL DEFAULT 0,        -- missing from current provider
    is_downloaded BOOLEAN NOT NULL DEFAULT 0,    -- derived from filesystem scan
    last_synced_at DATETIME,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_chapters_manga_id ON chapters(manga_id);
CREATE INDEX IF NOT EXISTS idx_chapters_is_orphan ON chapters(is_orphan);
CREATE INDEX IF NOT EXISTS idx_chapters_is_downloaded ON chapters(is_downloaded);
```

### What Gets Removed

- `chapter_download` — filesystem is the source of truth for download state
- `download_job` — moves to filesystem-based queue (job manifests as files, simpler recovery)
- `pages` table — derived from filesystem scan, cache only with mtime invalidation

### Manga Table Adjustments

- `cover_asset` / `banner_asset` paths remain (filesystem-derived during scan)
- `content_provider_id` / `content_remote_id` mirror what `meta.json` `content` holds

---

## Page Source Resolution

Page bytes are resolved at read time from a fallback chain:

```
Reader page source order:
  1. Disk (library)      — library/<manga_id>/<chapter_id>/<index>.<ext>
  2. Cache (ephemeral)   — cache/pages/<provider_id>/<sha256(url)>.<ext>
  3. Provider (live)     — fetched on demand
```

**Per-page source state, not chapter-level.** Each page carries its own `Source ∈ {disk, cache, provider}` state. The reader queries filesystem stat + cache lookup to determine source at runtime. No DB schema change required.

**API contract** for page list:
```
GET /manga/{id}/chapter/{cid}/pages
Response: []Page{Index, Source, URL?}
  Source ∈ {disk, cache, provider}
  URL present only when Source = provider (transient signed URL)
```

**`is_downloaded` remains chapter-level** for listing/filtering UX. It is `true` when all pages have `Source = disk`. Per-page state is computed on demand via filesystem stat; it is not stored in the DB.

**"Refresh" = re-enqueue missing pages.** The Refresh UI action iterates the chapter's page list and enqueues `DownloadPageArgs` for every page where `Source ≠ disk`. This is not a separate job kind — it is the original download job re-issued. Existing pages are left alone; only missing pages are fetched.

---

## Refresh & Sync Flow

### Refresh from Provider (Replaces Current Card #484 Spec)

```
1. User clicks "Refresh" on manga
2. Read <manga_id>/meta.json content.provider_id
3. Fetch chapters from provider
4. Begin merge transaction:
   for each remote chapter:
     if provider.has_stable_chapter_id():
       match by content.chapter_ref
     else:
       match by normalized title/number pattern
       if no match: ordinal fallback, mark low confidence
     if match found in local:
       update chapter meta (number, title, page_count)
       replace pages wholesale (no correlation)
     else:
       insert new chapter (chapter_id = new ULID)
5. Compare local chapters to remote:
   for each local not in remote:
     set is_orphan = 1
6. Edge case: if remote returns 0 chapters:
   ABORT. Surface error to user. Local data untouched.
7. Update meta.json last_synced_at
8. Surface last refreshed time in manga detail / chapter list toolbar
9. Update DB to reflect new state
```

### Three-State Indicator (Chapter List UI)

| State | Color | Meaning |
|---|---|---|
| Synced | Green | Local matches provider |
| Warning | Yellow | Missing from provider but downloaded (read offline) |
| Error/Orphan | Red/Gray | Missing from provider, no local files |

Orphan count shown in chapter list toolbar (not surfaced on manga card). User can filter to show orphans only.

### Zero-Chapters Failure Handling

Refresh returning 0 chapters = **error state**, not success. Possible causes:
- Provider outage
- Content takedown
- Auth failure
- Geo-restriction

Behavior:
- Log warning, surface error to user
- Local data untouched (no wipe)
- User can retry, manual override, or unlink provider

### Page URL Persistence

Page URLs are **not persisted** in `chapter meta.json`. They are:
- Ephemeral: signed URLs, token-bound, may expire
- Provider-authoritative: returned fresh on each chapter open
- Fetched live on reader open, with short cache (see `cache.md` §Page-List Cache)

**Chapter metadata that is persisted:**

| Field | Stored In | Rationale |
|---|---|---|
| `title` | `meta.json` | User-displayed, stable |
| `number` | `meta.json` | Normalized, stable |
| `volume` | `meta.json` | User-displayed, stable |
| `page_count` | `meta.json` | Derived from provider, set during refresh |
| `page_format` | `meta.json` | Set when first page downloaded |
| `content` | `meta.json` | `{ provider_id, chapter_ref }` for re-fetching |
| `downloaded_at` | `meta.json` | Written by indexer after page lands |
| `is_downloaded` | `kiyomi.db` | **Derived from filesystem scan**, not from meta.json |

**First-time add from provider:**
- Refresh flow writes `meta.json` per chapter with `page_count` and `page_format` set
- Page files are NOT downloaded unless user explicitly triggers download
- Page URLs not fetched until reader opens

**`is_downloaded` derivation:**
```
is_downloaded = (page files exist in chapter folder)
              ≠ meta.json downloaded_at  // meta.json may lag
```

This avoids the inconsistency where `downloaded_at` is set but pages were later deleted externally.

---

## Onboarding Paths

### Empty Library State (Onboarding CTA)

When the local library is empty, the UI displays a clean onboarding message with a **"Start Exploring"** call-to-action (CTA) button. Clicking this button redirects the user directly to the **Explore View**.

### Path 1: Explore & Add from Provider

All provider-based discoveries and library additions happen via the **Explore View**:
1. User opens Explore view (or is redirected via the empty library CTA).
2. User searches provider by tag or text, selects a manga.
3. Kiyomi displays a manga preview/details page.
4. User clicks "Add to Library" or picks a status from the dropdown:
   - **Plan to Read** — sets status to `plan_to_read`
   - **Currently Reading** — sets status to `reading`
   - **Already Read** — sets status to `completed`
   - **Add to Library** — status left unset
5. Kiyomi creates `library/<manga_id>/meta.json` with `content` populated.
6. Chapter list is fetched, and `library/<manga_id>/<chapter_id>/meta.json` is created per chapter.
7. Page files are NOT downloaded unless user triggers download.
8. Manga appears in library immediately.

### Path 2: Download (Existing)

1. User triggers download on chapter(s)
2. Page files written to `library/<manga_id>/<chapter_id>/NNN.<ext>`
3. `meta.json` `page_count`, `downloaded_at` updated
4. DB `chapters.is_downloaded = 1`

### Path 4: Mounting & Rebuilding an Existing Library

1. User points Kiyomi configuration to an existing `library/` directory.
2. Kiyomi scans the folder tree (e.g. on startup or via command-line database rebuild trigger).
3. Each manga and chapter `meta.json` file is read and indexed into the DB.
4. Local cover and banner files are resolved and matched.
5. Downloaded pages are verified to derive `is_downloaded = true` state.

---



---

## Migration Path from Current Architecture

### User-Facing Changes

1. Existing `downloads/` → move to `library/`
2. DB rebuild from filesystem (one-time migration)
3. Existing `manga` rows reconstitute into `library/<id>/meta.json`
4. Cover/banner files moved from `covers/`, `banners/` into manga folder
5. `shelves` field renamed to `collections` in meta.json (key name change)
5. Downloads moved from flat `downloads/<manga>/<chapter>/` to new structure

### Code Changes Made / Required

- `internal/library/library.go` — filesystem operations, manifest scanning, `collections` field, page list caching, and chapter progress updates
- `internal/api/library_handler.go` — API endpoints serving from filesystem-cached state and managing refresh/progress operations
- `plugin-sdk/provider.go` — defines `ContentProvider` interface including `HasStableChapterID()` flag and `FetchPages()` parameters
- `plugins/mangadex/` and `plugins/mangafox/` — built-in standalone plugins implementing the provider interfaces and handlers

### Backward Compatibility

- Old DB format can be migrated: scan `downloads/`, generate `meta.json`, move files
- One-time migration tool or auto-detect on startup
- After migration, old `downloads/` and `covers/` directories removed

---

## Provider Migration

When user switches manga's provider (e.g., source taken down, user prefers different):

1. User picks new provider in manga detail UI
2. Kiyomi re-fetches chapter list from new provider, writes new `content` in `meta.json`
3. Chapter correlation runs:
   - If `has_stable_chapter_id=true` for new provider: match by `chapter_ref`
   - If `has_stable_chapter_id=false`: fallback to title/number heuristic matching, low-confidence matches flagged
4. Orphan chapters handled manually by user (list via filter, keep or remove)

**Decisions:**
- Orphan chapters are kept permanently by default unless manually removed by the user. There is no automatic cleanup of orphan chapters, ensuring local files are preserved.
- When `chapter_ref` doesn't match and title/number heuristics fail, the chapter is marked as an orphan.
- Should we store old `content` history in `meta.json` for potential rollback? (Open Question)

## Bulk Chapter Operations

Chapter list supports checkbox-based multi-select with helpers:

**Selection helpers:**
- Select this chapter
- Select this and above
- Select this and below
- Select all downloaded
- Select all undownloaded
- Clear selection

**Actions on selection:**
- Mark as read / unread
- Download selected
- Delete download (remove pages from disk)
- (Future: move to collection, change reading direction)

Selection persists while navigating between chapter pages. Action buttons live in chapter list toolbar.

## Open Questions

1. Should chapter folders support loose files (jpg, png, webp) mixed? Or enforce single format?
2. Library scan on startup — full or incremental? Incremental is faster but adds complexity.
3. DB cache invalidation — mtime check on `meta.json`? Inotify/FSEvents for real-time?
4. Concurrent access — multiple Kiyomi instances on same library? Single-writer lock?
5. Storage backend abstraction — pure filesystem vs pluggable (S3, etc.)? Start filesystem-only, abstract later.
6. When orphan chapters have downloaded pages — should we offer a "download source-less chapter" action using cached page bytes as source?

---

## References

- Card #484: Feature: Chapter/Page Cache and Refresh
- Card #487: Feature: Import/Use already downloaded library
- Sibling design docs:
  - `docs/design/workers.md` — background job framework, download worker is library-aware
  - `docs/design/providers.md` — provider contract, `has_stable_chapter_id`
  - `docs/design/reader.md` — page source resolution model
  - `docs/design/cache.md` — page cache (middle tier: disk → cache → provider)
  - `docs/design/api.md` — REST surface for library
- Superseded docs (marked with deprecation notices at top):
  - `docs/developer/library_architecture.md` — prior DB-centric design, deprecated
  - `docs/developer/permanent_download_plan.md` — Phase 1 schema obsolete; worker/dispatcher concepts still relevant in spirit
- Related still-current docs:
  - `docs/developer/architecture.md` — system overview; "Disk Storage" section marked superseded
  - `docs/developer/sources_sdk.md` — provider SDK, will need `HasStableChapterID()` added
