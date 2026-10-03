package colonysim_test

import (
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/colonysim"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

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

func TestForecast_ValidUntil(t *testing.T) {
	extractorToStorage := &app.CharacterPlanet{
		LastUpdate: t0,
		Pins: []*app.PlanetPin{
			newExtractor(1, aqueousLiquids, 1081, 30*time.Minute, t0, t0.Add(4*time.Hour)),
			newStorage(2, app.EveGroupStorageFacilities, 12_000),
		},
		Routes: []*app.PlanetRoute{newRoute(1, 1, 2, aqueousLiquids, 10_000)},
	}
	storageToFactory := &app.CharacterPlanet{
		LastUpdate: t0,
		Pins: []*app.PlanetPin{
			newStorage(1, app.EveGroupStorageFacilities, 12_000, &app.PlanetPinContent{Type: aqueousLiquids, Amount: 9000}),
			newFactory(2, schematicWater),
			newStorage(3, app.EveGroupSpaceports, 10_000),
		},
		Routes: []*app.PlanetRoute{
			newRoute(1, 1, 2, aqueousLiquids, 3000),
			newRoute(2, 2, 3, water, 20),
		},
	}
	t.Run("should be valid until the next extractor cycle", func(t *testing.T) {
		f := colonysim.Forecast(extractorToStorage, t0.Add(65*time.Minute))
		assert.Equal(t, optional.New(t0.Add(90*time.Minute)), f.ValidUntil)
	})
	t.Run("should be valid until the next factory cycle", func(t *testing.T) {
		f := colonysim.Forecast(storageToFactory, t0.Add(45*time.Minute))
		assert.Equal(t, optional.New(t0.Add(60*time.Minute)), f.ValidUntil)
	})
	t.Run("should be valid until the expiry when no cycle ends before", func(t *testing.T) {
		cp := &app.CharacterPlanet{
			LastUpdate: t0,
			Pins: []*app.PlanetPin{
				// cycle ends after the expiry
				newExtractor(1, aqueousLiquids, 1081, 3*time.Hour, t0, t0.Add(2*time.Hour)),
				newStorage(2, app.EveGroupStorageFacilities, 12_000),
			},
			Routes: []*app.PlanetRoute{newRoute(1, 1, 2, aqueousLiquids, 10_000)},
		}
		f := colonysim.Forecast(cp, t0.Add(time.Hour))
		assert.Equal(t, optional.New(t0.Add(2*time.Hour)), f.ValidUntil)
	})
	t.Run("should never be valid after the work end", func(t *testing.T) {
		f := colonysim.Forecast(storageToFactory, t0.Add(45*time.Minute))
		v := f.ValidUntil.MustValue()
		assert.False(t, v.After(f.WorkEndsAt.MustValue()))
	})
	t.Run("should have no end when nothing runs anymore", func(t *testing.T) {
		f := colonysim.Forecast(extractorToStorage, t0.Add(5*time.Hour))
		assert.True(t, f.ValidUntil.IsEmpty())
	})
	t.Run("should have no end when colony is not setup", func(t *testing.T) {
		cp := &app.CharacterPlanet{
			LastUpdate: t0,
			Pins:       []*app.PlanetPin{newStorage(1, app.EveGroupStorageFacilities, 12_000)},
		}
		f := colonysim.Forecast(cp, t0.Add(time.Hour))
		assert.True(t, f.ValidUntil.IsEmpty())
	})
}

func TestForecast_StableUntilValidUntil(t *testing.T) {
	colonies := map[string]*app.CharacterPlanet{
		"extractor to storage": {
			LastUpdate: t0,
			Pins: []*app.PlanetPin{
				newExtractor(1, aqueousLiquids, 1081, 30*time.Minute, t0, t0.Add(4*time.Hour)),
				newStorage(2, app.EveGroupStorageFacilities, 12_000),
			},
			Routes: []*app.PlanetRoute{newRoute(1, 1, 2, aqueousLiquids, 10_000)},
		},
		"storage to factory to launchpad": {
			LastUpdate: t0,
			Pins: []*app.PlanetPin{
				newStorage(1, app.EveGroupStorageFacilities, 12_000, &app.PlanetPinContent{Type: aqueousLiquids, Amount: 9000}),
				newFactory(2, schematicWater),
				newStorage(3, app.EveGroupSpaceports, 10_000),
			},
			Routes: []*app.PlanetRoute{
				newRoute(1, 1, 2, aqueousLiquids, 3000),
				newRoute(2, 2, 3, water, 20),
			},
		},
		"storage full": {
			LastUpdate: t0,
			Pins: []*app.PlanetPin{
				newExtractor(1, aqueousLiquids, 1081, 30*time.Minute, t0, t0.Add(24*time.Hour)),
				newStorage(2, app.EveGroupStorageFacilities, 50),
			},
			Routes: []*app.PlanetRoute{newRoute(1, 1, 2, aqueousLiquids, 10_000)},
		},
		"extractor to storage to factory": {
			LastUpdate: t0,
			Pins: []*app.PlanetPin{
				newExtractor(1, aqueousLiquids, 5000, time.Hour, t0, t0.Add(12*time.Hour)),
				newStorage(2, app.EveGroupStorageFacilities, 12_000),
				newFactory(3, schematicWater),
				newStorage(4, app.EveGroupSpaceports, 10_000),
			},
			Routes: []*app.PlanetRoute{
				newRoute(1, 1, 2, aqueousLiquids, 10_000),
				newRoute(2, 2, 3, aqueousLiquids, 3000),
				newRoute(3, 3, 4, water, 20),
			},
		},
		"factory chain": {
			LastUpdate: t0,
			Pins: []*app.PlanetPin{
				newStorage(1, app.EveGroupStorageFacilities, 12_000,
					&app.PlanetPinContent{Type: aqueousLiquids, Amount: 6000},
					&app.PlanetPinContent{Type: suspendedPlasma, Amount: 6000},
				),
				newFactory(2, schematicWater),
				newFactory(3, schematicPlasmoids),
				newFactory(4, schematicSuperconductors),
				newStorage(5, app.EveGroupSpaceports, 10_000),
			},
			Routes: []*app.PlanetRoute{
				newRoute(1, 1, 2, aqueousLiquids, 3000),
				newRoute(2, 1, 3, suspendedPlasma, 3000),
				newRoute(3, 2, 4, water, 20),
				newRoute(4, 3, 4, plasmoids, 20),
				newRoute(5, 4, 5, superconductors, 5),
			},
		},
		"beyond horizon": {
			LastUpdate: t0,
			Pins: []*app.PlanetPin{
				newExtractor(1, aqueousLiquids, 1081, 30*time.Minute, t0, t0.Add(app.ColonyForecastHorizon+24*time.Hour)),
				newStorage(2, app.EveGroupStorageFacilities, 1_000_000),
			},
			Routes: []*app.PlanetRoute{newRoute(1, 1, 2, aqueousLiquids, 10_000)},
		},
	}
	offsets := []time.Duration{0, 10 * time.Minute, 45 * time.Minute, 65 * time.Minute, 3 * time.Hour, 5 * time.Hour, 40 * time.Hour}
	for name, cp := range colonies {
		for _, d := range offsets {
			t.Run(fmt.Sprintf("%s at %s", name, d), func(t *testing.T) {
				assertStableUntilValidUntil(t, cp, t0.Add(d))
			})
		}
	}
}

func FuzzForecast_StableUntilValidUntil(f *testing.F) {
	for seed := range uint64(50) {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, seed uint64) {
		cp, now := randomColony(seed)
		assertStableUntilValidUntil(t, cp, now)
	})
}

// assertStableUntilValidUntil asserts that forecasts until ValidUntil equal the forecast at now.
func assertStableUntilValidUntil(t *testing.T, cp *app.CharacterPlanet, now time.Time) {
	t.Helper()
	want := colonysim.Forecast(cp, now)
	var times []time.Time
	if v, ok := want.ValidUntil.Value(); ok {
		require.False(t, v.Before(now), "valid until is before now")
		for _, x := range []time.Time{now.Add(time.Nanosecond), now.Add(v.Sub(now) / 2), v.Add(-time.Nanosecond)} {
			if x.After(now) && x.Before(v) {
				times = append(times, x)
			}
		}
	} else {
		times = []time.Time{now.Add(time.Hour), now.Add(10 * 24 * time.Hour)}
	}
	normalize := func(f *app.ColonyForecast) app.ColonyForecast {
		x := *f
		x.Time = time.Time{}
		if want.WorksBeyondHorizon { // the horizon moves with the time of the forecast
			x.WorksBeyondHorizon = false
			x.WorkEndsAt = optional.Optional[time.Time]{}
		}
		return x
	}
	for _, x := range times {
		got := colonysim.Forecast(cp, x)
		if !assert.Equal(t, normalize(want), normalize(got), "forecast at %s differs from forecast at %s", x, now) {
			return
		}
	}
}

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
	return cp, t0.Add(duration(72 * time.Hour))
}

func containsType(s []*app.EveType, et *app.EveType) bool {
	for _, x := range s {
		if x.ID == et.ID {
			return true
		}
	}
	return false
}
