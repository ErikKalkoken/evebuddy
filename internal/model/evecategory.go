package model

const (
	EveCategoryIDBlueprint = 9
	EveCategoryIDShip      = 6
	EveCategoryIDSkill     = 16
	EveCategoryIDSKINs     = 91
)

// EveCategory is a category in Eve Online.
type EveCategory struct {
	ID          int32
	IsPublished bool
	Name        string
}
