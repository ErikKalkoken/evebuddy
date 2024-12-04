package app

import (
	"fyne.io/fyne/v2"
	"github.com/ErikKalkoken/evebuddy/internal/evehtml"
	"github.com/ErikKalkoken/evebuddy/internal/eveicon"
)

const (
	EveTypeAssetSafetyWrap             = 60
	EveTypeIHUB                        = 32458
	EveTypeInterplanetaryConsolidation = 2495
	EveTypeSolarSystem                 = 5
	EveTypeTCU                         = 32226
	EveTypePlanetTemperate             = 11
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

// Icon returns the icon for a type from the eveicon package
// and whether and icon exists for this type.
func (ep EveType) Icon() (fyne.Resource, bool) {
	if ep.IconID == 0 {
		return nil, false
	}
	res, ok := eveicon.GetResourceByIconID(ep.IconID)
	if !ok {
		return nil, false
	}
	return res, true
}
