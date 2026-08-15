package sdk

import (
	"github.com/hashicorp/go-plugin"
)

// Constants identifying the plugin protocol and service endpoints.
const (
	PluginProtocolVersion      = 1
	PluginMagicCookieKey       = "KIYOMI_PLUGIN"
	PluginMagicCookieValue     = "kiyomi_provider_v1"
	PluginServiceName          = "plugin"
	MetadataProviderPluginName = "metadata_provider"
	ContentProviderPluginName  = "content_provider"
	TrackerPluginName          = "tracker"
)

// HandshakeConfig is the standard handshake configuration required by HashiCorp go-plugin.
var HandshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  PluginProtocolVersion,
	MagicCookieKey:   PluginMagicCookieKey,
	MagicCookieValue: PluginMagicCookieValue,
}

// PluginMap constructs a HashiCorp go-plugin PluginSet with the given implementations.
func PluginMap(
	pluginImpl Plugin,
	metadata map[string]MetadataProvider,
	content map[string]ContentProvider,
	trackers map[string]Tracker,
) plugin.PluginSet {
	return plugin.PluginSet{
		PluginServiceName:          &PluginGRPCPlugin{Impl: pluginImpl},
		MetadataProviderPluginName: &MetadataProviderGRPCPlugin{Providers: metadata},
		ContentProviderPluginName:  &ContentProviderGRPCPlugin{Providers: content},
		TrackerPluginName:          &TrackerGRPCPlugin{Providers: trackers},
	}
}

// ServeOptions configures the plugin server when launching via ServePlugin.
type ServeOptions struct {
	Plugin            Plugin
	Providers         []any
	MetadataProviders map[string]MetadataProvider
	ContentProviders  map[string]ContentProvider
	Trackers          map[string]Tracker
}

// ServePlugin is the main entry point called by plugin binaries to start the HashiCorp go-plugin server.
func ServePlugin(opts ServeOptions) {
	metaMap := make(map[string]MetadataProvider)
	for k, v := range opts.MetadataProviders {
		metaMap[k] = v
	}

	contentMap := make(map[string]ContentProvider)
	for k, v := range opts.ContentProviders {
		contentMap[k] = v
	}

	trackerMap := make(map[string]Tracker)
	for k, v := range opts.Trackers {
		trackerMap[k] = v
	}

	// Auto-register providers from slice based on interface assertions
	for _, p := range opts.Providers {
		idGetter, hasID := p.(interface{ ID() string })
		if !hasID {
			continue
		}
		id := idGetter.ID()

		if m, ok := p.(MetadataProvider); ok {
			metaMap[id] = m
		}
		if c, ok := p.(ContentProvider); ok {
			contentMap[id] = c
		}
		if t, ok := p.(Tracker); ok {
			trackerMap[id] = t
		}
	}

	plugins := PluginMap(opts.Plugin, metaMap, contentMap, trackerMap)

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: HandshakeConfig,
		Plugins:         plugins,
		GRPCServer:      plugin.DefaultGRPCServer,
	})
}
