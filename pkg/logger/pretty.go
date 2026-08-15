package logger

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
)

// PrettyHandlerOptions configuration for PrettyHandler.
type PrettyHandlerOptions struct {
	Level   slog.Leveler
	NoColor bool
	Writer  io.Writer
}

// PrettyHandler is a custom slog.Handler that formats logs in a human-readable, colorized format.
type PrettyHandler struct {
	opts   PrettyHandlerOptions
	mu     *sync.Mutex
	out    io.Writer
	attrs  []slog.Attr
	groups []string
}

// NewPrettyHandler creates a new PrettyHandler.
func NewPrettyHandler(out io.Writer, opts *PrettyHandlerOptions) *PrettyHandler {
	if opts == nil {
		opts = &PrettyHandlerOptions{}
	}
	if opts.Level == nil {
		opts.Level = slog.LevelInfo
	}
	if opts.Writer != nil {
		out = opts.Writer
	}
	if out == nil {
		out = os.Stdout
	}

	noColor := opts.NoColor
	if os.Getenv("NO_COLOR") != "" {
		noColor = true
	}

	return &PrettyHandler{
		opts: PrettyHandlerOptions{
			Level:   opts.Level,
			NoColor: noColor,
			Writer:  out,
		},
		mu:  &sync.Mutex{},
		out: out,
	}
}

func (h *PrettyHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}

const (
	colorReset   = "\033[0m"
	colorFaint   = "\033[90m"
	colorCyan    = "\033[36m"
	colorGreen   = "\033[32m"
	colorYellow  = "\033[33m"
	colorRedBold = "\033[1;31m"
	colorBlue    = "\033[34m"
	colorBold    = "\033[1m"
)

func (h *PrettyHandler) Handle(_ context.Context, r slog.Record) error {
	buf := bytes.NewBuffer(make([]byte, 0, 256))

	// Timestamp format: 15:04:05.000
	ts := r.Time.Format("15:04:05.000")
	if h.opts.NoColor {
		buf.WriteString(ts)
		buf.WriteByte(' ')
	} else {
		buf.WriteString(colorFaint)
		buf.WriteString(ts)
		buf.WriteString(colorReset)
		buf.WriteByte(' ')
	}

	// Level tag
	levelStr := r.Level.String()
	if !h.opts.NoColor {
		switch r.Level {
		case slog.LevelDebug:
			buf.WriteString(colorCyan)
		case slog.LevelInfo:
			buf.WriteString(colorGreen)
		case slog.LevelWarn:
			buf.WriteString(colorYellow)
		case slog.LevelError:
			buf.WriteString(colorRedBold)
		default:
			buf.WriteString(colorFaint)
		}
	}
	buf.WriteString(fmt.Sprintf("[%5s]", levelStr))
	if !h.opts.NoColor {
		buf.WriteString(colorReset)
	}
	buf.WriteByte(' ')

	// Message
	if !h.opts.NoColor {
		buf.WriteString(colorBold)
		buf.WriteString(r.Message)
		buf.WriteString(colorReset)
	} else {
		buf.WriteString(r.Message)
	}

	// Attr helper
	writeAttr := func(a slog.Attr, prefix string) {
		a.Value = a.Value.Resolve()
		if a.Equal(slog.Attr{}) {
			return
		}
		key := prefix + a.Key
		buf.WriteByte(' ')
		if !h.opts.NoColor {
			buf.WriteString(colorBlue)
			buf.WriteString(key)
			buf.WriteString(colorReset)
			buf.WriteByte('=')
		} else {
			buf.WriteString(key)
			buf.WriteByte('=')
		}

		if err, ok := a.Value.Any().(error); ok {
			buf.WriteString(err.Error())
		} else {
			buf.WriteString(fmt.Sprintf("%v", a.Value.Any()))
		}
	}

	prefix := ""
	for _, g := range h.groups {
		prefix += g + "."
	}

	for _, a := range h.attrs {
		writeAttr(a, prefix)
	}

	r.Attrs(func(a slog.Attr) bool {
		writeAttr(a, prefix)
		return true
	})

	buf.WriteByte('\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.out.Write(buf.Bytes())
	return err
}

func (h *PrettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h2 := *h
	h2.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &h2
}

func (h *PrettyHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	h2 := *h
	h2.groups = append(append([]string{}, h.groups...), name)
	return &h2
}
