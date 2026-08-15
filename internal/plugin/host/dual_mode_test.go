package host_test

import (
	"context"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tubruk/kiyomi/internal/plugin/host"
	"github.com/tubruk/kiyomi/pkg/provider"
	provsdk "github.com/tubruk/kiyomi/pkg/provider/sdk"
	sdk "github.com/tubruk/kiyomi/plugin-sdk"
)

type mockBuiltinProvider struct {
	id   string
	name string
}

func (m *mockBuiltinProvider) ID() string                    { return m.id }
func (m *mockBuiltinProvider) Name() string                  { return m.name }
func (m *mockBuiltinProvider) Icon() string                  { return "" }
func (m *mockBuiltinProvider) Capabilities() []string        { return []string{"metadata", "content"} }
func (m *mockBuiltinProvider) ConfigKeys() []provsdk.ConfigKeySpec { return nil }
func (m *mockBuiltinProvider) RequiresAuth() bool           { return false }
func (m *mockBuiltinProvider) State() provsdk.ProviderState  { return provsdk.StateActive }
func (m *mockBuiltinProvider) IsBuiltIn() bool              { return true }

func TestDualModeExecution_BuiltinAndStandalonePlugins(t *testing.T) {
	tmpDir := t.TempDir()

	binExt := ""
	if runtime.GOOS == "windows" {
		binExt = ".exe"
	}

	dexBin := filepath.Join(tmpDir, "mangadex-plugin"+binExt)
	foxBin := filepath.Join(tmpDir, "mangafox-plugin"+binExt)

	// Resolve repository root directory
	repoRoot, err := filepath.Abs("../../..")
	require.NoError(t, err)

	dexPkg := filepath.Join(repoRoot, "plugins", "mangadex")
	foxPkg := filepath.Join(repoRoot, "plugins", "mangafox")

	// Build standalone MangaDex plugin binary
	cmdDex := exec.Command("go", "build", "-o", dexBin, dexPkg)
	outDex, err := cmdDex.CombinedOutput()
	require.NoError(t, err, "failed to compile mangadex plugin: %s", string(outDex))

	// Build standalone MangaFox plugin binary
	cmdFox := exec.Command("go", "build", "-o", foxBin, foxPkg)
	outFox, err := cmdFox.CombinedOutput()
	require.NoError(t, err, "failed to compile mangafox plugin: %s", string(outFox))

	// Create registry and register in-process built-in providers
	reg := provider.NewRegistry()
	builtInDex := &mockBuiltinProvider{id: "mangadex", name: "MangaDex Built-in"}
	builtInFox := &mockBuiltinProvider{id: "mangafox", name: "MangaFox Built-in"}

	reg.Register(builtInDex)
	reg.Register(builtInFox)

	// Verify built-in providers are registered under primary and @builtin handles
	dexP, ok := reg.Get("mangadex")
	require.True(t, ok)
	assert.Equal(t, "mangadex", dexP.ID())

	dexBuiltin, ok := reg.Get("mangadex@builtin")
	require.True(t, ok)
	assert.Equal(t, builtInDex, dexBuiltin)

	foxP, ok := reg.Get("mangafox")
	require.True(t, ok)
	assert.Equal(t, "mangafox", foxP.ID())

	foxBuiltin, ok := reg.Get("mangafox@builtin")
	require.True(t, ok)
	assert.Equal(t, builtInFox, foxBuiltin)

	// Create PluginManager and discover/start standalone plugin binaries
	mgr := host.NewPluginManager(host.ManagerOptions{
		PluginDir: tmpDir,
		Registry:  reg,
		HTTPConfig: sdk.GlobalHttpConfig{
			UserAgent:      "KiyomiDualModeTest/1.0",
			TimeoutSeconds: 15,
		},
	})
	defer mgr.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	err = mgr.ReloadAll(ctx)
	require.NoError(t, err)

	// Verify plugins started
	dexStatus, ok := mgr.GetPluginStatus("mangadex")
	require.True(t, ok)
	assert.Equal(t, host.StateRunning, dexStatus.State)

	foxStatus, ok := mgr.GetPluginStatus("mangafox")
	require.True(t, ok)
	assert.Equal(t, host.StateRunning, foxStatus.State)

	// Verify dual-mode routing:
	// 1. Primary handles still route to in-process built-ins by default
	primDex, ok := reg.Get("mangadex")
	require.True(t, ok)
	assert.Equal(t, builtInDex, primDex)

	primFox, ok := reg.Get("mangafox")
	require.True(t, ok)
	assert.Equal(t, builtInFox, primFox)

	// 2. Namespaced @builtin handles route to in-process built-ins
	hDexBuiltin, ok := reg.Get("mangadex@builtin")
	require.True(t, ok)
	assert.Equal(t, builtInDex, hDexBuiltin)

	// 3. Namespaced @<plugin-id> handles route to out-of-process gRPC plugin adapters
	hDexPlugin, ok := reg.Get("mangadex@mangadex")
	require.True(t, ok)
	assert.NotEqual(t, builtInDex, hDexPlugin)
	assert.Equal(t, "mangadex", hDexPlugin.ID())

	hFoxPlugin, ok := reg.Get("mangafox@mangafox")
	require.True(t, ok)
	assert.NotEqual(t, builtInFox, hFoxPlugin)
	assert.Equal(t, "mangafox", hFoxPlugin.ID())

	// 4. Test provider methods through gRPC adapter on namespaced handle
	metaAdapter, ok := reg.GetMetadata("mangadex@mangadex")
	require.True(t, ok)
	require.NotNil(t, metaAdapter)

	contentAdapter, ok := reg.GetContent("mangadex@mangadex")
	require.True(t, ok)
	assert.True(t, contentAdapter.HasStableChapterID())
	assert.Equal(t, float64(5), contentAdapter.RateLimit().RequestsPerSecond)

	// 5. Test user preference switching to out-of-process plugin
	reg.SetUserPreference("mangadex", "mangadex")
	prefDex, ok := reg.Get("mangadex")
	require.True(t, ok)
	assert.Equal(t, hDexPlugin, prefDex, "user preference should route primary handle to plugin")

	// 6. Test user preference switching back to builtin
	reg.SetUserPreference("mangadex", "builtin")
	restoredDex, ok := reg.Get("mangadex")
	require.True(t, ok)
	assert.Equal(t, builtInDex, restoredDex, "user preference should restore primary handle to builtin")

	// 7. Test Candidates and Collisions inspection
	cands := reg.Candidates()
	require.Contains(t, cands, "mangadex")
	assert.Len(t, cands["mangadex"], 2, "expected 2 candidates (builtin and standalone plugin)")
	require.Contains(t, cands, "mangafox")
	assert.Len(t, cands["mangafox"], 2, "expected 2 candidates (builtin and standalone plugin)")

	// 8. Test list info returns both primary and namespaced info appropriately
	primaryInfo := reg.ListInfo()
	assert.Len(t, primaryInfo, 2)

	allInfo := reg.ListAllInfo()
	assert.GreaterOrEqual(t, len(allInfo), 6) // mangadex, mangafox, mangadex@builtin, mangadex@mangadex, mangafox@builtin, mangafox@mangafox

	// 9. Verify stop
	err = mgr.Stop(ctx, "mangadex")
	require.NoError(t, err)

	_, ok = reg.Get("mangadex@mangadex")
	assert.False(t, ok, "mangadex@mangadex should be unregistered after stop")

	// Builtin should remain intact
	stillDex, ok := reg.Get("mangadex")
	require.True(t, ok)
	assert.Equal(t, builtInDex, stillDex)
}
