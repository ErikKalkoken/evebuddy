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

func TestColonyStatus(t *testing.T) {
	cases := []struct {
		name        string
		pins        []app.PinStatus
		hasProducer bool
		stopped     bool
		atRest      bool
		want        app.ColonyStatus
	}{
		{"empty colony has no activity", nil, false, false, true, app.ColonyNoActivity},
		{"only storages have no activity", []app.PinStatus{app.PinStatic, app.PinStatic}, false, false, true, app.ColonyNoActivity},
		{"full storage without producers has no activity", []app.PinStatus{app.PinStorageFull, app.PinStatic}, false, false, true, app.ColonyNoActivity},
		{"no activity even when stopped", []app.PinStatus{app.PinStatic}, false, true, true, app.ColonyNoActivity},
		{"not setup over stopped", []app.PinStatus{app.PinNotSetup, app.PinStatic}, true, true, true, app.ColonyNotSetup},
		{"input not routed is not setup", []app.PinStatus{app.PinInputNotRouted, app.PinExtracting}, true, false, false, app.ColonyNotSetup},
		{"output not routed is not setup", []app.PinStatus{app.PinOutputNotRouted, app.PinExtracting}, true, false, false, app.ColonyNotSetup},
		{"stopped over needs attention", []app.PinStatus{app.PinExtractorExpired, app.PinStatic}, true, true, true, app.ColonyStopped},
		{"stopped during wind-down", []app.PinStatus{app.PinFactoryIdle, app.PinStatic}, true, true, false, app.ColonyStopped},
		{"needs attention over extracting", []app.PinStatus{app.PinExtractorExpired, app.PinExtracting}, true, false, false, app.ColonyNeedsAttention},
		{"full storage needs attention", []app.PinStatus{app.PinExtracting, app.PinStorageFull}, true, false, false, app.ColonyNeedsAttention},
		{"inactive extractor needs attention", []app.PinStatus{app.PinExtractorInactive, app.PinProducing}, true, false, false, app.ColonyNeedsAttention},
		{"extracting over producing", []app.PinStatus{app.PinExtracting, app.PinProducing}, true, false, false, app.ColonyExtracting},
		{"producing over idle", []app.PinStatus{app.PinProducing, app.PinFactoryIdle}, true, false, false, app.ColonyProducing},
		{"producing while a factory waits for inputs", []app.PinStatus{app.PinFactoryIdle, app.PinStatic}, true, false, false, app.ColonyProducing},
		{"undefined when at rest but not stopped", []app.PinStatus{app.PinFactoryIdle, app.PinStatic}, true, false, true, app.ColonyStatusUndefined},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, colonyStatus(tc.pins, tc.hasProducer, tc.stopped, tc.atRest))
		})
	}
}

func TestSimulation_RunUntilWorkEnds(t *testing.T) {
	t.Run("should return when the colony stopped working", func(t *testing.T) {
		s := newSimulation(newExtractorColony(t0.Add(4 * time.Hour)))
		s.runUntil(t0.Add(5 * time.Hour))
		got, r := s.runUntilWorkEnds(t0.Add(24 * time.Hour))
		assert.Equal(t, runWorkEnded, r)
		assert.Equal(t, t0.Add(4*time.Hour), got)
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
