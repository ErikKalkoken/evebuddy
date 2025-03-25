package app

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
