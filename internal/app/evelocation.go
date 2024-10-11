package app

import (
	"fmt"
	"strings"
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

// DisplayName returns a user friendly name.
func (lc EveLocation) DisplayName() string {
	if lc.Name != "" {
		return lc.Name
	}
	return lc.alternativeName()
}

// DisplayName2 returns a user friendly name not including the location name.
func (lc EveLocation) DisplayName2() string {
	if lc.Name != "" {
		if lc.Variant() != EveLocationStructure {
			return lc.Name
		}
		p := strings.Split(lc.Name, " - ")
		if len(p) < 2 {
			return lc.Name
		}
		return p[1]
	}
	return lc.alternativeName()
}

func (lc EveLocation) alternativeName() string {
	switch lc.Variant() {
	case EveLocationUnknown:
		return "Unknown"
	case EveLocationAssetSafety:
		return "Asset Safety"
	case EveLocationSolarSystem:
		if lc.SolarSystem == nil {
			return fmt.Sprintf("Unknown solar system #%d", lc.ID)
		}
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
