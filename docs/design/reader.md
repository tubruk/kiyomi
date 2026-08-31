# Reader

## Overview

The reader is the user-facing reading experience in Kiyomi. It loads pages from local files when available, seamlessly falls back to ephemeral cache or remote content providers when not, tracks granular reading progress, and provides responsive navigation controls.

---

## Conceptual Model

```
┌─────────────────────────────────────────────────────────────────────────┐
│                             Reader Runtime                              │
│                                                                         │
│   ┌─────────────┐            ┌─────────────┐            ┌─────────────┐ │
│   │ Page Loader │            │  Navigator  │            │  Progress   │ │
│   │             │            │             │            │   Tracker   │ │
│   └──────┬──────┘            └──────┬──────┘            └──────┬──────┘ │
│          │                          │                          │        │
│          ▼                          ▼                          ▼        │
│   ┌───────────────────────────────────────────────────────────────────┐ │
│   │               3-Tier Page Source Resolution Engine                │ │
│   │            (Disk Library → Ephemeral Cache → Remote Proxy)        │ │
│   └───────────────────────────────────────────────────────────────────┘ │
└────────────────────────────────────┬────────────────────────────────────┘
                                     │
                                     ▼
     ┌───────────────────────────────┼───────────────────────────────┐
     ▼                               ▼                               ▼
┌──────────────┐             ┌──────────────┐             ┌─────────────────────┐
│ Tier 1: Disk │             │Tier 2: Cache │             │  Tier 3: Provider   │
│ Local Files  │             │Ephemeral Disk│             │  Remote Stream via  │
│  (library/)  │             │   (cache/)   │             │ Reverse Proxy (TLS) │
└──────────────┘             └──────────────┘             └─────────────────────┘
```

---

## Reading Modes

Kiyomi supports multiple reading layouts per chapter, driven by `library/<manga_id>/meta.json` `content.reading_mode`:

| Mode Enum | UI Display Label | Layout & Behavior |
|---|---|---|
| `rtl` | Right to Left (Manga) | Traditional manga; pages navigate right-to-left. |
| `ltr` | Left to Right (Comic) | Western comics; pages navigate left-to-right. |
| `vertical` | Vertical (Gapped) | Paged vertical layout with configurable margins between pages. |
| `longstrip` | Longstrip (Webtoon) | Continuous seamless vertical scroll without inter-page padding. |

The reading mode is configured per-manga in `manga.content.reading_mode`. If omitted or unspecified, the reader falls back to the user's default reading preference.

---

## 3-Tier Fallback Page Resolution

Pages are resolved on-demand through a resilient 3-tier fallback chain:

```
Page Resolution Order:
  1. Disk (Local Library)   — library/<manga_id>/<provider_id>/<chapter_id>/<index>.<ext>
  2. Cache (Ephemeral Disk) — cache/ (hashed key lookup with TTL and LRU eviction)
  3. Remote Provider (Live) — Fetched via backend reverse proxy with TLS fingerprinting & SSRF guard
```

### 1. Tier 1: Local Library Disk
- Target path: `<library_root>/<manga_id>/<provider_id>/<chapter_id>/<index>.<ext>`
- Supported formats: `.jpg`, `.jpeg`, `.png`, `.webp`, `.gif`, `.avif`.
- When found, the local file is served directly with `Cache-Control: public, max-age=86400`.

### 2. Tier 2: Ephemeral Disk Cache
- If the page image has not been pulled to the permanent local library, Kiyomi queries the ephemeral image disk cache (`cache/`).
- Cached items are indexed by remote image URL and managed with TTL expiration, maximum byte limits, and periodic background LRU cleanup workers.
- Avoids redundant network roundtrips for recently read chapters.

### 3. Tier 3: Live Remote Reverse Proxy
- If absent from both local library disk and ephemeral cache, the backend reverse proxy (`/api/v1/library/manga/{id}/chapters/{ch}/pages/{n}` or `/api/v1/proxy/image`) fetches the image stream live from the upstream provider.
- Successful responses are simultaneously written to the ephemeral cache and streamed to the client reader.

---

## Page Source Derivation & State

Each page carries an independent `source ∈ {disk, cache, provider}` state determined at read time:

| Source State | Detection Rule | UI Capabilities |
|---|---|---|
| `disk` (or `library`) | Local file exists at `library/<manga_id>/<provider_id>/<chapter_id>/<index>.<ext>` | Offline reading; fully persisted |
| `cache` | Disk absent, but item exists in ephemeral disk cache | "Save to Library" action enqueues pull |
| `provider` | Neither disk nor cache present; fetched live | "Pull Chapter" action enqueues background job |

Source state is tracked **per-page**, not aggregated at chapter level — a chapter may have partial pages downloaded on disk while the remainder stream on demand. The reader never blocks on pull operations; it reads whatever source is currently available.

---

## Security & Proxy Engine

All remote image requests routed through the reader reverse proxy are protected by security layers:

```
Incoming Request → SSRF Validation → Header & Cookie Injection → TLS Fingerprinted Transport → Upstream Provider
```

### 1. SSRF Protection
- Remote URLs are validated against the SSRF guard before any socket connection is attempted.
- Rejects private IP addresses (RFC 1918), IPv6 unique local addresses (RFC 4193), loopback addresses (`127.0.0.0/8`, `::1`), link-local metadata addresses (`169.254.169.254`, `fe80::/10`), and non-HTTP protocols.
- Private network access is blocked by default unless explicitly allowed via server configuration.

### 2. Direct Data URI Handling
- Data URIs (e.g. `data:image/jpeg;base64,...`) returned by specialized providers are decoded in-memory and served directly with appropriate `Content-Type` and `Content-Length` headers, bypassing the network transport.

### 3. TLS Browser Fingerprinting
- Upstream HTTP requests mirror realistic browser TLS Client Hello signatures using the TLS browser fingerprinting engine.
- Custom cipher suites, ALPN protocols, and elliptic curves evade anti-bot heuristics (Cloudflare, Akamai, DDOS-GUARD).
- Integrates stored provider cookies (`cf_clearance`, `__cf_bm`, session tokens) and upstream referer headers.

---

## Reading Progress Tracking

Reading progress is tracked independently per chapter and manga:

### Chapter Progress Schema:
```yaml
ChapterProgress:
  manga_id: string
  provider_id: string
  chapter_id: string
  is_read: boolean
  last_read_page: integer
  updated_at: timestamp
```

### Tracking Behavior:
- **Independent Namespaces**: Reading progress for a chapter in one provider namespace (e.g. `mangadex`) is isolated from other provider namespaces (e.g. `mangafox`).
- **Debounced Updates**: Progress is debounced during active scrolling and page flipping (flushed on page turn, chapter change, or reader close).
- **Crash Resilience**: Reading state persists to local storage immediately, enabling seamless resumption upon app restart.

---

## Navigation & Chapter Transitions

Navigation adapts seamlessly across reading modes and client input modalities:

- **Paged Navigation (`rtl`, `ltr`)**: Incremental forward/backward step actions driven by reading direction.
- **Continuous Scrolling (`vertical`, `longstrip`)**: Fluid vertical progression with position preservation.
- **Chapter Boundary Transitions**: Advancing past the final page or before the first page navigates to adjacent chapters in sequence.
- **Chapter Navigation Overlay**: Rapid jumping across the chapter list via an on-demand index menu or drawer.

### Seamless Transition & Preloading
- When approaching chapter boundaries, the reader preloads adjacent chapter assets in the background to ensure uninterrupted reading sessions.

---

## Offline Reading

When all pages of a chapter exist on disk (`library/<manga_id>/<provider_id>/<chapter_id>/`):
- Reader functions entirely offline with zero network connectivity.
- Reading progress updates are saved directly to the local database and filesystem manifests.

---

## Image Formats & Memory Management

- **Supported Formats**: JPEG, PNG, WebP, GIF, AVIF.
- **Viewport Virtualization**: Longstrip and vertical modes virtualize page rendering, decoding only visible pages plus a 2-page buffer in memory to keep browser RAM usage low during long chapters (100+ pages).

---

## References

- [Multi-Content Provider Library](./multi_content_provider_library.md) — Multi-provider filesystem layout and chapter directories.
- [Filesystem-First Library](./library.md) — Manga and chapter manifest schemas.
- [Background Job Queue](./background_jobs.md) — Asynchronous pull queue and job execution.
- [Content Pull System](./content_pull.md) — Chapter and page pull workflows.
- [Providers Design](./providers.md) — Provider SDK contracts and stream capabilities.
- [Anti-Bot Strategy](./antibot.md) — Cloudflare clearance and TLS fingerprint matching.
- [REST API](./api.md) — Reader page image proxy and chapter progress endpoints.

