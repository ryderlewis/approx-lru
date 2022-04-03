package lru

import (
	"sync"

	"github.com/bpowers/approx-lru/simplelru"
)

// Cache is a thread-safe fixed size LRU cache.
type Cache[K comparable, V any] struct {
	lru  simplelru.LRUCache[K, V]
	lock sync.RWMutex
}

// New creates an LRU of the given size.
func New[K comparable, V any](size int) (*Cache[K, V], error) {
	return NewWithEvict[K, V](size, nil)
}

// NewWithEvict constructs a fixed size cache with the given eviction
// callback.
func NewWithEvict[K comparable, V any](size int, onEvicted func(key K, value V)) (*Cache[K, V], error) {
	lru, err := simplelru.NewLRU[K, V](size, simplelru.EvictCallback[K, V](onEvicted))
	if err != nil {
		return nil, err
	}
	c := &Cache[K, V]{
		lru: lru,
	}
	return c, nil
}

// Purge is used to completely clear the cache.
func (c *Cache[K, V]) Purge() {
	c.lock.Lock()
	c.lru.Purge()
	c.lock.Unlock()
}

// Add adds a value to the cache. Returns true if an eviction occurred.
func (c *Cache[K, V]) Add(key K, value V) (evicted bool) {
	c.lock.Lock()
	evicted = c.lru.Add(key, value)
	c.lock.Unlock()
	return evicted
}

// Get looks up a key's value from the cache.
func (c *Cache[K, V]) Get(key K) (value V, ok bool) {
	c.lock.Lock()
	value, ok = c.lru.Get(key)
	c.lock.Unlock()
	return value, ok
}

// Contains checks if a key is in the cache, without updating the
// recent-ness or deleting it for being stale.
func (c *Cache[K, V]) Contains(key K) bool {
	c.lock.RLock()
	containKey := c.lru.Contains(key)
	c.lock.RUnlock()
	return containKey
}

// Peek returns the key value (or undefined if not found) without updating
// the "recently used"-ness of the key.
func (c *Cache[K, V]) Peek(key K) (value V, ok bool) {
	c.lock.RLock()
	value, ok = c.lru.Peek(key)
	c.lock.RUnlock()
	return value, ok
}

// ContainsOrAdd checks if a key is in the cache without updating the
// recent-ness or deleting it for being stale, and if not, adds the value.
// Returns whether found and whether an eviction occurred.
func (c *Cache[K, V]) ContainsOrAdd(key K, value V) (ok, evicted bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.lru.Contains(key) {
		return true, false
	}
	evicted = c.lru.Add(key, value)
	return false, evicted
}

// PeekOrAdd checks if a key is in the cache without updating the
// recent-ness or deleting it for being stale, and if not, adds the value.
// Returns whether found and whether an eviction occurred.
func (c *Cache[K, V]) PeekOrAdd(key K, value V) (previous V, ok, evicted bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	previous, ok = c.lru.Peek(key)
	if ok {
		return previous, true, false
	}

	evicted = c.lru.Add(key, value)
	return previous, false, evicted
}

// Remove removes the provided key from the cache.
func (c *Cache[K, V]) Remove(key K) (present bool) {
	c.lock.Lock()
	present = c.lru.Remove(key)
	c.lock.Unlock()
	return
}

// Resize changes the cache size.
func (c *Cache[K, V]) Resize(size int) (evicted int) {
	c.lock.Lock()
	evicted = c.lru.Resize(size)
	c.lock.Unlock()
	return evicted
}

// Len returns the number of items in the cache.
func (c *Cache[K, V]) Len() int {
	c.lock.RLock()
	length := c.lru.Len()
	c.lock.RUnlock()
	return length
}
