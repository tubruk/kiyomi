package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestGetInfo(t *testing.T) {
	e := echo.New()
	h := &Handler{
		buildInfo: BuildInfo{
			Version:   "1.2.3",
			Commit:    "abc1234",
			BuildTime: "2026-01-01T00:00:00Z",
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/info", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.getInfo(c); err != nil {
		t.Fatalf("getInfo returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp AppInfoResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.App != "Kiyomi" {
		t.Errorf("expected app=Kiyomi, got %q", resp.App)
	}
	if resp.Version != "1.2.3" {
		t.Errorf("expected version=1.2.3, got %q", resp.Version)
	}
	if resp.Commit != "abc1234" {
		t.Errorf("expected commit=abc1234, got %q", resp.Commit)
	}
	if resp.BuildTime != "2026-01-01T00:00:00Z" {
		t.Errorf("expected build_time=2026-01-01T00:00:00Z, got %q", resp.BuildTime)
	}
	if !strings.HasPrefix(resp.GoVersion, "go") {
		t.Errorf("expected GoVersion to start with 'go', got %q", resp.GoVersion)
	}
}
