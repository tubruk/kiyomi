package sdk

// ProviderState represents the lifecycle state of a provider.
type ProviderState int

const (
	StateRegistered ProviderState = iota
	StateActive
	StateDisabled
)

func (s ProviderState) String() string {
	switch s {
	case StateRegistered:
		return "registered"
	case StateActive:
		return "active"
	case StateDisabled:
		return "disabled"
	default:
		return "unknown"
	}
}

// ProviderInfo exposes provider metadata for registry queries.
type ProviderInfo struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Capabilities []string       `json:"capabilities"`
	State        ProviderState  `json:"state"`
	RequiresAuth bool           `json:"requiresAuth"`
	ConfigKeys   []ConfigKeySpec `json:"configKeys,omitempty"`
}
