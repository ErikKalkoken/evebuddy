package colonysim

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/evesde"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

func TestPin_CanActivateFactory(t *testing.T) {
	s, ok := evesde.PlanetSchematicByID(schematicWater)
	require.True(t, ok)
	cases := []struct {
		name                    string
		noSchematic             bool
		isActive                bool
		inputs                  int64
		hasReceivedInputs       bool
		receivedInputsLastCycle bool
		want                    bool
	}{
		{name: "active", isActive: true, want: true},
		{name: "no schematic", noSchematic: true, isActive: true, want: false},
		{name: "idle, received inputs this cycle", inputs: 3000, hasReceivedInputs: true, want: true},
		{name: "idle, received inputs last cycle", inputs: 3000, receivedInputsLastCycle: true, want: true},
		// same as RIFT: an idle factory with enough inputs is only restarted when receiving inputs
		{name: "idle with enough inputs, nothing received", inputs: 3000, want: false},
		{name: "idle without enough inputs, nothing received", inputs: 2999, want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := &pin{
				kind:                    kindFactory,
				schematic:               &s,
				demands:                 map[int64]int64{typeAqueousLiquids: 3000},
				contents:                map[int64]int64{typeAqueousLiquids: tc.inputs},
				isActive:                tc.isActive,
				hasReceivedInputs:       tc.hasReceivedInputs,
				receivedInputsLastCycle: tc.receivedInputsLastCycle,
			}
			if tc.noSchematic {
				p.schematic = nil
			}
			assert.Equal(t, tc.want, p.canActivate())
		})
	}
}

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
	t.Run("should match reference values", func(t *testing.T) {
		for _, tc := range cases {
			var got []int64
			for i := range len(tc.want) {
				runTime := install.Add(time.Duration(i+1) * tc.cycleTime)
				got = append(got, extractorOutput(tc.baseValue, install, runTime, tc.cycleTime))
			}
			assert.Equal(t, tc.want, got, "base %d", tc.baseValue)
		}
	})
	t.Run("should return zero without cycle time", func(t *testing.T) {
		assert.Equal(t, int64(0), extractorOutput(1081, t0, t0.Add(time.Hour), 0))
	})
}

func TestExtractorProgram(t *testing.T) {
	t.Run("should return output of each cycle", func(t *testing.T) {
		got := extractorProgram(1081, t0, t0.Add(4*time.Hour), 30*time.Minute)
		assert.Len(t, got, 8)
		for i, v := range got {
			runTime := t0.Add(time.Duration(i+1) * 30 * time.Minute)
			assert.Equal(t, extractorOutput(1081, t0, runTime, 30*time.Minute), v, "cycle %d", i)
		}
	})
	t.Run("should ignore incomplete last cycle", func(t *testing.T) {
		got := extractorProgram(1081, t0, t0.Add(100*time.Minute), 30*time.Minute)
		assert.Len(t, got, 3)
	})
	t.Run("should return nothing for invalid programs", func(t *testing.T) {
		assert.Empty(t, extractorProgram(1081, t0, t0.Add(4*time.Hour), 0))
		assert.Empty(t, extractorProgram(1081, time.Time{}, t0.Add(4*time.Hour), 30*time.Minute))
		assert.Empty(t, extractorProgram(1081, t0, t0, 30*time.Minute))
	})
}

func TestPin_InputBufferState(t *testing.T) {
	newGelMatrixFactory := func(contents map[int64]int64) *pin {
		pp := newFactory(1, schematicGelMatrix) // needs 10 oxides, biocells and superconductors
		for typeID, amount := range contents {
			pp.Contents = append(pp.Contents, &app.PlanetPinContent{Type: &app.EveType{ID: typeID}, Amount: amount})
		}
		p, ok := newPin(pp, t0)
		require.True(t, ok)
		return p
	}
	t.Run("should be zero without schematic", func(t *testing.T) {
		pp := newFactory(1, 0)
		pp.Schematic = optional.Optional[*app.EveSchematic]{}
		p, ok := newPin(pp, t0)
		require.True(t, ok)
		assert.Equal(t, 0.0, p.inputBufferState())
	})
	t.Run("should be highest for empty buffer", func(t *testing.T) {
		p := newGelMatrixFactory(nil)
		assert.InDelta(t, 1.0/3, p.inputBufferState(), 1e-12)
	})
	t.Run("should be lower for fuller buffers", func(t *testing.T) {
		partly := newGelMatrixFactory(map[int64]int64{typeOxides: 1, typeBiocells: 1, typeSuperconductors: 4})
		full := newGelMatrixFactory(map[int64]int64{typeOxides: 10, typeBiocells: 10, typeSuperconductors: 10})
		above := newGelMatrixFactory(map[int64]int64{typeOxides: 20, typeBiocells: 10, typeSuperconductors: 10})
		assert.InDelta(t, (1-0.6)/3, partly.inputBufferState(), 1e-12)
		assert.InDelta(t, -2.0/3, full.inputBufferState(), 1e-12)
		assert.Less(t, above.inputBufferState(), full.inputBufferState())
	})
	t.Run("should ignore types not needed", func(t *testing.T) {
		a := newGelMatrixFactory(map[int64]int64{typeOxides: 1})
		b := newGelMatrixFactory(map[int64]int64{typeOxides: 1, typeWater: 100})
		assert.Equal(t, a.inputBufferState(), b.inputBufferState())
	})
	t.Run("should not depend on map order", func(t *testing.T) {
		// the sum of these ratios depends on the order of adding them
		p := newGelMatrixFactory(map[int64]int64{typeOxides: 1, typeBiocells: 1, typeSuperconductors: 4})
		want := p.inputBufferState()
		for range 200 {
			if !assert.Equal(t, want, p.inputBufferState()) {
				return
			}
		}
	})
}

func TestPin_ByKind(t *testing.T) {
	s, ok := evesde.PlanetSchematicByID(schematicWater)
	require.True(t, ok)
	runTime := t0.Add(time.Hour)
	t.Run("storage", func(t *testing.T) {
		p := &pin{kind: kindStorage, isActive: true, contents: map[int64]int64{typeWater: 5}}
		assert.Zero(t, p.cycle())
		assert.False(t, p.canActivate())
		assert.Empty(t, p.run(runTime))
		assert.False(t, p.isActive)
		assert.Equal(t, runTime, p.lastRunTime)
		assert.Equal(t, map[int64]int64{typeWater: 5}, p.contents)
	})
	t.Run("extractor", func(t *testing.T) {
		p := &pin{kind: kindExtractor, cycleTime: 30 * time.Minute}
		assert.Equal(t, 30*time.Minute, p.cycle())
	})
	t.Run("extractor without product", func(t *testing.T) {
		p := &pin{kind: kindExtractor, isActive: true, baseValue: 1081, cycleTime: 30 * time.Minute, installTime: t0}
		assert.False(t, p.canActivate())
		assert.Empty(t, p.run(runTime))
		assert.Equal(t, runTime, p.lastRunTime)
	})
	t.Run("factory", func(t *testing.T) {
		p := &pin{kind: kindFactory, schematic: &s}
		assert.Equal(t, s.CycleTime, p.cycle())
	})
	t.Run("factory without schematic", func(t *testing.T) {
		p := &pin{kind: kindFactory}
		assert.Zero(t, p.cycle())
	})
}

func TestPin_RemoveCommodity(t *testing.T) {
	cases := []struct {
		name     string
		quantity int64
		want     int64
		contents map[int64]int64
	}{
		{"part", 3, 3, map[int64]int64{typeWater: 2}},
		{"all", 5, 5, map[int64]int64{}},
		{"more than stored", 8, 5, map[int64]int64{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := &pin{contents: map[int64]int64{typeWater: 5}}
			assert.Equal(t, tc.want, p.removeCommodity(typeWater, tc.quantity))
			assert.Equal(t, tc.contents, p.contents)
		})
	}
	t.Run("not stored", func(t *testing.T) {
		p := &pin{contents: map[int64]int64{typeWater: 5}}
		assert.Zero(t, p.removeCommodity(typeAqueousLiquids, 3))
		assert.Equal(t, map[int64]int64{typeWater: 5}, p.contents)
	})
}
