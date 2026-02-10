package storage

import (
	"github.com/ErikKalkoken/go-set"
	"golang.org/x/exp/constraints"
)

func convertNumericSet[Y constraints.Integer, X constraints.Integer](s set.Set[X]) []Y {
	s2 := make([]Y, 0)
	for v := range s.All() {
		s2 = append(s2, Y(v))
	}
	return s2
}
