package host

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-plugin"
	sdk "github.com/tubruk/kiyomi/plugin-sdk"
	v1 "github.com/tubruk/kiyomi/plugin-sdk/proto/v1"
)

// PluginManager coordinates discovery, execution, log interception, lifecycle,
// and zero-downtime hot-reloading of out-of-process Kiyomi plugin binaries.
type PluginManager struct {
	opts          ManagerOptions
	instances     map[string]*PluginInstance
	binaryPathMap map[string]string // pluginID -> executable binary path
	mu            sync.RWMutex
}

// NewPluginManager creates a new PluginManager with the provided options.
func NewPluginManager(opts ManagerOptions) *PluginManager {
	if opts.HostSDKVersion == "" {
		opts.HostSDKVersion = sdk.Version
	}
	if opts.LogBufferLimit <= 0 {
		opts.LogBufferLimit = DefaultLogBufferCapacity
	}
	return &PluginManager{
		opts:          opts,
		instances:     make(map[string]*PluginInstance),
		binaryPathMap: make(map[string]string),
	}
}

// DiscoverPlugins scans the configured PluginDir for executable binary files.
func (m *PluginManager) DiscoverPlugins() ([]string, error) {
	m.mu.RLock()
	dir := m.opts.PluginDir
	m.mu.RUnlock()

	if dir == "" {
		return nil, nil
	}

	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("plugin dir %q is not a directory", dir)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var binaries []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") || strings.HasSuffix(name, ".log") ||
			strings.HasSuffix(name, ".txt") || strings.HasSuffix(name, ".json") ||
			strings.HasSuffix(name, ".md") {
			continue
		}

		fullPath := filepath.Join(dir, name)
		fileInfo, err := entry.Info()
		if err != nil {
			continue
		}

		// Check if file is executable
		if fileInfo.Mode()&0111 != 0 {
			abs, err := filepath.Abs(fullPath)
			if err != nil {
				abs = fullPath
			}
			binaries = append(binaries, abs)
		}
	}

	return binaries, nil
}

// Start launches a plugin executable, completes handshake, runs Describe and Init,
// checks SDK compatibility, and registers its provider adapters into the registry.
func (m *PluginManager) Start(ctx context.Context, binaryPath string) (*PluginInstance, error) {
	inst, err := m.startInstance(ctx, binaryPath)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Stop any existing instance with the same pluginID
	if old, ok := m.instances[inst.PluginID]; ok && old != nil {
		if m.opts.Registry != nil {
			m.opts.Registry.UnregisterPlugin(inst.PluginID)
		}
		_ = old.Close()
	}

	// Register adapters into Registry
	if m.opts.Registry != nil {
		for _, adapter := range inst.Adapters {
			m.opts.Registry.Register(adapter)
		}
	}

	m.instances[inst.PluginID] = inst
	m.binaryPathMap[inst.PluginID] = binaryPath

	slog.Info("plugin started successfully",
		slog.String("plugin_id", inst.PluginID),
		slog.String("name", inst.Name),
		slog.String("version", inst.Version),
		slog.Int("pid", inst.PID),
		slog.Int("providers_count", len(inst.Adapters)),
	)

	return inst, nil
}

// startInstance spawns and initializes a PluginInstance without touching the manager registry map.
func (m *PluginManager) startInstance(ctx context.Context, binaryPath string) (*PluginInstance, error) {
	absPath, err := filepath.Abs(binaryPath)
	if err != nil {
		absPath = binaryPath
	}

	if _, err := os.Stat(absPath); err != nil {
		return nil, fmt.Errorf("plugin binary not found at %q: %w", absPath, err)
	}

	buffer := NewRingBuffer(m.opts.LogBufferLimit)
	interceptor := NewLogInterceptor(filepath.Base(absPath), buffer, slog.Default())

	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig:  sdk.HandshakeConfig,
		Plugins:          sdk.PluginMap(nil, nil, nil, nil),
		Cmd:              exec.Command(absPath),
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
		SyncStderr:       interceptor,
		SyncStdout:       interceptor,
	})

	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()
		return nil, fmt.Errorf("failed to connect to plugin %q: %w", absPath, err)
	}

	rawPlugin, err := rpcClient.Dispense(sdk.PluginServiceName)
	if err != nil {
		client.Kill()
		return nil, fmt.Errorf("failed to dispense plugin service for %q: %w", absPath, err)
	}

	plug, ok := rawPlugin.(sdk.Plugin)
	if !ok {
		client.Kill()
		return nil, fmt.Errorf("dispensed plugin does not implement sdk.Plugin")
	}

	desc, err := plug.Describe(ctx)
	if err != nil {
		client.Kill()
		return nil, fmt.Errorf("describe failed for plugin %q: %w", absPath, err)
	}

	// Update interceptor plugin ID to the declared plugin ID
	interceptor.SetPluginID(desc.PluginID)

	// Check SDK version compatibility
	if err := CheckSDKCompatibility(m.opts.HostSDKVersion, desc.SDKVersion); err != nil {
		client.Kill()
		return nil, fmt.Errorf("sdk compatibility check failed for plugin %q: %w", desc.PluginID, err)
	}

	// Prepare Init configuration
	cfg := sdk.PluginConfig{
		GlobalConfig:    m.opts.GlobalConfig,
		ProviderConfigs: m.opts.ProviderConfigs,
		HTTPConfig:      m.opts.HTTPConfig,
	}

	if err := plug.Init(ctx, cfg); err != nil {
		client.Kill()
		return nil, fmt.Errorf("init failed for plugin %q: %w", desc.PluginID, err)
	}

	// Dispense provider service clients
	var metaClient v1.MetadataProviderServiceClient
	if rawMeta, err := rpcClient.Dispense(sdk.MetadataProviderPluginName); err == nil && rawMeta != nil {
		if mc, ok := rawMeta.(v1.MetadataProviderServiceClient); ok {
			metaClient = mc
		}
	}

	var contentClient v1.ContentProviderServiceClient
	if rawContent, err := rpcClient.Dispense(sdk.ContentProviderPluginName); err == nil && rawContent != nil {
		if cc, ok := rawContent.(v1.ContentProviderServiceClient); ok {
			contentClient = cc
		}
	}

	var trackClient v1.TrackerServiceClient
	if rawTrack, err := rpcClient.Dispense(sdk.TrackerPluginName); err == nil && rawTrack != nil {
		if tc, ok := rawTrack.(v1.TrackerServiceClient); ok {
			trackClient = tc
		}
	}

	pid := 0
	if client.ReattachConfig() != nil {
		pid = client.ReattachConfig().Pid
	}

	inst := &PluginInstance{
		PluginID:       desc.PluginID,
		Name:           desc.PluginName,
		Version:        desc.PluginVersion,
		SDKVersion:     desc.SDKVersion,
		ExecutablePath: absPath,
		PID:            pid,
		State:          StateRunning,
		LoadedAt:       time.Now().UTC(),
		Client:         client,
		RPCClient:      rpcClient,
		PluginService:  plug,
		Descriptor:     desc,
		LogBuffer:      buffer,
	}

	adapters := make(map[string]*GRPCProviderAdapter, len(desc.Providers))
	for _, pDesc := range desc.Providers {
		adapter := NewGRPCProviderAdapter(pDesc, desc.PluginID, desc.PluginVersion, metaClient, contentClient, trackClient, &inst.ActiveCalls)
		adapters[pDesc.ID] = adapter
	}
	inst.Adapters = adapters

	return inst, nil
}

// Stop unregisters a plugin's providers, drains in-flight calls, and terminates its subprocess.
func (m *PluginManager) Stop(ctx context.Context, pluginID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	inst, ok := m.instances[pluginID]
	if !ok || inst == nil {
		return fmt.Errorf("plugin %q not found", pluginID)
	}

	if m.opts.Registry != nil {
		m.opts.Registry.UnregisterPlugin(pluginID)
	}

	if !inst.Drain(5 * time.Second) {
		slog.Warn("plugin instance did not drain in-flight calls before timeout", slog.String("plugin_id", pluginID))
	}
	_ = inst.Close()
	delete(m.instances, pluginID)

	slog.Info("plugin stopped", slog.String("plugin_id", pluginID))
	return nil
}

// Reload performs zero-downtime hot-reloading of a plugin:
// 1. Spawns and initializes a new plugin instance from the binary path.
// 2. Atomically swaps routing entries in the ProviderRegistry.
// 3. Drains in-flight gRPC calls on the old instance.
// 4. Terminates the old subprocess cleanly.
func (m *PluginManager) Reload(ctx context.Context, pluginID string) error {
	m.mu.RLock()
	oldInst, ok := m.instances[pluginID]
	binaryPath := m.binaryPathMap[pluginID]
	m.mu.RUnlock()

	if !ok || oldInst == nil {
		return fmt.Errorf("plugin %q not found for reload", pluginID)
	}
	if binaryPath == "" {
		binaryPath = oldInst.ExecutablePath
	}

	slog.Info("reloading plugin", slog.String("plugin_id", pluginID), slog.String("binary", binaryPath))

	// 1. Launch new instance
	newInst, err := m.startInstance(ctx, binaryPath)
	if err != nil {
		return fmt.Errorf("failed to launch new instance for plugin %q: %w", pluginID, err)
	}

	// 2. Atomic swap in registry
	m.mu.Lock()
	if m.opts.Registry != nil {
		for _, newAdapter := range newInst.Adapters {
			m.opts.Registry.Register(newAdapter)
		}
	}
	m.instances[pluginID] = newInst
	m.binaryPathMap[pluginID] = binaryPath
	m.mu.Unlock()

	// 3. Drain and terminate old instance in background
	go func(old *PluginInstance) {
		drained := old.Drain(5 * time.Second)
		if !drained {
			slog.Warn("old plugin instance did not drain in-flight calls before timeout", slog.String("plugin_id", old.PluginID))
		}
		_ = old.Close()
		slog.Info("old plugin instance drained and terminated", slog.String("plugin_id", old.PluginID))
	}(oldInst)

	slog.Info("plugin reloaded successfully", slog.String("plugin_id", pluginID), slog.String("version", newInst.Version))
	return nil
}

// ReloadAll reloads all currently running plugins and starts any newly discovered binaries.
func (m *PluginManager) ReloadAll(ctx context.Context) error {
	binaries, err := m.DiscoverPlugins()
	if err != nil {
		return fmt.Errorf("failed to discover plugins: %w", err)
	}

	m.mu.RLock()
	runningIDs := make([]string, 0, len(m.instances))
	for id := range m.instances {
		runningIDs = append(runningIDs, id)
	}
	m.mu.RUnlock()

	// Reload all running plugins
	for _, id := range runningIDs {
		if err := m.Reload(ctx, id); err != nil {
			slog.Error("failed to reload plugin", slog.String("plugin_id", id), slog.String("error", err.Error()))
		}
	}

	// Start any new binaries
	for _, binary := range binaries {
		m.mu.RLock()
		alreadyRunning := false
		for _, b := range m.binaryPathMap {
			if b == binary {
				alreadyRunning = true
				break
			}
		}
		m.mu.RUnlock()

		if !alreadyRunning {
			if _, err := m.Start(ctx, binary); err != nil {
				slog.Error("failed to start discovered plugin", slog.String("binary", binary), slog.String("error", err.Error()))
			}
		}
	}

	return nil
}

// GetPluginLogs returns the buffered log entries for a plugin.
func (m *PluginManager) GetPluginLogs(pluginID string) []PluginLogEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if inst, ok := m.instances[pluginID]; ok && inst != nil && inst.LogBuffer != nil {
		return inst.LogBuffer.Entries()
	}
	return nil
}

// GetPluginStatus returns the current status snapshot of a plugin.
func (m *PluginManager) GetPluginStatus(pluginID string) (PluginStatus, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if inst, ok := m.instances[pluginID]; ok && inst != nil {
		return inst.Status(), true
	}
	return PluginStatus{}, false
}

// ListPlugins returns status snapshots for all managed plugins.
func (m *PluginManager) ListPlugins() []PluginStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	statuses := make([]PluginStatus, 0, len(m.instances))
	for _, inst := range m.instances {
		if inst != nil {
			statuses = append(statuses, inst.Status())
		}
	}
	return statuses
}

// UpdatePluginConfig merges the provided configuration and re-initializes the running plugin instance.
func (m *PluginManager) UpdatePluginConfig(ctx context.Context, pluginID string, globalConfig map[string]string, providerConfigs map[string]map[string]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	inst, ok := m.instances[pluginID]
	if !ok || inst == nil {
		return fmt.Errorf("plugin %q not found", pluginID)
	}

	if m.opts.GlobalConfig == nil {
		m.opts.GlobalConfig = make(map[string]string)
	}
	for k, v := range globalConfig {
		m.opts.GlobalConfig[k] = v
	}

	if m.opts.ProviderConfigs == nil {
		m.opts.ProviderConfigs = make(map[string]map[string]string)
	}
	for pID, cfgMap := range providerConfigs {
		if m.opts.ProviderConfigs[pID] == nil {
			m.opts.ProviderConfigs[pID] = make(map[string]string)
		}
		for k, v := range cfgMap {
			m.opts.ProviderConfigs[pID][k] = v
		}
	}

	cfg := sdk.PluginConfig{
		GlobalConfig:    m.opts.GlobalConfig,
		ProviderConfigs: m.opts.ProviderConfigs,
		HTTPConfig:      m.opts.HTTPConfig,
	}

	if inst.PluginService != nil {
		if err := inst.PluginService.Init(ctx, cfg); err != nil {
			return fmt.Errorf("failed to re-init plugin %q: %w", pluginID, err)
		}
	}

	slog.Info("plugin configuration updated", slog.String("plugin_id", pluginID))
	return nil
}

// GetPluginConfig returns the stored global configuration and provider configurations for the given plugin.
func (m *PluginManager) GetPluginConfig(pluginID string) (map[string]string, map[string]map[string]string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	global := make(map[string]string)
	for k, v := range m.opts.GlobalConfig {
		global[k] = v
	}

	providers := make(map[string]map[string]string)
	if inst, ok := m.instances[pluginID]; ok && inst != nil {
		for _, pDesc := range inst.Descriptor.Providers {
			if cfg, exists := m.opts.ProviderConfigs[pDesc.ID]; exists {
				sub := make(map[string]string, len(cfg))
				for k, v := range cfg {
					sub[k] = v
				}
				providers[pDesc.ID] = sub
			}
		}
	}

	return global, providers
}

// RegisterInstanceForTest injects a plugin instance into the manager for testing purposes.
func (m *PluginManager) RegisterInstanceForTest(inst *PluginInstance) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if inst == nil {
		return
	}
	m.instances[inst.PluginID] = inst
	m.binaryPathMap[inst.PluginID] = inst.ExecutablePath
	if m.opts.Registry != nil {
		for _, adapter := range inst.Adapters {
			m.opts.Registry.Register(adapter)
		}
	}
}

// Close gracefully terminates all running plugin instances.
func (m *PluginManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, inst := range m.instances {
		if m.opts.Registry != nil {
			m.opts.Registry.UnregisterPlugin(id)
		}
		if inst != nil {
			if !inst.Drain(5 * time.Second) {
				slog.Warn("plugin instance did not drain in-flight calls before timeout", slog.String("plugin_id", id))
			}
			_ = inst.Close()
		}
	}
	m.instances = make(map[string]*PluginInstance)
	return nil
}
