// Package set implements a generic set container type.
package set

import (
	"errors"
	"fmt"
	"iter"
	"maps"
)

var ErrNotFound = errors.New("not found")

// Set is a container for a set of values.
type Set[T comparable] map[T]struct{}

// New returns a new Set.
func New[T comparable](vals ...T) Set[T] {
	s := Set[T]{}
	for _, v := range vals {
		s[v] = struct{}{}
	}
	return s
}

// NewFromSlice returns a new set from the elements of a slice.
func NewFromSlice[T comparable](slice []T) Set[T] {
	return New(slice...)
}

// Add adds an element to the set
func (s Set[T]) Add(v T) {
	s[v] = struct{}{}
}

// Clear removes all elements from a set.
func (s Set[T]) Clear() {
	for k := range s {
		delete(s, k)
	}
}

// Clone returns a clone of a set.
func (s Set[T]) Clone() Set[T] {
	return New(s.ToSlice()...)
}

// Contains reports wether an item is in this set.
func (s Set[T]) Contains(v T) bool {
	_, ok := s[v]
	return ok
}

// Difference returns a new set which elements from current set,
// that does not exist in other set.
func (s Set[T]) Difference(other Set[T]) Set[T] {
	n := New[T]()
	for v := range s {
		if !other.Contains(v) {
			n.Add(v)
		}
	}
	return n
}

// Equal reports wether two sets are equal.
func (s Set[T]) Equal(other Set[T]) bool {
	if s.Size() != other.Size() {
		return false
	}
	d := s.Difference(other)
	x := d.Size()
	return x == 0
}

// Intersect returns a new set which contains elements found in both sets only.
func (s Set[T]) Intersect(other Set[T]) Set[T] {
	n := New[T]()
	for v := range s {
		if other.Contains(v) {
			n.Add(v)
		}
	}
	return n
}

// IsDisjoint reports whether a set has any elements in common with another set.
func (s Set[T]) IsDisjoint(other Set[T]) bool {
	x := s.Intersect(other)
	return x.Size() == 0
}

// IsSubset reports whether a set is the subset of another set.
func (s Set[T]) IsSubset(other Set[T]) bool {
	x := s.Difference(other)
	return x.Size() == 0
}

// IsSuperset reports whether a set is the superset of another set.
func (s Set[T]) IsSuperset(other Set[T]) bool {
	x := other.Difference(s)
	return x.Size() == 0
}

// Remove removes an element from a set.
// It does nothing when the element doesn't exist.
func (s Set[T]) Remove(v T) {
	delete(s, v)
}

// Pop removes a random element from a set and returns it.
// Or if the set is empty an error is returned.
func (s Set[T]) Pop() (T, error) {
	for v := range s {
		delete(s, v)
		return v, nil
	}
	var x T
	return x, ErrNotFound
}

// Size returns the number of elements in a set.
func (s Set[T]) Size() int {
	return len(s)
}

func (s Set[T]) String() string {
	return fmt.Sprint(s.ToSlice())
}

// ToSlice converts a set to a slice and returns it.
// Note that the elements in the slice have no defined order.
func (s Set[T]) ToSlice() []T {
	slice := make([]T, 0, s.Size())
	for v := range s {
		slice = append(slice, v)
	}
	return slice
}

// Union returns a new set containing the combined elements from both sets.
func (s Set[T]) Union(other Set[T]) Set[T] {
	n := s.Clone()
	for v := range other {
		n.Add(v)
	}
	return n
}

// Values returns on iterator over all elements of a set.
// Note that sets are unordered, so elements will be returned in no particular order.
func (s Set[T]) Values() iter.Seq[T] {
	return maps.Keys(s)
}
