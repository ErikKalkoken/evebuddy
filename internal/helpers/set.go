// Package helpers contains generic helper functions and types,
// which can be used across all packages.
package helpers

// A generic set type
type Set[T comparable] struct {
	items map[T]struct{}
}

// Add an element to a set
func (s *Set[T]) Add(v T) {
	s.items[v] = struct{}{}
}

// Remove an element from a set
func (s *Set[T]) Remove(v T) {
	delete(s.items, v)
}

// Clear a set
func (s *Set[T]) Clear() {
	s.items = make(map[T]struct{})
}

// Return the size of a set
func (s *Set[T]) Size() int {
	return len(s.items)
}

// Report if item is in this set
func (s *Set[T]) Has(item T) bool {
	_, ok := s.items[item]
	return ok
}

// Convert a set to a slice
func (s *Set[T]) ToSlice() []T {
	slice := make([]T, s.Size())
	for v := range s.items {
		slice = append(slice, v)
	}
	return slice
}

// Return new empty set
func NewSet[T comparable]() *Set[T] {
	var s Set[T]
	s.items = make(map[T]struct{})
	return &s
}

// Return new set created from a slice
func NewSetFromSlice[T comparable](slice []T) *Set[T] {
	var s Set[T]
	s.items = make(map[T]struct{}, len(slice))
	for _, el := range slice {
		s.items[el] = struct{}{}
	}
	return &s
}
