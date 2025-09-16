// Package xslices contains helper functions for slices.
package xslices

// Filter returns a new slice containing the elements where applied function f returned true.
func Filter[S ~[]E, E any](s S, f func(E) bool) []E {
	s2 := make([]E, 0)
	for _, v := range s {
		if f(v) {
			s2 = append(s2, v)
		}
	}
	return s2
}

// Map returns a new slice with the results of function f applied to each element.
func Map[S ~[]X, X any, Y any](s S, f func(X) Y) []Y {
	s2 := make([]Y, len(s))
	for i, v := range s {
		s2[i] = f(v)
	}
	return s2
}

// Reduce applies f cumulatively to the elements of s, from left to right,
// so as to reduce the slice to a single value.
// If the slice is empty it will return the zero value of E.
func Reduce[S ~[]E, E any](s S, f func(E, E) E) E {
	var x E
	for i, v := range s {
		if i == 0 {
			x = v
			continue
		}
		x = f(x, v)
	}
	return x
}

// Deduplicate returns a new slice where all duplicate elements have been removed.
// The order of the elements is not changed, but the new slice can be shorter.
func Deduplicate[S ~[]E, E comparable](s S) []E {
	seen := make(map[E]bool)
	s2 := make([]E, 0)
	for _, v := range s {
		if seen[v] {
			continue
		}
		s2 = append(s2, v)
		seen[v] = true
	}
	return s2
}
