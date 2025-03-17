package storage

import "golang.org/x/exp/constraints"

// convertNumericSlice converts the type of a numeric slice and returns the new one.
func convertNumericSlice[Y constraints.Integer, X constraints.Integer](s []X) []Y {
	s2 := make([]Y, len(s))
	for i, v := range s {
		s2[i] = Y(v)
	}
	return s2
}
