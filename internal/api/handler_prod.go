//go:build !e2e

package api

import (
	"github.com/labstack/echo/v4"
	"github.com/tubruk/kiyomi/pkg/provider"
)

func registerE2EProviders(reg *provider.Registry) {}
func registerE2ERoutes(e *echo.Echo)              {}
