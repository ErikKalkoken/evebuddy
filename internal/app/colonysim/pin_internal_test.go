package colonysim

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/evesde"
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
	for _, tc := range cases {
		var got []int64
		for i := range len(tc.want) {
			runTime := install.Add(time.Duration(i+1) * tc.cycleTime)
			got = append(got, extractorOutput(tc.baseValue, install, runTime, tc.cycleTime))
		}
		assert.Equal(t, tc.want, got, "base %d", tc.baseValue)
	}
}

func TestExtractorOutput_WithoutCycleTime(t *testing.T) {
	assert.Equal(t, int64(0), extractorOutput(1081, t0, t0.Add(time.Hour), 0))
}
