package sdk

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	v1 "github.com/tubruk/kiyomi/plugin-sdk/proto/v1"
)

// MetadataProviderGRPCPlugin is the HashiCorp go-plugin implementation for MetadataProviderService.
type MetadataProviderGRPCPlugin struct {
	plugin.Plugin
	Providers map[string]MetadataProvider
}

func (p *MetadataProviderGRPCPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	v1.RegisterMetadataProviderServiceServer(s, &GRPCMetadataProviderServer{Providers: p.Providers})
	return nil
}

func (p *MetadataProviderGRPCPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return v1.NewMetadataProviderServiceClient(c), nil
}

// GRPCMetadataProviderServer dispatches gRPC requests to the appropriate MetadataProvider implementation.
type GRPCMetadataProviderServer struct {
	v1.UnimplementedMetadataProviderServiceServer
	Providers map[string]MetadataProvider
}

func (s *GRPCMetadataProviderServer) getProvider(id string) (MetadataProvider, error) {
	if s.Providers == nil {
		return nil, fmt.Errorf("no metadata providers registered")
	}
	p, ok := s.Providers[id]
	if !ok {
		return nil, fmt.Errorf("metadata provider %q not found", id)
	}
	return p, nil
}

func (s *GRPCMetadataProviderServer) Search(ctx context.Context, req *v1.SearchRequest) (*v1.SearchResponse, error) {
	p, err := s.getProvider(req.GetProviderId())
	if err != nil {
		return nil, err
	}

	opts := SearchOptions{
		Limit:  int(req.GetLimit()),
		Offset: int(req.GetOffset()),
		Mode:   req.GetMode(),
	}

	results, err := p.Search(ctx, req.GetQuery(), opts)
	if err != nil {
		return nil, err
	}

	resp := &v1.SearchResponse{
		Results: make([]*v1.SearchResult, len(results)),
	}

	for i, r := range results {
		resp.Results[i] = &v1.SearchResult{
			RemoteId:     r.RemoteID,
			Title:        r.Title,
			Aliases:      r.Aliases,
			CoverUrl:     r.CoverURL,
			Url:          r.URL,
			Availability: string(r.Availability),
		}
	}

	return resp, nil
}

func (s *GRPCMetadataProviderServer) Details(ctx context.Context, req *v1.DetailsRequest) (*v1.DetailsResponse, error) {
	p, err := s.getProvider(req.GetProviderId())
	if err != nil {
		return nil, err
	}

	m, err := p.Details(ctx, req.GetRemoteId())
	if err != nil {
		return nil, err
	}

	return &v1.DetailsResponse{
		Metadata: &v1.MangaMetadata{
			RemoteId:      m.RemoteID,
			Title:         m.Title,
			Aliases:       m.Aliases,
			CoverUrl:      m.CoverURL,
			Synopsis:      m.Synopsis,
			Status:        m.Status,
			Author:        m.Author,
			Artist:        m.Artist,
			Genres:        m.Genres,
			TotalChapters: int32(m.TotalChapters),
			ReadingMode:   string(m.ReadingMode),
			Score:         m.Score,
			Url:           m.URL,
			Availability:  string(m.Availability),
		},
	}, nil
}

func (s *GRPCMetadataProviderServer) Cover(ctx context.Context, req *v1.CoverRequest) (*v1.CoverResponse, error) {
	p, err := s.getProvider(req.GetProviderId())
	if err != nil {
		return nil, err
	}

	img, err := p.Cover(ctx, req.GetRemoteId(), ImageSize(req.GetSize()))
	if err != nil {
		return nil, err
	}

	return &v1.CoverResponse{
		Cover: &v1.ImageRef{
			Url:      img.URL,
			Width:    int32(img.Width),
			Height:   int32(img.Height),
			MimeType: img.MIMEType,
			Headers:  img.Headers,
		},
	}, nil
}

func (s *GRPCMetadataProviderServer) Aliases(ctx context.Context, req *v1.AliasesRequest) (*v1.AliasesResponse, error) {
	p, err := s.getProvider(req.GetProviderId())
	if err != nil {
		return nil, err
	}

	aliases, err := p.Aliases(ctx, req.GetRemoteId())
	if err != nil {
		return nil, err
	}

	return &v1.AliasesResponse{
		Aliases: aliases,
	}, nil
}

// GRPCMetadataProviderClient wraps a gRPC client to implement MetadataProvider for a specific provider ID.
type GRPCMetadataProviderClient struct {
	client     v1.MetadataProviderServiceClient
	providerID string
}

// NewMetadataProviderClient binds a gRPC client stub to a specific provider ID.
func NewMetadataProviderClient(client v1.MetadataProviderServiceClient, providerID string) MetadataProvider {
	return &GRPCMetadataProviderClient{
		client:     client,
		providerID: providerID,
	}
}

func (c *GRPCMetadataProviderClient) Search(ctx context.Context, query string, opts SearchOptions) ([]SearchResult, error) {
	resp, err := c.client.Search(ctx, &v1.SearchRequest{
		ProviderId: c.providerID,
		Query:      query,
		Limit:      int32(opts.Limit),
		Offset:     int32(opts.Offset),
		Mode:       opts.Mode,
	})
	if err != nil {
		return nil, err
	}

	protoResults := resp.GetResults()
	results := make([]SearchResult, len(protoResults))
	for i, r := range protoResults {
		results[i] = SearchResult{
			RemoteID:     r.GetRemoteId(),
			Title:        r.GetTitle(),
			Aliases:      r.GetAliases(),
			CoverURL:     r.GetCoverUrl(),
			URL:          r.GetUrl(),
			Availability: ContentAvailability(r.GetAvailability()),
		}
	}

	return results, nil
}

func (c *GRPCMetadataProviderClient) Details(ctx context.Context, remoteID string) (MangaMetadata, error) {
	resp, err := c.client.Details(ctx, &v1.DetailsRequest{
		ProviderId: c.providerID,
		RemoteId:   remoteID,
	})
	if err != nil {
		return MangaMetadata{}, err
	}

	m := resp.GetMetadata()
	return MangaMetadata{
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
		ReadingMode:   ReadingMode(m.GetReadingMode()),
		Score:         m.GetScore(),
		URL:           m.GetUrl(),
		Availability:  ContentAvailability(m.GetAvailability()),
	}, nil
}

func (c *GRPCMetadataProviderClient) Cover(ctx context.Context, remoteID string, size ImageSize) (ImageRef, error) {
	resp, err := c.client.Cover(ctx, &v1.CoverRequest{
		ProviderId: c.providerID,
		RemoteId:   remoteID,
		Size:       string(size),
	})
	if err != nil {
		return ImageRef{}, err
	}

	img := resp.GetCover()
	return ImageRef{
		URL:      img.GetUrl(),
		Width:    int(img.GetWidth()),
		Height:   int(img.GetHeight()),
		MIMEType: img.GetMimeType(),
		Headers:  img.GetHeaders(),
	}, nil
}

func (c *GRPCMetadataProviderClient) Aliases(ctx context.Context, remoteID string) ([]string, error) {
	resp, err := c.client.Aliases(ctx, &v1.AliasesRequest{
		ProviderId: c.providerID,
		RemoteId:   remoteID,
	})
	if err != nil {
		return nil, err
	}
	return resp.GetAliases(), nil
}
