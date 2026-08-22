package host

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// LogInterceptor intercepts stdio output from a plugin subprocess, parses structured JSON
// log lines (preserving nested slog.Group attributes), populates the plugin's ring buffer,
// and emits tagged logs to the host structured logger.
type LogInterceptor struct {
	pluginID string
	buffer   *RingBuffer
	logger   *slog.Logger
	mu       sync.Mutex
	lineBuf  bytes.Buffer
}

// NewLogInterceptor creates a new LogInterceptor for a given plugin and ring buffer.
func NewLogInterceptor(pluginID string, buffer *RingBuffer, logger *slog.Logger) *LogInterceptor {
	if logger == nil {
		logger = slog.Default()
	}
	return &LogInterceptor{
		pluginID: pluginID,
		buffer:   buffer,
		logger:   logger,
	}
}

// SetPluginID updates the plugin ID associated with the interceptor in a thread-safe manner.
func (i *LogInterceptor) SetPluginID(id string) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.pluginID = id
}

// Write processes incoming byte stream from subprocess stdio, splitting into lines.
func (i *LogInterceptor) Write(p []byte) (n int, err error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	for _, b := range p {
		if b == '\n' {
			line := i.lineBuf.String()
			i.lineBuf.Reset()
			i.processLine(strings.TrimRight(line, "\r"))
		} else {
			i.lineBuf.WriteByte(b)
		}
	}

	return len(p), nil
}

// Flush processes any remaining un-terminated line in the buffer.
func (i *LogInterceptor) Flush() {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.lineBuf.Len() > 0 {
		line := i.lineBuf.String()
		i.lineBuf.Reset()
		i.processLine(strings.TrimRight(line, "\r"))
	}
}

func (i *LogInterceptor) processLine(line string) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return
	}

	var rawMap map[string]any
	if err := json.Unmarshal([]byte(trimmed), &rawMap); err == nil && rawMap != nil {
		// Valid JSON structured log from plugin
		var timestamp time.Time
		if timeStr, ok := rawMap["time"].(string); ok {
			if t, err := time.Parse(time.RFC3339Nano, timeStr); err == nil {
				timestamp = t
			} else if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
				timestamp = t
			}
		}
		if timestamp.IsZero() {
			timestamp = time.Now().UTC()
		}

		levelStr := "INFO"
		if lvl, ok := rawMap["level"].(string); ok && lvl != "" {
			levelStr = strings.ToUpper(lvl)
		}

		msg := ""
		if m, ok := rawMap["msg"].(string); ok {
			msg = m
		} else if m, ok := rawMap["message"].(string); ok {
			msg = m
		}

		// Extract remaining custom fields
		fields := make(map[string]any)
		for k, v := range rawMap {
			if k != "time" && k != "level" && k != "msg" && k != "message" {
				fields[k] = v
			}
		}

		entry := PluginLogEntry{
			Timestamp: timestamp,
			Level:     levelStr,
			Message:   msg,
			Fields:    fields,
			Raw:       trimmed,
		}

		if i.buffer != nil {
			i.buffer.Push(entry)
		}

		i.logToHost(levelStr, msg, fields)
		return
	}

	// Plain text log line
	entry := PluginLogEntry{
		Timestamp: time.Now().UTC(),
		Level:     "INFO",
		Message:   trimmed,
		Raw:       trimmed,
	}

	if i.buffer != nil {
		i.buffer.Push(entry)
	}

	i.logToHost("INFO", trimmed, nil)
}

func (i *LogInterceptor) logToHost(levelStr, msg string, fields map[string]any) {
	if i.logger == nil {
		return
	}

	lvl := parseSlogLevel(levelStr)
	attrs := []any{slog.String("plugin_id", i.pluginID)}

	if len(fields) > 0 {
		attrs = append(attrs, fieldsToSlogAttrs(fields)...)
	}

	i.logger.Log(context.Background(), lvl, msg, attrs...)
}

func parseSlogLevel(lvl string) slog.Level {
	switch strings.ToUpper(lvl) {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// fieldsToSlogAttrs recursively converts a nested map into slog attributes, preserving nested groups.
func fieldsToSlogAttrs(fields map[string]any) []any {
	var attrs []any
	for k, v := range fields {
		if subMap, ok := v.(map[string]any); ok {
			groupAttrs := fieldsToSlogAttrs(subMap)
			slogAttrs := make([]slog.Attr, 0, len(groupAttrs))
			for _, ga := range groupAttrs {
				if sa, ok := ga.(slog.Attr); ok {
					slogAttrs = append(slogAttrs, sa)
				}
			}
			attrs = append(attrs, slog.Group(k, convertAnySliceToAny(slogAttrs)...))
		} else {
			attrs = append(attrs, slog.Any(k, v))
		}
	}
	return attrs
}

func convertAnySliceToAny(attrs []slog.Attr) []any {
	res := make([]any, len(attrs))
	for i, a := range attrs {
		res[i] = a
	}
	return res
}
