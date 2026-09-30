package evesde_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/evesde"
)

func TestPlanetSchematicByID(t *testing.T) {
	t.Run("should return basic schematic", func(t *testing.T) {
		// Water: 3000 Aqueous Liquids -> 20 Water
		got, ok := evesde.PlanetSchematicByID(121)
		if assert.True(t, ok) {
			want := evesde.PlanetSchematic{
				ID:             121,
				CycleTime:      30 * time.Minute,
				Inputs:         []evesde.PlanetSchematicInput{{TypeID: 2268, Quantity: 3000}},
				OutputTypeID:   3645,
				OutputQuantity: 20,
			}
			assert.Equal(t, want, got)
		}
	})
	t.Run("should return schematic with multiple inputs", func(t *testing.T) {
		// Superconductors: 40 Plasmoids + 40 Water -> 5 Superconductors
		got, ok := evesde.PlanetSchematicByID(65)
		if assert.True(t, ok) {
			want := evesde.PlanetSchematic{
				ID:        65,
				CycleTime: time.Hour,
				Inputs: []evesde.PlanetSchematicInput{
					{TypeID: 2389, Quantity: 40},
					{TypeID: 3645, Quantity: 40},
				},
				OutputTypeID:   9838,
				OutputQuantity: 5,
			}
			assert.Equal(t, want, got)
		}
	})
	t.Run("should report when not found", func(t *testing.T) {
		_, ok := evesde.PlanetSchematicByID(0)
		assert.False(t, ok)
	})
	t.Run("should not allow changing the embedded data", func(t *testing.T) {
		s1, _ := evesde.PlanetSchematicByID(121)
		s1.Inputs[0].Quantity = 1
		s2, _ := evesde.PlanetSchematicByID(121)
		assert.Equal(t, int64(3000), s2.Inputs[0].Quantity)
	})
}
