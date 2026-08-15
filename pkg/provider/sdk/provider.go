package sdk

import (
	"sync"
)

// Provider is the top-level interface for an external platform (e.g. MangaFox, AniList, MyAnimeList).
type Provider interface {
	ID() string
	Name() string
	Icon() string
	Capabilities() []string // "metadata" | "content" | "tracking"
	ConfigKeys() []ConfigKeySpec
	RequiresAuth() bool
	State() ProviderState
}

// ConcurrencyLimiter is an optional interface for providers that limit concurrent requests.
type ConcurrencyLimiter interface {
	ConcurrencyLimit() int
}

// ProviderStateMachine manages lifecycle state transitions for providers.
type ProviderStateMachine struct {
	mu    sync.RWMutex
	state map[string]ProviderState
}

func NewProviderStateMachine() *ProviderStateMachine {
	return &ProviderStateMachine{
		state: make(map[string]ProviderState),
	}
}

// SetState transitions a provider to a new state.
func (m *ProviderStateMachine) SetState(id string, s ProviderState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state[id] = s
}

// GetState returns the current state of a provider.
func (m *ProviderStateMachine) GetState(id string) ProviderState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if s, ok := m.state[id]; ok {
		return s
	}
	return StateRegistered
}
