package app

import (
	"cmp"

	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	npcCorporationIDBegin = 1_000_000
	npcCorporationIDEnd   = 2_000_000
	npcCharacterIDBegin   = 3_000_000
	npcCharacterIDEnd     = 4_000_000
)

// An EveEntity in EveOnline.
type EveEntity struct {
	Category EveEntityCategory
	ID       int32
	Name     string
}

func (ee EveEntity) CategoryDisplay() string {
	titler := cases.Title(language.English)
	return titler.String(ee.Category.String())
}

// IsValid reports whether an entity is valid.
func (ee EveEntity) IsValid() bool {
	return ee.Category.IsKnown()
}

// IsCharacter reports whether an entity is a character.
func (ee EveEntity) IsCharacter() bool {
	return ee.Category == EveEntityCharacter
}

// IsNPC reports whether an entity is a NPC.
//
// This function only works for characters and corporations and returns an empty value for anything else..
func (ee EveEntity) IsNPC() optional.Optional[bool] {
	switch ee.Category {
	case EveEntityCharacter:
		return optional.New(ee.ID >= npcCharacterIDBegin && ee.ID < npcCharacterIDEnd)
	case EveEntityCorporation:
		return optional.New(ee.ID >= npcCorporationIDBegin && ee.ID < npcCorporationIDEnd)
	}
	return optional.Optional[bool]{}
}

func (ee *EveEntity) Compare(other *EveEntity) int {
	return cmp.Compare(ee.Name, other.Name)
}

type EveEntityCategory int

// Supported categories of EveEntity
const (
	EveEntityUndefined EveEntityCategory = iota
	EveEntityAlliance
	EveEntityCharacter
	EveEntityConstellation
	EveEntityCorporation
	EveEntityFaction
	EveEntityInventoryType
	EveEntityMailList
	EveEntityRegion
	EveEntitySolarSystem
	EveEntityStation
	EveEntityUnknown
)

// IsKnown reports whether a category is known.
func (eec EveEntityCategory) IsKnown() bool {
	return eec != EveEntityUndefined && eec != EveEntityUnknown
}

func (eec EveEntityCategory) String() string {
	switch eec {
	case EveEntityUndefined:
		return "undefined"
	case EveEntityAlliance:
		return "alliance"
	case EveEntityCharacter:
		return "character"
	case EveEntityConstellation:
		return "constellation"
	case EveEntityCorporation:
		return "corporation"
	case EveEntityFaction:
		return "faction"
	case EveEntityInventoryType:
		return "inventory type"
	case EveEntityMailList:
		return "mailing list"
	case EveEntityRegion:
		return "region"
	case EveEntitySolarSystem:
		return "solar system"
	case EveEntityStation:
		return "station"
	case EveEntityUnknown:
		return "unknown"
	default:
		return "?"
	}
}

// ToEveImage returns the corresponding category string for the EveImage service.
// Will return an empty string when category is not supported.
func (eec EveEntityCategory) ToEveImage() string {
	switch eec {
	case EveEntityAlliance:
		return "alliance"
	case EveEntityCharacter:
		return "character"
	case EveEntityCorporation:
		return "corporation"
	case EveEntityFaction:
		return "faction"
	case EveEntityInventoryType:
		return "inventory_type"
	default:
		return ""
	}
}
