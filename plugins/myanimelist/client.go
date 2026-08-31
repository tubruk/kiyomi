package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

func (p *MyAnimeListPlugin) getBaseURL() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.baseURL != "" {
		return p.baseURL
	}
	return DefaultBase
}

// SetBaseURL overrides the upstream MyAnimeList API endpoint base URL (useful for testing).
func (p *MyAnimeListPlugin) SetBaseURL(url string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.baseURL = strings.TrimRight(url, "/")
}

// SetClientID updates the MAL client ID (useful for testing or dynamic config).
func (p *MyAnimeListPlugin) SetClientID(clientID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.clientID = clientID
}

func (p *MyAnimeListPlugin) newRequest(ctx context.Context, method, endpoint string) (*http.Request, error) {
	p.mu.RLock()
	ua := p.userAgent
	clientID := p.clientID
	p.mu.RUnlock()

	if clientID == "" {
		return nil, fmt.Errorf("myanimelist client_id is not configured (set KIYOMI_MAL_CLIENT_ID or provider setting)")
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-MAL-CLIENT-ID", clientID)
	return req, nil
}

func (p *MyAnimeListPlugin) doRequest(req *http.Request) (*http.Response, error) {
	p.mu.RLock()
	client := p.client
	p.mu.RUnlock()

	if client == nil {
		client = http.DefaultClient
	}
	return client.Do(req)
}
