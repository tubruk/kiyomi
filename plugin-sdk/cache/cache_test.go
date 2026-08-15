package cache

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheBasicGetSet(t *testing.T) {
	c := New[string, string]()
	defer c.Close()

	c.Set("key1", "val1", 0)
	val, ok := c.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "val1", val)

	_, ok = c.Get("nonexistent")
	assert.False(t, ok)

	c.Delete("key1")
	_, ok = c.Get("key1")
	assert.False(t, ok)
}

func TestCacheTTLExpiration(t *testing.T) {
	c := New[string, int]()
	defer c.Close()

	c.Set("temp", 42, 50*time.Millisecond)

	val, ok := c.Get("temp")
	assert.True(t, ok)
	assert.Equal(t, 42, val)

	time.Sleep(70 * time.Millisecond)

	_, ok = c.Get("temp")
	assert.False(t, ok)
}

func TestCacheGetOrSet(t *testing.T) {
	c := New[string, string]()
	defer c.Close()

	calls := 0
	fetcher := func() (string, time.Duration, error) {
		calls++
		return "computed", 50 * time.Millisecond, nil
	}

	val, err := c.GetOrSet("session", fetcher)
	require.NoError(t, err)
	assert.Equal(t, "computed", val)
	assert.Equal(t, 1, calls)

	// Second call before expiry uses cached value
	val, err = c.GetOrSet("session", fetcher)
	require.NoError(t, err)
	assert.Equal(t, "computed", val)
	assert.Equal(t, 1, calls)

	// After expiry, calls fetcher again
	time.Sleep(70 * time.Millisecond)
	val, err = c.GetOrSet("session", fetcher)
	require.NoError(t, err)
	assert.Equal(t, "computed", val)
	assert.Equal(t, 2, calls)

	// Fetcher returning error
	errFetcher := func() (string, time.Duration, error) {
		return "", 0, errors.New("fetch error")
	}
	_, err = c.GetOrSet("errKey", errFetcher)
	assert.Error(t, err)
}

func TestCachePurgeAndClear(t *testing.T) {
	c := New[string, string]()
	defer c.Close()

	c.Set("k1", "v1", 20*time.Millisecond)
	c.Set("k2", "v2", 1*time.Hour)
	assert.Equal(t, 2, c.Len())

	time.Sleep(30 * time.Millisecond)
	c.PurgeExpired()
	assert.Equal(t, 1, c.Len())

	c.Clear()
	assert.Equal(t, 0, c.Len())
}

func TestCacheConcurrency(t *testing.T) {
	c := NewWithCleanup[int, int](50 * time.Millisecond)
	defer c.Close()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			c.Set(id, id*2, 20*time.Millisecond)
			c.Get(id)
		}(i)
	}
	wg.Wait()
}
