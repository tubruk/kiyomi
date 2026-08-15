package host

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/go-plugin"
	sdk "github.com/tubruk/kiyomi/plugin-sdk"
)

// PluginInstance represents an active or stopped plugin subprocess instance.
type PluginInstance struct {
	mu             sync.RWMutex
	PluginID       string
	Name           string
	Version        string
	SDKVersion     string
	ExecutablePath string
	PID            int
	State          PluginState
	ErrorMessage   string
	LoadedAt       time.Time

	Client        *plugin.Client
	RPCClient     plugin.ClientProtocol
	PluginService sdk.Plugin
	Descriptor    sdk.PluginDescriptor
	Adapters      map[string]*GRPCProviderAdapter
	LogBuffer     *RingBuffer
	ActiveCalls   int64
}

// Drain blocks until all active in-flight gRPC calls on this instance complete or until timeout expires.
// Returns true if all calls drained cleanly, false if timeout was reached.
func (p *PluginInstance) Drain(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		if atomic.LoadInt64(&p.ActiveCalls) <= 0 {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(25 * time.Millisecond)
	}
}

// Close gracefully terminates the plugin subprocess after draining in-flight calls.
func (p *PluginInstance) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.State = StateStopped
	if p.Client != nil {
		p.Client.Kill()
	}
	return nil
}

// Status returns a snapshot of the plugin status.
func (p *PluginInstance) Status() PluginStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return PluginStatus{
		PluginID:             p.PluginID,
		PluginName:           p.Name,
		PluginVersion:        p.Version,
		SDKVersion:           p.SDKVersion,
		ExecutablePath:       p.ExecutablePath,
		PID:                  p.PID,
		State:                p.State,
		ErrorMessage:         p.ErrorMessage,
		LoadedAt:             p.LoadedAt,
		Providers:            p.Descriptor.Providers,
		PluginSettingsSchema: p.Descriptor.PluginSettingsSchema,
	}
}
