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
| | `/library/manga/{id}/refresh` | `POST` | Refresh chapter metadata from active content provider |
| | `/library/manga/{id}/pull` | `POST` | Enqueue background job (`pull_manga`) to pull all missing chapters |
| | `/library/manga/{id}` | `PUT` | Full replacement update of manga manifest metadata |
| | `/library/manga/{id}` | `PATCH` | Partial update of manga metadata and user tracking fields |
| | `/library/manga/{id}` | `DELETE` | Delete manga and associated files from local library |
| **Provider Bindings** | `/library/manga/{id}/providers` | `GET` | List bound providers for a manga |
| | `/library/manga/{id}/providers` | `POST` | Bind a new metadata/content provider to manga |
| | `/library/manga/{id}/providers/{providerId}/{providerMangaId}` | `DELETE` | Remove a provider binding from manga |
| | `/library/manga/{id}/content` | `PATCH` | Switch active content provider namespace |
| **Chapters & Pages** | `/library/manga/{id}/chapters` | `GET` | List chapters for manga (optional `?provider_id=...`) |
| | `/library/manga/{id}/providers/{providerId}/chapters/{ch}` | `GET` | Get single chapter manifest metadata |
| | `/library/manga/{id}/providers/{providerId}/chapters/{ch}` | `POST` | Create or update chapter manifest metadata |
| | `/library/manga/{id}/providers/{providerId}/chapters/{ch}/progress` | `PATCH` | Update reading progress (`is_read`, `last_read_page`) |
| | `/library/manga/{id}/providers/{providerId}/chapters/{ch}` | `DELETE` | Delete chapter manifest and downloaded files |
| | `/library/manga/{id}/providers/{providerId}/chapters/{ch}/files` | `DELETE` | Delete downloaded page images only (preserves chapter metadata) |
| | `/chapters/{ch}/pages` | `GET` | Get resolved page list (`index`, `url`, `source`) |
| | `/library/manga/{id}/providers/{providerId}/chapters/{ch}/pull` | `POST` | Enqueue background job (`pull_chapter`) for single chapter |
| **Batch Operations** | `/library/manga/{id}/providers/{providerId}/chapters/progress` | `PATCH` | Batch update reading progress for multiple chapters |
| | `/library/manga/{id}/providers/{providerId}/chapters/pull` | `POST` | Batch enqueue pull jobs for multiple chapters |
| | `/library/manga/{id}/providers/{providerId}/chapters/refresh` | `POST` | Batch refresh chapter manifests from upstream provider |
| | `/library/manga/{id}/providers/{providerId}/chapters/files/delete` | `POST` | Batch delete page images for multiple chapters |
| | `/library/manga/{id}/providers/{providerId}/chapters/files` | `DELETE` | Batch delete page images (alternative DELETE verb) |
| | `/library/manga/{id}/providers/{providerId}/chapters/delete` | `POST` | Batch delete chapter manifests and files |
| | `/library/manga/{id}/providers/{providerId}/chapters` | `DELETE` | Batch delete chapter manifests and files (alternative DELETE verb) |
| **Reverse Proxy** | `/library/manga/{id}/chapters/{ch}/pages/{n}` | `GET` | Serve page image via 3-tier resolution (disk → cache → remote) |
| | `/proxy/image` | `GET` | Direct image proxy with SSRF guard and TLS fingerprinting |
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

### 5. Refresh Manga Chapters (`POST /library/manga/{id}/refresh`)
Queries the active content provider for updated chapter listings, creates manifests for newly released chapters, and flags orphaned entries.

**Response (`200 OK`):**
```json
{
  "added": 3,
  "orphaned": 0,
  "updated": 0,
  "provider_id": "mangadex",
  "manga_id": "01JABCD1234EFGH5678IJKL90M"
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

### 7. Update Library Manga (`PUT /library/manga/{id}`) & (`PATCH /library/manga/{id}`)
- `PUT` performs a full replacement of the manga manifest.
- `PATCH` applies partial updates to metadata overrides and user tracking fields.

**Supported User Tracking Fields:**
- `user_status` (`string`): `unread`, `reading`, `completed`, `on_hold`, `dropped`, `plan_to_read`.
- `user_favorite` (`boolean`): Favorite / starred toggle.
- `user_rating` (`number`): Score rating (`0.0` to `10.0`, `0` = unrated).
- `user_notes` (`string`): Freeform personal notes.
- Metadata overrides (`title`, `aliases`, `description`, `authors`, `artists`, `tags`, `collections`, `content.reading_mode`).

---

## Provider Bindings Endpoints

### 1. List Providers (`GET /library/manga/{id}/providers`)
Returns all metadata and content providers bound to this manga.

### 2. Add Provider Binding (`POST /library/manga/{id}/providers`)
Binds an external provider to the library entry.

**Request:**
```json
{
  "provider_id": "mangafox",
  "provider_manga_id": "frieren_beyond_journeys_end",
  "manga_title": "Sousou no Frieren",
  "set_as_content": true
}
```

### 3. Remove Provider Binding (`DELETE /library/manga/{id}/providers/{providerId}/{providerMangaId}`)
Unbinds a provider from the manga. If the provider is currently the active content provider, the operation is rejected unless another content-capable provider remains.

### 4. Switch Content Provider (`PATCH /library/manga/{id}/content`)
Switches the active content provider namespace used for chapter indexing and reading.

**Request:**
```json
{
  "provider_id": "mangafox",
  "provider_manga_id": "frieren_beyond_journeys_end"
}
```

---

## Chapters & Pages Endpoints

### 1. List Chapters (`GET /library/manga/{id}/chapters`)
Returns the chapter list for a manga. When `?provider_id=` is omitted, it defaults to the active content provider. Chapters belonging to other provider namespaces are flagged with `"orphaned": true`.

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

### 2. Chapter Progress (`PATCH /library/manga/{id}/providers/{providerId}/chapters/{ch}/progress`)
Updates the read status and last read page for a single chapter.

**Request:**
```json
{
  "is_read": true,
  "last_read_page": 18
}
```

### 3. Pull Chapter (`POST /library/manga/{id}/providers/{providerId}/chapters/{ch}/pull`)
Enqueues a background `pull_chapter` job.

**Response (`202 Accepted`):**
```json
{
  "job_id": "84d79169-2f5a-4b92-93cb-339fa8a5a40b",
  "chapter_id": "ch-101"
}
```

### 4. Delete Chapter Files (`DELETE /library/manga/{id}/providers/{providerId}/chapters/{ch}/files`)
Deletes downloaded page image files from the chapter directory while preserving the chapter manifest and reading progress.

### 5. Get Chapter Pages (`GET /chapters/{chapterId}/pages`)
Resolves page image URLs and source locations (`disk`, `cache`, or `provider`).

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

---

## Batch Chapter Operations

Multi-select batch endpoints for chapter management:

| Operation | Path | Request Body |
|---|---|---|
| **Batch Progress** | `PATCH /library/manga/{id}/providers/{providerId}/chapters/progress` | `{ "chapter_ids": ["ch-1", "ch-2"], "is_read": true, "last_read_page": 0 }` |
| **Batch Pull** | `POST /library/manga/{id}/providers/{providerId}/chapters/pull` | `{ "chapter_ids": ["ch-1", "ch-2", "ch-3"] }` |
| **Batch Refresh** | `POST /library/manga/{id}/providers/{providerId}/chapters/refresh` | `{ "chapter_ids": ["ch-1", "ch-2"] }` |
| **Batch Delete Files** | `POST /library/manga/{id}/providers/{providerId}/chapters/files/delete` (or `DELETE .../files`) | `{ "chapter_ids": ["ch-1", "ch-2"] }` |
| **Batch Delete Chapters** | `POST /library/manga/{id}/providers/{providerId}/chapters/delete` (or `DELETE .../chapters`) | `{ "chapter_ids": ["ch-1", "ch-2"] }` |

---

## Reverse Proxy & Page Image Streaming

### 1. Chapter Page Image Proxy (`GET /library/manga/{id}/chapters/{ch}/pages/{n}`)
Resolves and streams a specific page image via the 3-tier resolution engine:
1. **Disk**: Inspects local chapter directory for downloaded page file.
2. **Ephemeral Cache**: Checks the disk image cache.
3. **Live Remote Proxy**: Streams from upstream provider via TLS fingerprinting transport and SSRF guard.

### 2. Direct Image Proxy (`GET /proxy/image`)
Direct proxy endpoint for remote covers, banners, and image assets. Enforces SSRF validation and TLS fingerprint matching.

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
