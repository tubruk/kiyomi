package host

import (
	"fmt"
	"sync"
)

// DefaultLogBufferCapacity is the default number of log lines retained per plugin.
const DefaultLogBufferCapacity = 200

// RingBuffer is a thread-safe circular buffer for PluginLogEntry items.
type RingBuffer struct {
	mu       sync.RWMutex
	entries  []PluginLogEntry
	capacity int
	start    int
	size     int
}

// NewRingBuffer initializes a new RingBuffer with the given maximum capacity.
func NewRingBuffer(capacity int) *RingBuffer {
	if capacity <= 0 {
		capacity = DefaultLogBufferCapacity
	}
	return &RingBuffer{
		entries:  make([]PluginLogEntry, capacity),
		capacity: capacity,
		start:    0,
		size:     0,
	}
}

// Push appends a new entry to the ring buffer, overwriting the oldest entry if full.
func (r *RingBuffer) Push(entry PluginLogEntry) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.size < r.capacity {
		idx := (r.start + r.size) % r.capacity
		r.entries[idx] = entry
		r.size++
	} else {
		r.entries[r.start] = entry
		r.start = (r.start + 1) % r.capacity
	}
}

// Entries returns a copy of all buffered log entries in chronological order (oldest to newest).
func (r *RingBuffer) Entries() []PluginLogEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]PluginLogEntry, r.size)
	for i := 0; i < r.size; i++ {
		idx := (r.start + i) % r.capacity
		result[i] = r.entries[idx]
	}
	return result
}

// Lines returns formatted string representations of all log entries in chronological order.
func (r *RingBuffer) Lines() []string {
	entries := r.Entries()
	lines := make([]string, len(entries))
	for i, e := range entries {
		if e.Raw != "" {
			lines[i] = e.Raw
		} else {
			lines[i] = fmt.Sprintf("[%s] [%s] %s", e.Timestamp.Format("2006-01-02T15:04:05.000Z07:00"), e.Level, e.Message)
		}
	}
	return lines
}

// Len returns the current number of buffered entries.
func (r *RingBuffer) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.size
}

// Clear removes all entries from the buffer.
func (r *RingBuffer) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.start = 0
	r.size = 0
}
