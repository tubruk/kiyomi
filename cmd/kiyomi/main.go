package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	machineryConfig "github.com/RichardKnop/machinery/v2/config"
	"github.com/tubruk/kiyomi/internal/api"
	"github.com/tubruk/kiyomi/internal/config"
	"github.com/tubruk/kiyomi/internal/library"
	"github.com/tubruk/kiyomi/internal/queue"
	"github.com/tubruk/kiyomi/internal/queue/inmemory"
	"github.com/tubruk/kiyomi/internal/queue/machinery"
	"github.com/tubruk/kiyomi/internal/queue/sqlite"
	"github.com/tubruk/kiyomi/pkg/logger"
	"github.com/tubruk/kiyomi/pkg/provider/sdk"
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

	// Seed process-wide DNS resolver list from config (env var or config file).
	if len(cfg.DNSResolvers) > 0 {
		sdk.SetGlobalDNSResolvers(cfg.DNSResolvers)
	}

	// Setup shared structured logger with pretty colored format by default
	logger.Setup(logger.Options{
		Level:  cfg.LogLevel,
		Format: cfg.LogFormat,
	})

	slog.Info("Kiyomi starting...", slog.String("home", cfg.Home), slog.String("port", cfg.Port))

	// Initialize filesystem library manager
	lib := library.NewLibrary(cfg.LibraryDir)
	lib.SetAllowPrivateNetworks(cfg.AllowPrivateNetworks)

	// Create Echo instance
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Register panic recovery and error logging middleware, and custom HTTP error handler
	e.Use(logger.EchoPanicRecovery())
	e.Use(logger.EchoErrorLogger())
	e.HTTPErrorHandler = logger.EchoHTTPErrorHandler

	// Initialize background job queue dependencies
	var jobStore queue.JobStore
	var enqueuer queue.Enqueuer
	var worker queue.Worker

	switch cfg.QueueDriver {
	case "sqlite":
		sqliteDriver, err := sqlite.NewDriver(cfg.QueueDBPath, cfg.QueueConcurrency)
		if err != nil {
			slog.Error("failed to initialize sqlite queue driver", slog.String("error", err.Error()))
			os.Exit(1)
		}
		if cfg.QueueCleanupEnabled {
			sqliteDriver.SetCleanupConfig(cfg.QueueCleanupInterval, queue.CleanupOptions{
				CompletedTTL: cfg.QueueCompletedTTL,
				FailedTTL:    cfg.QueueFailedTTL,
				BatchSize:    cfg.QueueCleanupBatchSize,
			})
		}
		jobStore = sqliteDriver
		enqueuer = sqliteDriver
		worker = sqliteDriver
	case "inmemory":
		inmemDriver := inmemory.NewDriver(cfg.QueueConcurrency)
		jobStore = inmemDriver
		enqueuer = inmemDriver
		worker = inmemDriver
	case "machinery":
		sqliteStore, err := sqlite.NewDriver(cfg.QueueDBPath, cfg.QueueConcurrency)
		if err != nil {
			slog.Error("failed to initialize sqlite store for machinery", slog.String("error", err.Error()))
			os.Exit(1)
		}
		if cfg.QueueCleanupEnabled {
			sqliteStore.SetCleanupConfig(cfg.QueueCleanupInterval, queue.CleanupOptions{
				CompletedTTL: cfg.QueueCompletedTTL,
				FailedTTL:    cfg.QueueFailedTTL,
				BatchSize:    cfg.QueueCleanupBatchSize,
			})
		}
		jobStore = sqliteStore

		mcfg := &machineryConfig.Config{
			Broker:          os.Getenv("KIYOMI_MACHINERY_BROKER"),
			DefaultQueue:    "kiyomi_jobs",
			ResultBackend:   os.Getenv("KIYOMI_MACHINERY_BACKEND"),
			ResultsExpireIn: int(cfg.QueueCompletedTTL.Seconds()),
		}
		if mcfg.Broker == "" {
			mcfg.Broker = "redis://localhost:6379"
		}
		if mcfg.ResultBackend == "" {
			mcfg.ResultBackend = "redis://localhost:6379"
		}
		machineryDriver, err := machinery.NewDriver(mcfg, "kiyomi-worker")
		if err != nil {
			slog.Error("failed to initialize machinery queue driver", slog.String("error", err.Error()))
			os.Exit(1)
		}
		enqueuer = machineryDriver
		worker = machineryDriver
	default:
		slog.Error("unknown queue driver", slog.String("driver", cfg.QueueDriver))
		os.Exit(1)
	}

	// Register API routes
	apiHandler := api.NewHandler(cfg, lib, jobStore, enqueuer)
	// Attach build metadata
	apiHandler.SetBuildInfo(api.BuildInfo{
		Version:   version,
		Commit:    commit,
		BuildTime: buildTime,
	})
	apiHandler.RegisterRoutes(e)

	// Register pull queue handlers before the worker starts so it can
	// dispatch pull_* jobs the moment they are enqueued.
	if worker != nil {
		pullHandlers := queue.NewPullHandlers(lib, apiHandler.Registry(), apiHandler.HTTPClient(), apiHandler.FingerprintStore(), enqueuer, jobStore)
		pullHandlers.SetImageCache(apiHandler.ImageCache())
		pullHandlers.SetAllowPrivateNetworks(cfg.AllowPrivateNetworks)
		pullHandlerMap := map[string]queue.JobHandler{
			queue.JobTypePullManga:   queue.JobHandlerFunc(pullHandlers.HandlePullManga),
			queue.JobTypePullChapter: queue.JobHandlerFunc(pullHandlers.HandlePullChapter),
			queue.JobTypePullPage:    queue.JobHandlerFunc(pullHandlers.HandlePullPage),
			queue.JobTypePullCover:   queue.JobHandlerFunc(pullHandlers.HandlePullCover),
		}
		for jobType, h := range pullHandlerMap {
			if err := worker.Register(jobType, h); err != nil {
				slog.Error("failed to register queue handler", slog.String("type", jobType), slog.String("error", err.Error()))
				os.Exit(1)
			}
		}
	}

	// Discover and start plugins in plugin directory
	if pm := apiHandler.PluginManager(); pm != nil {
		if err := pm.ReloadAll(context.Background()); err != nil {
			slog.Warn("initial plugin discovery encountered warnings", slog.String("error", err.Error()))
		}
	}

	// Configure per-group concurrency limits for the pull pipeline now that
	// e2e + plugin content providers are registered. Cover jobs share a
	// dedicated bucket; per-provider buckets throttle fan-out against a
	// single upstream.
	if worker != nil && apiHandler != nil {
		queue.ApplyPullGroupLimits(worker, apiHandler.Registry().ListContent())
	}

	// Embed Web UI routes (if build directory exists, otherwise serves dummy)
	webui.Register(e)

	// Start server in a goroutine
	go func() {
		slog.Info("Starting HTTP server", slog.String("port", cfg.Port))
		if err := e.Start(fmt.Sprintf(":%s", cfg.Port)); err != nil && err != http.ErrServerClosed {
			slog.Error("server shut down unexpectedly", slog.String("error", err.Error()))
		}
	}()

	// Start queue worker in a background goroutine
	if worker != nil {
		go func() {
			slog.Info("Starting background queue worker...")
			if err := worker.Start(context.Background()); err != nil {
				slog.Error("queue worker failed to start", slog.String("error", err.Error()))
			}
		}()
	}

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down HTTP server gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		slog.Error("server shutdown failed", slog.String("error", err.Error()))
	} else {
		slog.Info("Server exited gracefully")
	}

	if worker != nil {
		slog.Info("Stopping background queue worker...")
		workerCtx, workerCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer workerCancel()
		if err := worker.Stop(workerCtx); err != nil {
			slog.Error("queue worker shutdown failed", slog.String("error", err.Error()))
		} else {
			slog.Info("Queue worker stopped gracefully")
		}
	}

	if pm := apiHandler.PluginManager(); pm != nil {
		_ = pm.Close()
	}

	if closer, ok := jobStore.(interface{ Close() error }); ok {
		_ = closer.Close()
	}
}

