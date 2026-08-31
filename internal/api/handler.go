package api

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/tubruk/kiyomi/internal/cache"
	"github.com/tubruk/kiyomi/internal/config"
	"github.com/tubruk/kiyomi/internal/library"
	"github.com/tubruk/kiyomi/internal/plugin/host"
	"github.com/tubruk/kiyomi/internal/queue"
	"github.com/tubruk/kiyomi/internal/upstream"
	"github.com/tubruk/kiyomi/pkg/dnsresolver"
	"github.com/tubruk/kiyomi/pkg/fingerprint"
	"github.com/tubruk/kiyomi/pkg/provider"
	"github.com/tubruk/kiyomi/pkg/provider/sdk"
	plugin_sdk "github.com/tubruk/kiyomi/plugin-sdk"
)

// Handler handles API requests.
type Handler struct {
	cfg           *config.Config
	lib           *library.Library
	httpClient    *http.Client
	registry      *provider.Registry
	pluginManager *host.PluginManager
	fpStore       fingerprint.Store
	imageCache    *cache.DiskCache
	reqBuilder    *upstream.RequestBuilder
	buildInfo     BuildInfo
	jobStore      queue.JobStore
	enqueuer      queue.Enqueuer
}

// NewHandler creates a new Handler instance with library and config.
func NewHandler(cfg *config.Config, lib *library.Library, jobStore queue.JobStore, enqueuer queue.Enqueuer) *Handler {
	fpStore := fingerprint.NewMemoryStore()

	// Create an HTTP client configured with TLS fingerprinting transport and transient retry
	transport := fingerprint.NewTransport(func() (fingerprint.TLSProfile, bool) {
		return fingerprint.TLSProfileDefault, false
	})

	// Apply DNS override if configured. Preserves the fingerprint chain.
	if cfg != nil && len(cfg.DNSResolvers) > 0 {
		if specs, _, err := dnsresolver.ParseList(strings.Join(cfg.DNSResolvers, ",")); err == nil {
			transport.DialContext = dnsresolver.DialFunc(specs)
		} else {
			slog.Warn("api: invalid dns resolver urls, falling back to system resolver",
				slog.String("error", err.Error()),
			)
		}
	}

	jar, _ := cookiejar.New(nil)

	client := &http.Client{
		Transport: sdk.NewRetryTransport(transport),
		Timeout:   15 * time.Second,
		Jar:       jar,
	}

	reg := provider.NewRegistry()
	registerE2EProviders(reg)

	var ic *cache.DiskCache
	if cfg != nil && cfg.CacheDir != "" {
		var err error
		ic, err = cache.NewDiskCache(cfg.CacheDir, cfg.CacheImageTTL, cfg.CacheMaxBytes)
		if err != nil {
			slog.Error("api: failed to initialize image cache", slog.String("error", err.Error()))
		} else if ic != nil {
			ic.StartCleanupWorker(context.Background(), 1*time.Hour)
		}
	}

	var pm *host.PluginManager
	if cfg != nil && cfg.PluginDir != "" {
		pm = host.NewPluginManager(host.ManagerOptions{
			PluginDir: cfg.PluginDir,
			Registry:  reg,
			HTTPConfig: plugin_sdk.GlobalHttpConfig{
				ProxyURL:       "", // future: KIYOMI_PROXY
				UserAgent:      "",
				TimeoutSeconds: 0,
				DNSResolvers:   cfg.DNSResolvers,
			},
		})
	}

	reqBuilder := upstream.NewRequestBuilder(fpStore, reg, client)

	return &Handler{
		cfg:           cfg,
		lib:           lib,
		httpClient:    client,
		registry:      reg,
		pluginManager: pm,
		fpStore:       fpStore,
		imageCache:    ic,
		reqBuilder:    reqBuilder,
		jobStore:      jobStore,
		enqueuer:      enqueuer,
	}
}

// ImageCache returns the handler's disk image cache instance.
func (h *Handler) ImageCache() *cache.DiskCache {
	return h.imageCache
}

// SetPluginManager sets the plugin manager instance.
func (h *Handler) SetPluginManager(pm *host.PluginManager) {
	h.pluginManager = pm
}

// PluginManager returns the current plugin manager instance.
func (h *Handler) PluginManager() *host.PluginManager {
	return h.pluginManager
}

// Registry returns the provider registry instance.
func (h *Handler) Registry() *provider.Registry {
	return h.registry
}

// FingerprintStore returns the configured fingerprint store.
func (h *Handler) FingerprintStore() fingerprint.Store {
	return h.fpStore
}

// RequestBuilder returns the upstream request builder instance.
func (h *Handler) RequestBuilder() *upstream.RequestBuilder {
	return h.reqBuilder
}

// HTTPClient returns the configured HTTP client used by the handler. Useful
// for sharing the same transport/TLS-fingerprint/retry chain with background
// workers (e.g. download handlers).
func (h *Handler) HTTPClient() *http.Client {
	return h.httpClient
}

// SetBuildInfo sets the build metadata on the handler.
func (h *Handler) SetBuildInfo(info BuildInfo) {
	h.buildInfo = info
}

// RegisterRoutes registers all RESTful HTTP routes.
func (h *Handler) RegisterRoutes(e *echo.Echo) {
	registerE2ERoutes(e)
	v1 := e.Group("/api/v1")

	// System Info
	v1.GET("/info", h.getInfo)

	// System Cache
	v1.GET("/system/cache", h.getCacheStats)
	v1.POST("/system/cache/clear", h.clearCache)

	// Content Providers
	v1.GET("/providers", h.listContentProviders)
	v1.GET("/providers/:providerId/manga", h.getProviderMangaCatalog)
	v1.GET("/providers/:providerId/manga/:remoteId", h.getProviderMangaDetails)
	v1.GET("/providers/:providerId/manga/:remoteId/chapters", h.getProviderMangaChapters)
	v1.GET("/providers/:providerId/manga/:remoteId/chapters/:chapterId/pages", h.getChapterPages)
	v1.GET("/providers/:providerId/popular", h.getPopularManga)
	v1.GET("/providers/:providerId/latest", h.getLatestManga)
	v1.GET("/providers/:providerId/search", h.searchManga)

	// Provider Fingerprint Management
	v1.GET("/providers/:providerId/fingerprint", h.handleGetFingerprint)
	v1.PUT("/providers/:providerId/fingerprint", h.handlePutFingerprint)
	v1.DELETE("/providers/:providerId/fingerprint", h.handleDeleteFingerprint)

	// Plugin Management
	v1.GET("/plugins", h.listPlugins)
	v1.POST("/plugins/reload", h.reloadPlugins)
	v1.GET("/plugins/:id/logs", h.getPluginLogs)
	v1.POST("/plugins/:id/config", h.updatePluginConfig)
	v1.GET("/plugins/collisions", h.listCollisions)
	v1.POST("/plugins/preference", h.setPluginPreference)

	// Central Local Library Manga
	v1.GET("/library/manga", h.listLibraryManga)
	v1.GET("/library/manga/:mangaId", h.getLibraryManga)
	v1.POST("/library/manga", h.createLibraryManga)
	v1.POST("/library/manga/import", h.importProviderManga)
	v1.POST("/library/manga/:mangaId/refresh", h.refreshLibraryManga)
	v1.POST("/library/manga/:mangaId/pull", h.pullManga)
	v1.PUT("/library/manga/:mangaId", h.updateLibraryManga)
	v1.PATCH("/library/manga/:mangaId", h.patchLibraryManga)
	v1.DELETE("/library/manga/:mangaId", h.deleteLibraryManga)

	// Provider Bindings
	v1.GET("/library/manga/:mangaId/providers", h.listProviders)
	v1.POST("/library/manga/:mangaId/providers", h.addProvider)
	v1.DELETE("/library/manga/:mangaId/providers/:providerId/:providerMangaId", h.removeProvider)
	v1.PATCH("/library/manga/:mangaId/content", h.switchContentProvider)

	// Chapters & Pages
	v1.GET("/library/manga/:mangaId/chapters", h.listChapters)
	v1.GET("/library/manga/:mangaId/providers/:providerId/chapters/:chapterId", h.getChapter)
	v1.POST("/library/manga/:mangaId/providers/:providerId/chapters/:chapterId", h.saveChapter)
	v1.PATCH("/library/manga/:mangaId/providers/:providerId/chapters/:chapterId/progress", h.patchChapterProgress)
	v1.DELETE("/library/manga/:mangaId/providers/:providerId/chapters/:chapterId", h.deleteChapter)
	v1.DELETE("/library/manga/:mangaId/providers/:providerId/chapters/:chapterId/files", h.deleteChapterFiles)
	v1.GET("/chapters/:chapterId/pages", h.getChapterPages)
	v1.POST("/library/manga/:mangaId/providers/:providerId/chapters/:chapterId/pull", h.pullChapter)

	// Batch Chapters Operations
	v1.PATCH("/library/manga/:mangaId/providers/:providerId/chapters/progress", h.patchChaptersProgressBatch)
	v1.POST("/library/manga/:mangaId/providers/:providerId/chapters/pull", h.pullChaptersBatch)
	v1.POST("/library/manga/:mangaId/providers/:providerId/chapters/refresh", h.refreshChaptersBatch)
	v1.POST("/library/manga/:mangaId/providers/:providerId/chapters/files/delete", h.deleteChapterFilesBatch)
	v1.DELETE("/library/manga/:mangaId/providers/:providerId/chapters/files", h.deleteChapterFilesBatch)
	v1.POST("/library/manga/:mangaId/providers/:providerId/chapters/delete", h.deleteChaptersBatch)
	v1.DELETE("/library/manga/:mangaId/providers/:providerId/chapters", h.deleteChaptersBatch)

	// Fingerprinted Page Image Reverse Proxy
	v1.GET("/library/manga/:mangaId/chapters/:chapterId/pages/:pageIndex", h.proxyPageImage)
	v1.GET("/proxy/image", h.proxyImageDirect)

	// Background Jobs
	v1.GET("/jobs", h.listJobs)
	v1.POST("/jobs", h.enqueueJob)
	v1.DELETE("/jobs", h.cleanupJobs)
	v1.POST("/jobs/:id/cancel", h.cancelJob)
	v1.DELETE("/jobs/:id", h.deleteJob)
}
