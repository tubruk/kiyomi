# Content Pull System

## Overview

Content pull acquires manga content from upstream providers and persists it into the local Kiyomi library. This is a high-level feature document; the underlying job queue infrastructure is documented in [background_jobs.md](./background_jobs.md).

---

## Pull Naming Rationale

"Pull" is used instead of "download" to distinguish server-to-upstream acquisition from client-initiated downloads. "Download" implies the client is pulling a file to their device. "Pull" conveys the server fetching content from an external source into the local library, avoiding user confusion about the direction of data flow.

---

## Pull Operations

### `pull_manga`

Fetches the chapter list from a provider and schedules pulls for all chapters not yet in the library.

**Trigger**: User-initiated "Pull manga" action, or background sync when library auto-pull is enabled.

**Flow**:
1. Query provider API for chapter list for given manga.
2. Compare against existing local manifest.
3. Enqueue `pull_chapter` jobs for each missing chapter.

### `pull_chapter`

Resolves the full page list for a chapter and schedules individual page pulls.

**Trigger**: Enqueued by `pull_manga`, or user-initiated "Pull chapter".

**Flow**:
1. Query provider API for page URLs.
2. Store resolved page list in local manifest.
3. Enqueue `pull_page` jobs for each page.

### `pull_page`

Fetches a single page image from the upstream provider and writes it to the local filesystem.

**Path structure**: `<library_root>/<manga_id>/<provider_id>/<chapter_id>/<page_number>.<ext>`

**Behavior**:
- Fetch page image via HTTP.
- Write to target path atomically (write-then-rename to avoid partial reads).
- Update local page manifest on success.
- Retry with backoff on transient failures.
- Mark failed after max retries; log error details.

---

## Provider Integration

Each provider implements a common interface contract:

- **ListChapters**: Returns all chapter metadata for a manga from the provider.
- **ListPages**: Returns resolved page image URLs for a chapter.
- **FetchPage**: Performs the HTTP fetch for a single page.

Providers may implement additional methods for authentication, rate limiting, and Cloudflare challenge handling.

---

## Concurrency

Pull operations are scoped to a **provider concurrency key**: `pull:<provider_id>`. Only `N` pulls from the same provider run concurrently, preventing IP bans or rate-limit hits from upstream providers.

The job queue concurrency system (documented in background_jobs.md) enforces this via concurrency group keys on the handler.

---

## Error Handling

| Error type | Behavior |
|------------|----------|
| Transient (timeout, 5xx) | Retry with exponential backoff |
| Permanent (404, auth failure) | Fail immediately, mark job failed |
| Provider-specific (CF challenge, captcha) | Handled by provider implementation; may defer or skip |

---

## Persistence

**On disk**: `<library_root>/<manga_id>/<provider_id>/<chapter_id>/<page_number>.<ext>`

**In manifest**: `manifest.json` at the chapter level stores resolved page metadata. This enables re-verification and re-pull of corrupt pages without re-resolving page URLs from the provider.
