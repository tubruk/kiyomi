package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// Options holds configuration for logger setup.
type Options struct {
	Level   string    // "debug", "info", "warn", "error"
	Format  string    // "pretty", "color", "json", "text"
	NoColor bool      // Disable color output for pretty handler
	Writer  io.Writer // Destination writer (defaults to os.Stdout)
}

// Setup initializes and sets the default global slog logger.
func Setup(opts Options) *slog.Logger {
	out := opts.Writer
	if out == nil {
		out = os.Stdout
	}

	var lvl slog.Level
	switch strings.ToLower(strings.TrimSpace(opts.Level)) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	var handler slog.Handler
	format := strings.ToLower(strings.TrimSpace(opts.Format))
	switch format {
	case "json":
		handler = slog.NewJSONHandler(out, &slog.HandlerOptions{Level: lvl})
	case "text":
		handler = slog.NewTextHandler(out, &slog.HandlerOptions{Level: lvl})
	default: // "pretty", "color", or empty
		handler = NewPrettyHandler(out, &PrettyHandlerOptions{
			Level:   lvl,
			NoColor: opts.NoColor,
			Writer:  out,
		})
	}

	l := slog.New(handler)
	slog.SetDefault(l)
	return l
}
