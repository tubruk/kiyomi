package sdk

import (
	"context"
	"io"
	"time"
)

// SearchOptions carries pagination and filter hints for a metadata search.
type SearchOptions struct {
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
	Mode   string `json:"mode,omitempty"` // e.g. "popular" or "latest"
}

// ContentAvailability represents the availability status of content on a provider.
type ContentAvailability string

const (
	AvailabilityAvailable   ContentAvailability = "available"
	AvailabilityUnavailable ContentAvailability = "unavailable"
	AvailabilityUnknown     ContentAvailability = "unknown"
)

// ReadingMode represents the reading layout/direction of a manga series.
type ReadingMode string

const (
	ReadingModeUnspecified ReadingMode = ""
	ReadingModeLTR         ReadingMode = "ltr"
	ReadingModeRTL         ReadingMode = "rtl"
	ReadingModeVertical    ReadingMode = "vertical"
	ReadingModeLongstrip   ReadingMode = "longstrip"
)

// ImageSize selects a cover image variant.
type ImageSize string

const (
	ImageSizeSmall  ImageSize = "small"
	ImageSizeMedium ImageSize = "medium"
	ImageSizeLarge  ImageSize = "large"
)

// SearchResult represents a single manga entry returned by a metadata search.
type SearchResult struct {
	RemoteID     string              `json:"remoteId"`
	Title        string              `json:"title"`
	Aliases      []string            `json:"aliases,omitempty"`
	CoverURL     string              `json:"coverUrl,omitempty"`
	URL          string              `json:"url,omitempty"`
	Availability ContentAvailability `json:"availability,omitempty"`
}

// MangaMetadata represents normalized metadata for a manga series returned by a Metadata provider.
type MangaMetadata struct {
	RemoteID      string              `json:"remoteId"`
	Title         string              `json:"title"`
	Aliases       []string            `json:"aliases,omitempty"`
	CoverURL      string              `json:"coverUrl,omitempty"`
	Synopsis      string              `json:"synopsis,omitempty"`
	Status        string              `json:"status,omitempty"`
	Author        string              `json:"author,omitempty"`
	Artist        string              `json:"artist,omitempty"`
	Genres        []string            `json:"genres,omitempty"`
	TotalChapters int                 `json:"totalChapters,omitempty"`
	ReadingMode   ReadingMode         `json:"readingMode,omitempty"`
	Score         float32             `json:"score,omitempty"`
	URL           string              `json:"url,omitempty"`
	Availability  ContentAvailability `json:"availability,omitempty"`
}

// ImageRef is a resolved cover image URL with optional dimension hints and headers.
type ImageRef struct {
	URL      string            `json:"url"`
	Width    int               `json:"width,omitempty"`
	Height   int               `json:"height,omitempty"`
	MIMEType string            `json:"mimeType,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`
}

// Chapter represents a chapter of a manga from a Content provider.
type Chapter struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	URL         string    `json:"url"`
	Number      float32   `json:"number"`
	UploadDate  time.Time `json:"uploadDate"`
	SourceOrder int       `json:"sourceOrder"`
}

// Page represents an individual page image in a chapter from a Content provider.
type Page struct {
	Index   int               `json:"index"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

// RateLimitHint signals the expected request rate for a content provider.
type RateLimitHint struct {
	RequestsPerSecond float64 `json:"requestsPerSecond,omitempty"`
	RequestsPerMinute float64 `json:"requestsPerMinute,omitempty"`
}

// RateLimitSpec specifies default rate limit parameters for a provider.
type RateLimitSpec struct {
	RequestsPerSecond    int32 `json:"requestsPerSecond,omitempty"`
	MaxConcurrentRequests int32 `json:"maxConcurrentRequests,omitempty"`
}

// SettingSpec defines a configuration schema item for a plugin or provider.
type SettingSpec struct {
	Key          string   `json:"key"`
	Label        string   `json:"label"`
	Description  string   `json:"description,omitempty"`
	Type         string   `json:"type"` // "string", "number", "boolean", "secret", "select"
	DefaultValue string   `json:"defaultValue,omitempty"`
	Options      []string `json:"options,omitempty"`
}

// ProviderDescriptor self-describes a provider embedded within a plugin binary.
type ProviderDescriptor struct {
	ID               string        `json:"id"`
	Name             string        `json:"name"`
	Description      string        `json:"description,omitempty"`
	Capabilities     []string      `json:"capabilities"` // "metadata", "content", "tracking"
	SettingsSchema   []SettingSpec `json:"settingsSchema,omitempty"`
	DefaultRateLimit RateLimitSpec `json:"defaultRateLimit,omitempty"`
}

// PluginDescriptor self-describes the plugin package and all its contained providers.
type PluginDescriptor struct {
	PluginID             string               `json:"pluginId"`
	PluginName           string               `json:"pluginName"`
	PluginVersion        string               `json:"pluginVersion"`
	SDKVersion           string               `json:"sdkVersion"`
	PluginSettingsSchema []SettingSpec        `json:"pluginSettingsSchema,omitempty"`
	Providers            []ProviderDescriptor `json:"providers"`
}

// GlobalHttpConfig configures host HTTP settings for a plugin.
type GlobalHttpConfig struct {
	ProxyURL       string   `json:"proxyUrl,omitempty"`
	UserAgent      string   `json:"userAgent,omitempty"`
	TimeoutSeconds int      `json:"timeoutSeconds,omitempty"`
	DNSResolvers   []string `json:"dnsResolvers,omitempty"`
}

// PluginConfig holds initialized settings passed from host to plugin during Init.
type PluginConfig struct {
	GlobalConfig    map[string]string            `json:"globalConfig,omitempty"`
	ProviderConfigs map[string]map[string]string `json:"providerConfigs,omitempty"`
	HTTPConfig      GlobalHttpConfig             `json:"httpConfig,omitempty"`
}

// UserCredentials holds authentication data for a Tracker provider.
type UserCredentials struct {
	AccessToken  string `json:"accessToken,omitempty"`
	RefreshToken string `json:"refreshToken,omitempty"`
	ExpiresAt    int64  `json:"expiresAt,omitempty"`
}

// Session represents an authenticated Tracker session.
type Session struct {
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken,omitempty"`
	ExpiresAt    time.Time `json:"expiresAt,omitempty"`
	UserID       string    `json:"userId,omitempty"`
}

// Progress represents reading progress on a Tracker provider.
type Progress struct {
	Status    string `json:"status,omitempty"`
	Score     int    `json:"score,omitempty"`
	Progress  int    `json:"progress"`
	UpdatedAt int64  `json:"updatedAt,omitempty"`
}

// Plugin is the top-level lifecycle interface for a Kiyomi plugin.
type Plugin interface {
	Describe(ctx context.Context) (PluginDescriptor, error)
	Init(ctx context.Context, config PluginConfig) error
}

// MetadataProvider is implemented by providers that supply series search, details, cover art, and aliases.
type MetadataProvider interface {
	Search(ctx context.Context, query string, opts SearchOptions) ([]SearchResult, error)
	Details(ctx context.Context, remoteID string) (MangaMetadata, error)
	Cover(ctx context.Context, remoteID string, size ImageSize) (ImageRef, error)
	Aliases(ctx context.Context, remoteID string) ([]string, error)
}

// ContentProvider is implemented by providers that supply chapter lists, page lists, and page image streams.
type ContentProvider interface {
	HasStableChapterID() bool
	FetchChapters(ctx context.Context, mangaRef string) ([]Chapter, error)
	FetchPages(ctx context.Context, mangaRef, chapterRef string) ([]Page, error)
	FetchPageStream(ctx context.Context, page Page) (io.ReadCloser, error)
	RateLimit() RateLimitHint
}

// Tracker is implemented by providers that sync reading progress to external accounts.
type Tracker interface {
	Authenticate(ctx context.Context, creds UserCredentials) (Session, error)
	PushProgress(ctx context.Context, remoteID string, n int) error
	FetchProgress(ctx context.Context, remoteID string) (Progress, error)
	IsAuthenticated(ctx context.Context) bool
}
