package model

const (
	EveCategoryBlueprint  = 9
	EveCategoryDrone      = 18
	EveCategoryDeployable = 22
	EveCategoryFighter    = 87
	EveCategoryShip       = 6
	EveCategorySkill      = 16
	EveCategorySKINs      = 91
	EveCategoryStarbase   = 23
	EveCategoryStation    = 3
	EveCategoryStructure  = 65
)

// EveCategory is a category in Eve Online.
type EveCategory struct {
	ID          int32
	IsPublished bool
	Name        string
}
