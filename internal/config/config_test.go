package config

import (
	"os"
	"testing"
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
}
