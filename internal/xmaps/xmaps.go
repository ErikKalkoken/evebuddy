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
