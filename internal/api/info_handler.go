package api

import (
	"net/http"
	"runtime"

	"github.com/labstack/echo/v4"
)

// BuildInfo holds build-time metadata injected via ldflags.
type BuildInfo struct {
	Version   string
	Commit    string
	BuildTime string
}

// AppInfoResponse is the JSON response for GET /api/v1/info.
type AppInfoResponse struct {
	App       string `json:"app"`
	Version   string `json:"version"`
	GoVersion string `json:"go_version"`
	BuildTime string `json:"build_time"`
	Commit    string `json:"commit"`
}

// getInfo handles GET /api/v1/info.
func (h *Handler) getInfo(c echo.Context) error {
	return c.JSON(http.StatusOK, AppInfoResponse{
		App:       "Kiyomi",
		Version:   h.buildInfo.Version,
		GoVersion: runtime.Version(),
		BuildTime: h.buildInfo.BuildTime,
		Commit:    h.buildInfo.Commit,
	})
}
