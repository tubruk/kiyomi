package sdk

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	v1 "github.com/tubruk/kiyomi/plugin-sdk/proto/v1"
)

// ContentProviderGRPCPlugin is the HashiCorp go-plugin implementation for ContentProviderService.
type ContentProviderGRPCPlugin struct {
	plugin.Plugin
	Providers map[string]ContentProvider
}

func (p *ContentProviderGRPCPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	v1.RegisterContentProviderServiceServer(s, &GRPCContentProviderServer{Providers: p.Providers})
	return nil
}

func (p *ContentProviderGRPCPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return v1.NewContentProviderServiceClient(c), nil
}

// GRPCContentProviderServer dispatches gRPC requests to the appropriate ContentProvider.
type GRPCContentProviderServer struct {
	v1.UnimplementedContentProviderServiceServer
	Providers map[string]ContentProvider
}

func (s *GRPCContentProviderServer) getProvider(id string) (ContentProvider, error) {
	if s.Providers == nil {
		return nil, fmt.Errorf("no content providers registered")
	}
	p, ok := s.Providers[id]
	if !ok {
		return nil, fmt.Errorf("content provider %q not found", id)
	}
	return p, nil
}

func (s *GRPCContentProviderServer) HasStableChapterID(ctx context.Context, req *v1.HasStableChapterIDRequest) (*v1.HasStableChapterIDResponse, error) {
	p, err := s.getProvider(req.GetProviderId())
	if err != nil {
		return nil, err
	}
	return &v1.HasStableChapterIDResponse{
		HasStableChapterId: p.HasStableChapterID(),
	}, nil
}

func (s *GRPCContentProviderServer) FetchChapters(ctx context.Context, req *v1.FetchChaptersRequest) (*v1.FetchChaptersResponse, error) {
	p, err := s.getProvider(req.GetProviderId())
	if err != nil {
		return nil, err
	}

	chapters, err := p.FetchChapters(ctx, req.GetMangaRef())
	if err != nil {
		return nil, err
	}

	resp := &v1.FetchChaptersResponse{
		Chapters: make([]*v1.Chapter, len(chapters)),
	}

	for i, ch := range chapters {
		resp.Chapters[i] = &v1.Chapter{
			Id:             ch.ID,
			Name:           ch.Name,
			Url:            ch.URL,
			Number:         ch.Number,
			UploadDateUnix: ch.UploadDate.Unix(),
			SourceOrder:    int32(ch.SourceOrder),
		}
	}

	return resp, nil
}

func (s *GRPCContentProviderServer) FetchPages(ctx context.Context, req *v1.FetchPagesRequest) (*v1.FetchPagesResponse, error) {
	p, err := s.getProvider(req.GetProviderId())
	if err != nil {
		return nil, err
	}

	pages, err := p.FetchPages(ctx, req.GetMangaRef(), req.GetChapterRef())
	if err != nil {
		return nil, err
	}

	resp := &v1.FetchPagesResponse{
		Pages: make([]*v1.Page, len(pages)),
	}

	for i, pg := range pages {
		resp.Pages[i] = &v1.Page{
			Index:   int32(pg.Index),
			Url:     pg.URL,
			Headers: pg.Headers,
		}
	}

	return resp, nil
}

func (s *GRPCContentProviderServer) FetchPageStream(req *v1.FetchPageStreamRequest, stream v1.ContentProviderService_FetchPageStreamServer) error {
	p, err := s.getProvider(req.GetProviderId())
	if err != nil {
		return err
	}

	pgProto := req.GetPage()
	pg := Page{
		Index:   int(pgProto.GetIndex()),
		URL:     pgProto.GetUrl(),
		Headers: pgProto.GetHeaders(),
	}

	rc, err := p.FetchPageStream(stream.Context(), pg)
	if err != nil {
		return err
	}
	defer rc.Close()

	buf := make([]byte, 32*1024)
	for {
		n, readErr := rc.Read(buf)
		if n > 0 {
			if sendErr := stream.Send(&v1.FetchPageStreamResponse{Chunk: buf[:n]}); sendErr != nil {
				return sendErr
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				return nil
			}
			return readErr
		}
	}
}

func (s *GRPCContentProviderServer) RateLimit(ctx context.Context, req *v1.RateLimitRequest) (*v1.RateLimitResponse, error) {
	p, err := s.getProvider(req.GetProviderId())
	if err != nil {
		return nil, err
	}

	hint := p.RateLimit()
	return &v1.RateLimitResponse{
		RateLimit: &v1.RateLimitHint{
			RequestsPerSecond: hint.RequestsPerSecond,
			RequestsPerMinute: hint.RequestsPerMinute,
		},
	}, nil
}

// GRPCContentProviderClient wraps a gRPC client stub to implement ContentProvider for a specific provider ID.
type GRPCContentProviderClient struct {
	client     v1.ContentProviderServiceClient
	providerID string
}

// NewContentProviderClient binds a gRPC client stub to a specific provider ID.
func NewContentProviderClient(client v1.ContentProviderServiceClient, providerID string) ContentProvider {
	return &GRPCContentProviderClient{
		client:     client,
		providerID: providerID,
	}
}

func (c *GRPCContentProviderClient) HasStableChapterID() bool {
	resp, err := c.client.HasStableChapterID(context.Background(), &v1.HasStableChapterIDRequest{
		ProviderId: c.providerID,
	})
	if err != nil {
		return false
	}
	return resp.GetHasStableChapterId()
}

func (c *GRPCContentProviderClient) FetchChapters(ctx context.Context, mangaRef string) ([]Chapter, error) {
	resp, err := c.client.FetchChapters(ctx, &v1.FetchChaptersRequest{
		ProviderId: c.providerID,
		MangaRef:   mangaRef,
	})
	if err != nil {
		return nil, err
	}

	protoChapters := resp.GetChapters()
	chapters := make([]Chapter, len(protoChapters))
	for i, ch := range protoChapters {
		chapters[i] = Chapter{
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

func (c *GRPCContentProviderClient) FetchPages(ctx context.Context, mangaRef, chapterRef string) ([]Page, error) {
	resp, err := c.client.FetchPages(ctx, &v1.FetchPagesRequest{
		ProviderId: c.providerID,
		MangaRef:   mangaRef,
		ChapterRef: chapterRef,
	})
	if err != nil {
		return nil, err
	}

	protoPages := resp.GetPages()
	pages := make([]Page, len(protoPages))
	for i, pg := range protoPages {
		pages[i] = Page{
			Index:   int(pg.GetIndex()),
			URL:     pg.GetUrl(),
			Headers: pg.GetHeaders(),
		}
	}

	return pages, nil
}

func (c *GRPCContentProviderClient) FetchPageStream(ctx context.Context, page Page) (io.ReadCloser, error) {
	streamCtx, cancel := context.WithCancel(ctx)
	stream, err := c.client.FetchPageStream(streamCtx, &v1.FetchPageStreamRequest{
		ProviderId: c.providerID,
		Page: &v1.Page{
			Index:   int32(page.Index),
			Url:     page.URL,
			Headers: page.Headers,
		},
	})
	if err != nil {
		cancel()
		return nil, err
	}

	return &pageStreamReader{
		stream: stream,
		cancel: cancel,
	}, nil
}

func (c *GRPCContentProviderClient) RateLimit() RateLimitHint {
	resp, err := c.client.RateLimit(context.Background(), &v1.RateLimitRequest{
		ProviderId: c.providerID,
	})
	if err != nil {
		return RateLimitHint{}
	}
	r := resp.GetRateLimit()
	return RateLimitHint{
		RequestsPerSecond: r.GetRequestsPerSecond(),
		RequestsPerMinute: r.GetRequestsPerMinute(),
	}
}

type pageStreamReader struct {
	stream v1.ContentProviderService_FetchPageStreamClient
	cancel context.CancelFunc
	buf    []byte
}

func (r *pageStreamReader) Read(p []byte) (n int, err error) {
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

func (r *pageStreamReader) Close() error {
	if r.cancel != nil {
		r.cancel()
	}
	return nil
}
