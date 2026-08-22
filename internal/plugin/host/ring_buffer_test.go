package host_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tubruk/kiyomi/internal/plugin/host"
)

func TestRingBuffer_PushAndOrder(t *testing.T) {
	buf := host.NewRingBuffer(3)
	assert.Equal(t, 0, buf.Len())

	buf.Push(host.PluginLogEntry{Message: "msg 1", Level: "INFO", Timestamp: time.Now()})
	buf.Push(host.PluginLogEntry{Message: "msg 2", Level: "INFO", Timestamp: time.Now()})
	assert.Equal(t, 2, buf.Len())

	entries := buf.Entries()
	require.Len(t, entries, 2)
	assert.Equal(t, "msg 1", entries[0].Message)
	assert.Equal(t, "msg 2", entries[1].Message)

	// Push 3rd (at capacity)
	buf.Push(host.PluginLogEntry{Message: "msg 3", Level: "WARN", Timestamp: time.Now()})
	assert.Equal(t, 3, buf.Len())

	// Push 4th (overwrites msg 1)
	buf.Push(host.PluginLogEntry{Message: "msg 4", Level: "ERROR", Timestamp: time.Now()})
	assert.Equal(t, 3, buf.Len())

	entries = buf.Entries()
	require.Len(t, entries, 3)
	assert.Equal(t, "msg 2", entries[0].Message)
	assert.Equal(t, "msg 3", entries[1].Message)
	assert.Equal(t, "msg 4", entries[2].Message)

	// Push 5th (overwrites msg 2)
	buf.Push(host.PluginLogEntry{Message: "msg 5", Level: "INFO", Timestamp: time.Now()})
	entries = buf.Entries()
	require.Len(t, entries, 3)
	assert.Equal(t, "msg 3", entries[0].Message)
	assert.Equal(t, "msg 4", entries[1].Message)
	assert.Equal(t, "msg 5", entries[2].Message)

	lines := buf.Lines()
	require.Len(t, lines, 3)

	buf.Clear()
	assert.Equal(t, 0, buf.Len())
	assert.Empty(t, buf.Entries())
}

func TestRingBuffer_ConcurrentAccess(t *testing.T) {
	buf := host.NewRingBuffer(50)
	var wg sync.WaitGroup

	// Writers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				buf.Push(host.PluginLogEntry{
					Message:   fmt.Sprintf("worker %d msg %d", workerID, j),
					Level:     "INFO",
					Timestamp: time.Now(),
				})
			}
		}(i)
	}

	// Readers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_ = buf.Entries()
				_ = buf.Lines()
				_ = buf.Len()
			}
		}()
	}

	wg.Wait()
	assert.Equal(t, 50, buf.Len())
}

func TestRingBuffer_LinesFormatting(t *testing.T) {
	buf := host.NewRingBuffer(3)
	ts := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	buf.Push(host.PluginLogEntry{
		Raw:       `{"time":"2026-08-22T12:00:00Z","level":"INFO","msg":"raw line"}`,
		Message:   "raw line",
		Level:     "INFO",
		Timestamp: ts,
	})
	buf.Push(host.PluginLogEntry{
		Message:   "formatted line",
		Level:     "WARN",
		Timestamp: ts,
	})

	lines := buf.Lines()
	require.Len(t, lines, 2)
	assert.Equal(t, `{"time":"2026-08-22T12:00:00Z","level":"INFO","msg":"raw line"}`, lines[0])
	assert.Contains(t, lines[1], "[WARN] formatted line")

	buf.Clear()
	assert.Equal(t, 0, buf.Len())
	assert.Empty(t, buf.Lines())
}
