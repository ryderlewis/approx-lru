// Package simplelru provides simple LRU implementation based on build-in container/list.
package simplelru

// LRUCache is the interface for simple LRU cache.
type LRUCache interface {
	// Adds a value to the cache, returns true if an eviction occurred and
	// updates the "recently used"-ness of the key.
	Add(key interface{}, value interface{}) bool

	// Returns key's value from the cache and
	// updates the "recently used"-ness of the key. #value, isFound
	Get(key interface{}) (value interface{}, ok bool)

	// Checks if a key exists in cache without updating the recent-ness.
	Contains(key interface{}) (ok bool)

	// Returns key's value without updating the "recently used"-ness of the key.
	Peek(key interface{}) (value interface{}, ok bool)

	// Removes a key from the cache.
	Remove(key interface{}) bool

	// Returns the number of items in the cache.
	Len() int

	// Clears all cache entries.
	Purge()

	// Resizes cache, returning number evicted
	Resize(int) int
}
