// Package xiter provides helper functions for iterators.
package xiter

import (
	"iter"
	"slices"
)

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

// Filter returns an iterator over the filtered items of an iterator.
func Filter[T any](it iter.Seq[T], f func(T) bool) iter.Seq[T] {
	return func(yield func(T) bool) {
		for v := range it {
			if f(v) {
				if !yield(v) {
					return
				}
			}
		}
	}
}

// FilterSlice returns an iterator over the elements of s where f is true.
func FilterSlice[S ~[]E, E any](s S, f func(E) bool) iter.Seq[E] {
	return Filter(slices.Values(s), f)
}
