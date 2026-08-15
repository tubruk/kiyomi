package host_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tubruk/kiyomi/internal/plugin/host"
	sdklogger "github.com/tubruk/kiyomi/plugin-sdk/logger"
)

func TestLogInterceptor_StructuredJSONWithNestedGroups(t *testing.T) {
	buf := host.NewRingBuffer(10)
	var hostLogOut bytes.Buffer
	hostLogger := slog.New(slog.NewJSONHandler(&hostLogOut, &slog.HandlerOptions{Level: slog.LevelDebug}))

	interceptor := host.NewLogInterceptor("test-plugin", buf, hostLogger)

	// Simulate plugin emitting structured JSON log using SDK logger
	var pluginStdErr bytes.Buffer
	pLogger := sdklogger.New(&pluginStdErr, &sdklogger.Options{Level: slog.LevelDebug})
	groupLogger := pLogger.WithGroup("request").With(slog.String("method", "GET"), slog.Int("status", 200))
	groupLogger.InfoContext(context.Background(), "fetch chapter successful", slog.Group("target", slog.String("manga_id", "123"), slog.Float64("chapter", 1.5)))

	// Write simulated stderr into interceptor
	_, err := interceptor.Write(pluginStdErr.Bytes())
	require.NoError(t, err)

	entries := buf.Entries()
	require.Len(t, entries, 1)

	entry := entries[0]
	assert.Equal(t, "INFO", entry.Level)
	assert.Equal(t, "fetch chapter successful", entry.Message)

	// Verify nested fields are preserved
	reqGroup, ok := entry.Fields["request"].(map[string]any)
	require.True(t, ok, "request field should be a map")
	assert.Equal(t, "GET", reqGroup["method"])
	assert.Equal(t, float64(200), reqGroup["status"])

	targetGroup, ok := reqGroup["target"].(map[string]any)
	require.True(t, ok, "target field should be a nested map")
	assert.Equal(t, "123", targetGroup["manga_id"])
	assert.Equal(t, 1.5, targetGroup["chapter"])

	// Verify host log output received the log tagged with plugin_id
	assert.Contains(t, hostLogOut.String(), `"plugin_id":"test-plugin"`)
	assert.Contains(t, hostLogOut.String(), `"fetch chapter successful"`)
}

func TestLogInterceptor_PlainTextAndChunkedWrites(t *testing.T) {
	buf := host.NewRingBuffer(10)
	interceptor := host.NewLogInterceptor("test-plugin", buf, nil)

	// Chunked write
	_, _ = interceptor.Write([]byte("panic: something went "))
	_, _ = interceptor.Write([]byte("wrong\n"))
	_, _ = interceptor.Write([]byte("line 2 without trailing newline"))
	interceptor.Flush()

	entries := buf.Entries()
	require.Len(t, entries, 2)
	assert.Equal(t, "panic: something went wrong", entries[0].Message)
	assert.Equal(t, "line 2 without trailing newline", entries[1].Message)
	assert.False(t, entries[0].Timestamp.IsZero())
}

func TestLogInterceptor_TimestampParsing(t *testing.T) {
	buf := host.NewRingBuffer(10)
	interceptor := host.NewLogInterceptor("test-plugin", buf, nil)

	jsonLine := `{"time":"2026-08-16T12:34:56.789Z","level":"WARN","msg":"warning message","count":42}` + "\n"
	_, err := interceptor.Write([]byte(jsonLine))
	require.NoError(t, err)

	entries := buf.Entries()
	require.Len(t, entries, 1)
	assert.Equal(t, "WARN", entries[0].Level)
	assert.Equal(t, "warning message", entries[0].Message)
	assert.Equal(t, 2026, entries[0].Timestamp.Year())
	assert.Equal(t, time.August, entries[0].Timestamp.Month())
	assert.Equal(t, float64(42), entries[0].Fields["count"])
}
