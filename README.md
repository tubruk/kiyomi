# Kiyomi

![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go)
![React](https://img.shields.io/badge/React-18-61DAFB?style=flat&logo=react)
![TypeScript](https://img.shields.io/badge/TypeScript-5-3178C6?style=flat&logo=typescript)
![License](https://img.shields.io/badge/License-MIT-blue.svg)

**Kiyomi** is a self-hosted, filesystem-first manga reader and manager server with an embedded web interface.

Kiyomi puts data ownership first: your manga library lives in plain files and directories on local disk with human-readable JSON manifests (`meta.json` and `pages.json`). If the server or database dies, your library structure and reading data remain 100% intact and recoverable.

---

## Features

- **Filesystem-First Library**: Stores manga details, chapter listings, and reading history as plain files on local disk, avoiding proprietary database lock-in.
- **Built-in Manga Sources**: Search, browse catalogs, and import titles directly from **MangaDex** and **MangaFox**.
- **Browser TLS Profiles**: Uses standard browser TLS signatures and request headers to ensure reliable image delivery from upstream provider servers.
- **Fast & Snappy Page Loading**: Automatically saves cover art, chapter page lists, and images locally as you read for instant re-reading and smooth browsing.
- **Web Reader**: Single-page and vertical long-strip (Webtoon) reading modes with touch/drag navigation, keyboard shortcuts, and theme presets.
- **Progress Tracking & Statuses**: Automatic reading progress updates (last read page, completed chapters) and library shelf filters (*Reading*, *Completed*, *Plan to Read*, *On Hold*, *Dropped*).
- **Single-File Binary**: Embedded web interface served directly from a lightweight Go binary with zero external runtime dependencies.

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

Clone the repository and start the server:

```bash
git clone https://github.com/tubruk/kiyomi.git
cd kiyomi
go run ./cmd/kiyomi
```

The web interface will be available at `http://localhost:8080`.

---

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

System architecture design documents, provider authoring guides, and E2E testing guides are available in [`docs/`](./docs).
