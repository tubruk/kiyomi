# Design Docs

Architecture and design specifications for the Kiyomi codebase.

| Document | Description |
| :--- | :--- |
| [Provider & Plugin Architecture](./provider_plugin_architecture.md) | Plugin system design: SDK interfaces, gRPC transport, lifecycle, and hot-reload |
| [Providers](./providers.md) | Built-in provider implementation notes |
| [Library Storage](./library.md) | SQLite schema and filesystem layout for the manga library |
| [Multi-Content Provider Library](./multi_content_provider_library.md) | Multi-content provider first-class architecture, directory layout, and uncorrelated chapters |
| [Background Jobs](./background_jobs.md) | Generic background job queue system, drivers, worker pool, lifecycle, and task execution |
| [Content Pull Pipeline](./content_pull.md) | Granular content acquisition pipeline, multi-step hierarchy, SSRF guard, and image deduplication |
| [API](./api.md) | REST API design and endpoint reference |
| [Reader](./reader.md) | Web reader architecture and reading modes |
| [Reading Progress](./reading_progress.md) | Progress tracking model and sync design |
| [Metadata Import](./metadata_import.md) | Metadata enrichment and import flow |
| [Provider Binding](./provider_binding.md) | Decisions for add/switch/remove content provider endpoints |
| [Anti-bot](./antibot.md) | TLS fingerprinting and header spoofing approach |
| [DNS Override](./dns_override.md) | `KIYOMI_DNS_RESOLVERS` env var: plain DNS / DoT / DoH override for backend + plugins |
| [DNS Override — Sequence](./dns_override_sequence.md) | End-to-end sequence diagram of the resolver wiring |

