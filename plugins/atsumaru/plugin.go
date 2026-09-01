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
	PluginID    = "atsumaru"
	PluginName  = "Atsumaru"
	Version     = "1.0.0"
	DefaultBase = "https://atsu.moe"
	DefaultUA   = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36"
)

// AtsumaruPlugin implements sdk.Plugin, sdk.MetadataProvider, and sdk.ContentProvider.
type AtsumaruPlugin struct {
	mu        sync.RWMutex
	baseURL   string
	userAgent string
	adultMode bool
	client    *http.Client
	logger    *slog.Logger
}

// NewAtsumaruPlugin creates a new AtsumaruPlugin instance.
func NewAtsumaruPlugin() *AtsumaruPlugin {
	return &AtsumaruPlugin{
		baseURL:   DefaultBase,
		userAgent: DefaultUA,
		client: &http.Client{
			Timeout: 20 * time.Second,
		},
		logger: sdklogger.New(os.Stderr, &sdklogger.Options{Level: slog.LevelInfo}),
	}
}

// Describe returns self-describing metadata for Atsumaru plugin and its providers.
func (p *AtsumaruPlugin) Describe(ctx context.Context) (sdk.PluginDescriptor, error) {
	return sdk.PluginDescriptor{
		PluginID:      PluginID,
		PluginName:    PluginName,
		PluginVersion: Version,
		SDKVersion:    sdk.Version,
		PluginSettingsSchema: []sdk.SettingSpec{
			{
				Key:          "adult_mode",
				Label:        "Include Adult Content (+18)",
				Description:  "Include 18+ adult titles in search and browse results",
				Type:         "boolean",
				DefaultValue: "false",
			},
		},
		Providers: []sdk.ProviderDescriptor{
			{
				ID:           PluginID,
				Name:         "Atsumaru",
				Description:  "Atsumaru JSON REST API provider for manga metadata and chapters",
				Capabilities: []string{"metadata", "content"},
				DefaultRateLimit: sdk.RateLimitSpec{
					RequestsPerSecond:     2,
					MaxConcurrentRequests: 4,
				},
				SettingsSchema: []sdk.SettingSpec{
					{
						Key:          "adult_mode",
						Label:        "Include Adult Content (+18)",
						Description:  "Include 18+ adult titles in search and browse results",
						Type:         "boolean",
						DefaultValue: "false",
					},
				},
			},
		},
	}, nil
}

// Init configures the plugin instance with settings received from the host process.
func (p *AtsumaruPlugin) Init(ctx context.Context, config sdk.PluginConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()

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

	// Read adult mode setting
	adultStr := ""
	if provCfg, ok := config.ProviderConfigs[PluginID]; ok {
		adultStr = provCfg["adult_mode"]
	}
	if adultStr == "" && config.GlobalConfig != nil {
		adultStr = config.GlobalConfig["adult_mode"]
	}
	if strings.EqualFold(adultStr, "true") || adultStr == "1" {
		p.adultMode = true
	} else if strings.EqualFold(adultStr, "false") || adultStr == "0" {
		p.adultMode = false
	}

	p.logger.Info("atsumaru plugin initialized",
		slog.String("plugin_id", PluginID),
		slog.String("version", Version),
		slog.Bool("adult_mode", p.adultMode),
	)

	return nil
}
