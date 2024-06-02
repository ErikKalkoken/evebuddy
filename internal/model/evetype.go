package model

import "github.com/ErikKalkoken/evebuddy/internal/eveonline/converter"

const (
	EveDogmaAttributeIDImplantSlot          = 331
	EveDogmaAttributeIDPrimarySkillID       = 182
	EveDogmaAttributeIDPrimarySkillLevel    = 277
	EveDogmaAttributeIDSecondarySkillID     = 183
	EveDogmaAttributeIDSecondarySkillLevel  = 278
	EveDogmaAttributeIDTertiarySkillID      = 184
	EveDogmaAttributeIDTertiarySkillLevel   = 279
	EveDogmaAttributeIDQuaternarySkillID    = 1285
	EveDogmaAttributeIDQuaternarySkillLevel = 1286
	EveDogmaAttributeIDQuinarySkillID       = 1289
	EveDogmaAttributeIDQuinarySkillLevel    = 1287
	EveDogmaAttributeIDSenarySkillID        = 1290
	EveDogmaAttributeIDSenarySkillLevel     = 1288

	EveTypeIDAssetSafetyWrap = 60
	EveTypeIDSolarSystem     = 5
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
	return converter.EveHTMLToPlain(et.Description)
}

func (et EveType) IsBlueprint() bool {
	return et.Group.Category.ID == EveCategoryIDBlueprint
}

func (et EveType) IsSKIN() bool {
	return et.Group.Category.ID == EveCategoryIDSKINs
}
