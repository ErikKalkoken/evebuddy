// Package set implements a set type.
package set

import "fmt"

// A set of values.
type Set[T comparable] struct {
	values map[T]struct{}
}

func (s *Set[T]) String() string {
	return fmt.Sprint(s.ToSlice())
}

// Add adds an element to the set
func (s *Set[T]) Add(v T) {
	s.values[v] = struct{}{}
}

// Remove removes an element from a set.
// It does nothing when the element doesn't exist.
func (s *Set[T]) Remove(v T) {
	delete(s.values, v)
}

// Clear removes all elements from a set.
func (s *Set[T]) Clear() {
	s.values = make(map[T]struct{})
}

// Size returns the number of elements in a set.
func (s *Set[T]) Size() int {
	return len(s.values)
}

// Has reports wether an item is in this set.
func (s *Set[T]) Has(item T) bool {
	_, ok := s.values[item]
	return ok
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

// New returns a new set.
func New[T comparable]() *Set[T] {
	return NewFromSlice([]T{})
}

// NewFromSlice returns a new set from a slice.
func NewFromSlice[T comparable](slice []T) *Set[T] {
	var s Set[T]
	s.values = make(map[T]struct{}, len(slice))
	for _, el := range slice {
		s.values[el] = struct{}{}
	}
	return &s
}