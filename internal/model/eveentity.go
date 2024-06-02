package model

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

// An EveEntity in EveOnline.
type EveEntity struct {
	Category EveEntityCategory
	ID       int32
	Name     string
}
