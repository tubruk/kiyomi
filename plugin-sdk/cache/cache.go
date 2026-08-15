package cache

import (
	"sync"
	"time"
)

type item[V any] struct {
	value     V
	expiresAt time.Time
	hasTTL    bool
}

func (i item[V]) isExpired(now time.Time) bool {
	if !i.hasTTL {
		return false
	}
	return now.After(i.expiresAt)
}

// Cache is a thread-safe in-memory cache supporting generic keys and values with TTL expiration.
type Cache[K comparable, V any] struct {
	mu     sync.RWMutex
	items  map[K]item[V]
	stopCh chan struct{}
}

// New creates an in-memory TTL cache without a background cleanup goroutine.
// Expired items are lazily evicted on Get and can be explicitly purged with PurgeExpired.
func New[K comparable, V any]() *Cache[K, V] {
	return &Cache[K, V]{
		items: make(map[K]item[V]),
	}
}

// NewWithCleanup creates an in-memory TTL cache that periodically evicts expired entries.
func NewWithCleanup[K comparable, V any](interval time.Duration) *Cache[K, V] {
	c := &Cache[K, V]{
		items:  make(map[K]item[V]),
		stopCh: make(chan struct{}),
	}
	if interval > 0 {
		go c.cleanupLoop(interval)
	}
	return c
}

func (c *Cache[K, V]) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.PurgeExpired()
		case <-c.stopCh:
			return
		}
	}
}

// Set stores a value with an optional TTL. If ttl <= 0, the item does not expire.
func (c *Cache[K, V]) Set(key K, value V, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var it item[V]
	it.value = value
	if ttl > 0 {
		it.hasTTL = true
		it.expiresAt = time.Now().Add(ttl)
	}
	c.items[key] = it
}

// Get retrieves a value by key. If the item does not exist or has expired, it returns the zero value and false.
func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.mu.RLock()
	it, ok := c.items[key]
	c.mu.RUnlock()

	if !ok {
		var zero V
		return zero, false
	}

	if it.isExpired(time.Now()) {
		// Lazily remove expired entry
		c.mu.Lock()
		if current, exists := c.items[key]; exists && current.isExpired(time.Now()) {
			delete(c.items, key)
		}
		c.mu.Unlock()

		var zero V
		return zero, false
	}

	return it.value, true
}

// GetOrSet returns the existing value or computes it with fn and stores it with the returned TTL.
func (c *Cache[K, V]) GetOrSet(key K, fn func() (V, time.Duration, error)) (V, error) {
	if val, ok := c.Get(key); ok {
		return val, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double check after acquiring write lock
	if it, ok := c.items[key]; ok && !it.isExpired(time.Now()) {
		return it.value, nil
	}

	val, ttl, err := fn()
	if err != nil {
		var zero V
		return zero, err
	}

	var it item[V]
	it.value = val
	if ttl > 0 {
		it.hasTTL = true
		it.expiresAt = time.Now().Add(ttl)
	}
	c.items[key] = it

	return val, nil
}

// Delete removes an item from the cache.
func (c *Cache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

// Clear removes all items from the cache.
func (c *Cache[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[K]item[V])
}

// Len returns the total number of items in the cache, including expired items that haven't been purged.
func (c *Cache[K, V]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// PurgeExpired deletes all expired entries from the cache.
func (c *Cache[K, V]) PurgeExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for k, it := range c.items {
		if it.isExpired(now) {
			delete(c.items, k)
		}
	}
}

// Close stops the background cleanup worker if configured.
func (c *Cache[K, V]) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.stopCh != nil {
		select {
		case <-c.stopCh:
		default:
			close(c.stopCh)
		}
	}
}
