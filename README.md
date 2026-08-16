# Kiyomi

![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go)
![React](https://img.shields.io/badge/React-18-61DAFB?style=flat&logo=react)
![TypeScript](https://img.shields.io/badge/TypeScript-5-3178C6?style=flat&logo=typescript)
![License](https://img.shields.io/badge/License-MIT-blue.svg)

**Kiyomi** is a self-hosted, plugin-extensible manga reader and manager with an embedded web interface.

Kiyomi puts data ownership first: your manga library lives in plain files and directories on local disk with human-readable JSON manifests (`meta.json` and `pages.json`). If the server dies, your reading history, metadata, and cached images remain fully intact and recoverable.

---

## Features

- **Self-hosted**: Runs on your own machine. No accounts, no cloud sync, no external services required.
- **Multi-source via plugins**: Comes with a set of built-in sources. Additional sources can be added by dropping a plugin binary into the plugins directory.
- **Local caching**: Cover art, chapter pages, and metadata are cached on disk after the first fetch. Subsequent reads are served locally.
- **Single binary**: The web UI is embedded in the Go binary. No separate frontend server, runtime, or database daemon to run.

---

## Getting Started

```bash
docker run -p 8080:8080 \
  -v ./data:/data \
  -e KIYOMI_HOME=/data \
  ghcr.io/tubruk/kiyomi:latest
```

Open `http://localhost:8080`. Data is stored under `./data` on the host.

For a persistent setup with plugins, see the [Docker Compose example](./docs/user/configuration.md#docker-compose-example).

To build from source:

```bash
git clone https://github.com/tubruk/kiyomi.git
cd kiyomi
go run ./cmd/kiyomi
```

---

## Configuration

Kiyomi is configured through environment variables. See [docs/user/configuration.md](./docs/user/configuration.md) for the full reference.

---

## Building a Provider Plugin

Any Go developer can extend Kiyomi by building a plugin binary that imports the Plugin SDK:

```go
import sdk "github.com/tubruk/kiyomi/plugin-sdk"
```

Implement one or more provider interfaces (`MetadataProvider`, `ContentProvider`, `Tracker`), then call `sdk.ServePlugin(...)` as your `main` entry point. Kiyomi loads compiled binaries from the plugins directory at startup.

See [docs/design/provider_plugin_architecture.md](./docs/design/provider_plugin_architecture.md) for the full spec and [`plugins/`](./plugins) for first-party examples.

---

## Documentation

- [User docs](./docs/user/) — configuration reference
- [Design docs](./docs/design/) — architecture and system design specs
- [Plugin Developer docs](./docs/plugin_developer/) — guide to building custom provider plugins
- [E2E testing](./docs/e2e/) — end-to-end test guides

