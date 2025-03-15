package app

import (
	"cmp"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

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

var titler = cases.Title(language.English)

// An EveEntity in EveOnline.
type EveEntity struct {
	Category EveEntityCategory
	ID       int32
	Name     string
}

func (ee EveEntity) CategoryDisplay() string {
	return titler.String(ee.Category.String())
}

func (ee EveEntity) IsCharacter() bool {
	return ee.Category == EveEntityCharacter
}

func (ee *EveEntity) Compare(other *EveEntity) int {
	return cmp.Compare(ee.Name, other.Name)
}
