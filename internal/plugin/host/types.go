package host

import (
	"time"

	"github.com/tubruk/kiyomi/pkg/provider"
	sdk "github.com/tubruk/kiyomi/plugin-sdk"
)

// PluginState represents the lifecycle state of a plugin.
type PluginState string

const (
	StateStopped   PluginState = "stopped"
	StateStarting  PluginState = "starting"
	StateRunning   PluginState = "running"
	StateError     PluginState = "error"
	StateReloading PluginState = "reloading"
)

// PluginLogEntry represents a structured log entry captured from a plugin subprocess.
type PluginLogEntry struct {
	Timestamp time.Time      `json:"timestamp"`
	Level     string         `json:"level"`
	Message   string         `json:"message"`
	Fields    map[string]any `json:"fields,omitempty"`
	Raw       string         `json:"raw,omitempty"`
}

// PluginStatus describes the current runtime status and metadata of a plugin instance.
type PluginStatus struct {
	PluginID             string                    `json:"pluginId"`
	PluginName           string                    `json:"pluginName"`
	PluginVersion        string                    `json:"pluginVersion"`
	SDKVersion           string                    `json:"sdkVersion"`
	ExecutablePath       string                    `json:"executablePath"`
	PID                  int                       `json:"pid"`
	State                PluginState               `json:"state"`
	ErrorMessage         string                    `json:"errorMessage,omitempty"`
	LoadedAt             time.Time                 `json:"loadedAt"`
	Providers            []sdk.ProviderDescriptor   `json:"providers"`
	PluginSettingsSchema []sdk.SettingSpec         `json:"pluginSettingsSchema,omitempty"`
}

// ManagerOptions provides configuration settings when initializing a PluginManager.
type ManagerOptions struct {
	PluginDir       string
	GlobalConfig    map[string]string
	ProviderConfigs map[string]map[string]string
	HTTPConfig      sdk.GlobalHttpConfig
	UserPreferences map[string]string // providerID -> preferredPluginID or "builtin"
	Registry        *provider.Registry
	HostSDKVersion  string
	LogBufferLimit  int
}
