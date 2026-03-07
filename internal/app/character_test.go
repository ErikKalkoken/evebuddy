package app_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestCharacter_IDorZero(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		c := new(app.Character)
		xassert.Equal(t, 0, c.IDorZero())
	})
	t.Run("not nil", func(t *testing.T) {
		c := &app.Character{ID: 42}
		xassert.Equal(t, 42, c.IDorZero())
	})
}

func TestCharacterPlanet_ExtractedTypes(t *testing.T) {
	extractorType := &app.EveType{Group: &app.EveGroup{ID: app.EveGroupExtractorControlUnits}}
	productType1a := &app.EveType{ID: 1}
	productType1b := &app.EveType{ID: 1}
	productType2 := &app.EveType{ID: 2}
	extractorPin1a := &app.PlanetPin{
		Type:                 extractorType,
		ExtractorProductType: optional.New(productType1a),
	}
	extractorPin1b := &app.PlanetPin{
		Type:                 extractorType,
		ExtractorProductType: optional.New(productType1b),
	}
	extractorPin2 := &app.PlanetPin{
		Type:                 extractorType,
		ExtractorProductType: optional.New(productType2),
	}
	processorType := &app.EveType{Group: &app.EveGroup{ID: app.EveGroupProcessors}}
	processorPin := &app.PlanetPin{
		Type: processorType,
	}
	t.Run("should return unique extracted types", func(t *testing.T) {
		// given
		cp := &app.CharacterPlanet{Pins: []*app.PlanetPin{
			extractorPin1a,
			extractorPin1b,
			extractorPin2,
			processorPin,
		}}
		// when
		x := cp.ExtractedTypes()
		// then
		got := make([]int64, 0)
		for _, o := range x {
			got = append(got, o.ID)
		}
		assert.ElementsMatch(t, []int64{productType1a.ID, productType2.ID}, got)
	})
	t.Run("should return empty when no extractor", func(t *testing.T) {
		// given
		cp := &app.CharacterPlanet{Pins: []*app.PlanetPin{processorPin}}
		// when
		x := cp.ExtractedTypes()
		// then
		assert.Len(t, x, 0)
	})
	t.Run("should return empty when extractor, but no extraction product", func(t *testing.T) {
		// given
		pin := &app.PlanetPin{
			Type: extractorType,
		}
		cp := &app.CharacterPlanet{Pins: []*app.PlanetPin{pin}}
		// when
		x := cp.ExtractedTypes()
		// then
		assert.Len(t, x, 0)
	})
}

func TestCharacterPlanet_ProducedSchematics(t *testing.T) {
	extractorType := &app.EveType{Group: &app.EveGroup{ID: app.EveGroupExtractorControlUnits}}
	extractorPin := &app.PlanetPin{
		Type: extractorType,
	}
	processorType := &app.EveType{Group: &app.EveGroup{ID: app.EveGroupProcessors}}
	schematic1a := &app.EveSchematic{ID: 1}
	processorPin1a := &app.PlanetPin{
		Type:      processorType,
		Schematic: optional.New(schematic1a),
	}
	schematic1b := &app.EveSchematic{ID: 1}
	processorPin1b := &app.PlanetPin{
		Type:      processorType,
		Schematic: optional.New(schematic1b),
	}
	schematic2 := &app.EveSchematic{ID: 2}
	processorPin2 := &app.PlanetPin{
		Type:      processorType,
		Schematic: optional.New(schematic2),
	}
	t.Run("should return produced schematics", func(t *testing.T) {
		// given
		cp := &app.CharacterPlanet{Pins: []*app.PlanetPin{
			extractorPin,
			processorPin1a,
			processorPin1b,
			processorPin2,
		}}
		// when
		x := cp.ProducedSchematics()
		// then
		got := make([]int64, 0)
		for _, o := range x {
			got = append(got, o.ID)
		}
		assert.ElementsMatch(t, []int64{schematic1a.ID, schematic2.ID}, got)

	})
	t.Run("should return empty when no processor", func(t *testing.T) {
		// given
		cp := &app.CharacterPlanet{Pins: []*app.PlanetPin{extractorPin}}
		// when
		x := cp.ProducedSchematics()
		// then
		assert.Len(t, x, 0)
	})
	t.Run("should return empty when producer, but no schematic", func(t *testing.T) {
		// given
		pin := &app.PlanetPin{
			Type: processorType,
		}
		cp := &app.CharacterPlanet{Pins: []*app.PlanetPin{pin}}
		// when
		x := cp.ExtractedTypes()
		// then
		assert.Len(t, x, 0)
	})
}

func TestCharacterPlanet_ExtractionsExpire(t *testing.T) {
	extractorType := &app.EveType{Group: &app.EveGroup{ID: app.EveGroupExtractorControlUnits}}
	processorType := &app.EveType{Group: &app.EveGroup{ID: app.EveGroupProcessors}}
	productType := &app.EveType{ID: 42}
	processorPin := &app.PlanetPin{Type: processorType}
	t.Run("should return earlist expiration date", func(t *testing.T) {
		// given
		et1 := time.Now().Add(5 * time.Hour).UTC()
		et2 := time.Now().Add(10 * time.Hour).UTC()
		cp := &app.CharacterPlanet{Pins: []*app.PlanetPin{
			{
				Type:                 extractorType,
				ExpiryTime:           optional.New(et2),
				ExtractorProductType: optional.New(productType),
			},
			{
				Type:                 extractorType,
				ExpiryTime:           optional.New(et1),
				ExtractorProductType: optional.New(productType),
			},
			processorPin,
		}}
		// when
		x := cp.ExtractionsEarliestExpiry()
		// then
		xassert.Equal(t, et1, x.MustValue())
	})
	t.Run("should return empty time when no expiration date", func(t *testing.T) {
		// given
		cp := &app.CharacterPlanet{Pins: []*app.PlanetPin{
			{
				Type: extractorType,
			},
			processorPin,
		}}
		// when
		x := cp.ExtractionsEarliestExpiry()
		// then
		assert.True(t, x.IsEmpty())
	})
}

func TestCharacterWalletTransaction_Total(t *testing.T) {
	cases := []struct {
		name      string
		IsBuy     bool
		UnitPrice float64
		Quantity  int64
		want      float64
	}{
		{"buy", true, 1.2, 3, -3.6},
		{"sell", false, 1.2, 3, 3.6},
		{"zero", false, 0, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			o := app.CharacterWalletTransaction{
				IsBuy:     tc.IsBuy,
				UnitPrice: tc.UnitPrice,
				Quantity:  tc.Quantity,
			}
			got := o.Total()
			assert.InDelta(t, tc.want, got, 0.1)
		})
	}
}
