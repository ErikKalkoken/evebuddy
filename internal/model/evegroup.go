package model

// EveGroup is a group in Eve Online.
type EveGroup struct {
	ID          int32
	Category    *EveCategory
	IsPublished bool
	Name        string
}
