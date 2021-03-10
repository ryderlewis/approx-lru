approx-lru
==========

This provides the `lru` package which implements a fixed-size
thread safe LRU cache. It is based on [hashicorp/golang-lru](https://github.com/hashicorp/golang-lru),
which is in turn based on the cache in Groupcache.

The major difference in `bobby-stripe/approx-lru` is that instead of strictly ordering all items in the
cache in a doubly-linked list ([like `golang-lru` does](https://github.com/hashicorp/golang-lru/blob/master/simplelru/lru.go#L14)),
`approx-lru` contains a fixed-sized array of cache entries with a `lastUsed` timestamp.  If the cache is
full and needs to evict an item, we randomly probe several entries, and evict the oldest.  This is the
same strategy as [Redis's allkeys-lru](https://redis.io/topics/lru-cache), and approximates a perfect LRU
with less bookkeeping and memory overhead (each linked list entry in Go is
[40-bytes](https://golang.org/src/container/list/list.go?s=406:874#L5), in addition to the data).

Documentation
=============

Full docs are available on [Godoc](http://godoc.org/github.com/bobby-stripe/approx-lru)

Example
=======

Using the LRU is very simple:

```go
l, _ := New(128)
for i := 0; i < 256; i++ {
    l.Add(i, nil)
}
if l.Len() != 128 {
    panic(fmt.Sprintf("bad len: %v", l.Len()))
}
```
