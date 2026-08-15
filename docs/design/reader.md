# Reader

> **Status**: Foundational design, scaffolding.

## Overview

The reader is the user-facing reading experience. It loads pages from local files when available, falls back to remote providers when not, tracks reading progress, and provides navigation controls.

## Conceptual Model

```
┌──────────────────────────────────────────────────────────┐
│                     Reader Runtime                         │
│                                                           │
│   ┌─────────────┐    ┌─────────────┐    ┌─────────────┐  │
│   │  Page Loader │    │  Navigator   │    │  Progress   │  │
│   │             │    │             │    │  Tracker    │  │
│   └─────────────┘    └─────────────┘    └─────────────┘  │
│         │                  │                  │            │
│         ▼                  ▼                  ▼            │
│   ┌─────────────────────────────────────────────────────┐ │
│   │           Page Source Resolver (cache → fs → remote)│ │
│   └─────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────┘
                              │
                              ▼
              ┌─────────────────────────────┐
              │   Library + Cache + Provider │
              └─────────────────────────────┘
```

## Reading Modes

Kiyomi supports multiple reading layouts per chapter, driven by `library/<manga_id>/meta.json` `content.reading_mode`:

| Mode Enum | UI Display Label | Layout & Behavior |
|---|---|---|
| `rtl` | Right to Left (Manga) | Traditional manga, pages navigate right-to-left |
| `ltr` | Left to Right (Comic) | Western comics, pages navigate left-to-right |
| `vertical` | Vertical (Gapped) | Paged vertical layout with margins between pages |
| `longstrip` | Longstrip (Webtoon) | Continuous seamless vertical scroll without page padding |

The reading mode is normalized per-manga in `manga.content.reading_mode`. If omitted or unspecified (`""`), the reader falls back to the user's default reading preference.

## Page Resolution

In **Phase 1**, since the library is metadata-only (no downloaded page files in `library/` folders yet), page images are resolved live by streaming from the remote provider through the Kiyomi backend reverse proxy (`/api/v1/library/manga/{id}/chapters/{ch}/pages/{n}`). The backend uses the TLS fingerprinting engine (`pkg/fingerprint`) to fetch remote page streams securely.

*Full Fallback Chain (Phase 2+)*:
```
Page resolution order:
  1. Disk (library)  — library/<manga_id>/<chapter_id>/<index>.<ext>
  2. Cache (ephemeral) — cache/pages/<provider_id>/<sha256(url)>.<ext>
  3. Provider (live)  — fetch from provider via backend proxy
```

## Page Source Resolution

Each page carries a `Source ∈ {disk, cache, provider}` state, determined at read time by filesystem stat + cache lookup. Source state is **per-page**, not aggregated at chapter level — a chapter may have some pages from disk, some from cache, and some from provider simultaneously.

**UI surfaces per-page source:**
- Disk pages: no action needed
- Cache pages: "Save to library" action available
- Provider pages: "Download" action available

Both actions enqueue `DownloadPageArgs` targeting the library path. The reader **never blocks on download** — it reads whatever source is currently available and re-observes state on the next page visit.

**Source derivation (no DB change):**
- `disk` — `stat(library/<manga_id>/<chapter_id>/<index>.<ext>)` succeeds
- `cache` — `disk` absent AND `stat(cache/pages/<provider_id>/<sha256(url)>.<ext>)` succeeds
- `provider` — neither disk nor cache present; fetch live

## Reading Progress Tracking

Per-chapter progress is stored in the database (NOT in filesystem, deliberately — user data):

```
ChapterProgress:
  manga_id
  chapter_id
  last_page       1-based, last page viewed
  total_pages     derived from chapter meta
  status          unread | in_progress | read
  updated_at
```

Aggregated per-manga progress (`reading_progress` table) caches the latest read chapter for fast "continue reading" surfaces.

Update cadence:
- Debounced — every 2 seconds while active, or on chapter close, or on page nav
- Resumable — process crash mid-chapter, last read position persists on next open

## Navigation

| Action | Web | Mobile |
|---|---|---|
| Next page | Arrow right / scroll / tap right | Tap right / swipe left |
| Previous page | Arrow left / scroll up | Tap left / swipe right |
| Next chapter | End of current | Swipe up on last page |
| Previous chapter | Beginning of current | Swipe down on first page |
| Jump to chapter | Quick chapter drawer | Sheet picker |
| Open settings | Reader toolbar | Toolbar |
| Exit reader | Back button | Back button |

Navigation state lives in the URL/route. Closing reader = leaving route. Reopening = same chapter and page from progress data.

## Chapter Transitions

When reader reaches end of chapter:

```
On last page reached:
  if next chapter exists in library:
    preload first page (image)
    on user action (button / swipe), navigate
  else:
    show "end of series" / "no next chapter" indicator
```

Preload keeps transitions snappy. No full chapter prefetch (bandwidth + memory cost outweighs UX gain for most).

## Offline Behavior

When all pages of a chapter are downloaded locally, reader runs fully offline:

```
Network state: offline
  - All reads from filesystem or cache
  - Progress writes queue (write-through to local DB)
  - No provider calls
```

Progress writes always succeed locally (DB is local). No cloud sync = no offline failure mode.

## Image Format Support

```
Primary:  JPG, PNG, WebP
Fallback: AVIF (decoded via browser)
Archive:  CBZ, CBR (browser-native or WASM unzip)
```

CBZ handling — open `.cbz` file, unzip in browser or via WASM unzip module, treat contents as page list. Caching strategy for CBZ differs (whole archive = single cache key).

## Caching Strategy

| Source | Cache? | TTL |
|---|---|---|
| Cover art | Yes | 7 days (re-validate weekly) |
| Page (live fetch) | Yes | Until library sync confirms filesystem copy |
| Page (filesystem) | No (read directly) | n/a |
| Thumbnail | Yes | 30 days |

Cache invalidation tied to library events: when a chapter is downloaded (filesystem copy), cache entries can be evicted. When removed from library, cache purges.

## Memory & Performance

Large chapters (1000+ pages, rare but possible) need careful handling:

```
Virtual scroll / lazy load:
  - Render only pages in viewport + 2-page buffer
  - Decoded images held in memory cache, LRU eviction
  - Disk-backed cache for "scrolled away" pages
```

Browser native image decoding. No custom decoder unless AVIF requires it.

## Accessibility

- Keyboard navigation (arrows, page up/down, home/end)
- Screen reader labels for page index, chapter name
- Reduced motion preference honored (no auto-flip animations)
- High contrast / dark mode via CSS variables

## Reading Settings (Per-User)

```
Settings (per manga or global):
  - reading direction (override manga default)
  - fit mode (width, height, original)
  - gap between pages (none, small, large)
  - background color (black, gray, sepia, white)
  - tap zones configuration
```

Per-manga override wins over global. Stored in user config, not in library metadata.

## Pre-rendering & SSR

Reader is a SPA route. Initial page render needs:
- Chapter metadata (from cache or filesystem)
- First page image (preloaded)
- Progress data (from DB)

Subsequent page navigation is client-side. No additional server roundtrip per page.

## Migration Considerations

Reader behavior depends on:
- Where pages live (filesystem vs cache vs remote)
- Reading direction (in `meta.json`)
- Progress data (in DB)

Migrating existing libraries to filesystem-first model requires:
- Cover/banner migration into `library/<manga_id>/`
- Chapter folder migration
- Progress remains in DB, no change

## Open Questions

1. Webtoon (long strip) reader for vertical mode — same component or separate?
2. Two-page spread mode for tablet/landscape?
3. Image preprocessing pipeline (resize, format conversion) at download time?
4. Reading statistics (time spent, pages per session)? Adds tracking, defer.
5. Cloud sync of progress across devices — via tracking providers, separate from library.

## References

- `docs/design/library.md` — page file layout, `meta.json` schema, page source model
- `docs/design/cache.md` — page cache (middle tier of fallback chain)
- `docs/design/workers.md` — download worker, `DownloadPageArgs`
- `docs/design/providers.md` — remote page fallback
- `docs/developer/design.md` — current UI design system, reader toolbar specs
