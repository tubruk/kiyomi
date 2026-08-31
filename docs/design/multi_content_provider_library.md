# Multi-Content Provider Library Architecture

## Overview

Kiyomi implements a **multi-content-provider first-class architecture** that allows a single manga entry in the user's library to bind to multiple upstream content and metadata providers concurrently.

Instead of attempting fragile cross-provider chapter correlation or performing destructive database migrations when switching providers, chapters from different content providers are treated as completely independent entities isolated in dedicated directory namespaces on the filesystem.

---

## 1. Provider Directory Namespaces

Chapter metadata and downloaded image files are partitioned by provider ID directly under the manga's root directory:

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

### Identifier Spaces

| Identifier | Description | Example |
|---|---|---|
| `<manga_id>` | Local manga identifier (ULID or clean URL-safe slug) | `01J0MANGAEXAMPLEID` |
| `<provider_id>` | Registered provider identifier or `local` for manual imports | `mangadex`, `mangafox`, `local` |
| `<chapter_id>` | Local chapter identifier (ULID or provider chapter reference) | `ch-001`, `01J0CHAPEXAMPLEID` |

### Special Provider: `local`

The `local` provider ID is reserved for chapters that are managed locally without an external content provider (e.g. manual CBZ/ZIP extractions, local PDF scans, or legacy offline archives).

---

## 2. Provider Manifest Integration

Kiyomi coordinates multi-provider metadata through the manga manifest and chapter manifests on disk.

> **Note**: See [Library Storage Architecture](./library.md) for full manifest schemas and comprehensive field-by-field definitions.

### Manga Manifest Provider Bindings (`<library_root>/<manga_id>/meta.json`)

The manga manifest records all bound providers alongside the active provider pointer:

```json
{
  "title": "Sample Manga",
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
  "last_read_chapter_id": "ch-001",
  "last_read_at": "2026-08-06T10:15:00Z"
}
```

#### Key Provider Fields
- `content`: Object identifying the current active content provider (`provider_id`, `provider_manga_id`, `reading_mode`, `last_synced_at`).
- `providers[]`: Array of all bound provider references (`provider_id`, `provider_manga_id`, `manga_title`).
- `last_read_chapter_id` / `last_read_at`: Manga-level pointer to the most recently read chapter across all providers.

### Chapter Manifest Provider Coordinates (`<library_root>/<manga_id>/<provider_id>/<chapter_id>/meta.json`)

Each chapter manifest stores its upstream provider coordinates and download/read status:

```json
{
  "title": "Chapter 1: Beginning",
  "number": 1.0,
  "content": {
    "provider_id": "mangadex",
    "chapter_ref": "ch-abc-001",
    "last_synced_at": "2026-08-06T10:00:00Z"
  },
  "page_count": 24,
  "is_downloaded": true,
  "orphaned": false,
  "is_read": false,
  "last_read_page": 0,
  "last_read_at": ""
}
```

#### Key Provider Fields
- `content.provider_id`: Identifies the originating content provider for this chapter namespace.
- `content.chapter_ref`: Opaque remote chapter reference used when fetching pages or syncing with upstream.
- `orphaned`: Boolean flag indicating whether the chapter is no longer present in the upstream provider's active catalog.

---

## 3. Reading Progress and Tracking Model

Reading progress is designed to be independent at the chapter level while remaining unified at the manga level:

1. **Independent Chapter Progress**:
   - Each chapter manifest (`<provider_id>/<chapter_id>/meta.json`) tracks its own read state (`is_read`), page progress (`last_read_page`), and read timestamp (`last_read_at`).
   - Progress recorded on one provider does not mutate or overwrite chapters in other provider namespaces.

2. **Manga-Level Unified Tracking**:
   - High-level progress (`last_read_chapter_id`, `last_read_at`, `user_status`) is maintained in the manga's `meta.json`.
   - When switching between providers, the user can easily see their last reading position regardless of which provider supplied that chapter.

---

## 4. Provider Binding and Switching Lifecycle

Treating chapters as provider-isolated entities simplifies provider management from a complex migration into a clean metadata pointer update:

```
┌─────────────────────────────────────────────────────────────┐
│                      Manga Manifest                         │
│  - providers: [ mangadex (ref 1), kitsu (ref 2) ]           │
│  - content: { provider_id: "mangadex" }                     │
└──────────────────────────────┬──────────────────────────────┘
                               │
            ┌──────────────────┴──────────────────┐
            ▼                                     ▼
┌───────────────────────┐             ┌───────────────────────┐
│ /mangadex/ namespace  │             │   /kitsu/ namespace   │
│  ├── ch-001/          │             │  ├── ch-001/          │
│  └── ch-002/          │             │  └── ch-002/          │
└───────────────────────┘             └───────────────────────┘
```

### Lifecycle Workflow:

1. **Bind Provider**:
   - Appends a new `{ provider_id, provider_manga_id, manga_title }` record to `providers[]` in the manga manifest.
2. **Switch Active Provider**:
   - Updates `content.provider_id` and `content.provider_manga_id` in the manga manifest to point to the selected provider.
3. **Manifest Acquisition & Synchronization**:
   - If switching to a provider for the first time, Kiyomi queries the provider API and writes chapter manifests into `<library_root>/<manga_id>/<provider_id>/<chapter_id>/`.
4. **Non-Destructive Preservation**:
   - Existing chapters and downloaded images from previous providers remain intact on disk under `<library_root>/<manga_id>/<previous_provider_id>/`.
   - Switching providers never purges local files or corrupts reading history.

---

## 5. Offline Orphan Detection

When synchronizing with external content providers, Kiyomi ensures local availability is never compromised by upstream catalog changes:

1. **Reconciliation & Orphan Flagging**:
   - When refreshing chapter lists from an active provider API, local chapter manifests under `<library_root>/<manga_id>/<provider_id>/` are compared against the upstream response.
   - Chapters present on disk but missing from the upstream catalog (due to licensing changes, group deletions, or takedowns) are marked with `"orphaned": true` in their `meta.json`.
2. **Offline Resilience**:
   - Orphaned chapters remain fully accessible and readable from disk.
   - The reader continues serving local page images even when the upstream provider source is unreachable or removed.
3. **Zero-Chapter Safety Guard**:
   - If a provider query returns zero chapters (indicating a provider network error, auth failure, or API disruption), reconciliation is aborted immediately.
   - Local manifests are preserved without marking chapters as orphaned.

---

## 6. Non-Destructive Chapter File Deletion and Maintenance

Kiyomi distinguishes between clearing downloaded media files and removing chapter metadata:

1. **Delete Chapter Files (Free Disk Space)**:
   - Removes downloaded page images (`*.jpg`, `*.png`, etc.) and `pages.json` from `<library_root>/<manga_id>/<provider_id>/<chapter_id>/`.
   - Preserves `meta.json` and reading history (`is_read`, `last_read_page`).
   - The chapter remains visible in the library and web reader, allowing live streaming or re-pulling on demand.
2. **Delete Chapter**:
   - Completely removes the chapter subdirectory `<library_root>/<manga_id>/<provider_id>/<chapter_id>/`.
3. **Manga-Level Asset Ownership**:
   - Series artwork (`cover.<ext>`, `banner.<ext>`) is stored at the manga root (`<library_root>/<manga_id>/`), ensuring consistent display across views.
   - Binding, unbinding, or deleting provider chapters never alters manga-level assets.

---

## References

- [Library Storage Architecture](./library.md) — Canonical filesystem-first library specification, directory layout, and full manifest schemas
- [Background Jobs](./background_jobs.md) — Generic background job scheduler and worker engine
- [Content Pull System](./content_pull.md) — High-level content pull workflow and naming rationale
- [Providers](./providers.md) — Built-in provider contracts and capability model
- [Reader](./reader.md) — Web reader architecture and reading mode specifications
- [REST API Reference](./api.md) — Complete library and provider HTTP API endpoints
