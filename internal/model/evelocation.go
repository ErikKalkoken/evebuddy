package model

import (
	"fmt"
	"time"
)

type EveLocationVariant int

const (
	EveLocationUnknown EveLocationVariant = iota
	EveLocationAssetSafety
	EveLocationStation
	EveLocationStructure
	EveLocationSolarSystem
)

const LocationUnknownID = 888 // custom ID to signify a location that is not known

// EveLocation is a location in Eve Online.
type EveLocation struct {
	ID          int64
	SolarSystem *EveSolarSystem
	Type        *EveType
	Name        string
	Owner       *EveEntity
	UpdatedAt   time.Time
}

func (lc EveLocation) NamePlus() string {
	if lc.Name != "" {
		return lc.Name
	}
	switch lc.Variant() {
	case EveLocationUnknown:
		return "Unknown"
	case EveLocationAssetSafety:
		return "Asset Safety"
	case EveLocationSolarSystem:
		return lc.SolarSystem.Name
	case EveLocationStructure:
		return fmt.Sprintf("Unknown structure #%d", lc.ID)
	}
	return fmt.Sprintf("Unknown location #%d", lc.ID)
}

func (lc EveLocation) Variant() EveLocationVariant {
	return LocationVariantFromID(lc.ID)
}

func LocationVariantFromID(id int64) EveLocationVariant {
	switch {
	case id == LocationUnknownID:
		return EveLocationUnknown
	case id == 2004:
		return EveLocationAssetSafety
	case id >= 30_000_000 && id < 33_000_000:
		return EveLocationSolarSystem
	case id >= 60_000_000 && id < 64_000_000:
		return EveLocationStation
	case id > 1_000_000_000_000:
		return EveLocationStructure
	default:
		return EveLocationUnknown
	}
}
