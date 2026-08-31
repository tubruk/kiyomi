# Multi-Content Provider First-Class Library Architecture

## Overview
This design outlines the transition of Kiyomi from a single-active-content-provider model to a **multi-content-provider first-class architecture**. Instead of correlating chapters when switching content providers, chapters from different content providers are treated as completely independent entities. 

This is reflected in the filesystem layout and API structure, isolating content and simplifying content provider management.

---

## 1. Directory Layout

Under the new layout, chapter files are grouped by content provider ID within the manga's directory:

```
<library_root>/
└── <manga_id>/
    ├── meta.json                     # Manga manifest (title, metadata, bound content providers)
    ├── cover.<ext>                   # Cover image
    └── <content_provider_id>/        # Grouped by content provider (e.g. mangadex, mangafox, local)
        └── <chapter_id>/             # Local chapter ULID (uncorrelated)
            ├── meta.json             # Chapter manifest
            ├── pages.json            # Page map
            ├── 001.jpg               # Page images (files)
            └── ...
```

### Manga Manifest Schema (`meta.json`):
```json
{
  "title": "Sample Manga",
  "providers": [
    { "provider_id": "mangadex", "provider_manga_id": "abc123", "manga_title": "Title" }
  ],
  "content": {
    "provider_id": "mangadex",
    "provider_manga_id": "abc123",
    "reading_mode": "longstrip",
    "last_synced_at": "2026-08-06T10:00:00Z"
  }
}
```

### Special Content Provider ID: `local`
The `local` provider ID is reserved for chapters that are manually imported by placing files directly in the library directory (e.g., CBZ extractions or local PDF scans).

---

## 2. Reading Progress and Read Status

Progress tracking is kept simple and isolated:

1. **Independent Chapter Progress**: Each `chapter_id` tracks its read status (`is_read`) and page progress (`last_read_page`) completely independently. There is no automated cross-content-provider synchronization of chapter-level progress.
2. **Manga-Level Tracking**: General progress (e.g., "last read chapter number/float") is tracked at the manga level. When switching content providers, the UI uses this last-read number as a helper/indicator for where to pick up.

---

## 3. UI/UX Journey

### Chapter List Toolbar
The manga detail screen features a content provider selector (dropdown or tab bar) displaying all bound content providers:

```
+--------------------------------------------------+
| [ Manga Title ]                                  |
|                                                  |
| Content Provider: [ MangaDex (Active) | v ]      |
|                                                  |
| Chapters:                                        |
| [X] Ch. 1 - Beginning          (Pulled)          |
| [ ] Ch. 2 - The Journey Begins (Stream)          |
+--------------------------------------------------+
```

* Selecting a content provider updates the chapter list to show that provider's release list.
* The "Pull (to library)" action pulls files specifically to the corresponding namespace: `<library_root>/<manga_id>/<content_provider_id>/<chapter_id>`.

---

## 4. How the "Switch Content Provider" Flow Changes

With chapters treated as completely unrelated, the concept of "switching" content providers simplifies from an intrusive database migration into a lightweight UI toggle:

### Step-by-Step Flow:
1. **Add/Bind Content Provider**: The user adds a new content provider binding to the manga. The provider is appended to the `providers` array in the manga-level `meta.json`.
2. **Toggle Selection**: The user selects the new content provider from the dropdown selector in the UI. 
3. **Fetch & Save**: 
   * Kiyomi checks the local filesystem under `<library_root>/<manga_id>/<content_provider_id>/` for existing chapter metadata.
   * If empty (first time selecting), it fetches the chapter list from the new provider and writes chapter manifests (`meta.json`) under `<library_root>/<manga_id>/<content_provider_id>/<chapter_id>/`.
4. **Default Setting**: The `content.provider_id` in the manga-level `meta.json` is updated to point to the selected provider, acting as the new default selection for future visits.
5. **No Deletion of Other Chapters**: 
   * No deletion of old chapters is performed.
   * Chapters from other content providers remain untouched and safe on disk under `<library_root>/<manga_id>/<old_content_provider_id>/`.

---

## 5. Architectural Decisions

### A. Live API and Local Folder Merging
* When displaying the chapter list for a selected content provider, the backend merges the live chapter list (retrieved from the provider's API) with the physical directories existing under `<library_root>/<manga_id>/<content_provider_id>/`.
* If a chapter directory exists on disk but is no longer returned by the provider's API (e.g., content takedown or offline mode), the chapter remains in the UI list, clearly marked as **"Offline Only"** or **"Orphaned"**.

### B. "Delete Files" Behavior
* When deleting chapter files, only the page images (`001.jpg`, etc.) and the `pages.json` index are removed from the disk.
* The chapter directory and its `meta.json` manifest must **not** be deleted. This ensures the chapter stays in the UI list, retains its read status/reading progress, and remains available for live streaming or repulling.

### C. Pull Queue Rate-Limiting
* The pull worker queue processes jobs sequentially per provider, incorporating throttling/delays based on the provider's `RateLimit()` specifications. This mitigates the risk of IP blocks or rate limit bans during large chapter pulls.

### D. Cover Art Ownership
* Cover art remains at the manga level (`<library_root>/<manga_id>/cover.<ext>`) to keep library displays consistent. 
* When binding multiple providers, the user can manually trigger a sync to fetch and apply a specific provider's cover art as the canonical library cover.
