package colonysim_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/colonysim"
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
		f := colonysim.Forecast(cp, t0.Add(65*time.Minute))
		assert.Equal(t, app.ColonyExtracting, f.Status)
		assert.Equal(t, app.PinExtracting, f.Pins[1].Status)
		assert.Equal(t, map[int64]int64{typeAqueousLiquids: 2467 + 2086}, f.Pins[2].Contents)
		assert.InDelta(t, float64(2467+2086)*0.01, f.Pins[2].CapacityUsed, 0.0001)
		assert.Equal(t, optional.New(12_000.0), f.Pins[2].Capacity)
		assert.Equal(t, optional.New(t0.Add(4*time.Hour)), f.WorkEndsAt)
		assert.False(t, f.WorksBeyondHorizon)
	})
	t.Run("should forecast state after expiry", func(t *testing.T) {
		f := colonysim.Forecast(cp, t0.Add(5*time.Hour))
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
		f := colonysim.Forecast(newPlanet(t0.Add(app.ColonyForecastHorizon+24*time.Hour)), t0)
		assert.Equal(t, app.ColonyExtracting, f.Status)
		assert.True(t, f.WorksBeyondHorizon)
		assert.True(t, f.WorkEndsAt.IsEmpty())
	})
	t.Run("should report work end just before the horizon", func(t *testing.T) {
		expiry := t0.Add(app.ColonyForecastHorizon - time.Hour)
		f := colonysim.Forecast(newPlanet(expiry), t0)
		assert.False(t, f.WorksBeyondHorizon)
		assert.Equal(t, optional.New(expiry), f.WorkEndsAt)
	})
	t.Run("should return completed when still working at the horizon", func(t *testing.T) {
		s := colonysim.New(newPlanet(t0.Add(48 * time.Hour)))
		horizon := t0.Add(24 * time.Hour)
		got, r := s.RunUntilWorkEnds(horizon)
		assert.Equal(t, colonysim.RunCompleted, r)
		assert.Equal(t, horizon, got)
	})
	t.Run("should return work ended when colony stops before the horizon", func(t *testing.T) {
		s := colonysim.New(newPlanet(t0.Add(4 * time.Hour)))
		got, r := s.RunUntilWorkEnds(t0.Add(24 * time.Hour))
		assert.Equal(t, colonysim.RunWorkEnded, r)
		assert.Equal(t, t0.Add(4*time.Hour), got)
	})
	t.Run("should return work ended when colony is not working", func(t *testing.T) {
		s := colonysim.New(newPlanet(t0.Add(4 * time.Hour)))
		s.RunUntil(t0.Add(5 * time.Hour))
		got, r := s.RunUntilWorkEnds(t0.Add(24 * time.Hour))
		assert.Equal(t, colonysim.RunWorkEnded, r)
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
		f := colonysim.Forecast(cp, t0.Add(45*time.Minute))
		assert.Equal(t, app.ColonyProducing, f.Status)
		assert.Equal(t, app.PinProducing, f.Pins[2].Status)
		assert.Equal(t, map[int64]int64{typeWater: 20}, f.Pins[3].Contents)
		assert.Equal(t, map[int64]int64{typeAqueousLiquids: 3000}, f.Pins[2].Contents) // buffer for next cycle
		assert.Equal(t, map[int64]int64{typeAqueousLiquids: 3000}, f.Pins[2].Demands)
		assert.Equal(t, int64(20), f.Pins[2].OutputQuantity)
		assert.Equal(t, int64(typeWater), f.Pins[2].OutputTypeID)
		assert.Empty(t, f.Pins[1].Contents)
		assert.Equal(t, optional.New(t0.Add(90*time.Minute)), f.WorkEndsAt)
	})
	t.Run("should forecast state after inputs ran out", func(t *testing.T) {
		f := colonysim.Forecast(cp, t0.Add(3*time.Hour))
		assert.Equal(t, app.ColonyIdle, f.Status)
		assert.Equal(t, app.PinFactoryIdle, f.Pins[2].Status)
		assert.Equal(t, map[int64]int64{typeWater: 60}, f.Pins[3].Contents)
		assert.Empty(t, f.Pins[2].Contents)
		assert.True(t, f.WorkEndsAt.IsEmpty())
	})
	t.Run("should forecast work end at the snapshot while factory is idle", func(t *testing.T) {
		f := colonysim.Forecast(cp, t0)
		assert.Equal(t, optional.New(t0.Add(90*time.Minute)), f.WorkEndsAt)
	})
	t.Run("should return work ended when colony stops within one cycle of the horizon", func(t *testing.T) {
		s := colonysim.New(cp)
		s.RunUntil(t0.Add(45 * time.Minute))
		got, r := s.RunUntilWorkEnds(t0.Add(100 * time.Minute))
		assert.Equal(t, colonysim.RunWorkEnded, r)
		assert.Equal(t, t0.Add(90*time.Minute), got)
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
	f := colonysim.Forecast(cp, t0.Add(10*time.Hour))
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
	f := colonysim.Forecast(cp, t0.Add(time.Hour))
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
		f := colonysim.Forecast(newPlanet(), t0.Add(time.Hour))
		assert.Len(t, f.Pins, 2)
		assert.Equal(t, app.ColonyExtracting, f.Status)
	})
	t.Run("should not go back in time", func(t *testing.T) {
		s := colonysim.New(newPlanet())
		s.RunUntil(t0.Add(-time.Hour))
		assert.Equal(t, t0, s.Time())
	})
	t.Run("should not change the source planet", func(t *testing.T) {
		cp := newPlanet()
		cp.Pins[1].Contents = []*app.PlanetPinContent{{Type: aqueousLiquids, Amount: 5}}
		colonysim.Forecast(cp, t0.Add(time.Hour))
		assert.Equal(t, int64(5), cp.Pins[1].Contents[0].Amount)
	})
}

func TestSimulation_ExtractorToStorageToFactory(t *testing.T) {
	cp := &app.CharacterPlanet{
		LastUpdate: t0,
		Pins: []*app.PlanetPin{
			newExtractor(1, aqueousLiquids, 1081, 30*time.Minute, t0, t0.Add(4*time.Hour)),
			newStorage(2, app.EveGroupStorageFacilities, 12_000),
			newFactory(3, schematicWater),
			newStorage(4, app.EveGroupSpaceports, 10_000),
		},
		Routes: []*app.PlanetRoute{
			newRoute(1, 1, 2, aqueousLiquids, 10_000),
			newRoute(2, 2, 3, aqueousLiquids, 3000),
			newRoute(3, 3, 4, water, 20),
		},
	}
	inputs := func(f *app.ColonyForecast) int64 {
		return f.Pins[2].Contents[typeAqueousLiquids] + f.Pins[3].Contents[typeAqueousLiquids]
	}
	t.Run("should forward extracted inputs from storage to factory", func(t *testing.T) {
		f := colonysim.Forecast(cp, t0.Add(65*time.Minute))
		assert.Equal(t, app.PinProducing, f.Pins[3].Status)
		assert.Equal(t, int64(2467+2086-3000), inputs(f), "first batch consumed")
	})
	t.Run("should process all extracted inputs", func(t *testing.T) {
		f := colonysim.Forecast(cp, t0.Add(10*time.Hour))
		// 18,002 extracted makes 6 batches with 2 left over
		assert.Equal(t, map[int64]int64{typeWater: 120}, f.Pins[4].Contents)
		assert.Equal(t, int64(2), inputs(f))
	})
	t.Run("should end work when extractor expires", func(t *testing.T) {
		f := colonysim.Forecast(cp, t0)
		assert.Equal(t, optional.New(t0.Add(4*time.Hour)), f.WorkEndsAt)
	})
}

func TestSimulation_FactoryActiveAtSnapshot(t *testing.T) {
	// factory cycle started before the snapshot and ends at t0+20min, input buffer empty
	newPlanet := func(stored int64) *app.CharacterPlanet {
		f := newFactory(2, schematicWater)
		f.LastCycleStart = optional.New(t0.Add(-10 * time.Minute))
		var contents []*app.PlanetPinContent
		if stored > 0 {
			contents = append(contents, &app.PlanetPinContent{Type: aqueousLiquids, Amount: stored})
		}
		return &app.CharacterPlanet{
			LastUpdate: t0,
			Pins: []*app.PlanetPin{
				newStorage(1, app.EveGroupStorageFacilities, 12_000, contents...),
				f,
				newStorage(3, app.EveGroupSpaceports, 10_000),
			},
			Routes: []*app.PlanetRoute{
				newRoute(1, 1, 2, aqueousLiquids, 3000),
				newRoute(2, 2, 3, water, 20),
			},
		}
	}
	t.Run("should deliver output of cycle running at snapshot", func(t *testing.T) {
		cp := newPlanet(9000)
		assert.Equal(t, app.PinProducing, colonysim.Forecast(cp, t0).Pins[2].Status)
		assert.Empty(t, colonysim.Forecast(cp, t0.Add(15*time.Minute)).Pins[3].Contents)
		assert.Equal(t, map[int64]int64{typeWater: 20}, colonysim.Forecast(cp, t0.Add(25*time.Minute)).Pins[3].Contents)
	})
	t.Run("should keep working when inputs are pulled after the running cycle", func(t *testing.T) {
		// the factory is idle for an instant between finishing a cycle and pulling inputs
		cp := newPlanet(9000)
		assert.Equal(t, optional.New(t0.Add(110*time.Minute)), colonysim.Forecast(cp, t0).WorkEndsAt)
		f := colonysim.Forecast(cp, t0.Add(3*time.Hour))
		assert.Equal(t, map[int64]int64{typeWater: 80}, f.Pins[3].Contents, "running batch + 3 from storage")
		assert.Empty(t, f.Pins[1].Contents)
	})
	t.Run("should deliver one phantom batch for factory that never ran (known limitation, same as RIFT)", func(t *testing.T) {
		cp := newPlanet(0)
		assert.Equal(t, optional.New(t0.Add(20*time.Minute)), colonysim.Forecast(cp, t0).WorkEndsAt)
		f := colonysim.Forecast(cp, t0.Add(3*time.Hour))
		assert.Equal(t, map[int64]int64{typeWater: 20}, f.Pins[3].Contents)
		assert.Equal(t, app.PinFactoryIdle, f.Pins[2].Status)
	})
}

func TestSimulation_FactoryChain(t *testing.T) {
	const (
		typeSuspendedPlasma      = 2308
		typePlasmoids            = 2389
		typeSuperconductors      = 9838
		schematicPlasmoids       = 122
		schematicSuperconductors = 65
	)
	suspendedPlasma := newType(typeSuspendedPlasma, 1032, 0.01, 0)
	plasmoids := newType(typePlasmoids, 1042, 0.38, 0)
	superconductors := newType(typeSuperconductors, 1034, 0.75, 0)
	t.Run("should make P2 from two P1 factories", func(t *testing.T) {
		cp := &app.CharacterPlanet{
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
		}
		// P1 factories make 2 batches each until t0+60min, then the P2 factory runs one 1h cycle
		assert.Equal(t, optional.New(t0.Add(2*time.Hour)), colonysim.Forecast(cp, t0).WorkEndsAt)
		f := colonysim.Forecast(cp, t0.Add(3*time.Hour))
		assert.Equal(t, map[int64]int64{typeSuperconductors: 5}, f.Pins[5].Contents)
		for _, id := range []int64{1, 2, 3, 4} {
			assert.Empty(t, f.Pins[id].Contents, "pin %d", id)
		}
	})
	t.Run("should route output to the factory with the fuller input buffer first", func(t *testing.T) {
		fuller := newFactory(4, schematicSuperconductors)
		fuller.Contents = []*app.PlanetPinContent{{Type: water, Amount: 20}}
		cp := &app.CharacterPlanet{
			LastUpdate: t0,
			Pins: []*app.PlanetPin{
				newStorage(1, app.EveGroupStorageFacilities, 12_000, &app.PlanetPinContent{Type: aqueousLiquids, Amount: 3000}),
				newFactory(2, schematicWater),
				newFactory(3, schematicSuperconductors),
				fuller,
			},
			Routes: []*app.PlanetRoute{
				newRoute(1, 1, 2, aqueousLiquids, 3000),
				newRoute(2, 2, 3, water, 20), // lower route ID
				newRoute(3, 2, 4, water, 20),
			},
		}
		f := colonysim.Forecast(cp, t0.Add(35*time.Minute))
		assert.Empty(t, f.Pins[3].Contents)
		assert.Equal(t, map[int64]int64{typeWater: 40}, f.Pins[4].Contents)
	})
}

func TestSimulation_OutputToSeveralStorages(t *testing.T) {
	newPlanet := func(contents2, contents3 int64) *app.CharacterPlanet {
		newContents := func(amount int64) []*app.PlanetPinContent {
			if amount == 0 {
				return nil
			}
			return []*app.PlanetPinContent{{Type: aqueousLiquids, Amount: amount}}
		}
		return &app.CharacterPlanet{
			LastUpdate: t0,
			Pins: []*app.PlanetPin{
				newExtractor(1, aqueousLiquids, 1081, 30*time.Minute, t0, t0.Add(4*time.Hour)),
				newStorage(2, app.EveGroupStorageFacilities, 12_000, newContents(contents2)...),
				newStorage(3, app.EveGroupStorageFacilities, 12_000, newContents(contents3)...),
			},
			Routes: []*app.PlanetRoute{
				newRoute(1, 1, 2, aqueousLiquids, 10_000),
				newRoute(2, 1, 3, aqueousLiquids, 10_000),
			},
		}
	}
	t.Run("should split output evenly with the remainder going to the lower route ID", func(t *testing.T) {
		f := colonysim.Forecast(newPlanet(0, 0), t0.Add(35*time.Minute))
		assert.Equal(t, int64(1234), f.Pins[2].Contents[typeAqueousLiquids]) // 2467 / 2 rounded up
		assert.Equal(t, int64(1233), f.Pins[3].Contents[typeAqueousLiquids])
	})
	t.Run("should serve the storage with less free space first", func(t *testing.T) {
		f := colonysim.Forecast(newPlanet(0, 1000), t0.Add(35*time.Minute))
		assert.Equal(t, int64(1233), f.Pins[2].Contents[typeAqueousLiquids])
		assert.Equal(t, int64(1000+1234), f.Pins[3].Contents[typeAqueousLiquids])
	})
}

func TestSimulation_FactoryBufferAtSnapshot(t *testing.T) {
	newPlanet := func(stored int64) *app.CharacterPlanet {
		f := newFactory(2, schematicWater)
		f.Contents = []*app.PlanetPinContent{{Type: aqueousLiquids, Amount: 1500}}
		var contents []*app.PlanetPinContent
		if stored > 0 {
			contents = append(contents, &app.PlanetPinContent{Type: aqueousLiquids, Amount: stored})
		}
		return &app.CharacterPlanet{
			LastUpdate: t0,
			Pins: []*app.PlanetPin{
				newStorage(1, app.EveGroupStorageFacilities, 12_000, contents...),
				f,
				newStorage(3, app.EveGroupSpaceports, 10_000),
			},
			Routes: []*app.PlanetRoute{
				newRoute(1, 1, 2, aqueousLiquids, 3000),
				newRoute(2, 2, 3, water, 20),
			},
		}
	}
	t.Run("should top up a partial buffer from storage and start", func(t *testing.T) {
		f := colonysim.Forecast(newPlanet(1500), t0.Add(35*time.Minute))
		assert.Equal(t, map[int64]int64{typeWater: 20}, f.Pins[3].Contents)
		assert.Empty(t, f.Pins[1].Contents)
	})
	t.Run("should stay idle and keep a partial buffer without more inputs", func(t *testing.T) {
		cp := newPlanet(0)
		f := colonysim.Forecast(cp, t0.Add(3*time.Hour))
		assert.Equal(t, app.PinFactoryIdle, f.Pins[2].Status)
		assert.Equal(t, map[int64]int64{typeAqueousLiquids: 1500}, f.Pins[2].Contents)
		assert.Empty(t, f.Pins[3].Contents)
		assert.True(t, colonysim.Forecast(cp, t0).WorkEndsAt.IsEmpty())
	})
}

func TestSimulation_ExtractorToFactory(t *testing.T) {
	cp := &app.CharacterPlanet{
		LastUpdate: t0,
		Pins: []*app.PlanetPin{
			newExtractor(1, aqueousLiquids, 1081, 30*time.Minute, t0, t0.Add(4*time.Hour)),
			newFactory(2, schematicWater),
			newStorage(3, app.EveGroupSpaceports, 10_000),
		},
		Routes: []*app.PlanetRoute{
			newRoute(1, 1, 2, aqueousLiquids, 3000),
			newRoute(2, 2, 3, water, 20),
		},
	}
	t.Run("should fill the factory directly and drop what does not fit", func(t *testing.T) {
		// 2467 + 533 of 2086 fill the buffer at t0+60min, the rest has nowhere to go
		f := colonysim.Forecast(cp, t0.Add(65*time.Minute))
		assert.Equal(t, app.PinProducing, f.Pins[2].Status)
		assert.Empty(t, f.Pins[2].Contents)
		assert.Empty(t, f.Pins[3].Contents)
	})
	t.Run("should deliver the first batch", func(t *testing.T) {
		f := colonysim.Forecast(cp, t0.Add(95*time.Minute))
		assert.Equal(t, map[int64]int64{typeWater: 20}, f.Pins[3].Contents)
	})
}

func TestSimulation_TypeWithoutVolume(t *testing.T) {
	weightless := newType(9998, 1032, 0, 0)
	cp := &app.CharacterPlanet{
		LastUpdate: t0,
		Pins: []*app.PlanetPin{
			newExtractor(1, weightless, 1081, 30*time.Minute, t0, t0.Add(4*time.Hour)),
			newStorage(2, app.EveGroupCommandCenters, 0), // 500 m3
		},
		Routes: []*app.PlanetRoute{newRoute(1, 1, 2, weightless, 10_000)},
	}
	f := colonysim.Forecast(cp, t0.Add(5*time.Hour))
	assert.Equal(t, map[int64]int64{9998: 18_002}, f.Pins[2].Contents, "all 8 cycles stored")
	assert.Equal(t, app.PinStatic, f.Pins[2].Status)
}

func TestSimulation_FactoryBeyondHorizon(t *testing.T) {
	// enough inputs for 1500 cycles of 30 minutes, longer than the horizon
	cp := &app.CharacterPlanet{
		LastUpdate: t0,
		Pins: []*app.PlanetPin{
			newStorage(1, app.EveGroupStorageFacilities, 50_000, &app.PlanetPinContent{Type: aqueousLiquids, Amount: 4_500_000}),
			newFactory(2, schematicWater),
			newStorage(3, app.EveGroupSpaceports, 20_000),
		},
		Routes: []*app.PlanetRoute{
			newRoute(1, 1, 2, aqueousLiquids, 3000),
			newRoute(2, 2, 3, water, 20),
		},
	}
	f := colonysim.Forecast(cp, t0)
	assert.True(t, f.WorksBeyondHorizon)
	assert.True(t, f.WorkEndsAt.IsEmpty())
}

func TestSimulation_PinStatus(t *testing.T) {
	expiry := t0.Add(4 * time.Hour)
	toStorage := newRoute(1, 1, 2, aqueousLiquids, 10_000)
	fromStorage := newRoute(1, 2, 1, aqueousLiquids, 3000)
	toLaunchpad := newRoute(2, 1, 3, water, 20)
	cases := []struct {
		name   string
		pin    func() *app.PlanetPin // pin 1, next to storage 2 and launchpad 3
		routes []*app.PlanetRoute
		want   app.PinStatus
	}{
		{
			name: "extractor without product",
			pin: func() *app.PlanetPin {
				p := newExtractor(1, aqueousLiquids, 1081, 30*time.Minute, t0, expiry)
				p.ExtractorProductType = optional.Optional[*app.EveType]{}
				return p
			},
			routes: []*app.PlanetRoute{toStorage},
			want:   app.PinNotSetup,
		},
		{
			name:   "extractor without cycle time",
			pin:    func() *app.PlanetPin { return newExtractor(1, aqueousLiquids, 1081, 0, t0, expiry) },
			routes: []*app.PlanetRoute{toStorage},
			want:   app.PinNotSetup,
		},
		{
			name:   "extractor without quantity",
			pin:    func() *app.PlanetPin { return newExtractor(1, aqueousLiquids, 0, 30*time.Minute, t0, expiry) },
			routes: []*app.PlanetRoute{toStorage},
			want:   app.PinNotSetup,
		},
		{
			name: "extractor without program",
			pin: func() *app.PlanetPin {
				p := newExtractor(1, aqueousLiquids, 1081, 30*time.Minute, t0, expiry)
				p.InstallTime = optional.Optional[time.Time]{}
				p.ExpiryTime = optional.Optional[time.Time]{}
				return p
			},
			routes: []*app.PlanetRoute{toStorage},
			want:   app.PinNotSetup,
		},
		{
			name:   "extractor with output not routed",
			pin:    func() *app.PlanetPin { return newExtractor(1, aqueousLiquids, 1081, 30*time.Minute, t0, expiry) },
			routes: nil,
			want:   app.PinOutputNotRouted,
		},
		{
			name: "extractor without last cycle start",
			pin: func() *app.PlanetPin {
				p := newExtractor(1, aqueousLiquids, 1081, 30*time.Minute, t0, expiry)
				p.LastCycleStart = optional.Optional[time.Time]{}
				return p
			},
			routes: []*app.PlanetRoute{toStorage},
			want:   app.PinExtractorInactive,
		},
		{
			name: "factory without schematic",
			pin: func() *app.PlanetPin {
				p := newFactory(1, schematicWater)
				p.Schematic = optional.Optional[*app.EveSchematic]{}
				return p
			},
			routes: []*app.PlanetRoute{fromStorage, toLaunchpad},
			want:   app.PinNotSetup,
		},
		{
			name:   "factory with schematic unknown to the SDE",
			pin:    func() *app.PlanetPin { return newFactory(1, 999_999) },
			routes: []*app.PlanetRoute{fromStorage, toLaunchpad},
			want:   app.PinNotSetup,
		},
		{
			name: "factory with schematic from factory fallback",
			pin: func() *app.PlanetPin {
				p := newFactory(1, schematicWater)
				p.Schematic = optional.Optional[*app.EveSchematic]{}
				p.FactorySchematic = optional.New(&app.EveSchematic{ID: schematicWater})
				return p
			},
			routes: []*app.PlanetRoute{fromStorage, toLaunchpad},
			want:   app.PinFactoryIdle,
		},
		{
			name:   "factory with output not routed",
			pin:    func() *app.PlanetPin { return newFactory(1, schematicWater) },
			routes: []*app.PlanetRoute{fromStorage},
			want:   app.PinOutputNotRouted,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cp := &app.CharacterPlanet{
				LastUpdate: t0,
				Pins: []*app.PlanetPin{
					tc.pin(),
					newStorage(2, app.EveGroupStorageFacilities, 12_000),
					newStorage(3, app.EveGroupSpaceports, 10_000),
				},
				Routes: tc.routes,
			}
			assert.True(t, colonysim.New(cp).RunUntil(t0.Add(time.Minute)), "not aborted")
			f := colonysim.Forecast(cp, t0.Add(time.Minute))
			assert.Equal(t, tc.want, f.Pins[1].Status)
		})
	}
}

func TestSimulation_API(t *testing.T) {
	cp := &app.CharacterPlanet{
		LastUpdate: t0,
		Pins: []*app.PlanetPin{
			newExtractor(1, aqueousLiquids, 1081, 30*time.Minute, t0, t0.Add(4*time.Hour)),
			newStorage(2, app.EveGroupStorageFacilities, 12_000),
		},
		Routes: []*app.PlanetRoute{newRoute(1, 1, 2, aqueousLiquids, 10_000)},
	}
	t.Run("should start at the snapshot", func(t *testing.T) {
		s := colonysim.New(cp)
		assert.Equal(t, t0, s.Time())
		f := s.Forecast()
		assert.Equal(t, t0, f.Time)
		assert.Empty(t, f.Pins[2].Contents)
	})
	t.Run("should advance time when running", func(t *testing.T) {
		s := colonysim.New(cp)
		assert.True(t, s.RunUntil(t0.Add(65*time.Minute)))
		assert.Equal(t, t0.Add(65*time.Minute), s.Time())
		assert.Equal(t, t0.Add(65*time.Minute), s.Forecast().Time)
	})
	t.Run("should do nothing when running until current time", func(t *testing.T) {
		s := colonysim.New(cp)
		s.RunUntil(t0.Add(time.Hour))
		assert.True(t, s.RunUntil(t0.Add(time.Hour)))
		assert.Equal(t, t0.Add(time.Hour), s.Time())
	})
	t.Run("should continue from previous run", func(t *testing.T) {
		s1 := colonysim.New(cp)
		s1.RunUntil(t0.Add(time.Hour))
		s1.RunUntil(t0.Add(2 * time.Hour))
		s2 := colonysim.New(cp)
		s2.RunUntil(t0.Add(2 * time.Hour))
		assert.Equal(t, s2.Forecast(), s1.Forecast())
	})
	t.Run("should set time of forecast", func(t *testing.T) {
		f := colonysim.Forecast(cp, t0.Add(65*time.Minute))
		assert.Equal(t, t0.Add(65*time.Minute), f.Time)
	})
	t.Run("should return snapshot not affected by later runs", func(t *testing.T) {
		s := colonysim.New(cp)
		s.RunUntil(t0.Add(time.Hour))
		f := s.Forecast()
		s.RunUntil(t0.Add(2 * time.Hour))
		assert.Equal(t, map[int64]int64{typeAqueousLiquids: 2467 + 2086}, f.Pins[2].Contents)
	})
}

func TestSimulation_ActivityAndLastRun(t *testing.T) {
	t.Run("extractor", func(t *testing.T) {
		cp := &app.CharacterPlanet{
			LastUpdate: t0,
			Pins: []*app.PlanetPin{
				newExtractor(1, aqueousLiquids, 1081, 30*time.Minute, t0, t0.Add(4*time.Hour)),
				newStorage(2, app.EveGroupStorageFacilities, 12_000),
			},
			Routes: []*app.PlanetRoute{newRoute(1, 1, 2, aqueousLiquids, 10_000)},
		}
		f := colonysim.Forecast(cp, t0.Add(65*time.Minute))
		assert.True(t, f.Pins[1].IsActive)
		assert.Equal(t, optional.New(t0.Add(time.Hour)), f.Pins[1].LastRunTime)
		assert.False(t, f.Pins[2].IsActive)
		assert.True(t, f.Pins[2].LastRunTime.IsEmpty())

		f = colonysim.Forecast(cp, t0.Add(5*time.Hour))
		assert.False(t, f.Pins[1].IsActive)
		assert.Equal(t, optional.New(t0.Add(4*time.Hour)), f.Pins[1].LastRunTime)
	})
	t.Run("factory", func(t *testing.T) {
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
		f := colonysim.Forecast(cp, t0.Add(45*time.Minute))
		assert.True(t, f.Pins[2].IsActive)
		assert.Equal(t, optional.New(t0.Add(30*time.Minute)), f.Pins[2].LastRunTime)

		f = colonysim.Forecast(cp, t0.Add(3*time.Hour))
		assert.False(t, f.Pins[2].IsActive)
		// an idle factory keeps checking for inputs every cycle (same as RIFT)
		assert.Equal(t, optional.New(t0.Add(3*time.Hour)), f.Pins[2].LastRunTime)
	})
}
