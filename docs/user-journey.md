# User Journey — Kiyomi

> Kiyomi is a self-hosted manga library for people who already know what they want to read. Library owns the data; providers are replaceable.

---

## Mental Model

**You own your library.** Everything lives in `$KIYOMI_HOME/library/` — plain folders, plain files. Backup is `cp -r`. If the database dies, rebuild it from disk.

**Providers are services, not landlords.** A manga lives in your library even if its provider goes down. You can switch to a different provider for the same manga later.

**On-Demand Caching (Stream-First).** Pages are loaded directly from the provider when you read, and cached transiently. Offline downloading is low priority and postponed.

---

## Journey Map

| # | Journey | Trigger | Priority |
|---|---|---|---|
| 1 | Add manga to library | "I want to read X" | High |
| 2 | Read manga | "Time to read" | High |
| 3 | Download chapters | "Save for offline / better reading" | Postponed |
| 4 | Download entire manga | "I want to keep this forever" | Postponed |
| 5 | Organize | "Sort my library" | Medium |
| 6 | Discover | "Show me something new to read" | High |
| 7 | Switch provider | "This source is gone" | Medium |
| 8 | Handle orphans | "Something disappeared from the source" | Low |
| 9 | Refresh chapter list | "New chapters came out" | High |
| 10 | Library manga details | "Inspect manga metadata and chapters" | High |

---

## Journey 1 — Add Manga to Library

**Trigger:** User knows a manga title, wants it in their library.

**Flow:**

1. User opens Kiyomi → Library view (empty or existing)
2. User clicks "Add" → search modal opens
3. User types title → Kiyomi queries selected provider(s)
4. Results displayed with cover, title, provider badge
5. User picks the right result → Kiyomi creates `library/<manga_id>/meta.json`
6. Kiyomi fetches chapter list from provider → writes `meta.json` per chapter
7. Manga appears in Library immediately (chapters not downloaded yet)

**What the user sees:**
- Manga card in Library with cover art
- Chapter count badge
- "Not downloaded" state — normal at this point

**What happened on disk:**
```
library/<manga_id>/
  meta.json              ← manga manifest (source bound)
  cover.<ext>            ← downloaded if available
  <chapter_id>/
    meta.json            ← chapter manifest (page_count, number, title)
```

**Edge cases:**
- Provider returns 0 chapters → error surfaced, manga not added, local state untouched
- Network offline → search fails, user can still browse already-added manga

---

## Journey 2 — Read Manga

**Trigger:** User opens a manga from Library.

**Flow:**

1. User clicks manga card → Chapter list view
2. Each chapter shows: title, number, download state icon
   - Green dot = all pages on disk
   - Yellow dot = some pages in cache
   - No dot = pages on provider
3. User clicks chapter → Reader opens
4. Reader resolves each page from first available source:
   ```
   disk (library/) → cache → provider
   ```
5. Per-page source shown in reader UI:
   - Disk pages: no action
   - Cache pages: "Save to library" button
   - Provider pages: "Download" button
6. User navigates pages → progress saved (debounced, 2s)
7. User closes chapter → progress persisted, last page position remembered

**Reader states:**
- All pages on disk → fully offline, no network needed
- Some pages missing → live fetches from cache or provider as user reads
- Provider down → cached pages still work; missing pages show error state

**Progress tracking:**
- Per-chapter: last page read, status (unread / in_progress / read)
- "Continue reading" surface picks up where user left off
- Progress survives crashes and restarts

**Edge cases:**
- Chapter folder missing from disk (user deleted externally) → `is_downloaded=0`, reader falls back to cache/provider
- Page gap detected during read → gap flagged, user notified but reading continues

---

## Journey 3 — Download Chapters

**Trigger:** User wants chapters available offline for a flight, commute, or just to avoid provider slowdowns.

**Flow:**

1. User opens manga → Chapter list
2. User selects chapters via checkbox:
   - Tap chapter → select it
   - "Select this and above" / "Select this and below" helpers
   - "Select all downloaded" / "Select all undownloaded"
   - Clear selection
3. User clicks action (Download, Mark read, Delete download) → chapters queued
4. Background worker fetches each page, writes to `library/<manga_id>/<chapter_id>/NNN.ext`
5. Progress visible per chapter (e.g., "12/24 pages")
6. On completion: `meta.json` updated, `is_downloaded=1` in DB

**What the user can download:**
- Single chapter
- Range of chapters ("chapters 10–20")
- All unread chapters
- All chapters

**Downloaded chapters show green dot in chapter list.**

**Cancellable:** User can cancel a download job mid-flight. Partial pages left on disk are cleaned up.

**Retry:** Failed page downloads go to DLQ after max attempts. User can retry individually or re-trigger chapter download.

**Edge cases:**
- Disk full → download pauses, user notified, can free space and resume
- Network drops → job retries on reconnect
- Provider rate-limits → per-provider concurrency enforced, queue waits

---

## Journey 4 — Download Entire Manga (Archive)

**Trigger:** User loved the manga, wants it permanently offline for rereading.

**Flow:**

1. User opens manga detail
2. User clicks "Download All" → confirms
3. All chapters queued as download jobs (one page = one job)
4. Progress aggregated at manga level
5. On completion: entire manga on disk, green dots on all chapters

**"Download All" vs "Download Chapters":** Same mechanism, different scope. Download All is a bulk action.

**No automatic re-download of already-downloaded chapters.** If user re-triggers "Download All," existing pages are skipped (idempotent). Only missing pages are fetched.

---

## Journey 5 — Organize

**Trigger:** User wants to sort, label, and filter their library.

### Status

Each manga has a **status** (user-facing, never overwritten by provider):

| Status | Meaning |
|---|---|
| Reading | Currently reading |
| Completed | Finished all chapters |
| On Hold | Paused mid-series |
| Dropped | User quit |
| Planned | Added but not started |

User sets status on manga detail view or bulk-edit in Library.

### Collections (renamed from Shelves)

Collections are user-owned groups. A manga can be in multiple collections. Examples:
- "Reread favorites"
- "Weekend reads"
- "Dark manga"
- "Comedy only"

User creates, renames, deletes collections. Manga added to or removed from collections via manga detail or bulk-edit.

Collections are personal. Provider sync does not touch them.

### Tags

Tags describe the manga's content. Set by provider (on add) or user-edited. Examples:
- `type:manga`, `demographic:shoujo`, `genre:isekai`, `genre:romance`

User can add or remove tags. Tags are not owned by the user — they reflect what the manga is about, not what the manga means to the user (that's a collection).

### Filtering & Sorting

Library view supports:
- Filter by: status, collection, tag, download state, provider
- Sort by: title, last read, date added, last updated

---

## Journey 6 — Explore

**Trigger:** User wants to find new manga based on tags or interests.

**Flow:**

1. User opens Discover / Search
2. User enters tag filter (e.g., `genre:isekai`) or free-text search
3. Kiyomi queries provider(s) for matching results
4. Results shown with cover, title, tags
5. User previews manga details (synopsis, tags, chapter count)
6. User picks action from dropdown:
   - **Plan to Read** → adds manga to library, sets status to `plan_to_read`
   - **Currently Reading** → adds manga, sets status to Reading
   - **Already Read** → adds manga, sets status to Completed
   - **Add to Library** → adds manga, keeps status unset
7. Manga added, chapter list fetched (Journey 1)

**"Add to library" does not download chapters.** User must explicitly download to read offline.

**Explore vs Library:** Explore pulls from providers. Library is the user's owned collection.

**Edge cases:**
- Provider down → Explore disabled, user notified
- No results for tag → empty state, try different tag

---

## Journey 7 — Switch Provider

**Trigger:** A manga's provider is taken down, rate-limited, or the user prefers a different source.

**Flow:**

1. User opens manga detail
2. User sees current provider badge
3. User clicks "Change provider" → list of available providers shown
4. User picks new provider
5. Kiyomi re-fetches chapter list from new provider
6. New `source` written to `library/<manga_id>/meta.json`
7. Chapter IDs compared:
   - Matching chapters (by `has_stable_chapter_id` strategy) → merged
   - New chapters → added
   - Missing chapters → marked orphan
8. If all chapters match and downloaded, nothing changes for the user
9. If chapters are new, user notified: "X new chapters found"

**Provider switch is safe.** Library files survive. Only `meta.json` and DB index updated.

**Edge cases:**
- New provider has no stable chapter IDs → correlation uses title/number heuristics, low-confidence matches flagged
- User switches back to original provider later → re-correlation attempted, orphans may be recovered

---

## Journey 8 — Handle Orphans

**Trigger:** A manga's provider removed chapters (takendown, licensing), or user switched providers.

**What is an orphan?** A chapter in the local library that no longer exists on the current provider.

**How user sees it:**
- Chapter list shows red/gray marker on orphaned chapters
- Orphaned chapters with downloaded pages retain their files on disk
- Orphaned chapters without downloaded pages show no pages

**User actions:**
- "List all orphans" → filter in chapter list view
- "Keep anyway" → orphan stays, user can still read if pages downloaded
- "Remove" → chapter folder deleted from disk, removed from DB

**Orphaned downloaded chapters remain readable** — files are on disk, reader falls back to library source. Orphaned undownloaded chapters cannot be fetched (provider data gone).

**No automatic removal.** User explicitly decides what to do with orphans.

---

## Journey 9 — Refresh Chapter List

**Trigger:** User wants latest chapter list from provider ("new chapters came out").

**Flow:**

1. User opens manga detail or chapter list
2. User clicks "Refresh"
3. Kiyomi fetches chapter list from current provider
4. Kiyomi merges with local state:
   - New chapters → added
   - Removed chapters → marked orphan
   - Updated metadata (title, number) → synced
5. UI updates to show new state

**Refresh is manual only.** No background auto-refresh in v1.

**First add includes a refresh** — when user adds manga, Kiyomi fetches chapter list automatically. Subsequent refreshes are user-triggered.

**Edge cases:**
- Provider returns 0 chapters → error surfaced, local state untouched, user can retry
- No new chapters → UI shows "Up to date"

---

## Journey 10 — Library Manga Details (`library-manga-details`)

**Trigger:** User clicks on a manga card in their library to inspect its metadata and chapter list.

**Flow:**

1. User opens Kiyomi → Library view
2. User clicks a manga card (e.g., "Alpha Manga")
3. Kiyomi navigates to `/manga/<manga_id>` and renders the manga detail view
4. User inspects manga information:
   - **Cover Image**: Rendered prominently in the hero card
   - **Title & Aliases**: Primary title and alternate titles/aliases (if available)
   - **Author & Artist**: Displays creator credits (merged into "Author / Artist: ..." when the same person)
   - **Tags**: Genre and categorical badges/pills
   - **Synopsis**: Summary/description of the series
   - **Reading Status**: Current reading status (e.g., Reading, Completed, Plan to Read) in the status selector
   - **Chapter List**: All chapters available for the series with direct reading links

**What the user sees:**
- Hero card with cover thumbnail, title, aliases, merged author/artist, tags, synopsis, and reading status selector
- Chapter list with links to read each chapter

**Read-only guarantee:**
- Browsing library manga details is entirely read-only and does not mutate local library data.

---

## System Journeys (Background)

These happen without user action but matter for UX expectations.

### Library Scan

Kiyomi scans `$KIYOMI_HOME/library/` on startup and periodically. For each `meta.json` found, DB index is updated. Orphan files (chapter folder exists, manga folder gone) flagged.

Scan is incremental by default (mtime check on `meta.json`). Full scan available on demand.

### Cache Eviction

Cache pages expire by TTL (default 7 days). When a chapter is downloaded to library, its cached pages are invalidated.

Cache does not corrupt library data. Library files are never evicted.

### Progress Sync

Reading progress stored in `kiyomi.db`, not on filesystem. Single-writer SQLite, survives restarts.

---

## Data Ownership Summary

| Data | Owner | Survives provider outage | Survives DB loss |
|---|---|---|---|
| Manga files (pages) | Library | Yes | Yes (on disk) |
| Manga metadata | Library (`meta.json`) | Yes | Rebuild from disk |
| Chapter metadata | Library (`meta.json`) | Yes | Rebuild from disk |
| Tags | Provider + user | Yes | Rebuild from disk |
| Collections | User | Yes | Rebuild from disk |
| Reading progress | User (DB) | N/A | No (intentional) |
| Provider credentials | User config | Yes | Yes |
| Cache | Kiyomi | No | No (transient) |

**Rule:** If the user added it, it's theirs. If Kiyomi fetched it for performance, it's replaceable.
