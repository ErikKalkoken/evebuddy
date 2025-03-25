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
