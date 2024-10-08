package app

const (
	TimeDefaultFormat = "2006.01.02 15:04"
)

// EntityShort is a short representation of an entity.
type EntityShort[T comparable] struct {
	ID   T
	Name string
}

// Position is a position in 3D space.
type Position struct {
	X float64
	Y float64
	Z float64
}
