---
name: kiyomi-provider
description: Guidelines, architectures, interfaces, and testing practices for developing metadata, content, and tracking providers (extensions) in Kiyomi.
---

# Kiyomi Provider / Extension Development Skill

This skill defines the operational standards, interface specifications, utility helper usage, and testing methodologies for building metadata, content, and tracking providers (extensions) in the Kiyomi project.

---

## 1. Provider Capabilities & Interfaces

All Kiyomi providers live in `pkg/provider/` and implement capability interfaces defined in [`pkg/provider/sdk/capabilities.go`](file:///Users/akhyar.amarullah/Projects/github.com/tubruk/kiyomi/pkg/provider/sdk/capabilities.go).

Every provider MUST implement the base `sdk.Provider` interface:
```go
type Provider interface {
	ID() string
	Name() string
	Icon() string
	Capabilities() []string
	ConfigKeys() []ConfigKeySpec
	RequiresAuth() bool
	State() ProviderState
}
```

### 1.1 `sdk.Metadata` Capability
Supplies series search, details, cover art, and title aliases:
```go
type Metadata interface {
	Provider
	Search(ctx context.Context, query string, opts SearchOptions) ([]SearchResult, error)
	Details(ctx context.Context, remoteID string) (MangaMetadata, error)
	Cover(ctx context.Context, remoteID string, size ImageSize) (ImageRef, error)
	Aliases(ctx context.Context, remoteID string) ([]string, error)
}
```

### 1.2 `sdk.Content` Capability
Supplies chapter lists, page lists, and page image streams:
```go
type Content interface {
	Provider
	HasStableChapterID() bool
	FetchChapters(ctx context.Context, mangaRef string) ([]Chapter, error)
	FetchPages(ctx context.Context, mangaRef, chapterRef string) ([]Page, error)
	FetchPageStream(ctx context.Context, page Page) (io.ReadCloser, error)
	RateLimit() RateLimitHint
}
```

### 1.3 `sdk.Tracking` Capability
Synchronizes reading progress to external accounts (e.g. AniList, MyAnimeList, Kitsu):
```go
type Tracking interface {
	Provider
	Authenticate(ctx context.Context, creds UserCredentials) (Session, error)
	PushProgress(ctx context.Context, remoteID string, n int) error
	FetchProgress(ctx context.Context, remoteID string) (Progress, error)
	IsAuthenticated() bool
}
```

---

## 2. HTTP Scraper Design with `sdk.HttpSource`

When building scrapers targeting web pages, embed `*sdk.HttpSource` from [`pkg/provider/sdk/http_source.go`](file:///Users/akhyar.amarullah/Projects/github.com/tubruk/kiyomi/pkg/provider/sdk/http_source.go). It provides built-in:
* **Cookie Management**: Set via `ProviderConfig.Cookies` (e.g., bypassing age gates or persistent settings).
* **TLS Fingerprinting**: Configured automatically via `WithFingerprintStore` to evade bot detection.
* **DNS Resolvers & Proxies**: Pluggable outbound transports configuration.
* **Document Fetching**: `GetDocument(ctx, targetURL)` returns a parses `*goquery.Document`.
* **URL Resolution**: `ResolveURL(relativePath)` resolves relative hrefs/srcs against `BaseURL`.

### Initialization Example:
```go
cfg := sdk.ProviderConfig{
	ID:       "mysource",
	Name:     "MySource",
	BaseURL:  "https://example.com",
	Language: "en",
	Cookies: map[string]string{
		"https://example.com": "isAdult=1",
	},
}
base, err := sdk.NewHttpSource(cfg)
```

---

## 3. Declarative HTML Extraction Helpers

Do not write raw DOM traversal boilerplate. Use the selector utilities in [`pkg/provider/sdk/selector.go`](file:///Users/akhyar.amarullah/Projects/github.com/tubruk/kiyomi/pkg/provider/sdk/selector.go):
* **`sdk.ExtractText(selection, selector)`**: Retrieves trimmed text contents.
* **`sdk.ExtractAttr(selection, selector, attr)`**: Safely retrieves an attribute value.
* **`sdk.ExtractImageURL(selection, selector)`**: Automatically tries fallback lazy-load attributes (`src`, `data-src`, `data-original`, `data-lazy-src`).
* **`sdk.ParseChapterNumber(title)`**: Strips volume markers and parses floating point numbers.
* **`sdk.ParseDate(dateStr)`**: Standardizes various date string formats to Unix milliseconds.

---

## 4. Provider Registration

Register new providers within the central `providerRegistry` inside [`cmd/kiyomi/main.go`](file:///Users/akhyar.amarullah/Projects/github.com/tubruk/kiyomi/cmd/kiyomi/main.go):
```go
prov, err := mysource.NewProvider(mysource.Options{
	Store:     providerConfigStore,
	FpStore:   fpStore,
	Transport: http.DefaultTransport,
	Registry:  providerRegistry,
})
if err != nil {
	slog.Error("failed to initialize MySource provider", slog.String("error", err.Error()))
	os.Exit(1)
}
providerRegistry.Register(prov)
```

---

## 5. Provider Authoring Guidelines

When implementing a provider, authors MUST adhere to the following rules:

### 5.1 Purpose of the Provider SDK: Abstracting Upstream Behavior
- **Core Principle**: The primary purpose of the Provider SDK is to abstract away each provider's internal implementation details and ever-changing upstream behaviors (e.g., HTML scraping quirks, site layout variations, custom date string formats, or anti-bot protections).
- **Internal Normalization**: All upstream site peculiarities (such as parsing `"Today"`, `"Yesterday"`, or `"Oct 12,2023"` into standard `time.Time`) MUST be handled and normalized internally within the provider package.
- **Strict Native Types**: Provider methods MUST only return standard SDK types (such as native `time.Time` for `UploadDate`, clean `sdk.Chapter`, and `sdk.Page`). Raw upstream structures or unparsed strings MUST NEVER leak past the provider boundary into API handlers, database storage, or frontend UI callers.

### 5.2 URL-Safe Clean Chapter IDs (`chapter_ref`)
- Chapter IDs (`chapter_ref` / `remote_id`) MUST be clean, plain, URL-safe strings without Base64 encoding (e.g., plain UUIDs or transformed path strings like `ch-1` or `v01~c001~1.html`).
- Base64 encoding/decoding (`sdk.EncodeID` / `sdk.DecodeID`) MUST NOT be used for chapter IDs.
- Any raw upstream identifier containing slashes, query parameters, spaces, or non-ASCII characters MUST be sanitized, encoded, or converted into a URL-safe format inside the provider implementation itself.
- **No Leakage**: ID transformations are strictly an internal provider concern and MUST NEVER leak into REST API handlers or frontend UI callers.

---

## 6. Testing & Verification

1. **Unit Tests**: Place tests in `<provider>_test.go` and use standard Go assertions.
2. **HTTP Mocking**: Do not make outbound requests during testing. Mock target pages or JSON responses.
3. **Execution**:
   * Run unit tests from the repository root: `go test -v ./pkg/provider/...`
   * Run verification before completing provider updates: `go test -v ./...`
