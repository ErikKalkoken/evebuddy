package app

import (
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type CharacterPlanet struct {
	ID           int64
	CharacterID  int32
	EvePlanet    *EvePlanet
	LastUpdate   time.Time
	Pins         []*PlanetPin
	UpgradeLevel int
}

type PlanetPin struct {
	ID                   int64
	Contents             []*PlanetPinContent
	ExpiryTime           optional.Optional[time.Time]
	ExtractorProductType *EveType
	FactorySchematic     *EveSchematic
	InstallTime          optional.Optional[time.Time]
	LastCycleStart       optional.Optional[time.Time]
	Schematic            *EveSchematic
	Type                 *EveType
}

type PlanetPinContent struct {
	Amount int
	Type   *EveType
}
