package lru

import (
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
	l, err := NewSharded(128*1024, defaultShardCount)
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	trace := makeTrace(b.N * 2)

	b.ResetTimer()

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		seed := newRand().Intn(len(trace))

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
