# REST API

## Overview

Kiyomi exposes a RESTful HTTP API consumed by the Web UI and external clients. The API serves as the formal boundary and contract between the server subsystems (library storage, background job queue, content/metadata providers, plugins, and proxy engine) and user interfaces or automation tools.

---

## Transport & Architecture

```
┌─────────────────────────────────────────────────────────┐
│                      HTTP Transport                     │
│                                                         │
│   Routing Engine → Request Validation → Service Layer   │
│                                                         │
│   - Request Context & ID Tracking                       │
│   - Cross-Origin Resource Sharing (CORS)                │
│   - Standard Error Envelope Formatting                  │
│   - Selective Failure Logging                           │
└─────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────┐
│                     Service Layer                       │
│                                                         │
│   - Library & Manifest Storage                          │
│   - Provider Registry & Plugin Host                     │
│   - Background Job Queue & Concurrency Scheduler        │
│   - Reverse Proxy & SSRF Guard                          │
│   - Ephemeral Image Disk Cache                          │
└─────────────────────────────────────────────────────────┘
```

- **Protocol**: REST over HTTP/1.1.
- **Wire Format**: JSON request and response bodies.
- **Base Prefix**: All API endpoints are served under `/api/v1/`.

---

## Resource Model

| Category | Endpoint URI | Method | Description |
|---|---|---|---|
| **System Info** | `/info` | `GET` | Server version, build metadata, and runtime status |
| **System Cache** | `/system/cache` | `GET` | Disk image cache statistics (item count, total bytes, hit rates) |
| | `/system/cache/clear` | `POST` | Flush and purge ephemeral disk image cache |
| **Content Providers** | `/providers` | `GET` | List all registered content and metadata providers |
| | `/providers/{providerId}/manga` | `GET` | Paginated catalog search and mode browsing (`q`, `mode`, `page`) |
| | `/providers/{providerId}/popular` | `GET` | Popular manga catalog feed |
| | `/providers/{providerId}/latest` | `GET` | Latest updated manga catalog feed |
| | `/providers/{providerId}/search` | `GET` | Text search across provider catalog (`q`) |
| | `/providers/{providerId}/manga/{remoteId}` | `GET` | Upstream manga details, synopsis, and metadata |
| | `/providers/{providerId}/manga/{remoteId}/chapters` | `GET` | Upstream chapter release list |
| **Provider Fingerprint** | `/providers/{providerId}/fingerprint` | `GET` | Retrieve active TLS fingerprint and cookie state |
| | `/providers/{providerId}/fingerprint` | `PUT` | Update Cloudflare/anti-bot clearance cookies and fingerprint |
| | `/providers/{providerId}/fingerprint` | `DELETE` | Clear stored session cookies and fingerprint |
| **Plugin Management** | `/plugins` | `GET` | List loaded provider plugins and capabilities |
| | `/plugins/reload` | `POST` | Hot-reload plugins from disk |
| | `/plugins/{id}/logs` | `GET` | Retrieve runtime log output for a specific plugin |
| | `/plugins/{id}/config` | `POST` | Update configuration settings for a plugin |
| | `/plugins/collisions` | `GET` | List provider ID collisions between built-in and plugin sources |
| | `/plugins/preference` | `POST` | Set preferred provider implementation for colliding IDs |
| **Central Local Library Manga** | `/library/manga` | `GET` | List all local library manga entries |
| | `/library/manga/{id}` | `GET` | Get detailed metadata for a local manga entry |
| | `/library/manga` | `POST` | Create a new local library manga entry |
| | `/library/manga/import` | `POST` | Import manga from provider by remote ID with initial manifests |
| | `/library/manga/merge` | `POST` | Merge two existing library manga records |
| | `/library/manga/{id}/pull` | `POST` | Enqueue background job (`pull_manga`) to pull all missing chapters |
| | `/library/manga/{id}` | `PATCH` | Partial update of manga metadata overrides and user tracking fields |
| | `/library/manga/{id}` | `DELETE` | Delete manga and associated files from local library |
| **Manga Concern Files** | `/library/manga/{id}/metadata` | `GET` | Retrieve series metadata manifest (`metadata.json`) |
| | `/library/manga/{id}/metadata` | `PATCH` | Update series metadata manifest |
| | `/library/manga/{id}/metadata/refresh` | `POST` | Enqueue background job to refresh metadata from upstream provider |
| | `/library/manga/{id}/user_state` | `GET` | Retrieve user reading state and personal notes (`user_state.json`) |
| | `/library/manga/{id}/user_state` | `PATCH` | Update user reading state and personal notes |
| | `/library/manga/{id}/bindings` | `GET` | Retrieve bound provider references and active content pointer |
| | `/library/manga/{id}/bindings` | `POST` | Bind a new metadata/content provider to manga |
| | `/library/manga/{id}/bindings/{providerId}/{remoteId}` | `DELETE` | Remove a provider binding from manga |
| | `/library/manga/{id}/bindings/content` | `PATCH` | Switch active content provider namespace |
| **Chapters & Pages** | `/library/manga/{id}/chapters` | `GET` | List chapters for manga (optional `?provider_id=...`, `?orphaned=...`) |
| | `/library/manga/{id}/chapters` | `POST` | Create or import single chapter manifest metadata |
| | `/library/manga/{id}/chapters/{chapterId}` | `GET` | Get single chapter manifest metadata |
| | `/library/manga/{id}/chapters/{chapterId}` | `DELETE` | Delete chapter manifest and downloaded files |
| | `/library/manga/{id}/chapters/{chapterId}/progress` | `PATCH` | Update reading progress (`is_read`, `last_read_page`) |
| | `/library/manga/{id}/chapters/{chapterId}/pull` | `POST` | Enqueue background job (`pull_chapter`) for single chapter |
| | `/library/manga/{id}/chapters/{chapterId}/refresh` | `POST` | Refresh single chapter manifest from provider |
| | `/library/manga/{id}/chapters/{chapterId}/files` | `GET` | Inspect downloaded page files for chapter |
| | `/library/manga/{id}/chapters/{chapterId}/files` | `DELETE` | Delete downloaded page images only (preserves chapter metadata) |
| | `/library/manga/{id}/chapters/{chapterId}/pages` | `GET` | Get resolved page list (`index`, `url`, `source`) |
| | `/library/manga/{id}/chapters/{chapterId}/pages/{pageIndex}` | `GET` | Serve page image via 3-tier resolution (disk → cache → remote) |
| **Batch Chapter Operations** | `/library/manga/{id}/batch/chapters/progress` | `POST` | Batch update reading progress for multiple chapters |
| | `/library/manga/{id}/batch/chapters/pull` | `POST` | Batch enqueue pull jobs for multiple chapters |
| | `/library/manga/{id}/batch/chapters/refresh` | `POST` | Batch refresh chapter manifests from upstream provider |
| | `/library/manga/{id}/batch/chapters/delete` | `POST` | Batch delete chapter manifests and downloaded files |
| | `/library/manga/{id}/batch/chapter-files/delete` | `POST` | Batch delete page images for multiple chapters (preserves manifests) |
| **Reverse Proxy** | `/proxy/image` | `GET` | Direct image proxy with SSRF guard and TLS fingerprinting |
| **Background Jobs** | `/jobs` | `GET` | List background jobs with filters (`status`, `type`, `metadata.*`) |
| | `/jobs` | `POST` | Submit generic background job |
| | `/jobs/{id}/cancel` | `POST` | Cancel/abort pending or running job and child jobs |
| | `/jobs/{id}` | `DELETE` | Permanently remove job and child records (or cancel via `?action=cancel`) |
| | `/jobs` | `DELETE` | Bulk cleanup finished jobs (`?status=completed,failed`) |

---

## Central Local Library Manga Endpoints

### 1. List Library Manga (`GET /library/manga`)
Returns an array of local manga entries with metadata summary and active content provider.

**Response (`200 OK`):**
```json
[
  {
    "id": "01JABCD1234EFGH5678IJKL90M",
    "title": "Frieren: Beyond Journey's End",
    "cover": "https://example.com/covers/cover.jpg",
    "content_provider_id": "mangadex",
    "reading_mode": "longstrip",
    "meta": {
      "title": "Frieren: Beyond Journey's End",
      "user_status": "reading",
      "user_favorite": true,
      "user_rating": 9.5
    }
  }
]
```

### 2. Get Library Manga Details (`GET /library/manga/{id}`)
Returns full metadata manifest, aliases, tags, creator credits, and bound provider information.

### 3. Create Library Manga (`POST /library/manga`)
Creates a local manga entry manually without fetching from an external provider.

**Request:**
```json
{
  "id": "01JABCD1234EFGH5678IJKL90M",
  "meta": {
    "title": "Custom Local Series",
    "authors": ["Author Name"],
    "user_status": "plan_to_read"
  },
  "content": {
    "provider_id": "local",
    "reading_mode": "rtl"
  }
}
```

### 4. Import Manga from Provider (`POST /library/manga/import`)
Fetches manga details and chapter lists from an upstream provider, initializing local library manifests.

**Request:**
```json
{
  "provider_id": "mangadex",
  "remote_id": "a1c7c817-4e59-4220-9e80-77114d5e2197",
  "user_status": "reading"
}
```

**Response (`201 Created`):** Returns the initialized manga entity.

### 5. Merge Library Manga (`POST /library/manga/merge`)
Merges two library manga records into a single entry, combining chapter manifests and bindings.

**Request:**
```json
{
  "source_id": "01JOLDMANGA1234567890ABCD",
  "target_id": "01JABCD1234EFGH5678IJKL90M"
}
```

### 6. Pull Manga (`POST /library/manga/{id}/pull`)
Enqueues a background `pull_manga` job that resolves all chapters and schedules page downloads.

**Response (`202 Accepted`):**
```json
{
  "job_id": "3c988a38-51b6-4df0-ba9e-5e3e2cf63470",
  "provider_id": "mangadex"
}
```

### 7. Partial Update Library Manga (`PATCH /library/manga/{id}`)
Applies partial updates to metadata overrides and user tracking fields.

**Supported User Tracking Fields:**
- `user_status` (`string`): `unread`, `reading`, `completed`, `on_hold`, `dropped`, `plan_to_read`.
- `user_favorite` (`boolean`): Favorite / starred toggle.
- `user_rating` (`number`): Score rating (`0.0` to `10.0`, `0` = unrated).
- `user_notes` (`string`): Freeform personal notes.
- Metadata overrides (`title`, `aliases`, `description`, `authors`, `artists`, `tags`, `collections`, `content.reading_mode`).

---

## Per-Concern Manga Endpoints

The manga manifest is split into three concern files (`metadata.json`, `user_state.json`, `bindings.json`). The following endpoints provide targeted read and write access per concern without touching the others.

### 1. Get Metadata Concern (`GET /library/manga/{id}/metadata`)
Returns the content of `metadata.json` — provider-supplied series metadata (title, description, authors, tags, cover_url, etc.).

**Response (`200 OK`):**
```json
{
  "title": "Frieren: Beyond Journey's End",
  "aliases": ["Frieren", "葬送のフリーレン"],
  "description": "Inside the kingdom of Valor, Hermit...",
  "authors": ["Kanehito, Yamada"],
  "artists": ["Matsumoto, Tsukasa"],
  "tags": ["type:manga", "demographic:shounen", "Adventure", "Fantasy"],
  "cover_url": "https://uploads.mangadex.org/covers/abc/cover.jpg",
  "external_links": [...]
}
```

### 2. Replace Metadata Concern (`PATCH /library/manga/{id}/metadata`)
Replaces the entire content of `metadata.json`. Used when importing or overwriting provider-supplied metadata. Does not affect `user_state.json` or `bindings.json`.

**Request:**
```json
{
  "title": "Frieren: Beyond Journey's End",
  "description": "Updated description...",
  "authors": ["Kanehito, Yamada"],
  "tags": ["type:manga", "Adventure"]
}
```

### 3. Get User State Concern (`GET /library/manga/{id}/user_state`)
Returns the content of `user_state.json` — user reading state fields (status, rating, favorite, notes, last_read_chapter_id, etc.).

**Response (`200 OK`):**
```json
{
  "user_status": "reading",
  "user_rating": 9.5,
  "user_favorite": true,
  "user_notes": "Beautiful story about time and mortality.",
  "last_read_chapter_id": "ch-14",
  "last_read_at": "2026-08-15T21:00:00Z",
  "added_at": "2026-07-01T00:00:00Z",
  "updated_at": "2026-08-15T21:00:00Z"
}
```

### 4. Replace User State Concern (`PATCH /library/manga/{id}/user_state`)
Replaces the entire content of `user_state.json`. Used for bulk user-state updates. Does not affect `metadata.json` or `bindings.json`.

**Request:**
```json
{
  "user_status": "completed",
  "user_rating": 10.0,
  "user_favorite": true,
  "last_read_chapter_id": "ch-65",
  "last_read_at": "2026-09-01T18:00:00Z"
}
```

### 5. Get Bindings Concern (`GET /library/manga/{id}/bindings`)
Returns the content of `bindings.json` — the provider bindings list and active content source pointer.

**Response (`200 OK`):**
```json
{
  "content": {
    "provider_id": "mangadex",
    "provider_manga_id": "a1b2c3d4",
    "reading_mode": "longstrip",
    "last_synced_at": "2026-08-06T10:00:00Z"
  },
  "providers": [
    {
      "provider_id": "mangadex",
      "provider_manga_id": "a1b2c3d4",
      "manga_title": "Frieren: Beyond Journey's End"
    }
  ]
}
```

### 6. Enqueue Metadata Refresh (`POST /library/manga/{id}/metadata/refresh`)
Enqueues a `refresh_metadata` background job that fetches the latest metadata from a provider's metadata capability and overwrites `metadata.json`. Does not update `user_state.json` or `bindings.json`.

**Response (`202 Accepted`):**
```json
{
  "job_id": "8f3b6c2a-9e12-4d57-b184-fa6d7e29c0a1",
  "manga_id": "01JABCD1234EFGH5678IJKL90M",
  "provider_id": "mangadex"
}
```

---

## Provider Bindings Endpoints

Manga entries support multi-provider bindings. Bindings define which external provider sources are attached to a library entry and specify the active content provider.

### 1. List Bindings (`GET /library/manga/{id}/bindings`)
Returns all metadata and content provider references bound to this manga.

### 2. Add Provider Binding (`POST /library/manga/{id}/bindings`)
Binds an external provider to the library entry. Updates `bindings.json`.

**Request:**
```json
{
  "provider_id": "mangafox",
  "remote_id": "frieren_beyond_journeys_end",
  "manga_title": "Sousou no Frieren",
  "set_as_content": true
}
```

### 3. Remove Provider Binding (`DELETE /library/manga/{id}/bindings/{providerId}/{remoteId}`)
Unbinds a provider from the manga. If the provider is currently the active content provider, the operation is rejected unless another content-capable provider remains.

### 4. Switch Content Provider (`PATCH /library/manga/{id}/bindings/content`)
Switches the active content provider namespace used for chapter indexing and reading.

**Request:**
```json
{
  "provider_id": "mangafox",
  "remote_id": "frieren_beyond_journeys_end"
}
```

---

## Chapters & Pages Endpoints

Chapters belong to a manga collection. Because each chapter record internally tracks its authoritative `provider_id`, chapter endpoints do not include `{providerId}` in the URL path, eliminating sibling routing collisions and ensuring clean hierarchical REST semantics.

### 1. List Chapters (`GET /library/manga/{id}/chapters`)
Returns the chapter list for a manga. Supports query parameters `?provider_id=...` to filter by provider and `?orphaned=true|false`. When `?provider_id=` is omitted, it defaults to the active content provider.

**Response (`200 OK`):**
```json
{
  "chapters": [
    {
      "id": "ch-101",
      "manga_id": "01JABCD1234EFGH5678IJKL90M",
      "title": "Chapter 101",
      "number": 101.0,
      "volume": 11,
      "uploadDate": "2026-08-01T00:00:00Z",
      "sourceOrder": 101,
      "provider_id": "mangadex",
      "is_downloaded": true,
      "downloaded_pages": 18,
      "page_count": 18,
      "downloaded_at": "2026-08-02T12:00:00Z",
      "meta": {
        "title": "Chapter 101",
        "number": 101.0,
        "is_read": true,
        "last_read_page": 18,
        "orphaned": false
      }
    }
  ]
}
```

### 2. Create or Import Chapter (`POST /library/manga/{id}/chapters`)
Saves or manually registers chapter metadata in the library.

### 3. Get Chapter Details (`GET /library/manga/{id}/chapters/{chapterId}`)
Returns detailed metadata manifest for an individual chapter.

### 4. Delete Chapter (`DELETE /library/manga/{id}/chapters/{chapterId}`)
Deletes chapter manifest and all associated downloaded page files.

### 5. Update Chapter Progress (`PATCH /library/manga/{id}/chapters/{chapterId}/progress`)
Updates the read status and last read page for a single chapter.

**Request:**
```json
{
  "is_read": true,
  "last_read_page": 18
}
```

### 6. Pull Chapter (`POST /library/manga/{id}/chapters/{chapterId}/pull`)
Enqueues an asynchronous background job (`pull_chapter`) to download page images for the chapter.

**Response (`202 Accepted`):**
```json
{
  "job_id": "84d79169-2f5a-4b92-93cb-339fa8a5a40b",
  "chapter_id": "ch-101"
}
```

### 7. Refresh Chapter (`POST /library/manga/{id}/chapters/{chapterId}/refresh`)
Refreshes chapter manifest metadata from its upstream provider.

### 8. Chapter Files (`GET /library/manga/{id}/chapters/{chapterId}/files` & `DELETE /library/manga/{id}/chapters/{chapterId}/files`)
- `GET`: Lists downloaded image files, file sizes, and verification checksums for the chapter.
- `DELETE`: Purges downloaded page images from disk while preserving the chapter metadata manifest and user reading progress.

### 9. Get Chapter Pages Manifest (`GET /library/manga/{id}/chapters/{chapterId}/pages`)
Resolves page image URLs and current storage locations (`disk`, `cache`, or `provider`).

**Response (`200 OK`):**
```json
{
  "pages": [
    {
      "index": 1,
      "url": "https://example.com/data/01.jpg",
      "source": "library"
    },
    {
      "index": 2,
      "url": "https://example.com/data/02.jpg",
      "source": "provider"
    }
  ]
}
```

### 10. Serve Chapter Page Image (`GET /library/manga/{id}/chapters/{chapterId}/pages/{pageIndex}`)
Resolves and streams a specific page image via the 3-tier resolution engine:
1. **Disk**: Inspects local chapter directory for downloaded page file.
2. **Ephemeral Cache**: Checks the disk image cache.
3. **Live Remote Proxy**: Streams from upstream provider via TLS fingerprinting transport and SSRF guard.

---

## Batch Chapter Operations

To maintain absolute zero routing collisions in radix trie routing engines (such as Echo), batch operations are strictly isolated from individual resource paths (`/chapters/{chapterId}/...`) into a dedicated batch sub-resource namespace under `/library/manga/{id}/batch/chapters/...`.

| Operation | Path | Request Body | Description |
|---|---|---|---|
| **Batch Progress** | `POST /library/manga/{id}/batch/chapters/progress` | `{ "chapter_ids": ["ch-1", "ch-2"], "is_read": true, "last_read_page": 0 }` | Bulk updates read status and last read page |
| **Batch Pull** | `POST /library/manga/{id}/batch/chapters/pull` | `{ "chapter_ids": ["ch-1", "ch-2"] }` | Bulk enqueues chapter pull download jobs |
| **Batch Refresh** | `POST /library/manga/{id}/batch/chapters/refresh` | `{ "chapter_ids": ["ch-1", "ch-2"] }` | Bulk refreshes chapter manifests from provider |
| **Batch Delete Chapters** | `POST /library/manga/{id}/batch/chapters/delete` | `{ "chapter_ids": ["ch-1", "ch-2"] }` | Bulk deletes chapter manifests and downloaded files |
| **Batch Delete Files** | `POST /library/manga/{id}/batch/chapter-files/delete` | `{ "chapter_ids": ["ch-1", "ch-2"] }` | Bulk deletes downloaded page images only |

---

## Reverse Proxy & Direct Image Streaming

### 1. Direct Image Proxy (`GET /proxy/image`)
Direct proxy endpoint for remote covers, banners, and external image assets (`?url=...`). Enforces SSRF validation against loopback and RFC 1918 private ranges, and performs browser TLS fingerprint matching.

### 2. Chapter Page Streaming
Chapter page images are streamed through the dedicated resource route:
`GET /library/manga/{id}/chapters/{chapterId}/pages/{pageIndex}`
Backed by the 3-tier resolution engine (Disk → Ephemeral Cache → Upstream TLS Proxy).

---

## Background Jobs Endpoints

Background tasks (e.g. `pull_manga`, `pull_chapter`, `pull_page`, `metadata_refresh`) are managed through the generic job queue:

### 1. List Jobs (`GET /jobs`)
Retrieves job records with query filters (`status`, `type`, `parent_id`, `all`, `metadata.*`).

**Response (`200 OK`):**
```json
[
  {
    "id": "84d79169-2f5a-4b92-93cb-339fa8a5a40b",
    "parent_id": "3c988a38-51b6-4df0-ba9e-5e3e2cf63470",
    "type": "pull_chapter",
    "status": "running",
    "max_retries": 3,
    "retries": 0,
    "concurrency_group": "pull:mangadex",
    "metadata": {
      "manga_id": "01JABCD...",
      "provider_id": "mangadex",
      "chapter_id": "ch-101"
    },
    "created_at": "2026-08-31T05:00:00Z",
    "updated_at": "2026-08-31T05:00:02Z"
  }
]
```

### 2. Submit Generic Job (`POST /jobs`)
Submits a generic background job payload for scheduling.

### 3. Cancel Job (`POST /jobs/{id}/cancel` or `DELETE /jobs/{id}?action=cancel`)
Aborts a pending or running job and recursively cancels all child jobs.

### 4. Delete Job (`DELETE /jobs/{id}`)
Permanently deletes the job record and its descendants from the job database.

### 5. Cleanup Jobs (`DELETE /jobs?status=...`)
Bulk deletes finished jobs by status criteria (`completed`, `failed`, `finished`, `all`).

---

## Error Handling & Status Codes

All API errors return a standard JSON error envelope:

```json
{
  "error": "human-readable diagnostic error message",
  "provider_id": "optional provider identifier",
  "kind": "optional error classification (transient | auth | permanent | rate_limit)"
}
```

| HTTP Status | Semantics |
|---|---|
| `200 OK` | Request succeeded; payload returned in body. |
| `201 Created` | Resource successfully created. |
| `202 Accepted` | Asynchronous job scheduled and enqueued. |
| `204 No Content` | Request succeeded with no body returned. |
| `400 Bad Request` | Input validation failed or malformed JSON payload. |
| `404 Not Found` | Requested manga, chapter, job, or provider does not exist. |
| `409 Conflict` | Resource conflict (e.g. duplicate provider binding, removing last content provider). |
| `429 Too Many Requests` | Upstream or local rate limit exceeded. |
| `500 Internal Server Error` | Unhandled server error. |
| `502 Bad Gateway` | Upstream provider failure or network error. |
| `503 Service Unavailable` | Subsystem not configured or temporarily offline. |

---

## Security & Observability

1. **SSRF Guard**: Direct and proxy image requests validate destination addresses against private, loopback, and cloud metadata IP ranges.
2. **TLS Fingerprint Spoofing**: Upstream requests replicate standard browser TLS Client Hello signatures and headers.
3. **Structured Error Logging**: Handlers log failed requests (`status >= 400` or internal errors) with route URI, provider ID, status code, and underlying error cause.

---

## References

- [Multi-Content Provider Library](./multi_content_provider_library.md) — Filesystem layout and provider namespaces.
- [Background Job Queue](./background_jobs.md) — Generic background task runner and queue schema.
- [Content Pull System](./content_pull.md) — Chapter and page pull architecture.
- [Reader Design](./reader.md) — 3-tier page resolution and reading modes.
- [Providers Design](./providers.md) — Provider SDK contracts and capabilities.
- [Anti-Bot Strategy](./antibot.md) — Browser fingerprinting and session management.
