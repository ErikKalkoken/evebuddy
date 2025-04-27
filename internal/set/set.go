// Package set provides a generic set container type.
package set

import (
	"errors"
	"fmt"
	"iter"
	"maps"
)

var ErrNotFound = errors.New("not found")

// Set is a container for a set of values of type T.
//
// The zero value of Set is an empty set and ready for use.
//
// For comparing sets you must use the[Set.Equal] method.
//
// Sets must not be used concurrently.
type Set[T comparable] struct {
	m map[T]struct{}
}

// New returns a new set.
// It can optionally be initialized with a list of values vals.
func New[T comparable](vals ...T) Set[T] {
	s := Set[T]{}
	s.init()
	for _, v := range vals {
		s.m[v] = struct{}{}
	}
	return s
}

// NewFromSlice returns a new set created from the elements of slice x.
func NewFromSlice[T comparable](x []T) Set[T] {
	return New(x...)
}

// Collect returns a new set created from the elements of iterable seq.
func Collect[T comparable](seq iter.Seq[T]) Set[T] {
	s := New[T]()
	for v := range seq {
		s.Add(v)
	}
	return s
}

// Add adds element v to set s.
func (s *Set[T]) Add(v T) {
	if s.m == nil {
		s.init()
	}
	s.m[v] = struct{}{}
}

// init initializes set s.
func (s *Set[T]) init() {
	s.m = make(map[T]struct{})
}

// Clear removes all elements from set s.
func (s Set[T]) Clear() {
	if s.m == nil {
		return
	}
	for k := range s.m {
		delete(s.m, k)
	}
}

// Clone returns a new set, which is a clone clone of a set s.
func (s Set[T]) Clone() Set[T] {
	return New(s.ToSlice()...)
}

// Contains reports whether element v is in set s.
func (s Set[T]) Contains(v T) bool {
	_, ok := s.m[v]
	return ok
}

// Difference returns a new set with the difference between the sets s and u.
func (s Set[T]) Difference(u Set[T]) Set[T] {
	n := New[T]()
	for v := range s.m {
		if !u.Contains(v) {
			n.Add(v)
		}
	}
	return n
}

// Discard discards element v from set s if it is present.
// It does nothing when s does not contain v or when s is empty.
func (s Set[T]) Discard(v T) {
	delete(s.m, v)
}

// Equal reports whether sets s and u are equal.
func (s Set[T]) Equal(u Set[T]) bool {
	if s.Size() != u.Size() {
		return false
	}
	if s.IsEmpty() && u.IsEmpty() {
		return true
	}
	for v := range s.m {
		if !u.Contains(v) {
			return false
		}
	}
	return true
}

// Intersect returns a new set which contains the intersection between the sets s and u.
func (s Set[T]) Intersect(u Set[T]) Set[T] {
	n := New[T]()
	for v := range s.m {
		if u.Contains(v) {
			n.Add(v)
		}
	}
	return n
}

// IsDisjoint reports whether set s has any elements in common with set u.
func (s Set[T]) IsDisjoint(u Set[T]) bool {
	x := s.Intersect(u)
	return x.Size() == 0
}

// IsEmpty reports whether set s is empty.
func (s Set[T]) IsEmpty() bool {
	return s.Size() == 0
}

// IsSubset reports whether set s is the subset of set u.
func (s Set[T]) IsSubset(u Set[T]) bool {
	x := s.Difference(u)
	return x.Size() == 0
}

// IsSuperset reports whether set s is the superset of set u.
func (s Set[T]) IsSuperset(u Set[T]) bool {
	x := u.Difference(s)
	return x.Size() == 0
}

// MustPop removes a random element from set s and returns it when s is not empty.
// It panics if s is empty.
func (s Set[T]) MustPop() T {
	x, err := s.Pop()
	if err != nil {
		panic(err)
	}
	return x
}

// MustRemove removes element v from set s.
// It panics if s does not contain v.
func (s Set[T]) MustRemove(v T) {
	err := s.Remove(v)
	if err != nil {
		panic(err)
	}
}

// Pop removes a random element from set s and returns it when s is not empty.
// When s is empty it returns [ErrNotFound].
func (s Set[T]) Pop() (T, error) {
	for v := range s.m {
		delete(s.m, v)
		return v, nil
	}
	var x T
	return x, ErrNotFound
}

// Remove removes element v from set s.
// Returns [ErrNotFound] if v is not present.
func (s Set[T]) Remove(v T) error {
	if !s.Contains(v) {
		return ErrNotFound
	}
	s.Discard(v)
	return nil
}

// Size returns the number of elements in set s. An empty set returns 0.
func (s Set[T]) Size() int {
	return len(s.m)
}

// String returns a string representation of set s.
func (s Set[T]) String() string {
	return fmt.Sprint(s.ToSlice())
}

// Difference returns a new set with the difference between the sets s and u.
func (s Set[T]) SymetricDifference(u Set[T]) Set[T] {
	return s.Union(u).Difference(s.Intersect(u))
}

// ToSlice creates a new slice from the elements of set s and returns it.
//
// Note that the order of elements is undefined.
func (s Set[T]) ToSlice() []T {
	slice := make([]T, 0, s.Size())
	for v := range s.m {
		slice = append(slice, v)
	}
	return slice
}

// Union returns a new set containing the combined elements from the sets s and u.
func (s Set[T]) Union(u Set[T]) Set[T] {
	n := s.Clone()
	for v := range u.m {
		n.Add(v)
	}
	return n
}

// Values returns on iterator over all elements of set s.
//
// Note that the order of elements is undefined.
func (s Set[T]) Values() iter.Seq[T] {
	return maps.Keys(s.m)
}
