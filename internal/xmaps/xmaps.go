// Package xmaps provides an extension to the maps package.
package xmaps

import (
	"cmp"
	"iter"
	"maps"
	"slices"
)

// OrderedMap represents a variant of map with ordered keys.
type OrderedMap[K cmp.Ordered, V any] map[K]V

// All returns an iterator over key-value pairs from m. The keys are sorted.
func (o OrderedMap[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(k K, v V) bool) {
		keys := slices.Sorted(maps.Keys(o))
		for _, key := range keys {
			if !yield(key, o[key]) {
				return
			}
		}
	}
}

// Invert returns a new map with the keys and values of m swapped.
// If m contains duplicate values, an unspecified one of their keys is kept.
func Invert[M ~map[K]V, K, V comparable](m M) map[V]K {
	out := make(map[V]K, len(m))
	for k, v := range m {
		out[v] = k
	}
	return out
}

// Merge returns a new map with all entries from ms; later maps win on
// duplicate keys. The merge is shallow.
func Merge[M ~map[K]V, K comparable, V any](ms ...M) M {
	size := 0
	for _, m := range ms {
		size += len(m)
	}
	out := make(map[K]V, size)
	for _, m := range ms {
		maps.Copy(out, m)
	}
	return out
}
