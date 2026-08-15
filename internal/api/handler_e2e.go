//go:build e2e

package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/labstack/echo/v4"
	"github.com/tubruk/kiyomi/pkg/provider"
	"github.com/tubruk/kiyomi/pkg/provider/mock"
)

func getMockFixturesDir() string {
	return os.Getenv("KIYOMI_MOCK_FIXTURES")
}

func registerE2EProviders(reg *provider.Registry) {
	fixturesDir := getMockFixturesDir()
	reg.Register(mock.New(fixturesDir))
}

func registerE2ERoutes(e *echo.Echo) {
	fixturesDir := getMockFixturesDir()
	v1 := e.Group("/api/v1/mock")

	v1.GET("/:remoteID/:chapterID/:pageIndex", func(c echo.Context) error {
		remoteID := c.Param("remoteID")
		chapterID := c.Param("chapterID")
		pageIndex := c.Param("pageIndex")

		if fixturesDir != "" {
			filename := fmt.Sprintf("manga-%s-%s-%s.png", remoteID, chapterID, pageIndex)
			fpath := filepath.Join(fixturesDir, filename)
			if data, err := os.ReadFile(fpath); err == nil {
				return c.Blob(http.StatusOK, "image/png", data)
			}
			mangaDir := filepath.Join(fixturesDir, fmt.Sprintf("manga-%s", remoteID))
			fpath = filepath.Join(mangaDir, fmt.Sprintf("%s.png", pageIndex))
			if data, err := os.ReadFile(fpath); err == nil {
				return c.Blob(http.StatusOK, "image/png", data)
			}
		}

		return c.Blob(http.StatusOK, "image/png", transparent1x1PNG)
	})

	v1.GET("/covers/:remoteID", func(c echo.Context) error {
		remoteID := c.Param("remoteID")

		if fixturesDir != "" {
			fpath := filepath.Join(fixturesDir, fmt.Sprintf("cover-%s.png", remoteID))
			if data, err := os.ReadFile(fpath); err == nil {
				return c.Blob(http.StatusOK, "image/png", data)
			}
		}

		return c.Blob(http.StatusOK, "image/png", transparent1x1PNG)
	})
}

var transparent1x1PNG = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
	0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4, 0x89, 0x00, 0x00, 0x00,
	0x0d, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00, 0x00, 0x00, 0x00, 0x49,
	0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
}
