package storage

// convertNumericSlice converts the type of a numeric slice and returns the new one.
func convertNumericSlice[T int64 | int32 | int, V int64 | int32 | int](s []T) []V {
	s2 := make([]V, len(s))
	for i, v := range s {
		s2[i] = V(v)
	}
	return s2
}
