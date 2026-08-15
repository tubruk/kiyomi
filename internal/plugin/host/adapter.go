package host

import (
	"context"
	"fmt"
	"io"
	"sync/atomic"
	"time"

	provsdk "github.com/tubruk/kiyomi/pkg/provider/sdk"
	sdk "github.com/tubruk/kiyomi/plugin-sdk"
	v1 "github.com/tubruk/kiyomi/plugin-sdk/proto/v1"
)

// GRPCProviderAdapter adapts gRPC client services to both pkg/provider/sdk and plugin-sdk interfaces.
type GRPCProviderAdapter struct {
	providerID   string
	name         string
	icon         string
	pluginID     string
	version      string
	isBuiltIn    bool
	capabilities []string
	configKeys   []provsdk.ConfigKeySpec
	rateLimit    provsdk.RateLimitHint

	metaClient    v1.MetadataProviderServiceClient
	contentClient v1.ContentProviderServiceClient
	trackClient   v1.TrackerServiceClient

	activeCalls *int64
}

// NewGRPCProviderAdapter creates a new host provider adapter for an out-of-process gRPC plugin provider.
func NewGRPCProviderAdapter(
	desc sdk.ProviderDescriptor,
	pluginID string,
	version string,
	metaClient v1.MetadataProviderServiceClient,
	contentClient v1.ContentProviderServiceClient,
	trackClient v1.TrackerServiceClient,
	activeCalls *int64,
) *GRPCProviderAdapter {
	if activeCalls == nil {
		var zero int64
		activeCalls = &zero
	}

	configKeys := make([]provsdk.ConfigKeySpec, len(desc.SettingsSchema))
	for i, s := range desc.SettingsSchema {
		configKeys[i] = provsdk.ConfigKeySpec{
			Key:         s.Key,
			Type:        s.Type,
			Description: s.Description,
			Examples:    s.Options,
		}
	}

	rps := float64(desc.DefaultRateLimit.RequestsPerSecond)
	if rps <= 0 {
		rps = 5.0
	}

	return &GRPCProviderAdapter{
		providerID:    desc.ID,
		name:          desc.Name,
		icon:          "icon.png",
		pluginID:      pluginID,
		version:       version,
		isBuiltIn:     false,
		capabilities:  desc.Capabilities,
		configKeys:    configKeys,
		rateLimit:     provsdk.RateLimitHint{RequestsPerSecond: rps, RequestsPerMinute: rps * 60},
		metaClient:    metaClient,
		contentClient: contentClient,
		trackClient:   trackClient,
		activeCalls:   activeCalls,
	}
}

func (a *GRPCProviderAdapter) startCall() func() {
	if a.activeCalls != nil {
		atomic.AddInt64(a.activeCalls, 1)
		return func() {
			atomic.AddInt64(a.activeCalls, -1)
		}
	}
	return func() {}
}

func (a *GRPCProviderAdapter) hasCapability(cap string) bool {
	for _, c := range a.capabilities {
		if c == cap {
			return true
		}
	}
	return false
}

// --- Provider Core Interface (pkg/provider/sdk.Provider) ---

func (a *GRPCProviderAdapter) ID() string {
	return a.providerID
}

func (a *GRPCProviderAdapter) Name() string {
	return a.name
}

func (a *GRPCProviderAdapter) Icon() string {
	return a.icon
}

func (a *GRPCProviderAdapter) Capabilities() []string {
	return a.capabilities
}

func (a *GRPCProviderAdapter) ConfigKeys() []provsdk.ConfigKeySpec {
	return a.configKeys
}

func (a *GRPCProviderAdapter) RequiresAuth() bool {
	return a.hasCapability("tracking")
}

func (a *GRPCProviderAdapter) State() provsdk.ProviderState {
	return provsdk.StateActive
}

// Extra metadata accessors
func (a *GRPCProviderAdapter) PluginID() string { return a.pluginID }
func (a *GRPCProviderAdapter) Version() string  { return a.version }
func (a *GRPCProviderAdapter) IsBuiltIn() bool  { return a.isBuiltIn }

// --- Metadata Provider Methods (pkg/provider/sdk.Metadata & plugin-sdk.MetadataProvider) ---

func (a *GRPCProviderAdapter) Search(ctx context.Context, query string, opts provsdk.SearchOptions) ([]provsdk.SearchResult, error) {
	if a.metaClient == nil {
		return nil, fmt.Errorf("provider %q does not support metadata search", a.providerID)
	}
	done := a.startCall()
	defer done()

	resp, err := a.metaClient.Search(ctx, &v1.SearchRequest{
		ProviderId: a.providerID,
		Query:      query,
		Limit:      int32(opts.Limit),
		Offset:     int32(opts.Offset),
		Mode:       opts.Mode,
	})
	if err != nil {
		return nil, err
	}

	protoResults := resp.GetResults()
	results := make([]provsdk.SearchResult, len(protoResults))
	for i, r := range protoResults {
		results[i] = provsdk.SearchResult{
			RemoteID:     r.GetRemoteId(),
			Title:        r.GetTitle(),
			Aliases:      r.GetAliases(),
			CoverURL:     r.GetCoverUrl(),
			URL:          r.GetUrl(),
			Availability: provsdk.ContentAvailability(r.GetAvailability()),
		}
	}

	return results, nil
}

func (a *GRPCProviderAdapter) Details(ctx context.Context, remoteID string) (provsdk.MangaMetadata, error) {
	if a.metaClient == nil {
		return provsdk.MangaMetadata{}, fmt.Errorf("provider %q does not support metadata details", a.providerID)
	}
	done := a.startCall()
	defer done()

	resp, err := a.metaClient.Details(ctx, &v1.DetailsRequest{
		ProviderId: a.providerID,
		RemoteId:   remoteID,
	})
	if err != nil {
		return provsdk.MangaMetadata{}, err
	}

	m := resp.GetMetadata()
	if m == nil {
		return provsdk.MangaMetadata{}, fmt.Errorf("empty metadata returned from provider %q", a.providerID)
	}

	return provsdk.MangaMetadata{
		RemoteID:      m.GetRemoteId(),
		Title:         m.GetTitle(),
		Aliases:       m.GetAliases(),
		CoverURL:      m.GetCoverUrl(),
		Synopsis:      m.GetSynopsis(),
		Status:        m.GetStatus(),
		Author:        m.GetAuthor(),
		Artist:        m.GetArtist(),
		Genres:        m.GetGenres(),
		TotalChapters: int(m.GetTotalChapters()),
		ReadingMode:   provsdk.ReadingMode(m.GetReadingMode()),
		Score:         m.GetScore(),
		URL:           m.GetUrl(),
		Availability:  provsdk.ContentAvailability(m.GetAvailability()),
	}, nil
}

func (a *GRPCProviderAdapter) Cover(ctx context.Context, remoteID string, size provsdk.ImageSize) (provsdk.ImageRef, error) {
	if a.metaClient == nil {
		return provsdk.ImageRef{}, fmt.Errorf("provider %q does not support cover lookup", a.providerID)
	}
	done := a.startCall()
	defer done()

	resp, err := a.metaClient.Cover(ctx, &v1.CoverRequest{
		ProviderId: a.providerID,
		RemoteId:   remoteID,
		Size:       string(size),
	})
	if err != nil {
		return provsdk.ImageRef{}, err
	}

	cov := resp.GetCover()
	if cov == nil {
		return provsdk.ImageRef{}, fmt.Errorf("empty cover returned from provider %q", a.providerID)
	}

	return provsdk.ImageRef{
		URL:      cov.GetUrl(),
		Width:    int(cov.GetWidth()),
		Height:   int(cov.GetHeight()),
		MIMEType: cov.GetMimeType(),
	}, nil
}

func (a *GRPCProviderAdapter) Aliases(ctx context.Context, remoteID string) ([]string, error) {
	if a.metaClient == nil {
		return nil, fmt.Errorf("provider %q does not support aliases", a.providerID)
	}
	done := a.startCall()
	defer done()

	resp, err := a.metaClient.Aliases(ctx, &v1.AliasesRequest{
		ProviderId: a.providerID,
		RemoteId:   remoteID,
	})
	if err != nil {
		return nil, err
	}
	return resp.GetAliases(), nil
}

// --- Content Provider Methods (pkg/provider/sdk.Content & plugin-sdk.ContentProvider) ---

func (a *GRPCProviderAdapter) HasStableChapterID() bool {
	if a.contentClient == nil {
		return false
	}
	done := a.startCall()
	defer done()

	resp, err := a.contentClient.HasStableChapterID(context.Background(), &v1.HasStableChapterIDRequest{
		ProviderId: a.providerID,
	})
	if err != nil {
		return false
	}
	return resp.GetHasStableChapterId()
}

func (a *GRPCProviderAdapter) FetchChapters(ctx context.Context, mangaRef string) ([]provsdk.Chapter, error) {
	if a.contentClient == nil {
		return nil, fmt.Errorf("provider %q does not support chapters", a.providerID)
	}
	done := a.startCall()
	defer done()

	resp, err := a.contentClient.FetchChapters(ctx, &v1.FetchChaptersRequest{
		ProviderId: a.providerID,
		MangaRef:   mangaRef,
	})
	if err != nil {
		return nil, err
	}

	protoChapters := resp.GetChapters()
	chapters := make([]provsdk.Chapter, len(protoChapters))
	for i, ch := range protoChapters {
		chapters[i] = provsdk.Chapter{
			ID:          ch.GetId(),
			Name:        ch.GetName(),
			URL:         ch.GetUrl(),
			Number:      ch.GetNumber(),
			UploadDate:  time.Unix(ch.GetUploadDateUnix(), 0),
			SourceOrder: int(ch.GetSourceOrder()),
		}
	}

	return chapters, nil
}

func (a *GRPCProviderAdapter) FetchPages(ctx context.Context, mangaRef, chapterRef string) ([]provsdk.Page, error) {
	if a.contentClient == nil {
		return nil, fmt.Errorf("provider %q does not support pages", a.providerID)
	}
	done := a.startCall()
	defer done()

	resp, err := a.contentClient.FetchPages(ctx, &v1.FetchPagesRequest{
		ProviderId: a.providerID,
		MangaRef:   mangaRef,
		ChapterRef: chapterRef,
	})
	if err != nil {
		return nil, err
	}

	protoPages := resp.GetPages()
	pages := make([]provsdk.Page, len(protoPages))
	for i, pg := range protoPages {
		pages[i] = provsdk.Page{
			Index: int(pg.GetIndex()),
			URL:   pg.GetUrl(),
		}
	}

	return pages, nil
}

func (a *GRPCProviderAdapter) FetchPageStream(ctx context.Context, page provsdk.Page) (io.ReadCloser, error) {
	if a.contentClient == nil {
		return nil, fmt.Errorf("provider %q does not support page streaming", a.providerID)
	}
	done := a.startCall()

	streamCtx, cancel := context.WithCancel(ctx)
	stream, err := a.contentClient.FetchPageStream(streamCtx, &v1.FetchPageStreamRequest{
		ProviderId: a.providerID,
		Page: &v1.Page{
			Index: int32(page.Index),
			Url:   page.URL,
		},
	})
	if err != nil {
		done()
		cancel()
		return nil, err
	}

	return &adapterStreamReader{
		stream: stream,
		cancel: cancel,
		done:   done,
	}, nil
}

func (a *GRPCProviderAdapter) RateLimit() provsdk.RateLimitHint {
	if a.contentClient == nil {
		return a.rateLimit
	}
	done := a.startCall()
	defer done()

	resp, err := a.contentClient.RateLimit(context.Background(), &v1.RateLimitRequest{
		ProviderId: a.providerID,
	})
	if err != nil || resp.GetRateLimit() == nil {
		return a.rateLimit
	}

	return provsdk.RateLimitHint{
		RequestsPerSecond: resp.GetRateLimit().GetRequestsPerSecond(),
		RequestsPerMinute: resp.GetRateLimit().GetRequestsPerMinute(),
	}
}

// --- Tracking Provider Methods (pkg/provider/sdk.Tracking & plugin-sdk.Tracker) ---

func (a *GRPCProviderAdapter) Authenticate(ctx context.Context, creds provsdk.UserCredentials) (provsdk.Session, error) {
	if a.trackClient == nil {
		return provsdk.Session{}, fmt.Errorf("provider %q does not support tracking", a.providerID)
	}
	done := a.startCall()
	defer done()

	resp, err := a.trackClient.Authenticate(ctx, &v1.AuthenticateRequest{
		ProviderId: a.providerID,
		Credentials: &v1.UserCredentials{
			AccessToken:   creds.AccessToken,
			RefreshToken:  creds.RefreshToken,
			ExpiresAtUnix: creds.ExpiresAt,
		},
	})
	if err != nil {
		return provsdk.Session{}, err
	}

	sess := resp.GetSession()
	if sess == nil {
		return provsdk.Session{}, fmt.Errorf("empty session returned from provider %q", a.providerID)
	}

	return provsdk.Session{
		AccessToken:  sess.GetAccessToken(),
		RefreshToken: sess.GetRefreshToken(),
		ExpiresAt:    time.Unix(sess.GetExpiresAtUnix(), 0),
		UserID:       sess.GetUserId(),
	}, nil
}

func (a *GRPCProviderAdapter) PushProgress(ctx context.Context, remoteID string, n int) error {
	if a.trackClient == nil {
		return fmt.Errorf("provider %q does not support tracking", a.providerID)
	}
	done := a.startCall()
	defer done()

	_, err := a.trackClient.PushProgress(ctx, &v1.PushProgressRequest{
		ProviderId: a.providerID,
		RemoteId:   remoteID,
		Progress:   int32(n),
	})
	return err
}

func (a *GRPCProviderAdapter) FetchProgress(ctx context.Context, remoteID string) (provsdk.Progress, error) {
	if a.trackClient == nil {
		return provsdk.Progress{}, fmt.Errorf("provider %q does not support tracking", a.providerID)
	}
	done := a.startCall()
	defer done()

	resp, err := a.trackClient.FetchProgress(ctx, &v1.FetchProgressRequest{
		ProviderId: a.providerID,
		RemoteId:   remoteID,
	})
	if err != nil {
		return provsdk.Progress{}, err
	}

	p := resp.GetProgress()
	if p == nil {
		return provsdk.Progress{}, fmt.Errorf("empty progress returned from provider %q", a.providerID)
	}

	return provsdk.Progress{
		Status:    p.GetStatus(),
		Score:     int(p.GetScore()),
		Progress:  int(p.GetProgress()),
		UpdatedAt: p.GetUpdatedAtUnix(),
	}, nil
}

func (a *GRPCProviderAdapter) IsAuthenticated() bool {
	if a.trackClient == nil {
		return false
	}
	done := a.startCall()
	defer done()

	resp, err := a.trackClient.IsAuthenticated(context.Background(), &v1.IsAuthenticatedRequest{
		ProviderId: a.providerID,
	})
	if err != nil {
		return false
	}
	return resp.GetIsAuthenticated()
}

type adapterStreamReader struct {
	stream v1.ContentProviderService_FetchPageStreamClient
	cancel context.CancelFunc
	done   func()
	buf    []byte
	closed bool
}

func (r *adapterStreamReader) Read(p []byte) (n int, err error) {
	if len(r.buf) > 0 {
		n = copy(p, r.buf)
		r.buf = r.buf[n:]
		return n, nil
	}

	resp, err := r.stream.Recv()
	if err != nil {
		return 0, err
	}

	chunk := resp.GetChunk()
	n = copy(p, chunk)
	if n < len(chunk) {
		r.buf = chunk[n:]
	}
	return n, nil
}

func (r *adapterStreamReader) Close() error {
	if !r.closed {
		r.closed = true
		if r.cancel != nil {
			r.cancel()
		}
		if r.done != nil {
			r.done()
		}
	}
	return nil
}
