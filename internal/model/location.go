package model

import "time"

type LocationVariant int

const (
	LocationVariantUnknown LocationVariant = iota
	LocationVariantAssetSafety
	LocationVariantStation
	LocationVariantStructure
	LocationVariantSolarSystem
)

const LocationUnknownID = 888 // custom ID to signify a location that is not known

// Location is a structure in Eve Online.
type Location struct {
	ID          int64
	SolarSystem *EveSolarSystem
	Type        *EveType
	Name        string
	Owner       *EveEntity
	UpdatedAt   time.Time
}

func (l Location) Variant() LocationVariant {
	return LocationVariantFromID(l.ID)
}

func LocationVariantFromID(id int64) LocationVariant {
	switch {
	case id == LocationUnknownID:
		return LocationVariantUnknown
	case id == 2004:
		return LocationVariantAssetSafety
	case id >= 30_000_000 && id < 33_000_000:
		return LocationVariantSolarSystem
	case id >= 60_000_000 && id < 64_000_000:
		return LocationVariantStation
	case id > 1_000_000_000_000:
		return LocationVariantStructure
	default:
		return LocationVariantUnknown
	}
}
