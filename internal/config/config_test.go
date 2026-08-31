package config

import (
	"os"
	"testing"
	"time"
)

func TestConfigLoadAndValidate(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		cfg := Load()
		if err := cfg.Validate(); err != nil {
			t.Fatalf("expected valid config, got error: %v", err)
		}
		if cfg.CacheMaxBytes != 2*1024*1024*1024 {
			t.Fatalf("expected CacheMaxBytes=2GB (%d), got %d", int64(2*1024*1024*1024), cfg.CacheMaxBytes)
		}
		if !cfg.QueueCleanupEnabled {
			t.Fatal("expected QueueCleanupEnabled=true by default")
		}
		if cfg.QueueCleanupInterval != 1*time.Hour {
			t.Fatalf("expected QueueCleanupInterval=1h, got %v", cfg.QueueCleanupInterval)
		}
		if cfg.QueueCompletedTTL != 24*time.Hour {
			t.Fatalf("expected QueueCompletedTTL=24h, got %v", cfg.QueueCompletedTTL)
		}
		if cfg.QueueFailedTTL != 168*time.Hour {
			t.Fatalf("expected QueueFailedTTL=168h, got %v", cfg.QueueFailedTTL)
		}
		if cfg.QueueCleanupBatchSize != 500 {
			t.Fatalf("expected QueueCleanupBatchSize=500, got %d", cfg.QueueCleanupBatchSize)
		}
		if str := cfg.String(); str == "" {
			t.Fatal("expected non-empty String() representation")
		}
	})

	t.Run("env override", func(t *testing.T) {
		os.Setenv(EnvCacheMaxBytes, "1073741824")
		defer os.Unsetenv(EnvCacheMaxBytes)

		cfg := Load()
		if cfg.CacheMaxBytes != 1073741824 {
			t.Fatalf("expected CacheMaxBytes=1073741824, got %d", cfg.CacheMaxBytes)
		}
	})

	t.Run("plugin dir override", func(t *testing.T) {
		os.Setenv(EnvPluginDir, "/tmp/custom-plugins")
		defer os.Unsetenv(EnvPluginDir)

		cfg := Load()
		if cfg.PluginDir != "/tmp/custom-plugins" {
			t.Fatalf("expected PluginDir=/tmp/custom-plugins, got %s", cfg.PluginDir)
		}
	})

	t.Run("validate cache max bytes", func(t *testing.T) {
		cfg := Load()
		cfg.CacheMaxBytes = 0
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected validation error for CacheMaxBytes=0, got nil")
		}

		cfg.CacheMaxBytes = -100
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected validation error for CacheMaxBytes=-100, got nil")
		}
	})

	t.Run("queue env override", func(t *testing.T) {
		os.Setenv(EnvQueueDriver, "machinery")
		os.Setenv(EnvQueueDBPath, "/tmp/test_jobs.db")
		os.Setenv(EnvQueueConcurrency, "10")
		os.Setenv(EnvQueueCleanupEnabled, "false")
		os.Setenv(EnvQueueCleanupInterval, "30m")
		os.Setenv(EnvQueueCompletedTTL, "12h")
		os.Setenv(EnvQueueFailedTTL, "72h")
		os.Setenv(EnvQueueCleanupBatchSize, "100")
		defer func() {
			os.Unsetenv(EnvQueueDriver)
			os.Unsetenv(EnvQueueDBPath)
			os.Unsetenv(EnvQueueConcurrency)
			os.Unsetenv(EnvQueueCleanupEnabled)
			os.Unsetenv(EnvQueueCleanupInterval)
			os.Unsetenv(EnvQueueCompletedTTL)
			os.Unsetenv(EnvQueueFailedTTL)
			os.Unsetenv(EnvQueueCleanupBatchSize)
		}()

		cfg := Load()
		if cfg.QueueDriver != "machinery" {
			t.Fatalf("expected QueueDriver=machinery, got %s", cfg.QueueDriver)
		}
		if cfg.QueueDBPath != "/tmp/test_jobs.db" {
			t.Fatalf("expected QueueDBPath=/tmp/test_jobs.db, got %s", cfg.QueueDBPath)
		}
		if cfg.QueueConcurrency != 10 {
			t.Fatalf("expected QueueConcurrency=10, got %d", cfg.QueueConcurrency)
		}
		if cfg.QueueCleanupEnabled {
			t.Fatal("expected QueueCleanupEnabled=false, got true")
		}
		if cfg.QueueCleanupInterval != 30*time.Minute {
			t.Fatalf("expected QueueCleanupInterval=30m, got %v", cfg.QueueCleanupInterval)
		}
		if cfg.QueueCompletedTTL != 12*time.Hour {
			t.Fatalf("expected QueueCompletedTTL=12h, got %v", cfg.QueueCompletedTTL)
		}
		if cfg.QueueFailedTTL != 72*time.Hour {
			t.Fatalf("expected QueueFailedTTL=72h, got %v", cfg.QueueFailedTTL)
		}
		if cfg.QueueCleanupBatchSize != 100 {
			t.Fatalf("expected QueueCleanupBatchSize=100, got %d", cfg.QueueCleanupBatchSize)
		}
	})

	t.Run("validate queue cleanup settings", func(t *testing.T) {
		cfg := Load()
		cfg.QueueCleanupInterval = 0
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected validation error for QueueCleanupInterval=0, got nil")
		}

		cfg = Load()
		cfg.QueueCompletedTTL = 0
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected validation error for QueueCompletedTTL=0, got nil")
		}

		cfg = Load()
		cfg.QueueFailedTTL = 0
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected validation error for QueueFailedTTL=0, got nil")
		}

		cfg = Load()
		cfg.QueueCleanupBatchSize = 0
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected validation error for QueueCleanupBatchSize=0, got nil")
		}
	})
}
