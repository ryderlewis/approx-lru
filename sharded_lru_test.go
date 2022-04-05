package lru

import (
	"strconv"
	"sync"
	"testing"
	"unsafe"
)

func TestNewSharded(t *testing.T) {

}

func TestShardSize(t *testing.T) {
	if 128 != unsafe.Sizeof(shard{}) {
		t.Fatalf("expected shard to be 128-bytes in size")
	}
}

func BenchmarkLRU_BigSharded(b *testing.B) {
	var rngMu sync.Mutex
	rng := newRand()
	rngMu.Lock()
	l, err := NewSharded(128*1024, defaultShardCount)
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	type traceEntry struct {
		k string
		v int64
	}
	trace := make([]traceEntry, b.N*2)
	for i := 0; i < b.N*2; i++ {
		n := rng.Int63() % (4 * 128 * 1024)
		trace[i] = traceEntry{k: strconv.Itoa(int(n)), v: n}
	}
	rngMu.Unlock()

	b.ResetTimer()

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		rngMu.Lock()
		seed := rng.Intn(len(trace))
		rngMu.Unlock()

		var hit, miss int
		i := seed
		for pb.Next() {
			// use a predictable if rather than % len(trace) to eek a little more perf out
			if i >= len(trace) {
				i = 0
			}

			t := trace[i]
			if i%2 == 0 {
				l.Add(t.k, t.v)
			} else {
				if _, ok := l.Get(t.k); ok {
					hit++
				} else {
					miss++
				}
			}

			i++
		}
		// b.Logf("hit: %d miss: %d ratio: %f", hit, miss, float64(hit)/float64(miss))
	})
}
