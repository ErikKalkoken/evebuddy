// Package set defines a generic set type
package set

// Set is a generic set.
type Set[T comparable] struct {
	items map[T]struct{}
}

// Add adds an element to the set
func (s *Set[T]) Add(v T) {
	s.items[v] = struct{}{}
}

// Remove removes an element from a set.
// It does nothing when the element doesn't exist.
func (s *Set[T]) Remove(v T) {
	delete(s.items, v)
}

// Clear removes all elements from a set.
func (s *Set[T]) Clear() {
	s.items = make(map[T]struct{})
}

// Size returns the number of elements in a set.
func (s *Set[T]) Size() int {
	return len(s.items)
}

// Has reports wether an item is in this set.
func (s *Set[T]) Has(item T) bool {
	_, ok := s.items[item]
	return ok
}

// ToSlice converts a set to a slice and returns it.
// Note that the elements in the slice have no defined order.
func (s *Set[T]) ToSlice() []T {
	slice := make([]T, 0, s.Size())
	for v := range s.items {
		slice = append(slice, v)
	}
	return slice
}

// Union returns a new set containing the combined elements from both sets.
func (s *Set[T]) Union(other *Set[T]) *Set[T] {
	n := New([]T{})
	for v := range s.items {
		n.Add(v)
	}

	for v := range other.items {
		n.Add(v)
	}
	return n
}

// Intersect returns a new set which contains elements found in both sets only.
func (s *Set[T]) Intersect(other *Set[T]) *Set[T] {
	n := New([]T{})
	for v := range s.items {
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
	n := New([]T{})
	for v := range s.items {
		if other.Has(v) {
			continue
		}
		n.Add(v)
	}
	return n
}

// New returns a new set from a slice.
func New[T comparable](slice []T) *Set[T] {
	var s Set[T]
	s.items = make(map[T]struct{}, len(slice))
	for _, el := range slice {
		s.items[el] = struct{}{}
	}
	return &s
}
