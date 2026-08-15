package main

import (
	"context"
	"net/http"
	"strings"
)

func (p *MangaDexPlugin) getBaseURL() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.baseURL != "" {
		return p.baseURL
	}
	return DefaultBase
}

// SetBaseURL overrides the upstream MangaDex API endpoint base URL (useful for testing).
func (p *MangaDexPlugin) SetBaseURL(url string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.baseURL = strings.TrimRight(url, "/")
}

func (p *MangaDexPlugin) newRequest(ctx context.Context, method, endpoint string) (*http.Request, error) {
	p.mu.RLock()
	ua := p.userAgent
	p.mu.RUnlock()

	req, err := http.NewRequestWithContext(ctx, method, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "application/json")
	return req, nil
}

func (p *MangaDexPlugin) doRequest(req *http.Request) (*http.Response, error) {
	p.mu.RLock()
	client := p.client
	p.mu.RUnlock()

	if client == nil {
		client = http.DefaultClient
	}
	return client.Do(req)
}
