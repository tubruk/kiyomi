# Kiyomi

![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go)
![React](https://img.shields.io/badge/React-18-61DAFB?style=flat&logo=react)
![TypeScript](https://img.shields.io/badge/TypeScript-5-3178C6?style=flat&logo=typescript)
![License](https://img.shields.io/badge/License-MIT-blue.svg)

**Kiyomi** is a self-hosted, filesystem-first manga reader and manager server with an embedded web interface.

Kiyomi puts data ownership first: your manga library lives in plain files and directories on disk (`$KIYOMI_HOME/library/`) with human-readable JSON manifests (`meta.json` and `pages.json`). If the server or database dies, your library structure and reading data remain 100% intact and recoverable.

---

## Features

- **Filesystem-First Library**: Stores manga details, chapter listings, and reading history as human-readable JSON manifests directly on disk.
- **Built-in Provider Extensions**: Search, browse popular/latest catalogs, and import titles directly from **MangaDex** and **MangaFox**.
- **Transparent Server-Side Image Caching**: Proxied cover art and chapter pages are automatically cached to disk (`$KIYOMI_HOME/cache/`) with SHA-256 URL sharding, singleflight request deduplication, and background LRU cleanup.
- **Permanent Lazy-Loaded Chapter Pages**: Page metadata lists are fetched on-demand when a chapter is opened, saved atomically as `pages.json`, and served instantly (<1ms) on subsequent reads.
- **TLS Fingerprint Impersonation**: Built-in transport mimicking Chrome and Firefox TLS fingerprints and headers to bypass anti-bot CDN protections.
- **Web Reader Experience**: Single-page and vertical long-strip (Webtoon) reading modes with drag navigation, keyboard shortcuts, and theme presets (Dark, Light, Sepia).
- **Progress Tracking & Statuses**: Automatic reading progress updates (last read page, chapter completion) and status categorization (*Reading*, *Completed*, *Plan to Read*, *On Hold*, *Dropped*).
- **In-App Diagnostic Inspection**: Error detail modals and structured backend logging to diagnose provider or network failures without dumping raw HTML tracebacks.

---

## Architecture

- **Backend**: Go REST API (`internal/api`), filesystem storage engine (`internal/library`), LRU image cache (`internal/cache`), and provider SDK (`pkg/provider`).
- **Frontend**: Vite SPA in `web/` (React 18, TanStack Router & Query, Tailwind CSS, shadcn/ui) embedded directly into the Go binary for single-executable deployments.

---

## Getting Started

### Prerequisites

- **Go** 1.23+
- **Bun** 1.1+ (for Web UI development)

### Quick Start

1. **Clone the repository**:
   ```bash
   git clone https://github.com/tubruk/kiyomi.git
   cd kiyomi
   ```

2. **Run the server**:
   ```bash
   go run ./cmd/kiyomi
   ```
   The web interface will be available at `http://localhost:8080`.

### Configuration Environment Variables

| Variable | Default | Description |
| :--- | :--- | :--- |
| `KIYOMI_HOME` | Current working directory | Root home directory for relative storage paths |
| `KIYOMI_PORT` | `8080` | HTTP server port |
| `KIYOMI_LIBRARY_DIR` | `<home>/library` | Path to manga library storage directory |
| `KIYOMI_CACHE_DIR` | `<home>/cache` | Path to image & metadata disk cache |
| `KIYOMI_CACHE_MAX_BYTES` | `2147483648` (2 GB) | Maximum size limit for image cache before LRU purging |
| `KIYOMI_CACHE_IMAGE_TTL` | `720h` (30 days) | Retention TTL for cached image files |

---

## Documentation

Comprehensive architecture design documents, provider authoring guides, and E2E testing guides are available in [`docs/`](./docs).
