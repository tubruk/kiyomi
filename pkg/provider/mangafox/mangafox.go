package mangafox

import (
	"net/http"

	"github.com/tubruk/kiyomi/pkg/fingerprint"
	"github.com/tubruk/kiyomi/pkg/provider/sdk"
)

const (
	ProviderID   = "mangafox"
	ProviderName = "MangaFox"
	BaseURL      = "https://fanfox.net"
	Language     = "en"
)

// Provider implements sdk.Metadata and sdk.Content for MangaFox.
type Provider struct {
	*sdk.HttpSource
}

var (
	_ sdk.Metadata           = (*Provider)(nil)
	_ sdk.Content            = (*Provider)(nil)
	_ sdk.ConcurrencyLimiter = (*Provider)(nil)
)

// NewProvider creates a new MangaFox provider instance.
func NewProvider(client *http.Client, fpStore ...fingerprint.Store) (*Provider, error) {
	cfg := sdk.ProviderConfig{
		ID:       ProviderID,
		Name:     ProviderName,
		BaseURL:  BaseURL,
		Language: Language,
		Cookies: map[string]string{
			"https://fanfox.net":   "isAdult=1",
			"https://m.fanfox.net": "readway=2",
		},
	}

	source, err := sdk.NewHttpSource(cfg)
	if err != nil {
		return nil, err
	}

	if client != nil {
		source.Client = client
	}

	if len(fpStore) > 0 && fpStore[0] != nil {
		source.WithFingerprintStore(fpStore[0])
	}

	return &Provider{HttpSource: source}, nil
}

func (p *Provider) ID() string   { return ProviderID }
func (p *Provider) Name() string { return ProviderName }

func (p *Provider) Icon() string {
	return "https://fanfox.net/favicon.ico"
}

func (p *Provider) Capabilities() []string { return []string{"metadata", "content"} }

func (p *Provider) ConfigKeys() []sdk.ConfigKeySpec {
	return nil
}

func (p *Provider) RequiresAuth() bool { return false }

func (p *Provider) State() sdk.ProviderState {
	return sdk.StateActive
}

func (p *Provider) ConcurrencyLimit() int { return 1 }
