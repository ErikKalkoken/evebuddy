package app_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

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
		x := cp.ProducedSchematics()
		// then
		assert.Len(t, x, 0)
	})
}

func TestCharacterPlanet_TypeNames(t *testing.T) {
	t.Run("should return names from contents, extractors and routes", func(t *testing.T) {
		cp := app.CharacterPlanet{
			Pins: []*app.PlanetPin{
				{Contents: []*app.PlanetPinContent{{Type: &app.EveType{ID: 1, Name: "Alpha"}}}},
				{ExtractorProductType: optional.New(&app.EveType{ID: 2, Name: "Bravo"})},
			},
			Routes: []*app.PlanetRoute{{ContentType: &app.EveType{ID: 3, Name: "Charlie"}}},
		}
		assert.Equal(t, map[int64]string{1: "Alpha", 2: "Bravo", 3: "Charlie"}, cp.TypeNames())
	})
	t.Run("should ignore missing types", func(t *testing.T) {
		cp := app.CharacterPlanet{
			Pins:   []*app.PlanetPin{{Contents: []*app.PlanetPinContent{{}}}},
			Routes: []*app.PlanetRoute{{}},
		}
		assert.Empty(t, cp.TypeNames())
	})
}

func TestCharacterPlanet_TypeVolumes(t *testing.T) {
	t.Run("should return volumes from contents, extractors and routes", func(t *testing.T) {
		cp := app.CharacterPlanet{
			Pins: []*app.PlanetPin{
				{Contents: []*app.PlanetPinContent{{Type: &app.EveType{ID: 1, Volume: optional.New(0.5)}}}},
				{ExtractorProductType: optional.New(&app.EveType{ID: 2, Volume: optional.New(0.01)})},
			},
			Routes: []*app.PlanetRoute{{ContentType: &app.EveType{ID: 3}}}, // no volume
		}
		assert.Equal(t, map[int64]float64{1: 0.5, 2: 0.01, 3: 0}, cp.TypeVolumes())
	})
	t.Run("should ignore missing types", func(t *testing.T) {
		cp := app.CharacterPlanet{
			Pins:   []*app.PlanetPin{{Contents: []*app.PlanetPinContent{{}}}},
			Routes: []*app.PlanetRoute{{}},
		}
		assert.Empty(t, cp.TypeVolumes())
	})
}

func TestPlanetPin_ProcessorSchematic(t *testing.T) {
	schematic := &app.EveSchematic{ID: 1}
	factorySchematic := &app.EveSchematic{ID: 2}
	t.Run("should return schematic", func(t *testing.T) {
		pp := app.PlanetPin{Schematic: optional.New(schematic), FactorySchematic: optional.New(factorySchematic)}
		got, ok := pp.ProcessorSchematic()
		assert.True(t, ok)
		assert.Equal(t, schematic, got)
	})
	t.Run("should fall back to factory schematic", func(t *testing.T) {
		pp := app.PlanetPin{FactorySchematic: optional.New(factorySchematic)}
		got, ok := pp.ProcessorSchematic()
		assert.True(t, ok)
		assert.Equal(t, factorySchematic, got)
	})
	t.Run("should report false when no schematic", func(t *testing.T) {
		_, ok := app.PlanetPin{}.ProcessorSchematic()
		assert.False(t, ok)
	})
}
