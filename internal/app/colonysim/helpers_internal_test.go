package colonysim

import (
	"math/rand/v2"
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

const (
	typeSuspendedPlasma      = 2308
	typePlasmoids            = 2389
	typeSuperconductors      = 9838
	schematicPlasmoids       = 122
	schematicSuperconductors = 65
)

var (
	suspendedPlasma = newType(typeSuspendedPlasma, 1032, 0.01, 0)
	plasmoids       = newType(typePlasmoids, 1042, 0.38, 0)
	superconductors = newType(typeSuperconductors, 1034, 0.75, 0)
)

// randomColony returns a random colony and time for a forecast from a seed.
func randomColony(seed uint64) (*app.CharacterPlanet, time.Time) {
	r := rand.New(rand.NewPCG(seed, 0))
	pick := func(n int) int { return r.IntN(n) }
	duration := func(maxValue time.Duration) time.Duration {
		return time.Duration(r.Int64N(int64(maxValue))).Truncate(time.Second)
	}
	cp := &app.CharacterPlanet{LastUpdate: t0}
	type info struct {
		produces []*app.EveType // types the pin can send
		accepts  []*app.EveType // types the pin can receive; nil means all
		factory  bool
	}
	pins := make(map[int64]info)
	var id int64
	allTypes := []*app.EveType{aqueousLiquids, suspendedPlasma, water, plasmoids, superconductors}

	// extractors
	for range pick(3) {
		id++
		product := []*app.EveType{aqueousLiquids, suspendedPlasma}[pick(2)]
		cycle := []time.Duration{15 * time.Minute, 30 * time.Minute, time.Hour, 2 * time.Hour}[pick(4)]
		install := t0.Add(-duration(48 * time.Hour))
		expiry := install.Add(time.Duration(1+pick(60)) * cycle)
		p := newExtractor(id, product, int64(500+pick(5000)), cycle, install, expiry)
		if pick(4) > 0 {
			n := max(0, min(int(t0.Sub(install)/cycle), int(expiry.Sub(install)/cycle)))
			p.LastCycleStart = optional.New(install.Add(time.Duration(n) * cycle))
		}
		cp.Pins = append(cp.Pins, p)
		pins[id] = info{produces: []*app.EveType{product}, accepts: []*app.EveType{}}
	}
	// storages
	for range pick(3) {
		id++
		group := []int64{app.EveGroupStorageFacilities, app.EveGroupSpaceports, app.EveGroupCommandCenters}[pick(3)]
		var contents []*app.PlanetPinContent
		for _, et := range allTypes {
			if pick(3) == 0 {
				contents = append(contents, &app.PlanetPinContent{Type: et, Amount: int64(1 + pick(10_000))})
			}
		}
		capacity := []float64{50, 500, 12_000}[pick(3)]
		cp.Pins = append(cp.Pins, newStorage(id, group, capacity, contents...))
		pins[id] = info{produces: allTypes}
	}
	// factories
	schematics := []struct {
		id     int64
		inputs []*app.EveType
		output *app.EveType
		cycle  time.Duration
	}{
		{schematicWater, []*app.EveType{aqueousLiquids}, water, 30 * time.Minute},
		{schematicPlasmoids, []*app.EveType{suspendedPlasma}, plasmoids, 30 * time.Minute},
		{schematicSuperconductors, []*app.EveType{water, plasmoids}, superconductors, time.Hour},
	}
	for range pick(4) {
		id++
		if pick(8) == 0 {
			p := newFactory(id, 0)
			p.Schematic = optional.Optional[*app.EveSchematic]{} // not setup
			cp.Pins = append(cp.Pins, p)
			pins[id] = info{accepts: []*app.EveType{}}
			continue
		}
		s := schematics[pick(len(schematics))]
		p := newFactory(id, s.id)
		if pick(2) == 0 {
			p.LastCycleStart = optional.New(t0.Add(-duration(2 * s.cycle)))
		}
		for _, et := range s.inputs {
			if pick(3) == 0 {
				p.Contents = append(p.Contents, &app.PlanetPinContent{Type: et, Amount: int64(1 + pick(6000))})
			}
		}
		cp.Pins = append(cp.Pins, p)
		pins[id] = info{produces: []*app.EveType{s.output}, accepts: s.inputs, factory: true}
	}
	// routes
	var routeID int64
	for source := int64(1); source <= id; source++ {
		for destination := int64(1); destination <= id; destination++ {
			if source == destination || pick(3) > 0 {
				continue
			}
			for _, et := range pins[source].produces {
				accepts := pins[destination].accepts
				if accepts != nil && !containsType(accepts, et) {
					continue
				}
				routeID++
				cp.Routes = append(cp.Routes, newRoute(routeID, source, destination, et, int64(1+pick(5000))))
				break
			}
		}
	}
	var now time.Time
	switch pick(5) {
	case 0:
		now = t0.Add(-duration(2 * time.Hour)) // before the snapshot
	case 1:
		now = t0 // at the snapshot, before any pin ran
	case 2:
		now = t0.Add(duration(90 * 24 * time.Hour)) // long after the snapshot, so most colonies are at rest
	default:
		now = t0.Add(duration(72 * time.Hour))
	}
	return cp, now
}

func containsType(s []*app.EveType, et *app.EveType) bool {
	for _, x := range s {
		if x.ID == et.ID {
			return true
		}
	}
	return false
}

// exported for the external tests
const (
	TypeAqueousLiquids       = typeAqueousLiquids
	TypeWater                = typeWater
	SchematicWater           = schematicWater
	SchematicPlasmoids       = schematicPlasmoids
	SchematicSuperconductors = schematicSuperconductors
)

var (
	T0              = t0
	AqueousLiquids  = aqueousLiquids
	Water           = water
	NewType         = newType
	NewExtractor    = newExtractor
	NewStorage      = newStorage
	NewFactory      = newFactory
	NewRoute        = newRoute
	RandomColony    = randomColony
	SuspendedPlasma = suspendedPlasma
	Plasmoids       = plasmoids
	Superconductors = superconductors
)
