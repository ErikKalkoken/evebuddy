package model

import "github.com/ErikKalkoken/evebuddy/internal/eveonline/converter"

const (
	EveCategoryIDShip  = 6
	EveCategoryIDSkill = 16

	EveTypeIDSolarSystem     = 5
	EveTypeIDAssetSafetyWrap = 60

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
)

// EveCategory is a category in Eve Online.
type EveCategory struct {
	ID          int32
	IsPublished bool
	Name        string
}

// EveGroup is a group in Eve Online.
type EveGroup struct {
	ID          int32
	Category    *EveCategory
	IsPublished bool
	Name        string
}

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
func (t *EveType) DescriptionPlain() string {
	return converter.EveHTMLToPlain(t.Description)
}

// EveRegion is a region in Eve Online.
type EveRegion struct {
	Description string
	ID          int32
	Name        string
}

// EveConstellation is a constellation in Eve Online.
type EveConstellation struct {
	ID     int32
	Name   string
	Region *EveRegion
}

// EveSolarSystem is a solar system in Eve Online.
type EveSolarSystem struct {
	Constellation  *EveConstellation
	ID             int32
	Name           string
	SecurityStatus float64
}

// EveRace is a race in Eve Online.
type EveRace struct {
	Description string
	Name        string
	ID          int32
}

// FactionID returns the faction ID of a race.
func (r *EveRace) FactionID() (int32, bool) {
	m := map[int32]int32{
		1: 500001,
		2: 500002,
		4: 500003,
		8: 500004,
	}
	factionID, ok := m[r.ID]
	return factionID, ok
}

// Position is a position in 3D space.
type Position struct {
	X float64
	Y float64
	Z float64
}

// EntityShort is a short representation of an entity.
type EntityShort[T int | int32 | int64] struct {
	ID   T
	Name string
}
