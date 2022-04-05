package lru

import (
	"hash/maphash"
	"sync"

	"github.com/bpowers/approx-lru/simplelru"
)

const defaultShardCount = 256

type shard struct {
	mu       sync.Mutex
	lru      simplelru.LRU
	_padding [16]uint8
}

// Cache is a thread-safe fixed size LRU cache.
type ShardedCache struct {
	templateHash maphash.Hash
	shards       []shard
	size         int
}

// New creates an LRU of the given size.
func NewSharded(size, shardCount int) (*ShardedCache, error) {
	return NewShardedWithEvict(size, shardCount, nil)
}

// NewWithEvict constructs a fixed size cache with the given eviction
// callback.
func NewShardedWithEvict(size, shardCount int, onEvicted func(key interface{}, value interface{})) (*ShardedCache, error) {
	if shardCount <= 0 {
		shardCount = defaultShardCount
	}
	if size < shardCount {
		size = shardCount
	}
	perShardSize := size / shardCount
	size = perShardSize * shardCount
	c := &ShardedCache{
		shards: make([]shard, shardCount),
		size:   size,
	}
	c.templateHash.SetSeed(maphash.MakeSeed())
	for i := 0; i < shardCount; i++ {
		shard, err := simplelru.NewLRU(perShardSize, onEvicted)
		if err != nil {
			return nil, err
		}
		c.shards[i].lru = *shard
	}
	return c, nil
}

// Purge is used to completely clear the cache.
func (c *ShardedCache) Purge() {
	for i := 0; i < len(c.shards); i++ {
		shard := &c.shards[i]
		shard.mu.Lock()
		shard.lru.Purge()
		shard.mu.Unlock()
	}
}

func (c *ShardedCache) getShard(key string) *shard {
	hash := c.templateHash
	hash.WriteString(key)
	shardId := hash.Sum64() % uint64(len(c.shards))
	return &c.shards[shardId]
}

// Add adds a value to the cache. Returns true if an eviction occurred.
func (c *ShardedCache) Add(key string, value interface{}) (evicted bool) {
	shard := c.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	return shard.lru.Add(key, value)
}

// Get looks up a key's value from the cache.
func (c *ShardedCache) Get(key string) (value interface{}, ok bool) {
	shard := c.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	return shard.lru.Get(key)
}

// Contains checks if a key is in the cache, without updating the
// recent-ness or deleting it for being stale.
func (c *ShardedCache) Contains(key string) bool {
	shard := c.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	return shard.lru.Contains(key)
}

// Peek returns the key value (or undefined if not found) without updating
// the "recently used"-ness of the key.
func (c *ShardedCache) Peek(key string) (value interface{}, ok bool) {
	shard := c.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	return shard.lru.Peek(key)
}

// ContainsOrAdd checks if a key is in the cache without updating the
// recent-ness or deleting it for being stale, and if not, adds the value.
// Returns whether found and whether an eviction occurred.
func (c *ShardedCache) ContainsOrAdd(key string, value interface{}) (ok, evicted bool) {
	shard := c.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	if shard.lru.Contains(key) {
		return true, false
	}
	evicted = shard.lru.Add(key, value)
	return false, evicted
}

// PeekOrAdd checks if a key is in the cache without updating the
// recent-ness or deleting it for being stale, and if not, adds the value.
// Returns whether found and whether an eviction occurred.
func (c *ShardedCache) PeekOrAdd(key string, value interface{}) (previous interface{}, ok, evicted bool) {
	shard := c.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	previous, ok = shard.lru.Peek(key)
	if ok {
		return previous, true, false
	}

	evicted = shard.lru.Add(key, value)
	return previous, false, evicted
}

// Remove removes the provided key from the cache.
func (c *ShardedCache) Remove(key string) (present bool) {
	shard := c.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	return shard.lru.Remove(key)
}

// we don't support resize

// Len returns the number of items in the cache.
func (c *ShardedCache) Len() int {
	size := 0
	for i := 0; i < len(c.shards); i++ {
		shard := &c.shards[i]
		shard.mu.Lock()
		size += shard.lru.Len()
		shard.mu.Unlock()
	}
	return size
}
