package sdk

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	v1 "github.com/tubruk/kiyomi/plugin-sdk/proto/v1"
)

// PluginGRPCPlugin is the HashiCorp go-plugin implementation for PluginService.
type PluginGRPCPlugin struct {
	plugin.Plugin
	Impl Plugin
}

func (p *PluginGRPCPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	v1.RegisterPluginServiceServer(s, &GRPCPluginServer{Impl: p.Impl})
	return nil
}

func (p *PluginGRPCPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &GRPCPluginClient{client: v1.NewPluginServiceClient(c)}, nil
}

// GRPCPluginServer dispatches gRPC requests to a local Plugin implementation.
type GRPCPluginServer struct {
	v1.UnimplementedPluginServiceServer
	Impl Plugin
}

func (s *GRPCPluginServer) Describe(ctx context.Context, req *v1.DescribeRequest) (*v1.DescribeResponse, error) {
	if s.Impl == nil {
		return nil, fmt.Errorf("plugin implementation is nil")
	}

	desc, err := s.Impl.Describe(ctx)
	if err != nil {
		return nil, err
	}

	sdkVer := desc.SDKVersion
	if sdkVer == "" {
		sdkVer = Version
	}

	resp := &v1.DescribeResponse{
		SdkVersion:           sdkVer,
		PluginId:             desc.PluginID,
		PluginName:           desc.PluginName,
		PluginVersion:        desc.PluginVersion,
		PluginSettingsSchema: toProtoSettingSpecs(desc.PluginSettingsSchema),
		Providers:            make([]*v1.ProviderDescriptor, len(desc.Providers)),
	}

	for i, prov := range desc.Providers {
		resp.Providers[i] = &v1.ProviderDescriptor{
			Id:             prov.ID,
			Name:           prov.Name,
			Description:    prov.Description,
			Capabilities:   prov.Capabilities,
			SettingsSchema: toProtoSettingSpecs(prov.SettingsSchema),
			DefaultRateLimit: &v1.RateLimitSpec{
				RequestsPerSecond:     prov.DefaultRateLimit.RequestsPerSecond,
				MaxConcurrentRequests: prov.DefaultRateLimit.MaxConcurrentRequests,
			},
		}
	}

	return resp, nil
}

func (s *GRPCPluginServer) Init(ctx context.Context, req *v1.InitRequest) (*v1.InitResponse, error) {
	if s.Impl == nil {
		return &v1.InitResponse{Success: false, ErrorMessage: "plugin implementation is nil"}, nil
	}

	cfg := PluginConfig{
		GlobalConfig:    req.GetGlobalConfig(),
		ProviderConfigs: make(map[string]map[string]string),
	}

	for pID, pCfg := range req.GetProviderConfigs() {
		if pCfg != nil {
			cfg.ProviderConfigs[pID] = pCfg.GetSettings()
		}
	}

	if h := req.GetHttpConfig(); h != nil {
		cfg.HTTPConfig = GlobalHttpConfig{
			ProxyURL:       h.GetProxyUrl(),
			UserAgent:      h.GetUserAgent(),
			TimeoutSeconds: int(h.GetTimeoutSeconds()),
			DNSResolvers:   h.GetDnsResolvers(),
		}
	}

	if err := s.Impl.Init(ctx, cfg); err != nil {
		return &v1.InitResponse{Success: false, ErrorMessage: err.Error()}, nil
	}

	return &v1.InitResponse{Success: true}, nil
}

// GRPCPluginClient wraps a gRPC client to satisfy the Plugin interface.
type GRPCPluginClient struct {
	client v1.PluginServiceClient
}

// NewPluginClient creates a new Plugin backed by a gRPC PluginServiceClient.
func NewPluginClient(client v1.PluginServiceClient) Plugin {
	return &GRPCPluginClient{client: client}
}

func (c *GRPCPluginClient) Describe(ctx context.Context) (PluginDescriptor, error) {
	resp, err := c.client.Describe(ctx, &v1.DescribeRequest{})
	if err != nil {
		return PluginDescriptor{}, err
	}

	desc := PluginDescriptor{
		PluginID:             resp.GetPluginId(),
		PluginName:           resp.GetPluginName(),
		PluginVersion:        resp.GetPluginVersion(),
		SDKVersion:           resp.GetSdkVersion(),
		PluginSettingsSchema: fromProtoSettingSpecs(resp.GetPluginSettingsSchema()),
		Providers:            make([]ProviderDescriptor, len(resp.GetProviders())),
	}

	for i, p := range resp.GetProviders() {
		desc.Providers[i] = ProviderDescriptor{
			ID:             p.GetId(),
			Name:           p.GetName(),
			Description:    p.GetDescription(),
			Capabilities:   p.GetCapabilities(),
			SettingsSchema: fromProtoSettingSpecs(p.GetSettingsSchema()),
			DefaultRateLimit: RateLimitSpec{
				RequestsPerSecond:     p.GetDefaultRateLimit().GetRequestsPerSecond(),
				MaxConcurrentRequests: p.GetDefaultRateLimit().GetMaxConcurrentRequests(),
			},
		}
	}

	return desc, nil
}

func (c *GRPCPluginClient) Init(ctx context.Context, config PluginConfig) error {
	req := &v1.InitRequest{
		GlobalConfig:    config.GlobalConfig,
		ProviderConfigs: make(map[string]*v1.ProviderConfigMap),
		HttpConfig: &v1.GlobalHttpConfig{
			ProxyUrl:       config.HTTPConfig.ProxyURL,
			UserAgent:      config.HTTPConfig.UserAgent,
			TimeoutSeconds: int32(config.HTTPConfig.TimeoutSeconds),
			DnsResolvers:   config.HTTPConfig.DNSResolvers,
		},
	}

	for pID, settings := range config.ProviderConfigs {
		req.ProviderConfigs[pID] = &v1.ProviderConfigMap{
			Settings: settings,
		}
	}

	resp, err := c.client.Init(ctx, req)
	if err != nil {
		return err
	}
	if !resp.GetSuccess() {
		return fmt.Errorf("plugin init failed: %s", resp.GetErrorMessage())
	}
	return nil
}

func toProtoSettingSpecs(specs []SettingSpec) []*v1.SettingSpec {
	if specs == nil {
		return nil
	}
	out := make([]*v1.SettingSpec, len(specs))
	for i, s := range specs {
		out[i] = &v1.SettingSpec{
			Key:          s.Key,
			Label:        s.Label,
			Description:  s.Description,
			Type:         s.Type,
			DefaultValue: s.DefaultValue,
			Options:      s.Options,
		}
	}
	return out
}

func fromProtoSettingSpecs(specs []*v1.SettingSpec) []SettingSpec {
	if specs == nil {
		return nil
	}
	out := make([]SettingSpec, len(specs))
	for i, s := range specs {
		out[i] = SettingSpec{
			Key:          s.GetKey(),
			Label:        s.GetLabel(),
			Description:  s.GetDescription(),
			Type:         s.GetType(),
			DefaultValue: s.GetDefaultValue(),
			Options:      s.GetOptions(),
		}
	}
	return out
}
