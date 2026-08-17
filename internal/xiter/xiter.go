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
			if f(v) && !yield(v) {
				return
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

// MapSlice returns an iterator that maps each element of slice s to an element Y through applying f.
func MapSlice[S ~[]X, X any, Y any](s S, f func(X) Y) iter.Seq[Y] {
	return Map(slices.Values(s), f)
}

// MapSlice2 returns an iterator that maps each element of slice s to elements K, V through applying f.
func MapSlice2[S ~[]X, X, K, V any](s S, f func(X) (K, V)) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for _, v := range s {
			if !yield(f(v)) {
				return
			}
		}
	}
}

// Reduce applies f cumulatively to the elements of seq, from left to right.
// Returns zero value of T if seq is empty.
func Reduce[T any](seq iter.Seq[T], f func(T, T) T) T {
	var accumulator T
	first := true
	for v := range seq {
		if first {
			accumulator = v
			first = false
		} else {
			accumulator = f(accumulator, v)
		}
	}
	return accumulator
}

// Unique returns a lazy iterator yielding only unique elements in order of appearance.
func Unique[T comparable](seq iter.Seq[T]) iter.Seq[T] {
	return func(yield func(T) bool) {
		seen := make(map[T]struct{})
		for v := range seq {
			if _, exists := seen[v]; !exists {
				seen[v] = struct{}{}
				if !yield(v) {
					return
				}
			}
		}
	}
}
