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
| [Provider Binding](./provider-binding.md) | Decisions for add/switch/remove content provider endpoints |
| [Anti-bot](./antibot.md) | TLS fingerprinting and header spoofing approach |
| [DNS Override](./dns-override.md) | `KIYOMI_DNS_RESOLVERS` env var: plain DNS / DoT / DoH override for backend + plugins |
| [DNS Override — Sequence](./dns-override-sequence.md) | End-to-end sequence diagram of the resolver wiring |
