package lru

import (
	"testing"
	"unsafe"
)

func TestNewSharded(t *testing.T) {

}

func TestShardSize(t *testing.T) {
	if 128 != unsafe.Sizeof(shard[int]{}) {
		t.Fatalf("expected shard to be 128-bytes in size")
	}
}

func TestShardedCacheSize(t *testing.T) {
	expected := uintptr(128 * shardCount)
	actual := unsafe.Sizeof(*ShardedCache[int]{}.shards)
	if expected != actual {
		t.Fatalf("expected %d == %d", expected, actual)
	}
}
