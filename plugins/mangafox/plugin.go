package main

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"sync"
	"time"

	sdk "github.com/tubruk/kiyomi/plugin-sdk"
	sdklogger "github.com/tubruk/kiyomi/plugin-sdk/logger"
)

const (
	PluginID       = "mangafox"
	PluginName     = "MangaFox"
	Version        = "1.0.0"
	DefaultBaseURL = "https://fanfox.net"
	Language       = "en"
)

// MangaFoxPlugin implements sdk.Plugin, sdk.MetadataProvider, and sdk.ContentProvider.
type MangaFoxPlugin struct {
	mu        sync.RWMutex
	baseURL   string
	userAgent string
	client    *http.Client
	logger    *slog.Logger
}

// NewMangaFoxPlugin creates a new MangaFoxPlugin instance.
func NewMangaFoxPlugin() *MangaFoxPlugin {
	jar, _ := cookiejar.New(nil)
	u1, _ := url.Parse("https://fanfox.net")
	u2, _ := url.Parse("https://m.fanfox.net")
	if jar != nil {
		if u1 != nil {
			jar.SetCookies(u1, []*http.Cookie{{Name: "isAdult", Value: "1", Path: "/"}})
		}
		if u2 != nil {
			jar.SetCookies(u2, []*http.Cookie{{Name: "readway", Value: "2", Path: "/"}, {Name: "isAdult", Value: "1", Path: "/"}})
		}
	}

	return &MangaFoxPlugin{
		baseURL:   DefaultBaseURL,
		userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		client: &http.Client{
			Jar:     jar,
			Timeout: 20 * time.Second,
		},
		logger: sdklogger.New(os.Stderr, &sdklogger.Options{Level: slog.LevelInfo}),
	}
}

// Describe returns self-describing metadata for MangaFox plugin.
func (p *MangaFoxPlugin) Describe(ctx context.Context) (sdk.PluginDescriptor, error) {
	return sdk.PluginDescriptor{
		PluginID:      PluginID,
		PluginName:    PluginName,
		PluginVersion: Version,
		SDKVersion:    sdk.Version,
		Providers: []sdk.ProviderDescriptor{
			{
				ID:           PluginID,
				Name:         "MangaFox",
				Description:  "MangaFox / FanFox HTML scraper for manga metadata and chapters",
				Capabilities: []string{"metadata", "content"},
				DefaultRateLimit: sdk.RateLimitSpec{
					RequestsPerSecond:     1,
					MaxConcurrentRequests: 1,
				},
			},
		},
	}, nil
}

// Init configures the plugin instance with settings from the host.
func (p *MangaFoxPlugin) Init(ctx context.Context, config sdk.PluginConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	timeout := 20 * time.Second
	if config.HTTPConfig.TimeoutSeconds > 0 {
		timeout = time.Duration(config.HTTPConfig.TimeoutSeconds) * time.Second
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if config.HTTPConfig.ProxyURL != "" {
		if proxyURL, err := url.Parse(config.HTTPConfig.ProxyURL); err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}

	jar, _ := cookiejar.New(nil)
	u1, _ := url.Parse("https://fanfox.net")
	u2, _ := url.Parse("https://m.fanfox.net")
	if jar != nil {
		if u1 != nil {
			jar.SetCookies(u1, []*http.Cookie{{Name: "isAdult", Value: "1", Path: "/"}})
		}
		if u2 != nil {
			jar.SetCookies(u2, []*http.Cookie{{Name: "readway", Value: "2", Path: "/"}, {Name: "isAdult", Value: "1", Path: "/"}})
		}
	}

	p.client = &http.Client{
		Transport: transport,
		Jar:       jar,
		Timeout:   timeout,
	}

	if config.HTTPConfig.UserAgent != "" {
		p.userAgent = config.HTTPConfig.UserAgent
	}

	p.logger.Info("mangafox plugin initialized",
		slog.String("plugin_id", PluginID),
		slog.String("version", Version),
	)

	return nil
}
