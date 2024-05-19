package model

// EntityShort is a short representation of an entity.
type EntityShort[T int | int32 | int64] struct {
	ID   T
	Name string
}
