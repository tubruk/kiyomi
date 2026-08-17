# Configuration

Kiyomi is configured entirely through environment variables. All paths default to subdirectories of `KIYOMI_HOME`, which itself defaults to the current working directory.

## Paths

| Variable | Default | Description |
| :--- | :--- | :--- |
| `KIYOMI_HOME` | Current working directory | Base directory for all relative paths |
| `KIYOMI_PORT` | `8080` | HTTP server port |
| `KIYOMI_LIBRARY_DIR` | `<home>/library` | Manga library root |
| `KIYOMI_DOWNLOAD_DIR` | `<home>/library` | Downloaded chapter assets |
| `KIYOMI_CACHE_DIR` | `<home>/cache` | Image and metadata disk cache |
| `KIYOMI_PLUGIN_DIR` | `<home>/plugins` | External plugin binaries |
| `KIYOMI_PROVIDER_CONFIG` | `<home>/providers.json` | Per-provider settings |

## Cache

| Variable | Default | Description |
| :--- | :--- | :--- |
| `KIYOMI_CACHE_MAX_BYTES` | `2147483648` (2 GB) | Maximum cache size before LRU eviction |
| `KIYOMI_CACHE_IMAGE_TTL` | `720h` (30 days) | Image file retention |
| `KIYOMI_CACHE_PAGE_TTL` | `168h` (7 days) | Page list retention |
| `KIYOMI_CACHE_METADATA_TTL` | `12h` | Series metadata retention |
| `KIYOMI_CACHE_SEARCH_TTL` | `1h` | Search result retention |

## Behavior

| Variable | Default | Description |
| :--- | :--- | :--- |
| `KIYOMI_LOG_LEVEL` | `info` | Log threshold: `debug`, `info`, `warn`, `error` |
| `KIYOMI_LOG_FORMAT` | `pretty` | Log format: `pretty`, `json`, `text` |

## Docker Compose example

```yaml
services:
  kiyomi:
    image: ghcr.io/tubruk/kiyomi:latest
    ports:
      - "8080:8080"
    volumes:
      - ./data:/data
      - ./plugins:/data/plugins
    environment:
      KIYOMI_HOME: /data
```
