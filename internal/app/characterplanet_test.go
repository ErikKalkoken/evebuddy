package app_test

import (
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/stretchr/testify/assert"
)

func TestCharacterPlanetExtractedTypes(t *testing.T) {
	extractorType := &app.EveType{Group: &app.EveGroup{ID: app.EveGroupExtractorControlUnits}}
	productType1a := &app.EveType{ID: 1}
	productType1b := &app.EveType{ID: 1}
	productType2 := &app.EveType{ID: 2}
	extractorPin1a := &app.PlanetPin{
		Type:                 extractorType,
		ExtractorProductType: productType1a,
	}
	extractorPin1b := &app.PlanetPin{
		Type:                 extractorType,
		ExtractorProductType: productType1b,
	}
	extractorPin2 := &app.PlanetPin{
		Type:                 extractorType,
		ExtractorProductType: productType2,
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
		assert.Equal(t, []*app.EveType{productType1a, productType2}, x)
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

func TestCharacterPlanetProducedSchematics(t *testing.T) {
	extractorType := &app.EveType{Group: &app.EveGroup{ID: app.EveGroupExtractorControlUnits}}
	extractorPin := &app.PlanetPin{
		Type: extractorType,
	}
	processorType := &app.EveType{Group: &app.EveGroup{ID: app.EveGroupProcessors}}
	schematic1a := &app.EveSchematic{ID: 1}
	processorPin1a := &app.PlanetPin{
		Type:      processorType,
		Schematic: schematic1a,
	}
	schematic1b := &app.EveSchematic{ID: 1}
	processorPin1b := &app.PlanetPin{
		Type:      processorType,
		Schematic: schematic1b,
	}
	schematic2 := &app.EveSchematic{ID: 2}
	processorPin2 := &app.PlanetPin{
		Type:      processorType,
		Schematic: schematic2,
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
		assert.Equal(t, []*app.EveSchematic{schematic1a, schematic2}, x)
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

func TestCharacterPlanetExtractionsExpire(t *testing.T) {
	extractorType := &app.EveType{Group: &app.EveGroup{ID: app.EveGroupExtractorControlUnits}}
	processorType := &app.EveType{Group: &app.EveGroup{ID: app.EveGroupProcessors}}
	processorPin := &app.PlanetPin{Type: processorType}
	t.Run("should return final expiration date", func(t *testing.T) {
		// given
		et1 := time.Now().Add(5 * time.Hour).UTC()
		et2 := time.Now().Add(10 * time.Hour).UTC()
		cp := &app.CharacterPlanet{Pins: []*app.PlanetPin{
			{
				Type:       extractorType,
				ExpiryTime: optional.New(et2),
			},
			{
				Type:       extractorType,
				ExpiryTime: optional.New(et1),
			},
			processorPin,
		}}
		// when
		x := cp.ExtractionsExpire()
		// then
		assert.Equal(t, et2, x)
	})
	t.Run("should return zero time when no expiration date", func(t *testing.T) {
		// given
		cp := &app.CharacterPlanet{Pins: []*app.PlanetPin{
			{
				Type: extractorType,
			},
			processorPin,
		}}
		// when
		x := cp.ExtractionsExpire()
		// then
		assert.True(t, x.IsZero())
	})
}
