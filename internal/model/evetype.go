package model

import "github.com/ErikKalkoken/evebuddy/internal/evehtml"

const (
	EveTypeAssetSafetyWrap = 60
	EveTypeSolarSystem     = 5
)

// EveType is a type in Eve Online.
type EveType struct {
	ID             int32
	Group          *EveGroup
	Capacity       float32
	Description    string
	GraphicID      int32
	IconID         int32
	IsPublished    bool
	MarketGroupID  int32
	Mass           float32
	Name           string
	PackagedVolume float32
	PortionSize    int
	Radius         float32
	Volume         float32
}

// BodyPlain returns a mail's body as plain text.
func (et EveType) DescriptionPlain() string {
	return evehtml.ToPlain(et.Description)
}

func (et EveType) IsBlueprint() bool {
	return et.Group.Category.ID == EveCategoryBlueprint
}

func (et EveType) IsShip() bool {
	return et.Group.Category.ID == EveCategoryShip
}

func (et EveType) IsSKIN() bool {
	return et.Group.Category.ID == EveCategorySKINs
}

func (et EveType) HasFuelBay() bool {
	if et.Group.Category.ID != EveCategoryShip {
		return false
	}
	switch et.Group.ID {
	case EveGroupBlackOps,
		EveGroupCapitalIndustrialShip,
		EveGroupCarrier,
		EveGroupDreadnought,
		EveGroupForceAuxiliary,
		EveGroupJumpFreighter,
		EveGroupSuperCarrier,
		EveGroupTitan:
		return true
	}
	return false
}

func (et EveType) HasRender() bool {
	switch et.Group.Category.ID {
	case
		EveCategoryDrone,
		EveCategoryDeployable,
		EveCategoryFighter,
		EveCategoryShip,
		EveCategoryStation,
		EveCategoryStructure,
		EveCategoryStarbase:
		return true
	}
	return false
}
