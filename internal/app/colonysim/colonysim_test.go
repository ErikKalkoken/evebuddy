package colonysim

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

func TestExtractorOutput(t *testing.T) {
	// reference values computed with an independent port of the RIFT formula
	cases := []struct {
		baseValue int64
		cycleTime time.Duration
		want      []int64
	}{
		{1081, 30 * time.Minute, []int64{2467, 2086, 2039, 1994, 2095, 2558, 2611, 2152}},
		{6000, 2 * time.Hour, []int64{45801, 41958, 38709, 39127, 36462, 31413, 42578, 43744}},
		{30, 15 * time.Minute, []int64{42, 41, 39, 35, 32, 30, 29, 30}},
	}
	install := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	for _, tc := range cases {
		var got []int64
		for i := range len(tc.want) {
			runTime := install.Add(time.Duration(i+1) * tc.cycleTime)
			got = append(got, extractorOutput(tc.baseValue, install, runTime, tc.cycleTime))
		}
		assert.Equal(t, tc.want, got, "base %d", tc.baseValue)
	}
}

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

func TestSimulation_ExtractorToStorage(t *testing.T) {
	cp := &app.CharacterPlanet{
		LastUpdate: t0,
		Pins: []*app.PlanetPin{
			newExtractor(1, aqueousLiquids, 1081, 30*time.Minute, t0, t0.Add(4*time.Hour)),
			newStorage(2, app.EveGroupStorageFacilities, 12_000),
		},
		Routes: []*app.PlanetRoute{newRoute(1, 1, 2, aqueousLiquids, 10_000)},
	}
	t.Run("should forecast state while extracting", func(t *testing.T) {
		f := Forecast(cp, t0.Add(65*time.Minute))
		assert.Equal(t, app.ColonyExtracting, f.Status)
		assert.Equal(t, app.PinExtracting, f.Pins[1].Status)
		assert.Equal(t, map[int64]int64{typeAqueousLiquids: 2467 + 2086}, f.Pins[2].Contents)
		assert.InDelta(t, float64(2467+2086)*0.01, f.Pins[2].CapacityUsed, 0.0001)
		assert.Equal(t, optional.New(12_000.0), f.Pins[2].Capacity)
		assert.Equal(t, optional.New(t0.Add(4*time.Hour)), f.WorkEndsAt)
		assert.False(t, f.WorksBeyondHorizon)
	})
	t.Run("should forecast state after expiry", func(t *testing.T) {
		f := Forecast(cp, t0.Add(5*time.Hour))
		assert.Equal(t, app.ColonyNeedsAttention, f.Status)
		assert.Equal(t, app.PinExtractorExpired, f.Pins[1].Status)
		var total int64
		for _, x := range []int64{2467, 2086, 2039, 1994, 2095, 2558, 2611, 2152} {
			total += x
		}
		assert.Equal(t, map[int64]int64{typeAqueousLiquids: total}, f.Pins[2].Contents)
		assert.True(t, f.WorkEndsAt.IsEmpty())
		assert.False(t, f.WorksBeyondHorizon)
	})
}

func TestSimulation_Horizon(t *testing.T) {
	newPlanet := func(expiry time.Time) *app.CharacterPlanet {
		return &app.CharacterPlanet{
			LastUpdate: t0,
			Pins: []*app.PlanetPin{
				newExtractor(1, aqueousLiquids, 1081, 30*time.Minute, t0, expiry),
				newStorage(2, app.EveGroupStorageFacilities, 1_000_000),
			},
			Routes: []*app.PlanetRoute{newRoute(1, 1, 2, aqueousLiquids, 10_000)},
		}
	}
	t.Run("should report when colony works beyond the horizon", func(t *testing.T) {
		f := Forecast(newPlanet(t0.Add(app.ColonyForecastHorizon+24*time.Hour)), t0)
		assert.Equal(t, app.ColonyExtracting, f.Status)
		assert.True(t, f.WorksBeyondHorizon)
		assert.True(t, f.WorkEndsAt.IsEmpty())
	})
	t.Run("should report work end just before the horizon", func(t *testing.T) {
		expiry := t0.Add(app.ColonyForecastHorizon - time.Hour)
		f := Forecast(newPlanet(expiry), t0)
		assert.False(t, f.WorksBeyondHorizon)
		assert.Equal(t, optional.New(expiry), f.WorkEndsAt)
	})
	t.Run("should return completed when still working at the horizon", func(t *testing.T) {
		s := New(newPlanet(t0.Add(48 * time.Hour)))
		horizon := t0.Add(24 * time.Hour)
		got, r := s.RunUntilWorkEnds(horizon)
		assert.Equal(t, RunCompleted, r)
		assert.Equal(t, horizon, got)
	})
	t.Run("should return work ended when colony stops before the horizon", func(t *testing.T) {
		s := New(newPlanet(t0.Add(4 * time.Hour)))
		got, r := s.RunUntilWorkEnds(t0.Add(24 * time.Hour))
		assert.Equal(t, RunWorkEnded, r)
		assert.Equal(t, t0.Add(4*time.Hour), got)
	})
	t.Run("should return work ended when colony is not working", func(t *testing.T) {
		s := New(newPlanet(t0.Add(4 * time.Hour)))
		s.RunUntil(t0.Add(5 * time.Hour))
		got, r := s.RunUntilWorkEnds(t0.Add(24 * time.Hour))
		assert.Equal(t, RunWorkEnded, r)
		assert.Equal(t, t0.Add(5*time.Hour), got)
	})
}

func TestSimulation_StorageToFactoryToLaunchpad(t *testing.T) {
	cp := &app.CharacterPlanet{
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
	t.Run("should forecast state while producing", func(t *testing.T) {
		f := Forecast(cp, t0.Add(45*time.Minute))
		assert.Equal(t, app.ColonyProducing, f.Status)
		assert.Equal(t, app.PinProducing, f.Pins[2].Status)
		assert.Equal(t, map[int64]int64{typeWater: 20}, f.Pins[3].Contents)
		assert.Equal(t, map[int64]int64{typeAqueousLiquids: 3000}, f.Pins[2].Contents) // buffer for next cycle
		assert.Empty(t, f.Pins[1].Contents)
		assert.Equal(t, optional.New(t0.Add(90*time.Minute)), f.WorkEndsAt)
	})
	t.Run("should forecast state after inputs ran out", func(t *testing.T) {
		f := Forecast(cp, t0.Add(3*time.Hour))
		assert.Equal(t, app.ColonyIdle, f.Status)
		assert.Equal(t, app.PinFactoryIdle, f.Pins[2].Status)
		assert.Equal(t, map[int64]int64{typeWater: 60}, f.Pins[3].Contents)
		assert.Empty(t, f.Pins[2].Contents)
		assert.True(t, f.WorkEndsAt.IsEmpty())
	})
}

func TestSimulation_StorageFull(t *testing.T) {
	bulky := newType(9999, 1032, 1, 0)
	cp := &app.CharacterPlanet{
		LastUpdate: t0,
		Pins: []*app.PlanetPin{
			newExtractor(1, bulky, 30, 15*time.Minute, t0, t0.Add(24*time.Hour)),
			newStorage(2, app.EveGroupCommandCenters, 0), // falls back to default capacity
		},
		Routes: []*app.PlanetRoute{newRoute(1, 1, 2, bulky, 42)},
	}
	f := Forecast(cp, t0.Add(10*time.Hour))
	assert.Equal(t, app.ColonyNeedsAttention, f.Status)
	assert.Equal(t, app.PinStorageFull, f.Pins[2].Status)
	assert.Equal(t, optional.New(500.0), f.Pins[2].Capacity)
	assert.LessOrEqual(t, f.Pins[2].CapacityUsed, 500.0)
	assert.Greater(t, f.Pins[2].CapacityUsed, 500.0-42)
}

func TestSimulation_NotSetup(t *testing.T) {
	cp := &app.CharacterPlanet{
		LastUpdate: t0,
		Pins: []*app.PlanetPin{
			newStorage(1, app.EveGroupStorageFacilities, 12_000),
			newFactory(2, schematicWater),
		},
	}
	f := Forecast(cp, t0.Add(time.Hour))
	assert.Equal(t, app.ColonyNotSetup, f.Status)
	assert.Equal(t, app.PinInputNotRouted, f.Pins[2].Status)
	assert.Equal(t, app.PinStatic, f.Pins[1].Status)
}

func TestSimulation(t *testing.T) {
	newPlanet := func() *app.CharacterPlanet {
		return &app.CharacterPlanet{
			LastUpdate: t0,
			Pins: []*app.PlanetPin{
				newExtractor(1, aqueousLiquids, 1081, 30*time.Minute, t0, t0.Add(4*time.Hour)),
				newStorage(2, app.EveGroupStorageFacilities, 12_000),
				{ID: 3, Type: newType(1, 1, 0, 0)}, // unknown pin kind
			},
			Routes: []*app.PlanetRoute{
				newRoute(1, 1, 2, aqueousLiquids, 10_000),
				newRoute(2, 1, 99, aqueousLiquids, 10_000), // unknown pin
			},
		}
	}
	t.Run("should ignore unknown pins and routes to them", func(t *testing.T) {
		f := Forecast(newPlanet(), t0.Add(time.Hour))
		assert.Len(t, f.Pins, 2)
		assert.Equal(t, app.ColonyExtracting, f.Status)
	})
	t.Run("should not go back in time", func(t *testing.T) {
		s := New(newPlanet())
		s.RunUntil(t0.Add(-time.Hour))
		assert.Equal(t, t0, s.Time())
	})
	t.Run("should not change original when running clone", func(t *testing.T) {
		s := New(newPlanet())
		s.RunUntil(t0.Add(time.Hour))
		s2 := s.Clone()
		s2.RunUntil(t0.Add(2 * time.Hour))
		assert.Equal(t, t0.Add(time.Hour), s.Time())
		assert.Equal(t, int64(2467+2086), s.Forecast().Pins[2].Contents[typeAqueousLiquids])
		assert.Equal(t, int64(2467+2086+2039+1994), s2.Forecast().Pins[2].Contents[typeAqueousLiquids])
	})
	t.Run("should not change the source planet", func(t *testing.T) {
		cp := newPlanet()
		cp.Pins[1].Contents = []*app.PlanetPinContent{{Type: aqueousLiquids, Amount: 5}}
		Forecast(cp, t0.Add(time.Hour))
		assert.Equal(t, int64(5), cp.Pins[1].Contents[0].Amount)
	})
}
