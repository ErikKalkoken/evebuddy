package character

import "iter"

// Count adds an item counter to an iterator.
// This allows to range over an iterator with an index similar to ranging over a slice.
func Count[T any](it iter.Seq[T], start int) iter.Seq2[int, T] {
	return func(yield func(int, T) bool) {
		for v := range it {
			if !yield(start, v) {
				return
			}
			start++
		}
	}
}
