// Package xiter provides common iterator helper functions.
package xiter

import (
	"iter"
	"slices"
)

// Chain returns an iterator that returns the elements of each seq one after the other.
func Chain[T any](seqs ...iter.Seq[T]) iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, seq := range seqs {
			for v := range seq {
				if !yield(v) {
					return
				}
			}
		}
	}
}

// Count adds an item counter to an iterator.
// This allows to range over a sequence seq with an index similar to ranging over a slice.
func Count[T any](seq iter.Seq[T], start int) iter.Seq2[int, T] {
	return func(yield func(int, T) bool) {
		for v := range seq {
			if !yield(start, v) {
				return
			}
			start++
		}
	}
}

// Filter returns an iterator over the items of sequence seq where applied f returns true.
func Filter[T any](seq iter.Seq[T], f func(T) bool) iter.Seq[T] {
	return func(yield func(T) bool) {
		for v := range seq {
			if f(v) {
				if !yield(v) {
					return
				}
			}
		}
	}
}

// FilterSlice returns an iterator over the elements of s where applied f returns true.
func FilterSlice[S ~[]E, E any](s S, f func(E) bool) iter.Seq[E] {
	return Filter(slices.Values(s), f)
}

// Map returns an iterator that maps each element X of sequence seq to element Y through applying f.
func Map[X, Y any](seq iter.Seq[X], f func(X) Y) iter.Seq[Y] {
	return func(yield func(Y) bool) {
		for v := range seq {
			if !yield(f(v)) {
				return
			}
		}
	}
}

// Map returns an iterator that maps each element of slice s to an element Y through applying f.
func MapSlice[S ~[]X, X any, Y any](s S, f func(X) Y) iter.Seq[Y] {
	return Map(slices.Values(s), f)
}

// Map returns an iterator that maps each element of slice s to elements K, V through applying f.
func MapSlice2[X, K, V any](s []X, f func(X) (K, V)) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for _, v := range s {
			if !yield(f(v)) {
				return
			}
		}
	}
}
