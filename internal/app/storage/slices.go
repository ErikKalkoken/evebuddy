package storage

import (
	"github.com/ErikKalkoken/go-set"
	"golang.org/x/exp/constraints"
)

// convertNumericSlice converts the type of a numeric slice and returns the new one.
func convertNumericSlice[Y constraints.Integer, X constraints.Integer](s []X) []Y {
	s2 := make([]Y, len(s))
	for i, v := range s {
		s2[i] = Y(v)
	}
	return s2
}

func convertNumericSet[Y constraints.Integer, X constraints.Integer](s set.Set[X]) []Y {
	s2 := make([]Y, 0)
	for v := range s.All() {
		s2 = append(s2, Y(v))
	}
	return s2
}
