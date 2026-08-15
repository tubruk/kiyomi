package sdk

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	v1 "github.com/tubruk/kiyomi/plugin-sdk/proto/v1"
)

// TrackerGRPCPlugin is the HashiCorp go-plugin implementation for TrackerService.
type TrackerGRPCPlugin struct {
	plugin.Plugin
	Providers map[string]Tracker
}

func (p *TrackerGRPCPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	v1.RegisterTrackerServiceServer(s, &GRPCTrackerServer{Providers: p.Providers})
	return nil
}

func (p *TrackerGRPCPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return v1.NewTrackerServiceClient(c), nil
}

// GRPCTrackerServer dispatches gRPC requests to the appropriate Tracker provider.
type GRPCTrackerServer struct {
	v1.UnimplementedTrackerServiceServer
	Providers map[string]Tracker
}

func (s *GRPCTrackerServer) getProvider(id string) (Tracker, error) {
	if s.Providers == nil {
		return nil, fmt.Errorf("no tracker providers registered")
	}
	p, ok := s.Providers[id]
	if !ok {
		return nil, fmt.Errorf("tracker provider %q not found", id)
	}
	return p, nil
}

func (s *GRPCTrackerServer) Authenticate(ctx context.Context, req *v1.AuthenticateRequest) (*v1.AuthenticateResponse, error) {
	p, err := s.getProvider(req.GetProviderId())
	if err != nil {
		return nil, err
	}

	creds := UserCredentials{
		AccessToken:  req.GetCredentials().GetAccessToken(),
		RefreshToken: req.GetCredentials().GetRefreshToken(),
		ExpiresAt:    req.GetCredentials().GetExpiresAtUnix(),
	}

	sess, err := p.Authenticate(ctx, creds)
	if err != nil {
		return nil, err
	}

	return &v1.AuthenticateResponse{
		Session: &v1.Session{
			AccessToken:   sess.AccessToken,
			RefreshToken:  sess.RefreshToken,
			ExpiresAtUnix: sess.ExpiresAt.Unix(),
			UserId:        sess.UserID,
		},
	}, nil
}

func (s *GRPCTrackerServer) PushProgress(ctx context.Context, req *v1.PushProgressRequest) (*v1.PushProgressResponse, error) {
	p, err := s.getProvider(req.GetProviderId())
	if err != nil {
		return nil, err
	}

	if err := p.PushProgress(ctx, req.GetRemoteId(), int(req.GetProgress())); err != nil {
		return nil, err
	}

	return &v1.PushProgressResponse{Success: true}, nil
}

func (s *GRPCTrackerServer) FetchProgress(ctx context.Context, req *v1.FetchProgressRequest) (*v1.FetchProgressResponse, error) {
	p, err := s.getProvider(req.GetProviderId())
	if err != nil {
		return nil, err
	}

	prog, err := p.FetchProgress(ctx, req.GetRemoteId())
	if err != nil {
		return nil, err
	}

	return &v1.FetchProgressResponse{
		Progress: &v1.Progress{
			Status:        prog.Status,
			Score:         int32(prog.Score),
			Progress:      int32(prog.Progress),
			UpdatedAtUnix: prog.UpdatedAt,
		},
	}, nil
}

func (s *GRPCTrackerServer) IsAuthenticated(ctx context.Context, req *v1.IsAuthenticatedRequest) (*v1.IsAuthenticatedResponse, error) {
	p, err := s.getProvider(req.GetProviderId())
	if err != nil {
		return nil, err
	}

	return &v1.IsAuthenticatedResponse{
		IsAuthenticated: p.IsAuthenticated(ctx),
	}, nil
}

// GRPCTrackerClient wraps a gRPC client stub to implement Tracker for a specific provider ID.
type GRPCTrackerClient struct {
	client     v1.TrackerServiceClient
	providerID string
}

// NewTrackerClient binds a gRPC client stub to a specific provider ID.
func NewTrackerClient(client v1.TrackerServiceClient, providerID string) Tracker {
	return &GRPCTrackerClient{
		client:     client,
		providerID: providerID,
	}
}

func (c *GRPCTrackerClient) Authenticate(ctx context.Context, creds UserCredentials) (Session, error) {
	resp, err := c.client.Authenticate(ctx, &v1.AuthenticateRequest{
		ProviderId: c.providerID,
		Credentials: &v1.UserCredentials{
			AccessToken:   creds.AccessToken,
			RefreshToken:  creds.RefreshToken,
			ExpiresAtUnix: creds.ExpiresAt,
		},
	})
	if err != nil {
		return Session{}, err
	}

	s := resp.GetSession()
	return Session{
		AccessToken:  s.GetAccessToken(),
		RefreshToken: s.GetRefreshToken(),
		ExpiresAt:    time.Unix(s.GetExpiresAtUnix(), 0),
		UserID:       s.GetUserId(),
	}, nil
}

func (c *GRPCTrackerClient) PushProgress(ctx context.Context, remoteID string, n int) error {
	_, err := c.client.PushProgress(ctx, &v1.PushProgressRequest{
		ProviderId: c.providerID,
		RemoteId:   remoteID,
		Progress:   int32(n),
	})
	return err
}

func (c *GRPCTrackerClient) FetchProgress(ctx context.Context, remoteID string) (Progress, error) {
	resp, err := c.client.FetchProgress(ctx, &v1.FetchProgressRequest{
		ProviderId: c.providerID,
		RemoteId:   remoteID,
	})
	if err != nil {
		return Progress{}, err
	}

	p := resp.GetProgress()
	return Progress{
		Status:    p.GetStatus(),
		Score:     int(p.GetScore()),
		Progress:  int(p.GetProgress()),
		UpdatedAt: p.GetUpdatedAtUnix(),
	}, nil
}

func (c *GRPCTrackerClient) IsAuthenticated(ctx context.Context) bool {
	resp, err := c.client.IsAuthenticated(ctx, &v1.IsAuthenticatedRequest{
		ProviderId: c.providerID,
	})
	if err != nil {
		return false
	}
	return resp.GetIsAuthenticated()
}
