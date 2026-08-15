# Backlog: Upstream Reading Direction Hints Mapping

> **Status**: Proposed / Backlog  
> **Target Components**: `pkg/provider/sdk`, `pkg/provider/mangadex`, `pkg/provider/mangafox`, `internal/library`

---

## 1. Overview

Currently, Kiyomi's reader supports multiple reading layouts (`rtl`, `ltr`, `vertical`) driven by manga-level metadata. However, neither **MangaDex** nor **MangaFox** exposes an explicit top-level `readingDirection` attribute in their upstream metadata APIs.

This backlog item outlines the strategy to extract **implicit reading direction hints** from upstream provider data (such as original publication language, format tags, and genre classifications) and map them automatically to Kiyomi's reading direction defaults.

---

## 2. Upstream Data Analysis

### MangaDex (`pkg/provider/mangadex`)

* **Explicit Attribute**: None (MangaDex API v5 `data.attributes` does not contain a `readingDirection` key).
* **Implicit Hints**:
  1. **`originalLanguage`** (`data.attributes.originalLanguage`):
     * `"ja"` (Japanese): Traditional **Manga** $\rightarrow$ maps to `rtl` (Right-to-Left).
     * `"ko"` (Korean): **Manhwa** $\rightarrow$ maps to `vertical` (Webtoon / Long Strip).
     * `"zh"` / `"zh-hk"` (Chinese): **Manhua** $\rightarrow$ maps to `vertical` (Webtoon / Long Strip).
     * `"en"` (English) / Other Western: OEL / Comics $\rightarrow$ maps to `ltr` (Left-to-Right).
  2. **Format Tags** (`data.attributes.tags`):
     * **`Long Strip`** tag (Tag ID `3e2b8dae-350e-4ab8-a8ce-016e844b9f0d` under the `format` tag group): Explicitly indicates a continuous **vertical scrolling** format.
     * **`Web Comic`** tag: Secondary indicator for digital/scrolling formats.

### MangaFox / FanFox (`pkg/provider/mangafox`)

* **Explicit Attribute**: None in HTML metadata.
* **Implicit Hints**:
  1. **Genre Tags** (`.detail-info-right-tag a`):
     * **`Webtoons`** genre tag: Direct indicator for **vertical scrolling**.
  2. **Type Metadata**:
     * Japanese Manga $\rightarrow$ maps to `rtl`.
     * Korean Manhwa / Chinese Manhua $\rightarrow$ maps to `vertical`.

---

## 3. Proposed Architecture & Implementation Plan

### Phase 1: SDK Extension (`pkg/provider/sdk`)
Add an optional `ReadingDirection` field to `sdk.MangaMetadata`:

```go
type ReadingDirection string

const (
    ReadingDirectionUnspecified ReadingDirection = ""
    ReadingDirectionRTL         ReadingDirection = "rtl"
    ReadingDirectionLTR         ReadingDirection = "ltr"
    ReadingDirectionVertical    ReadingDirection = "vertical"
)

type MangaMetadata struct {
    // ...
    ReadingDirection ReadingDirection `json:"readingDirection,omitempty"`
}
```

### Phase 2: Provider Inferences
* **MangaDex Provider** (`pkg/provider/mangadex/metadata_provider.go`):
  * Parse `originalLanguage` from `apiResp.Data.Attributes.OriginalLanguage`.
  * Check `tags` for `"Long Strip"`.
  * Infer `ReadingDirection` (`Long Strip` or `ko`/`zh` $\rightarrow$ `vertical`; `ja` $\rightarrow$ `rtl`; `en` $\rightarrow$ `ltr`).
* **MangaFox Provider** (`pkg/provider/mangafox/metadata_provider.go`):
  * Check extracted `genres` slice for `"Webtoons"` $\rightarrow$ `vertical`.
  * Check type classification if present.

### Phase 3: Library & Reader Integration (`internal/library`, `web/src`)
* Persist the inferred `reading_direction` into SQLite database / `meta.json` when adding manga to the library.
* Use the provider-inferred hint as the default reading direction in the Web UI reader if no manual user preference override has been set.

---

## 4. Verification & Testing Strategy

1. **Unit Tests**: Add unit tests in `mangadex_test.go` and `mangafox_test.go` verifying that sample metadata JSON/HTML with `ko` language, `Long Strip` tag, or `Webtoons` genre correctly yields `ReadingDirectionVertical` or `ReadingDirectionRTL`.
2. **Backend Verification**: Run `go test -v ./...`.
3. **Frontend Verification**: Verify Web UI build with `bun run build` inside `web/`.
