package colonysim

import (
	"testing"

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
