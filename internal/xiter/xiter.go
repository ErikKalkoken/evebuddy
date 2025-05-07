// Package xiter provides helper functions with iterators.
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

// Filter returns an iterator over the items of iterator it where applied f returns true.
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

// FilterSlice returns an iterator over the elements of s where applied f returns true.
func FilterSlice[S ~[]E, E any](s S, f func(E) bool) iter.Seq[E] {
	return Filter(slices.Values(s), f)
}

// Map returns an iterator that applies function f to every element of iterator it, yielding the results.
func Map[X, Y any](it iter.Seq[X], f func(X) Y) iter.Seq[Y] {
	return func(yield func(Y) bool) {
		for v := range it {
			if !yield(f(v)) {
				return
			}
		}
	}
}

// MapSlice returns an iterator that aplies function f to every element of a slice s, yielding the results.
func MapSlice[S ~[]X, X any, Y any](s S, f func(X) Y) iter.Seq[Y] {
	return Map(slices.Values(s), f)
}

// Chain returns an iterator that returns the all elements of each seq one after the other.
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
