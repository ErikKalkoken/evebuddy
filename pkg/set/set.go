// Package set implements a generic set container type.
package set

import (
	"fmt"
	"iter"
	"maps"
)

// Set is a container for a set of values.
type Set[T comparable] struct {
	values map[T]struct{}
}

// New returns a new Set.
func New[T comparable]() *Set[T] {
	s := &Set[T]{values: make(map[T]struct{})}
	return s
}

// NewFromSlice returns a new set from a slice.
func NewFromSlice[T comparable](slice []T) *Set[T] {
	s := New[T]()
	for _, el := range slice {
		s.values[el] = struct{}{}
	}
	return s
}

// Add adds an element to the set
func (s *Set[T]) Add(v T) {
	s.values[v] = struct{}{}
}

// All returns on iterator over all elements of a set.
// Note that sets are unordered, so elements will be returned in no particular order.
func (s *Set[T]) All() iter.Seq[T] {
	return maps.Keys(s.values)
}

// Clear removes all elements from a set.
func (s *Set[T]) Clear() {
	s.values = make(map[T]struct{})
}

// Difference returns a new set which elements from current set,
// that does not exist in other set.
func (s *Set[T]) Difference(other *Set[T]) *Set[T] {
	n := NewFromSlice([]T{})
	for v := range s.values {
		if other.Has(v) {
			continue
		}
		n.Add(v)
	}
	return n
}

// Equal reports wether two sets are equal.
func (s *Set[T]) Equal(other *Set[T]) bool {
	if s.Size() != other.Size() {
		return false
	}
	d := s.Difference(other)
	x := d.Size()
	return x == 0
}

// Has reports wether an item is in this set.
func (s *Set[T]) Has(item T) bool {
	_, ok := s.values[item]
	return ok
}

// Intersect returns a new set which contains elements found in both sets only.
func (s *Set[T]) Intersect(other *Set[T]) *Set[T] {
	n := NewFromSlice([]T{})
	for v := range s.values {
		if !other.Has(v) {
			continue
		}
		n.Add(v)
	}
	return n
}

// Remove removes an element from a set.
// It does nothing when the element doesn't exist.
func (s *Set[T]) Remove(v T) {
	delete(s.values, v)
}

// Size returns the number of elements in a set.
func (s *Set[T]) Size() int {
	return len(s.values)
}

func (s *Set[T]) String() string {
	return fmt.Sprint(s.ToSlice())
}

// ToSlice converts a set to a slice and returns it.
// Note that the elements in the slice have no defined order.
func (s *Set[T]) ToSlice() []T {
	slice := make([]T, 0, s.Size())
	for v := range s.values {
		slice = append(slice, v)
	}
	return slice
}

// Union returns a new set containing the combined elements from both sets.
func (s *Set[T]) Union(other *Set[T]) *Set[T] {
	n := NewFromSlice([]T{})
	for v := range s.values {
		n.Add(v)
	}
	for v := range other.values {
		n.Add(v)
	}
	return n
}
