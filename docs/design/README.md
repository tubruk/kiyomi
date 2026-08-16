# Design Docs

Architecture and design specifications for the Kiyomi codebase.

| Document | Description |
| :--- | :--- |
| [Provider & Plugin Architecture](./provider_plugin_architecture.md) | Plugin system design: SDK interfaces, gRPC transport, lifecycle, and hot-reload |
| [Providers](./providers.md) | Built-in provider implementation notes |
| [Library Storage](./library.md) | SQLite schema and filesystem layout for the manga library |
| [API](./api.md) | REST API design and endpoint reference |
| [Reader](./reader.md) | Web reader architecture and reading modes |
| [Reading Progress](./reading_progress.md) | Progress tracking model and sync design |
| [Metadata Import](./metadata_import.md) | Metadata enrichment and import flow |
| [Anti-bot](./antibot.md) | TLS fingerprinting and header spoofing approach |
