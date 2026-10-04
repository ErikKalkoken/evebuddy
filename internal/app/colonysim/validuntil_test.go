package colonysim_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/colonysim"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

const (
	schematicPlasmoids       = colonysim.SchematicPlasmoids
	schematicSuperconductors = colonysim.SchematicSuperconductors
)

var (
	suspendedPlasma = colonysim.SuspendedPlasma
	plasmoids       = colonysim.Plasmoids
	superconductors = colonysim.Superconductors
	randomColony    = colonysim.RandomColony
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

func FuzzWorkEndsAt_MatchesForecast(f *testing.F) {
	for seed := range uint64(50) {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, seed uint64) {
		cp, _ := randomColony(seed)
		want := colonysim.Forecast(cp, cp.LastUpdate).WorkEndsAt
		got := colonysim.WorkEndsAt(cp, cp.LastUpdate.Add(app.ColonyForecastHorizon))
		assert.Equal(t, want, got)
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
