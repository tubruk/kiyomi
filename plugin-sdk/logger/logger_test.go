package logger

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONHandlerBasic(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, &Options{Level: slog.LevelInfo})

	log.Info("fetching manga", slog.String("provider", "mangadex"), slog.Int("page", 1))

	var parsed map[string]any
	err := json.Unmarshal(buf.Bytes(), &parsed)
	require.NoError(t, err)

	assert.Equal(t, "INFO", parsed["level"])
	assert.Equal(t, "fetching manga", parsed["msg"])
	assert.Equal(t, "mangadex", parsed["provider"])
	assert.Equal(t, float64(1), parsed["page"])
	assert.NotEmpty(t, parsed["time"])
}

func TestJSONHandlerNestedGroup(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, nil)

	scoped := log.WithGroup("http").With(slog.String("method", "GET"))
	scoped.Info("request sent",
		slog.Group("response",
			slog.Int("status", 200),
			slog.String("content_type", "application/json"),
		),
	)

	var parsed map[string]any
	err := json.Unmarshal(buf.Bytes(), &parsed)
	require.NoError(t, err)

	assert.Equal(t, "request sent", parsed["msg"])

	httpGroup, ok := parsed["http"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "GET", httpGroup["method"])

	respGroup, ok := httpGroup["response"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(200), respGroup["status"])
	assert.Equal(t, "application/json", respGroup["content_type"])
}

func TestJSONHandlerLevelFilter(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, &Options{Level: slog.LevelWarn})

	log.Debug("debug message")
	log.Info("info message")
	assert.Empty(t, buf.String())

	log.Warn("warn message")
	assert.Contains(t, buf.String(), "warn message")
}
