package mangadex

import (
	"context"
	"net/http"

	"github.com/tubruk/kiyomi/pkg/fingerprint"
	"github.com/tubruk/kiyomi/pkg/provider/sdk"
)

const (
	ProviderID = "mangadex"
	BaseURL    = "https://api.mangadex.org"
	UserAgent  = "Kiyomi/1.0.0 (https://github.com/tubruk/kiyomi)"
)

// Provider implements sdk.Content and sdk.Metadata for MangaDex.
type Provider struct {
	*sdk.HttpSource
}

// NewProvider creates a new MangaDex provider instance.
func NewProvider(client *http.Client, fpStore ...fingerprint.Store) *Provider {
	cfg := sdk.ProviderConfig{
		ID:        ProviderID,
		Name:      "MangaDex",
		BaseURL:   BaseURL,
		UserAgent: UserAgent,
	}

	source, err := sdk.NewHttpSource(cfg)
	if err != nil {
		panic(err)
	}

	if client != nil {
		source.Client = client
	}

	if len(fpStore) > 0 && fpStore[0] != nil {
		source.WithFingerprintStore(fpStore[0])
	}

	return &Provider{HttpSource: source}
}

func (p *Provider) ID() string {
	return ProviderID
}

func (p *Provider) Name() string {
	return "MangaDex"
}

func (p *Provider) Icon() string {
	return "https://mangadex.org/favicon.ico"
}

func (p *Provider) Capabilities() []string {
	return []string{"content", "metadata"}
}

func (p *Provider) ConfigKeys() []sdk.ConfigKeySpec {
	return nil
}

func (p *Provider) RequiresAuth() bool {
	return false
}

func (p *Provider) State() sdk.ProviderState {
	return sdk.StateActive
}

func (p *Provider) baseURL() string {
	if p.Config.BaseURL != "" {
		return p.Config.BaseURL
	}
	return BaseURL
}

func (p *Provider) newRequest(ctx context.Context, method, endpoint string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", "application/json")
	return req, nil
}
