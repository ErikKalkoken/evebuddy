package colonysim

import (
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

const (
	typeAqueousLiquids = 2268
	typeWater          = 3645
	schematicWater     = 121
)

var t0 = time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

func newType(id, groupID int64, volume, capacity float64) *app.EveType {
	et := &app.EveType{
		ID:    id,
		Group: &app.EveGroup{ID: groupID},
	}
	if volume > 0 {
		et.Volume = optional.New(volume)
	}
	if capacity > 0 {
		et.Capacity = optional.New(capacity)
	}
	return et
}

var (
	aqueousLiquids = newType(typeAqueousLiquids, 1032, 0.01, 0)
	water          = newType(typeWater, 1042, 0.38, 0)
)

func newExtractor(id int64, product *app.EveType, baseValue int64, cycle time.Duration, install, expiry time.Time) *app.PlanetPin {
	return &app.PlanetPin{
		ID:                   id,
		Type:                 newType(2848, app.EveGroupExtractorControlUnits, 0, 0),
		ExtractorProductType: optional.New(product),
		ExtractorQtyPerCycle: optional.New(baseValue),
		ExtractorCycleTime:   optional.New(cycle),
		InstallTime:          optional.New(install),
		ExpiryTime:           optional.New(expiry),
		LastCycleStart:       optional.New(install),
	}
}

func newStorage(id int64, groupID int64, capacity float64, contents ...*app.PlanetPinContent) *app.PlanetPin {
	return &app.PlanetPin{
		ID:       id,
		Type:     newType(2541, groupID, 0, capacity),
		Contents: contents,
	}
}

func newFactory(id int64, schematicID int64) *app.PlanetPin {
	return &app.PlanetPin{
		ID:        id,
		Type:      newType(2473, app.EveGroupProcessors, 0, 0),
		Schematic: optional.New(&app.EveSchematic{ID: schematicID}),
	}
}

func newRoute(id, source, destination int64, et *app.EveType, quantity int64) *app.PlanetRoute {
	return &app.PlanetRoute{
		RouteID:          id,
		SourcePinID:      source,
		DestinationPinID: destination,
		ContentType:      et,
		Quantity:         quantity,
	}
}

// exported for the external tests
const (
	TypeAqueousLiquids = typeAqueousLiquids
	TypeWater          = typeWater
	SchematicWater     = schematicWater
)

var (
	T0             = t0
	AqueousLiquids = aqueousLiquids
	Water          = water
	NewType        = newType
	NewExtractor   = newExtractor
	NewStorage     = newStorage
	NewFactory     = newFactory
	NewRoute       = newRoute
)
