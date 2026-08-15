# API

> **Status**: Foundational design, scaffolding.

## Overview

Kiyomi exposes a RESTful HTTP API consumed by the Web UI and any third-party client. The API is the contract between the server's internal subsystems (library, workers, providers) and external callers. Internal subsystems do not bypass the API; they share the same handler layer.

## Transport

```
┌─────────────────────────────────────────────────────────┐
│                      HTTP Server                         │
│                                                          │
│   Middleware stack → Route group → Handler → Service     │
│                                                          │
│   - Logging                                             │
│   - Recovery (panic → 500)                              │
│   - Request ID                                          │
│   - CORS                                                │
│   - Content negotiation                                 │
└─────────────────────────────────────────────────────────┘
                              │
                              ▼
                    ┌──────────────────┐
                    │  Service layer   │
                    │  (library, etc.) │
                    └──────────────────┘
```

REST over HTTP/1.1. JSON request/response bodies. No streaming endpoints for the core CRUD — WebSocket or SSE reserved for future progress streaming.

## Resource Model

API resources map to library concepts:

| Resource | URI | Owner |
|---|---|---|
| Library manga | `/library/manga` | Library |
| Manga details | `/library/manga/{id}` | Library |
| Chapter list | `/library/manga/{id}/chapters` | Library |
| Chapter detail | `/library/manga/{id}/chapters/{ch}` | Library |
| Page list | `/library/manga/{id}/chapters/{ch}/pages` | Library |
| Page image | `/library/manga/{id}/chapters/{ch}/pages/{n}` | Library/Reader |
| Reading progress | `/progress/manga/{id}` | Library |
| Chapter progress | `/progress/manga/{id}/chapters/{ch}` | Library |
| Providers | `/providers` | Provider registry |
| Provider search | `/providers/{id}/search` | Provider |
| Provider metadata | `/providers/{id}/manga/{rid}` | Provider |
| Explore | `/explore` | Provider |
| Downloads | `/downloads` | Workers |
| Jobs | `/jobs` | Workers |
| Cache | `/cache` | Cache |

Mutations are explicit HTTP verbs. Each URI is a noun, verb on URL is forbidden.

### Library Entry Management (`PATCH /library/manga/{id}`)

Clients can partially update user tracking and custom metadata fields for a library manga entry via `PATCH /api/v1/library/manga/{id}`:

```json
PATCH /api/v1/library/manga/01JABCD1234EFGH5678IJKL90M
Content-Type: application/json

{
  "user_status": "reading",
  "user_favorite": true,
  "user_rating": 8.0,
  "user_notes": "Favorite arc starts around chapter 15."
}
```

**Supported Fields:**
- `user_status` (`string`): One of `unread`, `reading`, `completed`, `on_hold`, `dropped`, `plan_to_read`.
- `user_favorite` (`boolean`): Toggles favorite / starred status.
- `user_rating` (`number`): Floating point score between `0.0` and `10.0` (`0` represents unrated).
- `user_notes` (`string`): Private freeform personal notes stored on local filesystem.
- Metadata overrides (`title`, `aliases`, `description`, `authors`, `artists`, `tags`, `collections`, `content.reading_mode`, etc.).

**Responses:**
- `200 OK`: Returns updated manga object `{ "id": "<id>", "meta": <MangaMeta> }`.
- `400 Bad Request`: If `user_status` is not a valid enum value or JSON is malformed.
- `404 Not Found`: If manga ID does not exist in local library.

## Request Lifecycle

```
1. Request arrives
2. Middleware: log, recover, request-id, CORS, content-type check
3. Route match → handler
4. Handler validates input (shape, range, refs)
5. Handler calls service layer
6. Service returns result or domain error
7. Handler maps to HTTP response:
   - Success → 2xx + JSON body
   - Domain error → 4xx + error envelope
   - Server error → 5xx + error envelope
8. Middleware: log response with timing
```

## Versioning

API version embedded in URI path:

```
/api/v1/library/manga
/api/v2/library/manga
```

Major versions coexist. Deprecation announced in advance; v{N-1} continues serving for at least one minor version of v{N}.

Versioning rule: any breaking change → new major version. Additive changes (new field, new endpoint) → existing version, no version bump.

## Content Negotiation

```
Request:
  Accept: application/json
  Content-Type: application/json

Response:
  Content-Type: application/json; charset=utf-8
```

No alternative formats (XML, msgpack) planned. JSON is the only wire format.

## Error Envelope

All error responses share one shape:

```
ErrorEnvelope:
  code         machine-readable identifier
  message      human-readable diagnostic
  details      optional structured context (validation errors, etc.)
  request_id   for log correlation
```

HTTP status code mirrors error kind:

| Status | When |
|---|---|
| 200 | Success |
| 201 | Resource created |
| 204 | Success, no body |
| 400 | Validation failure |
| 404 | Resource not found |
| 409 | Conflict (duplicate, stale state) |
| 422 | Domain rule violation |
| 429 | Rate limited |
| 500 | Server error |
| 502 | Provider error |
| 503 | Dependency unavailable |

## Pagination

List endpoints use cursor or offset pagination:

```
?page=1&limit=20       offset-based, default 20, max 100
?cursor=<token>&limit=20  cursor-based for large/unstable lists
```

Response includes pagination metadata:

```
Pagination:
  total       total count (when known)
  page        current page
  limit       page size
  has_next    boolean
  next_cursor or next_page
```

## Filtering & Sorting

Filter params are field-prefixed:

```
?filter[user_status]=reading
?filter[user_favorite]=true
?sort=-updated_at
```

Sort syntax: field name, `-` prefix for descending.

## Authentication & Authorization

Single-user local app. No auth on local-host. Remote access requires reverse proxy with auth (out of scope for API design).

When multi-user or remote access is added:
- Bearer token in `Authorization` header
- Token issued via login endpoint
- All endpoints require valid token except `/health`

## Idempotency

Mutations are idempotent where possible:
- POST /resource → uses client-provided UUID or server-generated
- PUT /resource/{id} → upsert semantics
- DELETE /resource/{id} → 204 either way (subsequent calls idempotent)

Job creation: client supplies `idempotency_key`, server dedupes within 24h window.

## Caching Headers

Library resources are mostly server-state, no client caching:

```
Cache-Control: no-store
```

Provider responses may include:

```
Cache-Control: max-age=300  (metadata cache respects this)
ETag: <hash>               (for conditional GET)
```

## Long-Running Operations

Downloads, library scans, refresh operations are async. API returns immediately:

```
POST /downloads
→ 202 Accepted
  Location: /jobs/<job_id>
  body: { job_id, status: "pending" }
```

Client polls `/jobs/<job_id>` or subscribes via SSE (future).

## Rate Limiting

Per-client rate limits (when remote access enabled):

```
RateLimit-Limit: 100
RateLimit-Remaining: 99
RateLimit-Reset: <unix-timestamp>
```

Per-provider limits are configured separately, enforced by workers framework.

## Streaming

Reader pages served as static files (already on disk). No streaming endpoint needed for pages.

Cover art served with proper `Content-Type` and caching headers.

## Observability

Every request logged:

```
Log entry:
  request_id
  method
  path
  status
  latency_ms
  user_agent
  ip
  error (if any)
```

Structured logging (JSON) for machine parsing. Log aggregation out of scope.

## OpenAPI / Schema

API schema published as OpenAPI 3.1 spec. Source of truth for both server validation and client generation.

```
docs/design/api/openapi.yaml  (canonical)
```

Generated types in client (TypeScript) and server (Go validation structs) derive from spec.

## Migration Path

Current API in `internal/api/handler.go` and `docs/developer/api.md`:
- URI patterns largely compatible with new design
- Response shapes need consolidation (current uses mixed envelopes)
- Error handling needs standardization (current ad-hoc)
- OpenAPI spec to be authored during Phase 1 of API redesign

## Open Questions

1. GraphQL or gRPC for internal services? Out of scope, REST over HTTP.
2. Server-Sent Events vs WebSocket for progress streaming? SSE simpler, defer choice.
3. Batch operations endpoint (`POST /batch/manga`) for bulk library imports?
4. Webhook system for external integrations (similar to Fizzy model)?

## References

- `docs/design/library.md` — primary domain
- `docs/design/workers.md` — job API endpoints
- `docs/design/providers.md` — provider query endpoints
- `docs/design/reader.md` — page image endpoints
- `docs/developer/api.md` — current API docs (will be superseded once OpenAPI spec lands)
