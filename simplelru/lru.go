package simplelru

import (
	crand "crypto/rand"
	"encoding/binary"
	"errors"
	"math/rand"
	"sort"
)

func newRand() *rand.Rand {
	seedBytes := make([]byte, 8)
	if _, err := crand.Read(seedBytes); err != nil {
		panic(err)
	}
	seed := binary.LittleEndian.Uint64(seedBytes)

	return rand.New(rand.NewSource(int64(seed)))
}

// EvictCallback is used to get a callback when a cache entry is evicted
type EvictCallback func(key interface{}, value interface{})

// TODO: move this to a file that is built only on 64-bit architectures and
// calculate the right size for 32-byte architectures
const LRUStructSize = 104

// LRU implements a non-thread safe fixed size LRU cache
type LRU struct {
	items   map[interface{}]int
	data    []entry
	counter int64
	size    int64
	rng     rand.Rand
	onEvict EvictCallback
}

const randomProbes = 8

// entry is used to hold a value in the evictList
type entry struct {
	lastUsed int64
	key      interface{}
	value    interface{}
}

// NewLRU constructs an LRU of the given size
func NewLRU(size int, onEvict EvictCallback) (*LRU, error) {
	if size <= 0 {
		return nil, errors.New("must provide a positive size")
	}
	c := &LRU{
		data:    make([]entry, 0, size),
		items:   make(map[interface{}]int, size),
		counter: 1,
		size:    int64(size),
		rng:     *newRand(),
		onEvict: onEvict,
	}
	return c, nil
}

func (c *LRU) getCounter() int64 {
	n := c.counter
	c.counter++
	if c.counter < 0 {
		panic("counter overflow; won't happen in practice :rip:")
	}
	return n
}

// Purge is used to completely clear the cache.
func (c *LRU) Purge() {
	for k, i := range c.items {
		if c.onEvict != nil {
			c.onEvict(k, c.data[i].value)
		}
	}
	c.data = c.data[0:0]
	c.items = make(map[interface{}]int)
}

//go:noinline
func (c *LRU) shuffle() {
	c.rng.Shuffle(len(c.data), func(i, j int) {
		c.items[c.data[i].key] = j
		c.items[c.data[j].key] = i

		c.data[i], c.data[j] = c.data[j], c.data[i]
	})
}

// Add adds a value to the cache.  Returns true if an eviction occurred.
func (c *LRU) Add(key interface{}, value interface{}) (evicted bool) {
	now := c.getCounter()
	// Check for existing item
	if i, ok := c.items[key]; ok {
		entry := &c.data[i]
		entry.lastUsed = now
		entry.value = value
		return false
	}

	// Add new item
	ent := entry{now, key, value}

	if int64(len(c.data)) < c.size {
		i := len(c.data)
		c.data = append(c.data, ent)
		c.items[key] = i
		// if we have filled up the cache for the first time, shuffle
		// the items to ensure they are randomly distributed in the array.
		// we need this to ensure our random probing is correct.
		if int64(len(c.data)) == c.size {
			c.shuffle()
		}
	} else {
		evicted = true
		i := c.removeOldest()
		c.data[i] = ent
		c.items[key] = i
	}

	return
}

// Get looks up a key's value from the cache.
func (c *LRU) Get(key interface{}) (value interface{}, ok bool) {
	if i, ok := c.items[key]; ok {
		entry := &c.data[i]
		entry.lastUsed = c.getCounter()
		return entry.value, true
	}
	return
}

// Contains checks if a key is in the cache, without updating the recent-ness
// or deleting it for being stale.
func (c *LRU) Contains(key interface{}) (ok bool) {
	_, ok = c.items[key]
	return ok
}

// Peek returns the key value (or undefined if not found) without updating
// the "recently used"-ness of the key.
func (c *LRU) Peek(key interface{}) (value interface{}, ok bool) {
	if i, ok := c.items[key]; ok {
		return c.data[i].value, true
	}
	return value, false
}

// Remove removes the provided key from the cache, returning if the
// key was contained.
func (c *LRU) Remove(key interface{}) (present bool) {
	if i, ok := c.items[key]; ok {
		c.removeElement(i, c.data[i])
		return true
	}
	return false
}

// Len returns the number of items in the cache.
func (c *LRU) Len() int {
	return len(c.items)
}

type byLastUsed []entry

func (a byLastUsed) Len() int           { return len(a) }
func (a byLastUsed) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byLastUsed) Less(i, j int) bool { return a[i].lastUsed > a[j].lastUsed }

// Resize changes the cache size.
func (c *LRU) Resize(size int) (evicted int) {
	diff := c.Len() - size
	if diff < 0 {
		diff = 0
	}
	// sort in descending order
	sort.Sort(byLastUsed(c.data))
	for i, entry := range c.data {
		if entry.lastUsed == 0 {
			continue
		}
		c.items[entry.key] = i
	}
	oldSize := len(c.data)
	for i := 0; i < diff; i++ {
		j := oldSize - 1 - i
		entry := c.data[j]
		if entry.lastUsed > 0 {
			c.removeElement(j, entry)
		}
	}
	c.size = int64(size)
	if size < oldSize {
		c.data = c.data[:size]
	} else {
		oldData := c.data
		c.data = make([]entry, oldSize, size)
		copy(c.data, oldData)
	}
	if len(c.data) != len(c.items) {
		panic("we mucked it up")
	}
	c.shuffle()
	return diff
}

// removeOldest removes the oldest item from the cache.
func (c *LRU) removeOldest() (off int) {
	size := c.Len()
	if size <= 0 {
		return -1
	}
	base := c.rng.Intn(size)
	oldestOff := base
	oldest := c.data[base]
	// if our offset does NOT result in us wrapping off the end of the array
	// (which is unlikely! should be predicted well), don't require `% size`
	// as that is expensive.  duplicate the whole loop to put the conditional
	// outside the loop rather than in it.
	if base+randomProbes-1 < size {
		for j := 1; j < randomProbes; j++ {
			off := base + j
			candidate := &c.data[off]
			if candidate.lastUsed < oldest.lastUsed {
				oldestOff = off
				oldest = *candidate
			}
		}
	} else {
		for j := 1; j < randomProbes; j++ {
			off := (base + j) % size
			candidate := &c.data[off]
			if candidate.lastUsed < oldest.lastUsed {
				oldestOff = off
				oldest = *candidate
			}
		}
	}

	// we could have found an empty slot
	if oldest.lastUsed != 0 {
		c.removeElement(oldestOff, oldest)
	}
	return oldestOff
}

// removeElement is used to remove a given list element from the cache
func (c *LRU) removeElement(i int, ent entry) {
	c.data[i] = entry{}
	delete(c.items, ent.key)
	if c.onEvict != nil {
		c.onEvict(ent.key, ent.value)
	}
}
