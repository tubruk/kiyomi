# Design Document: Reading Progress Tracking & Resume Journey

- **Author:** Kiyomi Engineering Team
- **Status:** Approved / Implemented
- **Last Updated:** 2026-08-15

---

## 1. Overview & Objectives

Reading progress tracking ensures users can seamlessly read manga across multiple sessions and devices without losing their position.

### Core Goals
1. **Automatic Page Level Tracking**: Save the exact 1-indexed page number (`last_read_page`) as the user reads.
2. **Automatic Chapter Completion**: Mark chapters as `is_read: true` when reaching the final page or proceeding to the next chapter.
3. **One-Click Resumption**: Provide a dynamic **"Resume Ch. X (p. Y)"** action on series detail pages and unread count badges on library cards.

---

## 2. Storage & Schema Specifications

Progress data is persisted directly in Kiyomi's library manifest directory (`$KIYOMI_LIBRARY_DIR/<manga_id>/`):

### 2.1 Chapter Metadata (`$KIYOMI_LIBRARY_DIR/<manga_id>/<chapter_id>/meta.json`)
```json
{
  "title": "Chapter 14",
  "number": 14.0,
  "volume": 2,
  "language": "en",
  "page_count": 24,
  "is_read": true,
  "last_read_page": 24,
  "last_read_at": "2026-08-15T21:00:00Z"
}
```

### 2.2 Manga Metadata (`$KIYOMI_LIBRARY_DIR/<manga_id>/meta.json`)
```json
{
  "title": "Chainsaw Man",
  "user_status": "reading",
  "last_read_chapter_id": "ch-14",
  "last_read_at": "2026-08-15T21:00:00Z"
}
```

---

## 3. API Contract Design

### `PATCH /api/v1/library/manga/:mangaId/chapters/:chapterId/progress`

#### Request Payload (Partial Updates Supported)
```json
{
  "is_read": true,
  "last_read_page": 18
}
```

#### Response (200 OK)
```json
{
  "id": "ch-14",
  "manga_id": "manga-01",
  "meta": {
    "title": "Chapter 14",
    "number": 14.0,
    "page_count": 24,
    "is_read": true,
    "last_read_page": 18,
    "last_read_at": "2026-08-15T21:00:00Z"
  }
}
```

---

## 4. Frontend Reader & UI Behavior Specs

1. **Debounced Sync**: Viewport active page `$P$` triggers `PATCH` progress sync after 1.5 seconds of scroll idle time.
2. **Scroll Restoration**: Opening an in-progress chapter (`is_read: false` & `last_read_page > 1`) automatically scrolls the canvas to `$P$`.
3. **Completion Trigger**: Reaching `$P == page_count` or clicking **"Next Chapter"** automatically submits `is_read: true`.
4. **Library Card Badges**: Shows `X unread` pill badge or green `Completed` pill when all chapters are read.
