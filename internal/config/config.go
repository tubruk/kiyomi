package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Environment variable names used to override defaults.
const (
	EnvHome        = "KIYOMI_HOME"
	EnvDownloadDir = "KIYOMI_DOWNLOAD_DIR"
	EnvCacheDir    = "KIYOMI_CACHE_DIR"
	EnvDBPath      = "KIYOMI_DB_PATH"
	EnvPort        = "KIYOMI_PORT"
	EnvLibraryDir  = "KIYOMI_LIBRARY_DIR"

	// EnvWebDir is the dev-mode override. When unset the server serves
	// the embedded web UI bundle; when set to a directory path
	// (relative values are joined to Home, absolute values used
	// verbatim) the server serves that filesystem directory instead so
	// a freshly built web/dist takes effect without recompiling Go.
	EnvWebDir = "KIYOMI_WEB_DIR"

	EnvProviderConfig   = "KIYOMI_PROVIDER_CONFIG"
	EnvOfflineMode      = "KIYOMI_OFFLINE_MODE"
	EnvCachePageTTL     = "KIYOMI_CACHE_PAGE_TTL"
	EnvCacheMetadataTTL = "KIYOMI_CACHE_METADATA_TTL"
	EnvCacheImageTTL    = "KIYOMI_CACHE_IMAGE_TTL"
	EnvCacheSearchTTL   = "KIYOMI_CACHE_SEARCH_TTL"
	EnvCacheMaxBytes    = "KIYOMI_CACHE_MAX_BYTES"
	EnvLogLevel         = "KIYOMI_LOG_LEVEL"
	EnvLogFormat        = "KIYOMI_LOG_FORMAT"
)

// Config holds all static, process-wide paths and runtime knobs for the
// Kiyomi server. A single Config is constructed once at startup in main and
// passed to the components that need it (API handler, web static server).
type Config struct {
	// Home is the base directory for all relative paths. Defaults to the
	// process's current working directory when KIYOMI_HOME is unset.
	Home string

	// DownloadDir is the absolute path on disk where page assets and other
	// downloaded content is cached. When the user supplies a relative path
	// (the default), it is resolved against Home. An absolute path is used
	// verbatim.
	DownloadDir string

	// CacheDir is the absolute path on disk where ephemeral assets are cached.
	// When the user supplies a relative path (the default), it is resolved against Home.
	CacheDir string

	// DBPath is the absolute filesystem path to the SQLite database file
	// (e.g. "<home>/kiyomi.db"). Relative values from KIYOMI_DB_PATH are
	// joined to Home; absolute values are used verbatim. The default is
	// "<home>/kiyomi.db".
	DBPath string

	// LibraryDir is the absolute filesystem path to the manga library root.
	// When the user supplies a relative path (the default), it is resolved
	// against Home. An absolute path is used verbatim. The default is
	// "<home>/library".
	LibraryDir string

	// WebDist is the dev-mode override for the web UI bundle source. When
	// empty (the default), the server serves the assets embedded in the
	// binary. When set, the server serves from this filesystem directory
	// instead, so a freshly built web/dist takes effect without
	// recompiling Go. Relative values from KIYOMI_WEB_DIR are joined to
	// Home; absolute values are used verbatim.
	WebDist string

	// Port the HTTP server binds to. Defaults to "8080".
	Port string

	// GlobalConcurrency is the maximum number of queue workers overall. Defaults to 4.
	GlobalConcurrency int
	// ProviderConcurrency is the default maximum number of workers per provider. Defaults to 2.
	ProviderConcurrency int
	// ProviderConfigPath is the filesystem path to the per-provider TOML config file.
	// Defaults to "<home>/providers.json". Relative values are joined to Home;
	// absolute values are used verbatim.
	ProviderConfigPath string

	// OfflineMode determines whether network calls to providers are disabled.
	OfflineMode bool

	// Per-category Cache TTLs
	CachePageTTL     time.Duration
	CacheMetadataTTL time.Duration
	CacheImageTTL    time.Duration
	CacheSearchTTL   time.Duration

	// CacheMaxBytes is the maximum disk space used by the image cache in bytes. Defaults to 2 GB.
	CacheMaxBytes int64

	// LogLevel specifies logging threshold ("debug", "info", "warn", "error"). Defaults to "info".
	LogLevel string
	// LogFormat specifies output style ("pretty", "json", "text"). Defaults to "pretty".
	LogFormat string
}

// Load builds a Config from the process environment. It never returns an
// error: missing or malformed values fall back to documented defaults so the
// server always starts. Failures are surfaced through the returned Config
// (with safe defaults) AND logged so operators can see what changed.
func Load() *Config {
	cfg := &Config{Port: "8080", GlobalConcurrency: 4, ProviderConcurrency: 2}

	cfg.LogLevel = strings.TrimSpace(os.Getenv(EnvLogLevel))
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}
	cfg.LogFormat = strings.TrimSpace(os.Getenv(EnvLogFormat))
	if cfg.LogFormat == "" {
		cfg.LogFormat = "pretty"
	}

	cfg.Home = resolveHome()

	if port := strings.TrimSpace(os.Getenv(EnvPort)); port != "" {
		cfg.Port = port
	} else {
		slog.Debug("config: using default port", slog.String("port", cfg.Port))
	}

	dl, dlSrc := resolveDownloadDir(cfg.Home)
	cfg.DownloadDir = dl
	switch dlSrc {
	case sourceEnvOverride:
		slog.Info("config: download dir overridden by env",
			slog.String("env", EnvDownloadDir),
			slog.String("path", cfg.DownloadDir),
		)
	case sourceRelativeDefault:
		slog.Debug("config: using default download dir relative to home",
			slog.String("home", cfg.Home),
			slog.String("path", cfg.DownloadDir),
		)
	}

	cache, cacheSrc := resolveCacheDir(cfg.Home)
	cfg.CacheDir = cache
	switch cacheSrc {
	case sourceEnvOverride:
		slog.Info("config: cache dir overridden by env",
			slog.String("env", EnvCacheDir),
			slog.String("path", cfg.CacheDir),
		)
	case sourceRelativeDefault:
		slog.Debug("config: using default cache dir relative to home",
			slog.String("home", cfg.Home),
			slog.String("path", cfg.CacheDir),
		)
	}

	dbPath, dbSrc := resolveDBPath(cfg.Home)
	cfg.DBPath = dbPath
	switch dbSrc {
	case sourceEnvOverride:
		slog.Info("config: db path overridden by env",
			slog.String("env", EnvDBPath),
			slog.String("path", cfg.DBPath),
		)
	case sourceRelativeDefault:
		slog.Debug("config: using default db path relative to home",
			slog.String("home", cfg.Home),
			slog.String("path", cfg.DBPath),
		)
	}

	webDist, webSrc := resolveWebDist(cfg.Home)
	cfg.WebDist = webDist
	if webSrc == sourceEnvOverride {
		slog.Info("config: web UI overridden to filesystem (dev mode)",
			slog.String("env", EnvWebDir),
			slog.String("path", cfg.WebDist),
		)
	} else {
		slog.Debug("config: serving embedded web UI bundle (default)")
	}

	libraryDir, librarySrc := resolveLibraryDir(cfg.Home)
	cfg.LibraryDir = libraryDir
	switch librarySrc {
	case sourceEnvOverride:
		slog.Info("config: library dir overridden by env",
			slog.String("env", EnvLibraryDir),
			slog.String("path", cfg.LibraryDir),
		)
	case sourceRelativeDefault:
		slog.Debug("config: using default library dir relative to home",
			slog.String("home", cfg.Home),
			slog.String("path", cfg.LibraryDir),
		)
	}

	providerConfigPath, providerConfigSrc := resolveProviderConfigPath(cfg.Home)
	cfg.ProviderConfigPath = providerConfigPath
	switch providerConfigSrc {
	case sourceEnvOverride:
		slog.Info("config: provider config path overridden by env",
			slog.String("env", EnvProviderConfig),
			slog.String("path", cfg.ProviderConfigPath),
		)
	case sourceRelativeDefault:
		slog.Debug("config: using default provider config path relative to home",
			slog.String("home", cfg.Home),
			slog.String("path", cfg.ProviderConfigPath),
		)
	}

	cfg.OfflineMode = parseBool(EnvOfflineMode, false)
	cfg.CachePageTTL = parseDuration(EnvCachePageTTL, 168*time.Hour)
	cfg.CacheMetadataTTL = parseDuration(EnvCacheMetadataTTL, 12*time.Hour)
	cfg.CacheImageTTL = parseDuration(EnvCacheImageTTL, 720*time.Hour)
	cfg.CacheSearchTTL = parseDuration(EnvCacheSearchTTL, 1*time.Hour)
	cfg.CacheMaxBytes = parsePositiveInt64(EnvCacheMaxBytes, 2*1024*1024*1024)

	return cfg
}

// IsAbs reports whether path is an absolute filesystem path. It understands
// both POSIX absolute paths ("/foo") and Windows-style drive-letter or UNC
// paths ("C:\\foo", "\\\\server\\share"). HOME-relative config values should
// be passed through filepath.IsAbs; on Linux/macOS that is enough, but on
// Windows an env var like "D:\\manga" is also absolute and must not be
// joined to Home.
func IsAbs(path string) bool {
	if path == "" {
		return false
	}
	return filepath.IsAbs(path)
}

// resolveHome returns the configured home directory. Falls back to the
// process's current working directory when KIYOMI_HOME is unset or blank.
func resolveHome() string {
	raw := strings.TrimSpace(os.Getenv(EnvHome))
	if raw == "" {
		wd, err := os.Getwd()
		if err != nil {
			// os.Getwd failing is essentially never recoverable. Return "." so
			// downstream path joins at least produce sensible values, and log
			// loudly so it shows up in the server banner.
			slog.Error("config: failed to determine working directory; falling back to \".\"",
				slog.String("error", err.Error()),
			)
			return "."
		}
		return wd
	}
	return raw
}

// resolveDownloadDir applies the relative-vs-absolute rule. The single
// override is KIYOMI_DOWNLOAD_DIR; no legacy fallback.
func resolveDownloadDir(home string) (string, configSource) {
	if v := strings.TrimSpace(os.Getenv(EnvDownloadDir)); v != "" {
		return resolveAgainstHome(home, v), sourceEnvOverride
	}
	return filepath.Join(home, "library"), sourceRelativeDefault
}

// resolveCacheDir applies the relative-vs-absolute rule. The single
// override is KIYOMI_CACHE_DIR.
func resolveCacheDir(home string) (string, configSource) {
	if v := strings.TrimSpace(os.Getenv(EnvCacheDir)); v != "" {
		return resolveAgainstHome(home, v), sourceEnvOverride
	}
	return filepath.Join(home, "cache"), sourceRelativeDefault
}

// resolveDBPath applies the relative-vs-absolute rule to the SQLite file
// path. Default is "<home>/kiyomi.db".
func resolveDBPath(home string) (string, configSource) {
	if v := strings.TrimSpace(os.Getenv(EnvDBPath)); v != "" {
		return resolveAgainstHome(home, v), sourceEnvOverride
	}
	return filepath.Join(home, "kiyomi.db"), sourceRelativeDefault
}

// resolveWebDist returns the dev-mode override for the web UI bundle
// source. Empty result means "serve the embedded bundle" (the
// default). Non-empty result is the absolute filesystem path the
// server should serve from instead. KIYOMI_WEB_DIR is consulted when
// set; relative values are joined to Home, absolute values are used
// verbatim.
func resolveWebDist(home string) (string, configSource) {
	v := strings.TrimSpace(os.Getenv(EnvWebDir))
	if v == "" {
		return "", sourceRelativeDefault
	}
	return resolveAgainstHome(home, v), sourceEnvOverride
}

// resolveLibraryDir returns the manga library root directory.
// KIYOMI_LIBRARY_DIR is consulted when set; relative values are joined
// to Home, absolute values are used verbatim. Default is "<home>/library".
func resolveLibraryDir(home string) (string, configSource) {
	if v := strings.TrimSpace(os.Getenv(EnvLibraryDir)); v != "" {
		return resolveAgainstHome(home, v), sourceEnvOverride
	}
	return filepath.Join(home, "library"), sourceRelativeDefault
}

// resolveProviderConfigPath returns the provider config file path.
// KIYOMI_PROVIDER_CONFIG is consulted when set; relative values are joined
// to Home, absolute values are used verbatim. Default is "<home>/providers.json".
func resolveProviderConfigPath(home string) (string, configSource) {
	if v := strings.TrimSpace(os.Getenv(EnvProviderConfig)); v != "" {
		return resolveAgainstHome(home, v), sourceEnvOverride
	}
	return filepath.Join(home, "providers.json"), sourceRelativeDefault
}

// resolveAgainstHome is the single chokepoint that turns a user-supplied
// path into an absolute path. Absolute inputs (including the home itself)
// pass through; relative inputs are joined to home. Empty inputs return
// home itself so callers never end up with an empty string.
func resolveAgainstHome(home, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return home
	}
	if IsAbs(raw) {
		return filepath.Clean(raw)
	}
	return filepath.Clean(filepath.Join(home, raw))
}

// Validate is a best-effort sanity check run at startup. It does not fail
// fatally: callers may choose to log warnings only. Today the checks are
// that Home, DownloadDir, and DBPath are non-empty; future fields can extend
// it without changing the public surface.
func (c *Config) Validate() error {
	if c == nil {
		return errors.New("config: nil")
	}
	if strings.TrimSpace(c.Home) == "" {
		return errors.New("config: home dir is empty")
	}
	if strings.TrimSpace(c.DownloadDir) == "" {
		return errors.New("config: download dir is empty")
	}
	if strings.TrimSpace(c.CacheDir) == "" {
		return errors.New("config: cache dir is empty")
	}
	if strings.TrimSpace(c.DBPath) == "" {
		return errors.New("config: db path is empty")
	}
	if strings.TrimSpace(c.LibraryDir) == "" {
		return errors.New("config: library dir is empty")
	}
	if c.CacheMaxBytes <= 0 {
		return errors.New("config: cache max bytes must be greater than zero")
	}
	return nil
}

// String renders a compact, operator-friendly view of the resolved paths.
// Safe to log at startup.
func (c *Config) String() string {
	if c == nil {
		return "<nil config>"
	}
	return fmt.Sprintf(
		"home=%s download_dir=%s cache_dir=%s db_path=%s library_dir=%s web_dist=%s port=%s global_concurrency=%d provider_concurrency=%d provider_config=%s cache_max_bytes=%d",
		c.Home, c.DownloadDir, c.CacheDir, c.DBPath, c.LibraryDir, c.WebDist, c.Port, c.GlobalConcurrency, c.ProviderConcurrency, c.ProviderConfigPath, c.CacheMaxBytes,
	)
}

// configSource tags how a derived value was obtained. Used internally for
// logging precedence decisions; not part of the public surface.
type configSource int

const (
	sourceRelativeDefault configSource = iota
	sourceEnvOverride
)

func parsePositiveInt(envName string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(envName))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		slog.Warn("config: invalid positive integer; using default", slog.String("env", envName), slog.String("value", raw), slog.Int("default", fallback))
		return fallback
	}
	slog.Info("config: concurrency overridden by env", slog.String("env", envName), slog.Int("value", value))
	return value
}

func parsePositiveInt64(envName string, fallback int64) int64 {
	raw := strings.TrimSpace(os.Getenv(envName))
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		slog.Warn("config: invalid positive int64; using default", slog.String("env", envName), slog.String("value", raw), slog.Int64("default", fallback))
		return fallback
	}
	slog.Info("config: cache max bytes overridden by env", slog.String("env", envName), slog.Int64("value", value))
	return value
}

func parseBool(envName string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(envName))
	if raw == "" {
		return fallback
	}
	val, err := strconv.ParseBool(raw)
	if err != nil {
		slog.Warn("config: invalid boolean; using default", slog.String("env", envName), slog.String("value", raw), slog.Bool("default", fallback))
		return fallback
	}
	slog.Info("config: boolean overridden by env", slog.String("env", envName), slog.Bool("value", val))
	return val
}

func parseDuration(envName string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(envName))
	if raw == "" {
		return fallback
	}
	dur, err := time.ParseDuration(raw)
	if err != nil || dur <= 0 {
		slog.Warn("config: invalid duration; using default", slog.String("env", envName), slog.String("value", raw), slog.Duration("default", fallback))
		return fallback
	}
	slog.Info("config: duration overridden by env", slog.String("env", envName), slog.Duration("value", dur))
	return dur
}
