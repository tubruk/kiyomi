package logger

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"
)

// Options configuration for SDK JSON logger handler.
type Options struct {
	Level slog.Leveler
}

// JSONHandler is a slog.Handler that formats logs as structured JSON lines across stdio pipes,
// preserving nested slog.Group attributes and maps without flattening.
type JSONHandler struct {
	opts   Options
	writer io.Writer
	mu     *sync.Mutex
	prefix string
	groups []string
	attrs  []slog.Attr
}

// New creates a new slog.Logger configured with the SDK's structured JSON handler outputting to w (or stderr if nil).
func New(w io.Writer, opts *Options) *slog.Logger {
	return slog.New(NewHandler(w, opts))
}

// NewHandler creates a new JSONHandler writing structured JSON records to w.
func NewHandler(w io.Writer, opts *Options) *JSONHandler {
	if w == nil {
		w = os.Stderr
	}
	opt := Options{}
	if opts != nil {
		opt = *opts
	}
	if opt.Level == nil {
		opt.Level = slog.LevelInfo
	}

	return &JSONHandler{
		opts:   opt,
		writer: w,
		mu:     &sync.Mutex{},
		groups: nil,
		attrs:  nil,
	}
}

func (h *JSONHandler) Enabled(_ context.Context, level slog.Level) bool {
	minLevel := slog.LevelInfo
	if h.opts.Level != nil {
		minLevel = h.opts.Level.Level()
	}
	return level >= minLevel
}

func (h *JSONHandler) Handle(_ context.Context, r slog.Record) error {
	payload := make(map[string]any)
	payload["time"] = r.Time.UTC().Format(time.RFC3339Nano)
	payload["level"] = r.Level.String()
	payload["msg"] = r.Message

	attrsMap := make(map[string]any)

	// Add handler pre-configured attributes
	for _, attr := range h.attrs {
		appendAttr(attrsMap, h.groups, attr)
	}

	// Add record dynamic attributes
	r.Attrs(func(attr slog.Attr) bool {
		appendAttr(attrsMap, h.groups, attr)
		return true
	})

	for k, v := range attrsMap {
		payload[k] = v
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	data = append(data, '\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err = h.writer.Write(data)
	return err
}

func appendAttr(target map[string]any, groups []string, attr slog.Attr) {
	attr.Value = attr.Value.Resolve()
	if attr.Equal(slog.Attr{}) {
		return
	}

	current := target
	for _, group := range groups {
		if sub, ok := current[group].(map[string]any); ok {
			current = sub
		} else {
			sub = make(map[string]any)
			current[group] = sub
			current = sub
		}
	}

	if attr.Value.Kind() == slog.KindGroup {
		groupAttrs := attr.Value.Group()
		groupMap := make(map[string]any)
		for _, a := range groupAttrs {
			appendAttr(groupMap, nil, a)
		}
		current[attr.Key] = groupMap
	} else {
		current[attr.Key] = attr.Value.Any()
	}
}

func (h *JSONHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, 0, len(h.attrs)+len(attrs))
	newAttrs = append(newAttrs, h.attrs...)
	newAttrs = append(newAttrs, attrs...)

	return &JSONHandler{
		opts:   h.opts,
		writer: h.writer,
		mu:     h.mu,
		groups: h.groups,
		attrs:  newAttrs,
	}
}

func (h *JSONHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	newGroups := make([]string, 0, len(h.groups)+1)
	newGroups = append(newGroups, h.groups...)
	newGroups = append(newGroups, name)

	return &JSONHandler{
		opts:   h.opts,
		writer: h.writer,
		mu:     h.mu,
		groups: newGroups,
		attrs:  h.attrs,
	}
}
