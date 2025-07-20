package app_test

import (
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/stretchr/testify/assert"
)

func TestCharacterContractDisplayName(t *testing.T) {
	cases := []struct {
		name     string
		contract *app.CharacterContract
		want     string
	}{
		{
			"courier contract",
			&app.CharacterContract{
				Type:             app.ContractTypeCourier,
				Volume:           10,
				StartSolarSystem: &app.EntityShort[int32]{Name: "Start"},
				EndSolarSystem:   &app.EntityShort[int32]{Name: "End"},
			},
			"Start >> End (10 m3)",
		},
		{
			"courier contract without solar systems",
			&app.CharacterContract{
				Type:   app.ContractTypeCourier,
				Volume: 10,
			},
			"? >> ? (10 m3)",
		},
		{
			"non-courier contract with multiple items",
			&app.CharacterContract{
				Type:  app.ContractTypeItemExchange,
				Items: []string{"first", "second"},
			},
			"[Multiple Items]",
		},
		{
			"non-courier contract with single items",
			&app.CharacterContract{
				Type:  app.ContractTypeItemExchange,
				Items: []string{"first"},
			},
			"first",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.contract.NameDisplay())
		})
	}
}

func TestCharacterNotification(t *testing.T) {
	t.Run("can convert type to title", func(t *testing.T) {
		x := &app.CharacterNotification{
			Type: "AlphaBravoCharlie",
		}
		y := x.TitleFake()
		assert.Equal(t, "Alpha Bravo Charlie", y)
	})
	t.Run("can deal with short name", func(t *testing.T) {
		x := &app.CharacterNotification{
			Type: "Alpha",
		}
		y := x.TitleFake()
		assert.Equal(t, "Alpha", y)
	})
}

func TestCharacterNotificationBodyPlain(t *testing.T) {
	t.Run("can return body as plain text", func(t *testing.T) {
		n := &app.CharacterNotification{
			Type: "Alpha",
			Body: optional.From("**alpha**"),
		}
		got, err := n.BodyPlain()
		if assert.NoError(t, err) {
			assert.Equal(t, "alpha\n", got.MustValue())
		}
	})
	t.Run("should return empty when body is empty", func(t *testing.T) {
		n := &app.CharacterNotification{
			Type: "Alpha",
		}
		got, err := n.BodyPlain()
		if assert.NoError(t, err) {
			assert.True(t, got.IsEmpty())
		}
	})
}

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
		got := make([]int32, 0)
		for _, o := range x {
			got = append(got, o.ID)
		}
		assert.ElementsMatch(t, []int32{productType1a.ID, productType2.ID}, got)
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
		got := make([]int32, 0)
		for _, o := range x {
			got = append(got, o.ID)
		}
		assert.ElementsMatch(t, []int32{schematic1a.ID, schematic2.ID}, got)

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
	productType := &app.EveType{ID: 42}
	processorPin := &app.PlanetPin{Type: processorType}
	t.Run("should return final expiration date", func(t *testing.T) {
		// given
		et1 := time.Now().Add(5 * time.Hour).UTC()
		et2 := time.Now().Add(10 * time.Hour).UTC()
		cp := &app.CharacterPlanet{Pins: []*app.PlanetPin{
			{
				Type:                 extractorType,
				ExpiryTime:           optional.From(et2),
				ExtractorProductType: productType,
			},
			{
				Type:                 extractorType,
				ExpiryTime:           optional.From(et1),
				ExtractorProductType: productType,
			},
			processorPin,
		}}
		// when
		x := cp.ExtractionsExpiryTime()
		// then
		assert.Equal(t, et2, x)
	})
	t.Run("should return expiration date in the past", func(t *testing.T) {
		// given
		et1 := time.Now().Add(-5 * time.Hour).UTC()
		cp := &app.CharacterPlanet{Pins: []*app.PlanetPin{
			{
				Type:                 extractorType,
				ExpiryTime:           optional.From(et1),
				ExtractorProductType: productType,
			},
			processorPin,
		}}
		// when
		x := cp.ExtractionsExpiryTime()
		// then
		assert.Equal(t, et1, x)
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
		x := cp.ExtractionsExpiryTime()
		// then
		assert.True(t, x.IsZero())
	})
}

func TestCharacterWalletTransaction_Total(t *testing.T) {
	cases := []struct {
		name      string
		IsBuy     bool
		UnitPrice float64
		Quantity  int32
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
