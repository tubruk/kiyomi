package provider

import (
	"strconv"
	"strings"
	"sync"

	"github.com/tubruk/kiyomi/pkg/provider/sdk"
)

// Candidate represents a registered provider implementation candidate for collision resolution.
type Candidate struct {
	Provider  sdk.Provider
	PluginID  string
	Version   string
	IsBuiltIn bool
}

// Registry manages thread-safe storage and lookup of providers by capability,
// supporting dual-mode (in-process built-in & out-of-process gRPC plugin adapters),
// collision resolution, namespaced routing handles, and zero-downtime hot-swapping.
type Registry struct {
	mu         sync.RWMutex
	provs      map[string]sdk.Provider
	metadata   map[string]sdk.Metadata
	content    map[string]sdk.Content
	tracking   map[string]sdk.Tracking
	candidates map[string][]Candidate // baseID -> candidates
	userPrefs  map[string]string      // baseID -> preferredPluginID or "builtin"
}

// NewRegistry creates a new initialized Registry instance.
func NewRegistry() *Registry {
	return &Registry{
		provs:      make(map[string]sdk.Provider),
		metadata:   make(map[string]sdk.Metadata),
		content:    make(map[string]sdk.Content),
		tracking:   make(map[string]sdk.Tracking),
		candidates: make(map[string][]Candidate),
		userPrefs:  make(map[string]string),
	}
}

// SetUserPreference sets explicit user preference for a provider ID (e.g. "plugin-b" or "builtin").
func (r *Registry) SetUserPreference(providerID, preferredSource string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.userPrefs[providerID] = preferredSource
	r.recomputeWinner(providerID)
}

// Register inspects implemented interfaces of p, tracks it as a candidate under p.ID(),
// resolves any collisions, and indexes winning & namespaced handles.
func (r *Registry) Register(p any) {
	prov, ok := p.(sdk.Provider)
	if !ok {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	baseID := prov.ID()
	pluginID := ""
	version := "0.0.0"
	isBuiltIn := true

	if pi, ok := p.(interface{ PluginID() string }); ok {
		pluginID = pi.PluginID()
		if pluginID != "" {
			isBuiltIn = false
		}
	}
	if v, ok := p.(interface{ Version() string }); ok && v.Version() != "" {
		version = v.Version()
	}
	if bi, ok := p.(interface{ IsBuiltIn() bool }); ok {
		isBuiltIn = bi.IsBuiltIn()
	}

	cand := Candidate{
		Provider:  prov,
		PluginID:  pluginID,
		Version:   version,
		IsBuiltIn: isBuiltIn,
	}

	list := r.candidates[baseID]
	updated := false
	for i, c := range list {
		if c.PluginID == pluginID && c.IsBuiltIn == isBuiltIn {
			list[i] = cand
			updated = true
			break
		}
	}
	if !updated {
		list = append(list, cand)
	}
	r.candidates[baseID] = list

	r.recomputeWinner(baseID)
}

// SwapProvider atomically replaces an old provider implementation with a new one for baseID.
func (r *Registry) SwapProvider(baseID string, oldProv, newProv sdk.Provider) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	list := r.candidates[baseID]
	found := false
	for i, c := range list {
		if c.Provider == oldProv {
			list[i].Provider = newProv
			if v, ok := newProv.(interface{ Version() string }); ok && v.Version() != "" {
				list[i].Version = v.Version()
			}
			found = true
			break
		}
	}
	if !found {
		return false
	}
	r.candidates[baseID] = list
	r.recomputeWinner(baseID)
	return true
}

// Unregister removes a provider by its primary or namespaced ID.
func (r *Registry) Unregister(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	baseID := id
	var targetPluginID string
	var isNamespaced bool

	if atIdx := strings.Index(id, "@"); atIdx >= 0 {
		baseID = id[:atIdx]
		targetPluginID = id[atIdx+1:]
		isNamespaced = true
	}

	list := r.candidates[baseID]
	newList := make([]Candidate, 0, len(list))
	for _, c := range list {
		if isNamespaced {
			if (c.IsBuiltIn && targetPluginID == "builtin") || (!c.IsBuiltIn && c.PluginID == targetPluginID) {
				continue
			}
		} else {
			if c.Provider.ID() == id {
				continue
			}
		}
		newList = append(newList, c)
	}

	if len(newList) == 0 {
		delete(r.candidates, baseID)
		r.clearRouting(baseID)
	} else {
		r.candidates[baseID] = newList
		r.recomputeWinner(baseID)
	}
}

// UnregisterPlugin removes all provider candidates associated with a pluginID.
func (r *Registry) UnregisterPlugin(pluginID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for baseID, list := range r.candidates {
		newList := make([]Candidate, 0, len(list))
		for _, c := range list {
			if !c.IsBuiltIn && c.PluginID == pluginID {
				continue
			}
			newList = append(newList, c)
		}
		if len(newList) == 0 {
			delete(r.candidates, baseID)
			r.clearRouting(baseID)
		} else {
			r.candidates[baseID] = newList
			r.recomputeWinner(baseID)
		}
	}
}

func (r *Registry) clearRouting(baseID string) {
	delete(r.provs, baseID)
	delete(r.metadata, baseID)
	delete(r.content, baseID)
	delete(r.tracking, baseID)

	// Clear namespaced keys
	prefix := baseID + "@"
	for k := range r.provs {
		if strings.HasPrefix(k, prefix) {
			delete(r.provs, k)
			delete(r.metadata, k)
			delete(r.content, k)
			delete(r.tracking, k)
		}
	}
}

// recomputeWinner must be called while holding r.mu.Lock().
// Collision resolution priority:
// 1. In-Process Built-in (highest tier default)
// 2. User Explicit Preference (if configured)
// 3. Highest SemVer Version
func (r *Registry) recomputeWinner(baseID string) {
	list := r.candidates[baseID]
	r.clearRouting(baseID)
	if len(list) == 0 {
		return
	}

	var winner Candidate
	pref := r.userPrefs[baseID]

	// 1. In-process Built-in check
	var builtinCand *Candidate
	for i := range list {
		if list[i].IsBuiltIn {
			builtinCand = &list[i]
			break
		}
	}

	// 2. User Explicit Preference check
	var preferredCand *Candidate
	if pref != "" {
		for i := range list {
			if (pref == "builtin" && list[i].IsBuiltIn) || (!list[i].IsBuiltIn && list[i].PluginID == pref) {
				preferredCand = &list[i]
				break
			}
		}
	}

	if builtinCand != nil && (pref == "" || pref == "builtin") {
		winner = *builtinCand
	} else if preferredCand != nil {
		winner = *preferredCand
	} else if builtinCand != nil {
		winner = *builtinCand
	} else {
		// 3. Highest SemVer Version among non-builtin candidates
		winner = list[0]
		for _, c := range list[1:] {
			if compareVersionStrings(c.Version, winner.Version) > 0 {
				winner = c
			}
		}
	}

	// Register winner under base ID
	r.indexProvider(baseID, winner.Provider)

	// Register all candidates under namespaced handles
	for _, c := range list {
		var handle string
		if c.IsBuiltIn {
			handle = baseID + "@builtin"
		} else if c.PluginID != "" {
			handle = baseID + "@" + c.PluginID
		}
		if handle != "" {
			r.indexProvider(handle, c.Provider)
		}
	}
}

func (r *Registry) indexProvider(handle string, p sdk.Provider) {
	r.provs[handle] = p

	if m, ok := p.(sdk.Metadata); ok {
		r.metadata[handle] = m
	} else {
		delete(r.metadata, handle)
	}

	if c, ok := p.(sdk.Content); ok {
		r.content[handle] = c
	} else {
		delete(r.content, handle)
	}

	if t, ok := p.(sdk.Tracking); ok {
		r.tracking[handle] = t
	} else {
		delete(r.tracking, handle)
	}
}

// Get returns the sdk.Provider for the given ID (supports primary and namespaced handles).
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

// ListContent returns a slice of all registered primary sdk.Content providers.
func (r *Registry) ListContent() []sdk.Content {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]sdk.Content, 0, len(r.content))
	for k, c := range r.content {
		if !strings.Contains(k, "@") {
			list = append(list, c)
		}
	}
	return list
}

// ListMetadata returns a slice of all registered primary sdk.Metadata providers.
func (r *Registry) ListMetadata() []sdk.Metadata {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]sdk.Metadata, 0, len(r.metadata))
	for k, m := range r.metadata {
		if !strings.Contains(k, "@") {
			list = append(list, m)
		}
	}
	return list
}

// ListInfo returns a slice of sdk.ProviderInfo for all registered primary providers.
func (r *Registry) ListInfo() []sdk.ProviderInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]sdk.ProviderInfo, 0, len(r.provs))
	for k, p := range r.provs {
		if !strings.Contains(k, "@") {
			list = append(list, sdk.ProviderInfo{
				ID:           p.ID(),
				Name:         p.Name(),
				Capabilities: p.Capabilities(),
				State:        p.State(),
				RequiresAuth: p.RequiresAuth(),
				ConfigKeys:   p.ConfigKeys(),
			})
		}
	}
	return list
}

// ListAllInfo returns a slice of sdk.ProviderInfo for all registered handles including namespaced ones.
func (r *Registry) ListAllInfo() []sdk.ProviderInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]sdk.ProviderInfo, 0, len(r.provs))
	for k, p := range r.provs {
		list = append(list, sdk.ProviderInfo{
			ID:           k,
			Name:         p.Name(),
			Capabilities: p.Capabilities(),
			State:        p.State(),
			RequiresAuth: p.RequiresAuth(),
			ConfigKeys:   p.ConfigKeys(),
		})
	}
	return list
}

// Candidates returns a map of all registered candidates keyed by base provider ID.
func (r *Registry) Candidates() map[string][]Candidate {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string][]Candidate, len(r.candidates))
	for k, v := range r.candidates {
		cp := make([]Candidate, len(v))
		copy(cp, v)
		result[k] = cp
	}
	return result
}

// compareVersionStrings provides internal SemVer comparison without external dependencies.
func compareVersionStrings(v1, v2 string) int {
	clean := func(v string) (int, int, int) {
		s := strings.TrimPrefix(strings.TrimSpace(v), "v")
		if dash := strings.Index(s, "-"); dash >= 0 {
			s = s[:dash]
		}
		parts := strings.Split(s, ".")
		var maj, min, pat int
		if len(parts) > 0 {
			maj, _ = strconv.Atoi(parts[0])
		}
		if len(parts) > 1 {
			min, _ = strconv.Atoi(parts[1])
		}
		if len(parts) > 2 {
			pat, _ = strconv.Atoi(parts[2])
		}
		return maj, min, pat
	}

	maj1, min1, pat1 := clean(v1)
	maj2, min2, pat2 := clean(v2)

	if maj1 != maj2 {
		if maj1 > maj2 {
			return 1
		}
		return -1
	}
	if min1 != min2 {
		if min1 > min2 {
			return 1
		}
		return -1
	}
	if pat1 != pat2 {
		if pat1 > pat2 {
			return 1
		}
		return -1
	}
	return 0
}
