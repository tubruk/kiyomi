package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/labstack/echo/v4"
	"github.com/tubruk/kiyomi/internal/api"
	"github.com/tubruk/kiyomi/internal/config"
	"github.com/tubruk/kiyomi/internal/library"
	"github.com/tubruk/kiyomi/pkg/logger"
	"github.com/tubruk/kiyomi/pkg/webui"
)

// Build metadata — set via ldflags at build time:
//   go build -ldflags "-X main.version=1.0.0 -X main.commit=abc1234 -X main.buildTime=2026-01-01T00:00:00Z"
var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
)

func main() {
	// Load config
	cfg := config.Load()

	// Setup shared structured logger with pretty colored format by default
	logger.Setup(logger.Options{
		Level:  cfg.LogLevel,
		Format: cfg.LogFormat,
	})

	slog.Info("Kiyomi starting...", slog.String("home", cfg.Home), slog.String("port", cfg.Port))

	// Initialize filesystem library manager
	lib := library.NewLibrary(cfg.LibraryDir)

	// Create Echo instance
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Register panic recovery and error logging middleware, and custom HTTP error handler
	e.Use(logger.EchoPanicRecovery())
	e.Use(logger.EchoErrorLogger())
	e.HTTPErrorHandler = logger.EchoHTTPErrorHandler

	// Register API routes
	apiHandler := api.NewHandler(cfg, lib)
	// Attach build metadata
	apiHandler.SetBuildInfo(api.BuildInfo{
		Version:   version,
		Commit:    commit,
		BuildTime: buildTime,
	})
	apiHandler.RegisterRoutes(e)

	// Discover and start plugins in plugin directory
	if pm := apiHandler.PluginManager(); pm != nil {
		if err := pm.ReloadAll(context.Background()); err != nil {
			slog.Warn("initial plugin discovery encountered warnings", slog.String("error", err.Error()))
		}
	}

	// Embed Web UI routes (if build directory exists, otherwise serves dummy)
	webui.Register(e)

	// Start server
	slog.Info("Starting HTTP server", slog.String("port", cfg.Port))
	if err := e.Start(fmt.Sprintf(":%s", cfg.Port)); err != nil {
		slog.Error("server shut down unexpectedly", slog.String("error", err.Error()))
	}
}

