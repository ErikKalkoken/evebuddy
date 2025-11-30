// Package set defines a Set type that holds a set of elements.
//
// # Key features
//
//   - Type safe
//   - Feature complete
//   - Usable zero value
//   - Supports standard iterators
//   - Prevents accidental comparison with == operator
//   - Idiomatic API (inspired by latest Go proposal for new set type)
//   - Full test coverage
//
// # Zero value
//
// The zero value of a [Set] is an empty set and ready to use.
//
// # Usage in tests
//
// For comparing sets in tests the [Set.Equal] method should be used.
// For example like this:
//
//	  if !got.Equal(want) {
//		   t.Errorf("got %q, wanted %q", got, want)
//	  }
//
// The popular approach of using reflect.DeepEqual for comparing objects
// should not be used for [Set],
// because it will incorrectly report zero sets as not equal to empty sets.
//
// For example a cleared set is empty and initialized
// and would be incorrectly reported as not equal to a zero set.
package set

import (
	"cmp"
	"fmt"
	"iter"
	"maps"
	"slices"
	"strings"
)

// A Set is an unordered collection of unique elements.
// The zero value of a Set is an empty set ready to use.
//
// Set is not safe for current use.
type Set[E comparable] struct {
	m map[E]struct{}
	_ nocmp
}

// Of returns a set of the elements v.
func Of[E comparable](v ...E) Set[E] {
	var s Set[E]
	s.Add(v...)
	return s
}

// Add adds elements v to set s.
func (s *Set[E]) Add(v ...E) {
	if s.m == nil {
		s.m = make(map[E]struct{})
	}
	for _, w := range v {
		s.m[w] = struct{}{}
	}
}

// AddSeq adds the values from seq to s.
func (s *Set[E]) AddSeq(seq iter.Seq[E]) {
	for v := range seq {
		s.Add(v)
	}
}

// Clear removes all elements from set s.
func (s *Set[E]) Clear() {
	clear(s.m)
}

// Clone returns a new set, which contains a shallow copy of all elements of set s.
func (s Set[E]) Clone() Set[E] {
	return Set[E]{m: maps.Clone(s.m)}
}

// Contains reports whether element v is in set s.
func (s Set[E]) Contains(v E) bool {
	_, ok := s.m[v]
	return ok
}

// ContainsAny reports whether any of the elements in seq are in s.
func (s *Set[E]) ContainsAny(seq iter.Seq[E]) bool {
	for v := range seq {
		if _, ok := s.m[v]; ok {
			return true
		}
	}
	return false
}

// ContainsAll reports whether all of the elements in seq are in s.
func (s *Set[E]) ContainsAll(seq iter.Seq[E]) bool {
	for v := range seq {
		if _, ok := s.m[v]; !ok {
			return false
		}
	}
	return true
}

// ContainsFunc reports whether at least one element e of s satisfies f(e).
func (s *Set[E]) ContainsFunc(f func(E) bool) bool {
	if f == nil || len(s.m) == 0 {
		return false
	}
	for v := range s.m {
		if f(v) {
			return true
		}
	}
	return false
}

// Delete removes elements v from set s.
// It returns the number of deleted elements.
// Elements that are not found in the set are ignored.
func (s Set[E]) Delete(v ...E) int {
	ln := len(s.m)
	for _, w := range v {
		delete(s.m, w)
	}
	return ln - len(s.m)
}

// DeleteFunc deletes the elements in s for which del returns true.
// It returns the number of deleted elements.
func (s *Set[E]) DeleteFunc(del func(E) bool) int {
	if del == nil {
		return 0
	}
	ln := len(s.m)
	for v := range s.m {
		if del(v) {
			delete(s.m, v)
		}
	}
	return ln - len(s.m)
}

// DeleteSeq deletes the elements in seq from s.
// Elements that are not present are ignored.
// It returns the number of deleted elements.
func (s *Set[E]) DeleteSeq(seq iter.Seq[E]) int {
	var c int
	for v := range seq {
		_, ok := s.m[v]
		if ok {
			delete(s.m, v)
			c++
		}
	}
	return c
}

// Equal reports whether sets s and u are equal.
func (s Set[E]) Equal(u Set[E]) bool {
	if len(s.m) != len(u.m) {
		return false
	}
	if len(s.m) == 0 && len(u.m) == 0 {
		return true
	}
	for v := range s.m {
		if !u.Contains(v) {
			return false
		}
	}
	return true
}

// Size returns the number of elements in set s. An empty set returns 0.
func (s Set[E]) Size() int {
	return len(s.m)
}

// String returns a string representation of set s.
// Sets are printed with curly brackets, e.g. {1 2}.
func (s Set[E]) String() string {
	var p []string
	for v := range s.All() {
		p = append(p, fmt.Sprint(v))
	}
	slices.Sort(p)
	return "{" + strings.Join(p, " ") + "}"
}

// Slice creates a new slice from the elements of set s and returns it.
//
// Note that the order of elements is undefined.
func (s Set[E]) Slice() []E {
	return slices.Collect(s.All())
}

// All returns on iterator over all elements of set s.
//
// Note that the order of elements is undefined.
func (s Set[E]) All() iter.Seq[E] {
	return maps.Keys(s.m)
}

// Collect returns a new [Set] created from the elements of iterable seq.
func Collect[E comparable](seq iter.Seq[E]) Set[E] {
	var r Set[E]
	for v := range seq {
		r.Add(v)
	}
	return r
}

// Difference constructs a new [Set] containing the elements of s that are not present in others.
func Difference[E comparable](s Set[E], others ...Set[E]) Set[E] {
	if len(others) == 0 {
		return s
	}
	var r Set[E]
	o := Union(others...)
	for v := range s.m {
		if !o.Contains(v) {
			r.Add(v)
		}
	}
	return r
}

// Intersection returns a new [Set] with elements common to all sets.
//
// When less then 2 sets are provided they will be assumed to be empty.
func Intersection[E comparable](sets ...Set[E]) Set[E] {
	var r Set[E]
	if len(sets) < 2 {
		return r
	}
L:
	for v := range sets[0].m {
		for _, s := range sets[1:] {
			if !s.Contains(v) {
				continue L
			}
		}
		r.Add(v)
	}
	return r
}

type comparableAndOrderable interface {
	cmp.Ordered
	comparable
}

// Max returns the maximal value in s. It panics if s is empty.
func Max[E comparableAndOrderable](s Set[E]) E {
	return slices.Max(s.Slice())
}

// MaxFunc returns the maximal value in s, using cmp to compare elements.
// It panics if x is empty.
// If there is more than one maximal element according to the cmp function, MaxFunc returns the first one.
func MaxFunc[E comparable](s Set[E], cmp func(a, b E) int) E {
	return slices.MaxFunc(s.Slice(), cmp)
}

// Min returns the minimal value in s. It panics if s is empty.
func Min[E comparableAndOrderable](s Set[E]) E {
	return slices.Min(s.Slice())
}

// MinFunc returns the minimal value in s, using cmp to compare elements.
// It panics if x is empty.
// If there is more than one minimal element according to the cmp function, MinFunc returns the first one.
func MinFunc[E comparable](s Set[E], cmp func(a, b E) int) E {
	return slices.MinFunc(s.Slice(), cmp)
}

// Union returns a new [Set] with the elements of all sets.
func Union[E comparable](sets ...Set[E]) Set[E] {
	var r Set[E]
	for _, s := range sets {
		for v := range s.m {
			r.Add(v)
		}
	}
	return r
}

// nocmp is an uncomparable struct. Embed this inside another struct to make it uncomparable.
type nocmp [0]func()
