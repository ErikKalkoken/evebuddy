package app

import (
	"maps"
	"slices"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type CharacterPlanet struct {
	ID           int64
	CharacterID  int32
	EvePlanet    *EvePlanet
	LastUpdate   time.Time
	LastNotified time.Time // expiry time that was last notified
	Pins         []*PlanetPin
	UpgradeLevel int
}

// ExtractedTypes returns a list of unique types currently being extracted.
func (cp CharacterPlanet) ExtractedTypes() []*EveType {
	types := make(map[int32]*EveType)
	for _, p := range cp.Pins {
		if p.Type.Group.ID != EveGroupExtractorControlUnits || p.ExtractorProductType == nil {
			continue
		}
		types[p.ExtractorProductType.ID] = p.ExtractorProductType
	}
	return slices.Collect(maps.Values(types))
}

// ExtractionsExpiryTime returns the final expiry time for all extractions.
// When no expiry data is found it will return a zero time.
func (cp CharacterPlanet) ExtractionsExpiryTime() time.Time {
	expireTimes := make([]time.Time, 0)
	for _, p := range cp.Pins {
		if p.Type.Group.ID != EveGroupExtractorControlUnits {
			continue
		}
		if x := p.ExpiryTime.ValueOrZero(); x.After(time.Now()) {
			expireTimes = append(expireTimes, x)
		}
	}
	if len(expireTimes) == 0 {
		return time.Time{}
	}
	slices.SortFunc(expireTimes, func(a, b time.Time) int {
		return b.Compare(a) // sort descending
	})
	return expireTimes[0]
}

// ProducedSchematics returns a list of unique schematics currently in production.
func (cp CharacterPlanet) ProducedSchematics() []*EveSchematic {
	schematics := make(map[int32]*EveSchematic)
	for _, p := range cp.Pins {
		if p.Type.Group.ID != EveGroupProcessors || p.Schematic == nil {
			continue
		}
		schematics[p.Schematic.ID] = p.Schematic
	}
	return slices.Collect(maps.Values(schematics))
}

type PlanetPin struct {
	ID                   int64
	ExpiryTime           optional.Optional[time.Time]
	ExtractorProductType *EveType
	FactorySchematic     *EveSchematic
	InstallTime          optional.Optional[time.Time]
	LastCycleStart       optional.Optional[time.Time]
	Schematic            *EveSchematic
	Type                 *EveType
}
