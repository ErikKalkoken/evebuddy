package colonysim

import (
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/evesde"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

func TestSimulation_Run(t *testing.T) {
	cp := newExtractorColony(t0.Add(4 * time.Hour))
	t.Run("should start at the snapshot", func(t *testing.T) {
		s := newSimulation(cp)
		assert.Equal(t, t0, s.simTime)
		f := s.forecast()
		assert.Equal(t, t0, f.Time)
		assert.Empty(t, f.Pins[2].Contents)
	})
	t.Run("should advance time when running", func(t *testing.T) {
		s := newSimulation(cp)
		assert.True(t, s.runUntil(t0.Add(65*time.Minute)))
		assert.Equal(t, t0.Add(65*time.Minute), s.simTime)
		assert.Equal(t, t0.Add(65*time.Minute), s.forecast().Time)
	})
	t.Run("should do nothing when running until current time", func(t *testing.T) {
		s := newSimulation(cp)
		s.runUntil(t0.Add(time.Hour))
		assert.True(t, s.runUntil(t0.Add(time.Hour)))
		assert.Equal(t, t0.Add(time.Hour), s.simTime)
	})
	t.Run("should continue from previous run", func(t *testing.T) {
		s1 := newSimulation(cp)
		s1.runUntil(t0.Add(time.Hour))
		s1.runUntil(t0.Add(2 * time.Hour))
		s2 := newSimulation(cp)
		s2.runUntil(t0.Add(2 * time.Hour))
		assert.Equal(t, s2.forecast(), s1.forecast())
	})
	t.Run("should return snapshot not affected by later runs", func(t *testing.T) {
		s := newSimulation(cp)
		s.runUntil(t0.Add(time.Hour))
		f := s.forecast()
		s.runUntil(t0.Add(2 * time.Hour))
		assert.Equal(t, map[int64]int64{typeAqueousLiquids: 2467 + 2086}, f.Pins[2].Contents)
	})
	t.Run("should not go back in time", func(t *testing.T) {
		s := newSimulation(cp)
		s.runUntil(t0.Add(-time.Hour))
		assert.Equal(t, t0, s.simTime)
	})
	t.Run("should advance time without changes when at rest", func(t *testing.T) {
		// idle factory without inputs
		cp := &app.CharacterPlanet{
			LastUpdate: t0,
			Pins: []*app.PlanetPin{
				newStorage(1, app.EveGroupStorageFacilities, 12_000),
				newFactory(2, schematicWater),
			},
			Routes: []*app.PlanetRoute{
				newRoute(1, 1, 2, aqueousLiquids, 3000),
				newRoute(2, 2, 1, water, 20),
			},
		}
		s := newSimulation(cp)
		require.True(t, s.runUntil(t0.Add(3*time.Hour)))
		require.True(t, s.atRest)
		want := s.forecast()
		assert.True(t, s.runUntil(t0.Add(10*time.Hour)))
		assert.Equal(t, t0.Add(10*time.Hour), s.simTime)
		got := s.forecast()
		want.Time = got.Time
		assert.Equal(t, want, got)
		_, ok := s.nextChange()
		assert.False(t, ok)
	})
}

func TestSimulation_ColonyStatus(t *testing.T) {
	// pins and their routes; the storage is always present
	type part struct {
		pin    func() *app.PlanetPin
		routes []*app.PlanetRoute
	}
	activeExtractor := part{
		pin: func() *app.PlanetPin {
			return newExtractor(1, aqueousLiquids, 1081, 30*time.Minute, t0, t0.Add(4*time.Hour))
		},
		routes: []*app.PlanetRoute{newRoute(1, 1, 10, aqueousLiquids, 10_000)},
	}
	expiredExtractor := part{
		pin: func() *app.PlanetPin {
			return newExtractor(2, aqueousLiquids, 1081, 30*time.Minute, t0.Add(-4*time.Hour), t0.Add(-time.Hour))
		},
		routes: []*app.PlanetRoute{newRoute(2, 2, 10, aqueousLiquids, 10_000)},
	}
	notSetupFactory := part{
		pin: func() *app.PlanetPin {
			p := newFactory(3, 0)
			p.Schematic = optional.Optional[*app.EveSchematic]{}
			return p
		},
	}
	producingFactory := part{
		pin: func() *app.PlanetPin {
			p := newFactory(4, schematicWater)
			p.LastCycleStart = optional.New(t0.Add(-10 * time.Minute))
			return p
		},
		routes: []*app.PlanetRoute{newRoute(3, 10, 4, aqueousLiquids, 3000), newRoute(4, 4, 10, water, 20)},
	}
	idleFactory := part{
		pin:    func() *app.PlanetPin { return newFactory(5, schematicWater) },
		routes: []*app.PlanetRoute{newRoute(5, 10, 5, aqueousLiquids, 3000), newRoute(6, 5, 10, water, 20)},
	}
	fullStorage := part{
		pin: func() *app.PlanetPin {
			return newStorage(6, app.EveGroupCommandCenters, 500)
		},
		routes: []*app.PlanetRoute{newRoute(7, 1, 6, aqueousLiquids, 100_000)}, // 1000 m3
	}
	cases := []struct {
		name  string
		parts []part
		want  app.ColonyStatus
	}{
		{"empty colony", nil, app.ColonyIdle},
		{"not setup over needs attention", []part{notSetupFactory, expiredExtractor}, app.ColonyNotSetup},
		{"needs attention over extracting", []part{expiredExtractor, activeExtractor}, app.ColonyNeedsAttention},
		{"full storage needs attention", []part{activeExtractor, fullStorage}, app.ColonyNeedsAttention},
		{"extracting over producing", []part{activeExtractor, producingFactory}, app.ColonyExtracting},
		{"producing over idle", []part{producingFactory, idleFactory}, app.ColonyProducing},
		{"idle", []part{idleFactory}, app.ColonyIdle},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cp := &app.CharacterPlanet{
				LastUpdate: t0,
				Pins:       []*app.PlanetPin{newStorage(10, app.EveGroupStorageFacilities, 12_000)},
			}
			for _, x := range tc.parts {
				cp.Pins = append(cp.Pins, x.pin())
				cp.Routes = append(cp.Routes, x.routes...)
			}
			got, _ := newSimulation(cp).colonyStatus(t0)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestSimulation_RunUntilWorkEnds(t *testing.T) {
	t.Run("should return work ended when colony is not working", func(t *testing.T) {
		s := newSimulation(newExtractorColony(t0.Add(4 * time.Hour)))
		s.runUntil(t0.Add(5 * time.Hour))
		got, r := s.runUntilWorkEnds(t0.Add(24 * time.Hour))
		assert.Equal(t, runWorkEnded, r)
		assert.Equal(t, t0.Add(5*time.Hour), got)
	})
	t.Run("should return work ended when colony stops within one cycle of the horizon", func(t *testing.T) {
		cp := newFactoryColony(9000)
		s := newSimulation(cp)
		s.runUntil(t0.Add(45 * time.Minute))
		got, r := s.runUntilWorkEnds(t0.Add(100 * time.Minute))
		assert.Equal(t, runWorkEnded, r)
		assert.Equal(t, t0.Add(90*time.Minute), got)
	})
}

func TestSimulation_IncompleteExtractorDoesNotAbort(t *testing.T) {
	cases := map[string]func(p *app.PlanetPin){
		"without cycle time": func(p *app.PlanetPin) { p.ExtractorCycleTime = optional.Optional[time.Duration]{} },
		"without quantity":   func(p *app.PlanetPin) { p.ExtractorQtyPerCycle = optional.Optional[int64]{} },
		"without product":    func(p *app.PlanetPin) { p.ExtractorProductType = optional.Optional[*app.EveType]{} },
		"without program": func(p *app.PlanetPin) {
			p.InstallTime = optional.Optional[time.Time]{}
			p.ExpiryTime = optional.Optional[time.Time]{}
		},
	}
	for name, modify := range cases {
		t.Run(name, func(t *testing.T) {
			cp := newExtractorColony(t0.Add(4 * time.Hour))
			modify(cp.Pins[0])
			assert.True(t, newSimulation(cp).runUntil(t0.Add(time.Hour)))
		})
	}
}

func TestIsAtRest(t *testing.T) {
	schematic := &evesde.PlanetSchematic{}
	// factory returns an idle factory which needs 40 water.
	factory := func(water int64) *pin {
		return &pin{kind: kindFactory, schematic: schematic, demands: map[int64]int64{typeWater: 40}, contents: map[int64]int64{typeWater: water}}
	}
	stock := func(kind pinKind, typeID, amount int64) *pin {
		return &pin{kind: kind, contents: map[int64]int64{typeID: amount}}
	}
	cases := []struct {
		name   string
		pins   []*pin // with IDs from 1
		routes []route
		want   bool
	}{
		{"no pins", nil, nil, true},
		{"active extractor", []*pin{{kind: kindExtractor, isActive: true}}, nil, false},
		{"active extractor past expiry", []*pin{{kind: kindExtractor, isActive: true, expiryTime: t0.Add(-time.Hour)}}, nil, false},
		{"inactive extractor", []*pin{{kind: kindExtractor}}, nil, true},
		{"active factory", []*pin{{kind: kindFactory, schematic: schematic, isActive: true}}, nil, false},
		{"factory received inputs last cycle", []*pin{{kind: kindFactory, schematic: schematic, receivedInputsLastCycle: true}}, nil, false},
		{"factory received inputs", []*pin{{kind: kindFactory, schematic: schematic, hasReceivedInputs: true}}, nil, false},
		{"idle factory", []*pin{{kind: kindFactory, schematic: schematic}}, nil, true},
		{"factory without schematic", []*pin{{kind: kindFactory, hasReceivedInputs: true, receivedInputsLastCycle: true}}, nil, true},
		{"storages", []*pin{
			{kind: kindCommandCenter, contents: map[int64]int64{typeWater: 1}},
			{kind: kindLaunchpad, contents: map[int64]int64{typeWater: 1}},
			{kind: kindStorage, contents: map[int64]int64{typeWater: 1}},
		}, nil, true},
		{"idle factory and active extractor", []*pin{
			{kind: kindFactory, schematic: schematic},
			{kind: kindExtractor, isActive: true},
		}, nil, false},
		{"factory can pull from storage", []*pin{stock(kindStorage, typeWater, 100), factory(0)},
			[]route{{id: 1, sourceID: 1, destinationID: 2, typeID: typeWater, quantity: 40}}, false},
		{"factory can pull from launchpad", []*pin{stock(kindLaunchpad, typeWater, 100), factory(0)},
			[]route{{id: 1, sourceID: 1, destinationID: 2, typeID: typeWater, quantity: 40}}, false},
		{"factory can pull from command center", []*pin{stock(kindCommandCenter, typeWater, 100), factory(0)},
			[]route{{id: 1, sourceID: 1, destinationID: 2, typeID: typeWater, quantity: 40}}, false},
		{"factory can pull missing rest of input", []*pin{stock(kindStorage, typeWater, 100), factory(39)},
			[]route{{id: 1, sourceID: 1, destinationID: 2, typeID: typeWater, quantity: 40}}, false},
		{"factory can pull through one of several routes", []*pin{stock(kindStorage, typeWater, 0), stock(kindStorage, typeWater, 100), factory(0)},
			[]route{
				{id: 1, sourceID: 1, destinationID: 3, typeID: typeWater, quantity: 40},
				{id: 2, sourceID: 2, destinationID: 3, typeID: typeWater, quantity: 40},
			}, false},
		{"stock not needed by factory", []*pin{stock(kindStorage, typePlasmoids, 100), factory(0)},
			[]route{{id: 1, sourceID: 1, destinationID: 2, typeID: typePlasmoids, quantity: 40}}, true},
		{"factory full", []*pin{stock(kindStorage, typeWater, 100), factory(40)},
			[]route{{id: 1, sourceID: 1, destinationID: 2, typeID: typeWater, quantity: 40}}, true},
		{"route without quantity", []*pin{stock(kindStorage, typeWater, 100), factory(0)},
			[]route{{id: 1, sourceID: 1, destinationID: 2, typeID: typeWater, quantity: 0}}, true},
		{"stock routed to other factory", []*pin{stock(kindStorage, typeWater, 100), factory(40), factory(0)},
			[]route{{id: 1, sourceID: 1, destinationID: 2, typeID: typeWater, quantity: 40}}, true},
		{"stock in factory", []*pin{factory(100), factory(0)},
			[]route{{id: 1, sourceID: 1, destinationID: 2, typeID: typeWater, quantity: 40}}, true},
		{"stock routed to storage", []*pin{stock(kindStorage, typeWater, 100), stock(kindStorage, typeWater, 0)},
			[]route{{id: 1, sourceID: 1, destinationID: 2, typeID: typeWater, quantity: 40}}, true},
		{"factory without schematic can pull", []*pin{stock(kindStorage, typeWater, 100), {kind: kindFactory, demands: map[int64]int64{typeWater: 40}}},
			[]route{{id: 1, sourceID: 1, destinationID: 2, typeID: typeWater, quantity: 40}}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := &simulation{pins: make(map[int64]*pin), routes: tc.routes}
			for i, p := range tc.pins {
				p.id = int64(i + 1)
				s.pins[p.id] = p
			}
			assert.Equal(t, tc.want, s.isAtRest())
		})
	}
}

// FuzzIsAtRest_WakeUpsChangeNothing verifies that pins of a colony at rest can no longer change it.
func FuzzIsAtRest_WakeUpsChangeNothing(f *testing.F) {
	for seed := range uint64(50) {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, seed uint64) {
		cp, now := randomColony(seed)
		s := newSimulation(cp)
		s.runUntil(now)
		if !s.atRest {
			return
		}
		type pinState struct {
			contents                map[int64]int64
			isActive                bool
			isRunnable              bool
			hasReceivedInputs       bool
			receivedInputsLastCycle bool
			lastCycleStartTime      time.Time
		}
		// the status only depends on this state and the time
		state := func() map[int64]pinState {
			m := make(map[int64]pinState)
			for id, p := range s.pins {
				m[id] = pinState{
					contents:                maps.Clone(p.contents),
					isActive:                p.isActive,
					isRunnable:              p.isRunnable(),
					hasReceivedInputs:       p.hasReceivedInputs,
					receivedInputsLastCycle: p.receivedInputsLastCycle,
					lastCycleStartTime:      p.lastCycleStartTime,
				}
			}
			return m
		}
		want := state()
		for round := range 3 {
			// wake up all pins which would still run, in order of their next run
			var pins []*pin
			for _, id := range s.pinIDs {
				if p := s.pins[id]; p.isRunnable() {
					pins = append(pins, p)
				}
			}
			next := func(p *pin) time.Time {
				t, ok := p.nextRunTime()
				if !ok || t.Before(s.simTime) {
					return s.simTime
				}
				return t
			}
			slices.SortStableFunc(pins, func(a, b *pin) int {
				return next(a).Compare(next(b))
			})
			for _, p := range pins {
				s.simTime = next(p)
				s.evaluatePin(p)
			}
			require.Equal(t, want, state(), "state changed in round %d", round+1)
		}
	})
}
