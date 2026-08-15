package provider

import (
	"sync"

	"github.com/tubruk/kiyomi/pkg/provider/sdk"
)

// Registry manages thread-safe storage and lookup of providers by capability.
type Registry struct {
	mu       sync.RWMutex
	provs    map[string]sdk.Provider
	metadata map[string]sdk.Metadata
	content  map[string]sdk.Content
	tracking map[string]sdk.Tracking
}

// NewRegistry creates a new initialized Registry instance.
func NewRegistry() *Registry {
	return &Registry{
		provs:    make(map[string]sdk.Provider),
		metadata: make(map[string]sdk.Metadata),
		content:  make(map[string]sdk.Content),
		tracking: make(map[string]sdk.Tracking),
	}
}

// Register inspects implemented interfaces of p and indexes them in provs, metadata, content, and tracking maps under p.ID().
func (r *Registry) Register(p any) {
	prov, ok := p.(sdk.Provider)
	if !ok {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	id := prov.ID()
	r.provs[id] = prov

	if m, ok := p.(sdk.Metadata); ok {
		r.metadata[id] = m
	} else {
		delete(r.metadata, id)
	}

	if c, ok := p.(sdk.Content); ok {
		r.content[id] = c
	} else {
		delete(r.content, id)
	}

	if t, ok := p.(sdk.Tracking); ok {
		r.tracking[id] = t
	} else {
		delete(r.tracking, id)
	}
}

// Get returns the sdk.Provider for the given ID.
func (r *Registry) Get(id string) (sdk.Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.provs[id]
	return p, ok
}

// GetMetadata returns the sdk.Metadata for the given ID.
func (r *Registry) GetMetadata(id string) (sdk.Metadata, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.metadata[id]
	return m, ok
}

// GetContent returns the sdk.Content for the given ID.
func (r *Registry) GetContent(id string) (sdk.Content, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.content[id]
	return c, ok
}

// GetTracking returns the sdk.Tracking for the given ID.
func (r *Registry) GetTracking(id string) (sdk.Tracking, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tracking[id]
	return t, ok
}

// ListContent returns a slice of all registered sdk.Content providers.
func (r *Registry) ListContent() []sdk.Content {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]sdk.Content, 0, len(r.content))
	for _, c := range r.content {
		list = append(list, c)
	}
	return list
}

// ListMetadata returns a slice of all registered sdk.Metadata providers.
func (r *Registry) ListMetadata() []sdk.Metadata {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]sdk.Metadata, 0, len(r.metadata))
	for _, m := range r.metadata {
		list = append(list, m)
	}
	return list
}

// ListInfo returns a slice of sdk.ProviderInfo for all registered providers.
func (r *Registry) ListInfo() []sdk.ProviderInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]sdk.ProviderInfo, 0, len(r.provs))
	for _, p := range r.provs {
		list = append(list, sdk.ProviderInfo{
			ID:           p.ID(),
			Name:         p.Name(),
			Capabilities: p.Capabilities(),
			State:        p.State(),
			RequiresAuth: p.RequiresAuth(),
			ConfigKeys:   p.ConfigKeys(),
		})
	}
	return list
}
