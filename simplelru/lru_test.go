package simplelru

import (
	"testing"
	"time"
)

func hackSleep() {
	// on macOS, UnixNanos() has a max resolution of microseconds.  Sleep
	// just a smidge here to ensure we evict the right item below.  In production
	// this wouldn't matter since we are an approximate LRU: if two items have a
	// timestamp within a micosecond of each other, either one would be old enough
	// to evict
	time.Sleep(1 * time.Microsecond)
}

func TestLRU(t *testing.T) {
	evictCounter := 0
	onEvicted := func(k interface{}, v interface{}) {
		if k != v {
			t.Fatalf("Evict values not equal (%v!=%v)", k, v)
		}
		evictCounter++
	}
	l, err := NewLRU(128, onEvicted)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	for i := 0; i < 256; i++ {
		l.Add(i, i)
		hackSleep()
	}
	if l.Len() != 128 {
		t.Fatalf("bad len: %v", l.Len())
	}

	if evictCounter != 128 {
		t.Fatalf("bad evict count: %v", evictCounter)
	}

	for k := range l.items {
		if v, ok := l.Get(k); !ok || v != k {
			t.Fatalf("bad key: %v", k)
		}
	}
	stale := 0
	for i := 0; i < 128; i++ {
		_, ok := l.Get(i)
		if ok {
			stale++
		}
	}
	// if we had a perfect LRU, this would be 0.  since we are approximating an LRU, this is slightly non-zero
	if stale > 20 {
		t.Fatalf("too many stale: %d", stale)
	}

	diedBeforeTheirTime := 0
	for i := 128; i < 256; i++ {
		_, ok := l.Get(i)
		if !ok {
			diedBeforeTheirTime++
		}
	}
	// if we had a perfect LRU, this would be 0.  since we are approximating an LRU, this is slightly non-zero
	if diedBeforeTheirTime > 20 {
		t.Fatalf("too many 'new' evicted early: %d", diedBeforeTheirTime)
	}

	for i := 128; i < 192; i++ {
		ok := l.Remove(i)
		if !ok {
			continue
		}
		ok = l.Remove(i)
		if ok {
			t.Fatalf("should not be contained")
		}
		_, ok = l.Get(i)
		if ok {
			t.Fatalf("should be deleted")
		}
	}

	l.Get(192) // expect 192 to be last key in l.Keys()

	/*for k := range l.items {
		if (i < 63 && k != i+193) || (i == 63 && k != 192) {
			t.Fatalf("out of order key: %v", k)
		}
	}*/

	l.Purge()
	if l.Len() != 0 {
		t.Fatalf("bad len: %v", l.Len())
	}
	if _, ok := l.Get(200); ok {
		t.Fatalf("should contain nothing")
	}
}

// Test that Add returns true/false if an eviction occurred
func TestLRU_Add(t *testing.T) {
	evictCounter := 0
	onEvicted := func(k interface{}, v interface{}) {
		evictCounter++
	}

	l, err := NewLRU(1, onEvicted)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if l.Add(1, 1) == true || evictCounter != 0 {
		t.Errorf("should not have an eviction")
	}
	if l.Add(2, 2) == false || evictCounter != 1 {
		t.Errorf("should have an eviction")
	}
}

// Test that Contains doesn't update recent-ness
func TestLRU_Contains(t *testing.T) {
	l, err := NewLRU(2, nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	l.Add(1, 1)
	hackSleep()
	l.Add(2, 2)
	if !l.Contains(1) {
		t.Errorf("1 should be contained")
	}

	hackSleep()
	l.Add(3, 3)
	if l.Contains(1) {
		t.Errorf("Contains should not have updated recent-ness of 1")
	}
}

// Test that Peek doesn't update recent-ness
func TestLRU_Peek(t *testing.T) {
	l, err := NewLRU(2, nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	l.Add(1, 1)
	hackSleep()
	l.Add(2, 2)
	if l.Len() != 2 {
		t.Errorf("expected Len to be 2")
	}
	if v, ok := l.Peek(1); !ok || v != 1 {
		t.Errorf("1 should be set to 1: %v, %v", v, ok)
	}

	l.Add(3, 3)
	if l.Contains(1) {
		t.Errorf("should not have updated recent-ness of 1")
	}
}

// Test that Resize can upsize and downsize
func TestLRU_Resize(t *testing.T) {
	onEvictCounter := 0
	onEvicted := func(k interface{}, v interface{}) {
		onEvictCounter++
	}
	l, err := NewLRU(2, onEvicted)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Downsize
	l.Add(1, 1)
	hackSleep()
	l.Add(2, 2)
	hackSleep()
	evicted := l.Resize(1)
	if evicted != 1 {
		t.Errorf("1 element should have been evicted: %v", evicted)
	}
	if onEvictCounter != 1 {
		t.Errorf("onEvicted should have been called 1 time: %v", onEvictCounter)
	}

	hackSleep()
	l.Add(3, 3)
	if l.Contains(1) {
		t.Errorf("Element 1 should have been evicted")
	}

	// Upsize
	evicted = l.Resize(2)
	if evicted != 0 {
		t.Errorf("0 elements should have been evicted: %v", evicted)
	}

	hackSleep()
	l.Add(4, 4)
	if !l.Contains(3) || !l.Contains(4) {
		t.Errorf("Cache should have contained 2 elements")
	}
}
