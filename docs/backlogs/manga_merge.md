# Backlog: Manga Merge

> **Status**: Proposed / Backlog
> **Target Components**: `web/src`, `internal/api`, `internal/library`

---

## Overview

Allow users to merge two library entries that represent the same manga (added from different providers or by accident). Merging combines `providers[]` from both sources, resolves metadata conflicts, and reconciles chapters.

## Problem

Library allows duplicate entries. Users may add the same manga twice via different providers (Kitsu vs MAL) without realizing it. Each entry has its own `providers[]`, chapters, and reading progress. Duplicates clutter the library and waste storage.

## Proposed UX

- **Detection**: title/author fuzzy match on add, low-priority banner: "Similar entry exists, merge?"
- **Manual merge**: manga detail shows "Similar entries" section with merge CTA
- **Step 1**: pick source + target
- **Step 2**: resolve metadata conflicts field-by-field
- **Step 3**: handle chapter overlap (move from source, dedup by title/number, or keep separate)
- **Step 4**: confirm + execute

## Constraints

- `providers[]` from both entries merged into target, deduped by `provider_id + manga_id`
- Reading progress preserved when chapter matches
- Source entry folders deleted after merge
- Reversible? No — destructive op, single confirmation step

## Open Questions

- Auto-detect on add vs only manual merge?
- Merge while one entry has unread chapters — preserve progress per chapter or move?
- Provider conflict: same provider with different `manga_id` in each entry — keep both, or pick one?
