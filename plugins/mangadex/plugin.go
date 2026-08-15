package main

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	sdk "github.com/tubruk/kiyomi/plugin-sdk"
	sdklogger "github.com/tubruk/kiyomi/plugin-sdk/logger"
)

const (
	PluginID    = "mangadex"
	PluginName  = "MangaDex"
	Version     = "1.0.0"
	DefaultBase = "https://api.mangadex.org"
	DefaultUA   = "Kiyomi/1.0.0 (https://github.com/tubruk/kiyomi)"
)

// MangaDexPlugin implements sdk.Plugin, sdk.MetadataProvider, and sdk.ContentProvider.
type MangaDexPlugin struct {
	mu        sync.RWMutex
	baseURL   string
	userAgent string
	dataSaver bool
	client    *http.Client
	logger    *slog.Logger
}

// NewMangaDexPlugin creates a new MangaDexPlugin instance with default options.
func NewMangaDexPlugin() *MangaDexPlugin {
	return &MangaDexPlugin{
		baseURL:   DefaultBase,
		userAgent: DefaultUA,
		client: &http.Client{
			Timeout: 20 * time.Second,
		},
		logger: sdklogger.New(os.Stderr, &sdklogger.Options{Level: slog.LevelInfo}),
	}
}

// Describe returns self-describing metadata for MangaDex plugin and its providers.
func (p *MangaDexPlugin) Describe(ctx context.Context) (sdk.PluginDescriptor, error) {
	return sdk.PluginDescriptor{
		PluginID:      PluginID,
		PluginName:    PluginName,
		PluginVersion: Version,
		SDKVersion:    sdk.Version,
		PluginSettingsSchema: []sdk.SettingSpec{
			{
				Key:          "data_saver",
				Label:        "Data Saver Mode",
				Description:  "Use compressed low-bandwidth image stream from MangaDex@Home",
				Type:         "boolean",
				DefaultValue: "false",
			},
		},
		Providers: []sdk.ProviderDescriptor{
			{
				ID:           PluginID,
				Name:         "MangaDex",
				Description:  "Official MangaDex REST API v5 provider for manga metadata and chapters",
				Capabilities: []string{"metadata", "content"},
				DefaultRateLimit: sdk.RateLimitSpec{
					RequestsPerSecond:     5,
					MaxConcurrentRequests: 5,
				},
				SettingsSchema: []sdk.SettingSpec{
					{
						Key:          "data_saver",
						Label:        "Data Saver Mode",
						Description:  "Use compressed low-bandwidth image stream from MangaDex@Home",
						Type:         "boolean",
						DefaultValue: "false",
					},
				},
			},
		},
	}, nil
}

// Init configures the plugin instance with settings received from the host process.
func (p *MangaDexPlugin) Init(ctx context.Context, config sdk.PluginConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Configure HTTP client options
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

	p.client = &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}

	if config.HTTPConfig.UserAgent != "" {
		p.userAgent = config.HTTPConfig.UserAgent
	}

	// Read data saver setting
	dataSaverStr := ""
	if provCfg, ok := config.ProviderConfigs[PluginID]; ok {
		dataSaverStr = provCfg["data_saver"]
	}
	if dataSaverStr == "" && config.GlobalConfig != nil {
		dataSaverStr = config.GlobalConfig["data_saver"]
	}
	if strings.EqualFold(dataSaverStr, "true") || dataSaverStr == "1" {
		p.dataSaver = true
	} else if strings.EqualFold(dataSaverStr, "false") || dataSaverStr == "0" {
		p.dataSaver = false
	}

	p.logger.Info("mangadex plugin initialized",
		slog.String("plugin_id", PluginID),
		slog.String("version", Version),
		slog.Bool("data_saver", p.dataSaver),
	)

	return nil
}
