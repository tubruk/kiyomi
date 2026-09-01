package main

import (
	"context"
	"net/http"
	"strings"
)

func (p *AtsumaruPlugin) getBaseURL() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.baseURL != "" {
		return p.baseURL
	}
	return DefaultBase
}

// SetBaseURL overrides the upstream API base URL (useful for testing).
func (p *AtsumaruPlugin) SetBaseURL(url string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.baseURL = strings.TrimRight(url, "/")
}

func (p *AtsumaruPlugin) newRequest(ctx context.Context, method, endpoint string) (*http.Request, error) {
	p.mu.RLock()
	ua := p.userAgent
	p.mu.RUnlock()

	req, err := http.NewRequestWithContext(ctx, method, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if ua != "" {
		req.Header.Set("User-Agent", ua)
	}
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func (p *AtsumaruPlugin) doRequest(req *http.Request) (*http.Response, error) {
	p.mu.RLock()
	client := p.client
	p.mu.RUnlock()

	if client == nil {
		client = http.DefaultClient
	}
	return client.Do(req)
}

// resolveImageURL turns a relative or absolute image path from Atsumaru into a fully qualified HTTPS URL.
func (p *AtsumaruPlugin) resolveImageURL(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if strings.HasPrefix(path, "//") {
		return "https:" + path
	}

	cleanPath := strings.TrimPrefix(path, "/")
	cleanPath = strings.TrimPrefix(cleanPath, "static/")

	return p.getBaseURL() + "/static/" + cleanPath
}
