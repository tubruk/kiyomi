package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	sdk "github.com/tubruk/kiyomi/plugin-sdk"
	sdkhttp "github.com/tubruk/kiyomi/plugin-sdk/http"
	sdklogger "github.com/tubruk/kiyomi/plugin-sdk/logger"
)

const (
	PluginID    = "myanimelist"
	PluginName  = "MyAnimeList"
	Version     = "1.0.0"
	DefaultBase = "https://api.myanimelist.net/v2"
	DefaultUA   = "Kiyomi/1.0.0 (https://github.com/tubruk/kiyomi)"
)

// MyAnimeListPlugin implements sdk.Plugin and sdk.MetadataProvider.
type MyAnimeListPlugin struct {
	mu        sync.RWMutex
	baseURL   string
	userAgent string
	clientID  string
	client    *http.Client
	logger    *slog.Logger
}

// NewMyAnimeListPlugin creates a new MyAnimeListPlugin instance with default options.
func NewMyAnimeListPlugin() *MyAnimeListPlugin {
	clientID := os.Getenv("KIYOMI_MAL_CLIENT_ID")
	return &MyAnimeListPlugin{
		baseURL:   DefaultBase,
		userAgent: DefaultUA,
		clientID:  clientID,
		client: &http.Client{
			Timeout: 20 * time.Second,
		},
		logger: sdklogger.New(os.Stderr, &sdklogger.Options{Level: slog.LevelInfo}),
	}
}

// Describe returns self-describing metadata for MyAnimeList plugin and its providers.
func (p *MyAnimeListPlugin) Describe(ctx context.Context) (sdk.PluginDescriptor, error) {
	return sdk.PluginDescriptor{
		PluginID:      PluginID,
		PluginName:    PluginName,
		PluginVersion: Version,
		SDKVersion:    sdk.Version,
		PluginSettingsSchema: []sdk.SettingSpec{
			{
				Key:         "client_id",
				Label:       "Client ID",
				Description: "MyAnimeList API Client ID (falls back to KIYOMI_MAL_CLIENT_ID env var)",
				Type:        "secret",
			},
		},
		Providers: []sdk.ProviderDescriptor{
			{
				ID:           PluginID,
				Name:         "MyAnimeList",
				Description:  "Official MyAnimeList API v2 provider for manga metadata and ranking",
				Capabilities: []string{"metadata"},
				DefaultRateLimit: sdk.RateLimitSpec{
					RequestsPerSecond:     2,
					MaxConcurrentRequests: 2,
				},
				SettingsSchema: []sdk.SettingSpec{
					{
						Key:         "client_id",
						Label:       "Client ID",
						Description: "MyAnimeList API Client ID (falls back to KIYOMI_MAL_CLIENT_ID env var)",
						Type:        "secret",
					},
				},
			},
		},
	}, nil
}

// Init configures the plugin instance with settings received from the host process.
func (p *MyAnimeListPlugin) Init(ctx context.Context, config sdk.PluginConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Configure HTTP client options
	timeout := 20 * time.Second
	if config.HTTPConfig.TimeoutSeconds > 0 {
		timeout = time.Duration(config.HTTPConfig.TimeoutSeconds) * time.Second
	}

	httpClient := sdkhttp.NewClient(
		sdkhttp.WithSDKGlobalHttpConfig(config.HTTPConfig),
		sdkhttp.WithTimeout(timeout),
	).StandardClient()

	p.client = httpClient

	if config.HTTPConfig.UserAgent != "" {
		p.userAgent = config.HTTPConfig.UserAgent
	}

	// Read client_id setting with fallback to existing env var
	clientID := ""
	if provCfg, ok := config.ProviderConfigs[PluginID]; ok {
		clientID = provCfg["client_id"]
	}
	if clientID == "" && config.GlobalConfig != nil {
		clientID = config.GlobalConfig["client_id"]
	}
	if clientID == "" {
		clientID = os.Getenv("KIYOMI_MAL_CLIENT_ID")
	}

	p.clientID = strings.TrimSpace(clientID)

	hasClientID := p.clientID != ""
	p.logger.Info("myanimelist plugin initialized",
		slog.String("plugin_id", PluginID),
		slog.String("version", Version),
		slog.Bool("has_client_id", hasClientID),
	)

	return nil
}
