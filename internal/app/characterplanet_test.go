package app_test

import (
	"reflect"
	"slices"
	"testing"
	"time"

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
	t.Run("should return schematic from factory details", func(t *testing.T) {
		// given
		schematic3 := &app.EveSchematic{ID: 3}
		pin := &app.PlanetPin{
			Type:             processorType,
			FactorySchematic: optional.New(schematic3),
		}
		cp := &app.CharacterPlanet{Pins: []*app.PlanetPin{pin}}
		// when
		x := cp.ProducedSchematics()
		// then
		if assert.Len(t, x, 1) {
			assert.Equal(t, schematic3.ID, x[0].ID)
		}
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

func TestCharacterPlanet_TypeGroupNames(t *testing.T) {
	t.Run("should return group names from contents, extractors and routes", func(t *testing.T) {
		cp := app.CharacterPlanet{
			Pins: []*app.PlanetPin{
				{Contents: []*app.PlanetPinContent{{Type: &app.EveType{ID: 1, Group: &app.EveGroup{Name: "Alpha"}}}}},
				{ExtractorProductType: optional.New(&app.EveType{ID: 2, Group: &app.EveGroup{Name: "Bravo"}})},
			},
			Routes: []*app.PlanetRoute{{ContentType: &app.EveType{ID: 3}}}, // no group
		}
		assert.Equal(t, map[int64]string{1: "Alpha", 2: "Bravo"}, cp.TypeGroupNames())
	})
	t.Run("should ignore missing types", func(t *testing.T) {
		cp := app.CharacterPlanet{
			Pins:   []*app.PlanetPin{{Contents: []*app.PlanetPinContent{{}}}},
			Routes: []*app.PlanetRoute{{}},
		}
		assert.Empty(t, cp.TypeGroupNames())
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

func TestPlanetPin_Designator(t *testing.T) {
	cases := []struct {
		id   int64
		want string
	}{
		{0, "11-111"},
		{1, "21-111"},
		{33, "Y1-111"}, // Z is never used
		{34, "12-111"},
		{1046793247463, "4V-PU5"}, // matches RIFT
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			assert.Equal(t, tc.want, app.PlanetPin{ID: tc.id}.Designator())
		})
	}
}

func TestCharacterPlanet_PinName(t *testing.T) {
	cp := app.CharacterPlanet{EvePlanet: &app.EvePlanet{Type: &app.EveType{Name: "Planet (Barren)"}}}
	t.Run("should return short type name with designator", func(t *testing.T) {
		p := &app.PlanetPin{ID: 1, Type: &app.EveType{Name: "Barren Extractor Control Unit"}}
		assert.Equal(t, app.PinTypeExtractor, cp.PinTypeName(p))
		assert.Equal(t, "Extractor 21-111", cp.PinName(p))
	})
	t.Run("should return type name without planet type for unknown types", func(t *testing.T) {
		p := &app.PlanetPin{ID: 1, Type: &app.EveType{Name: "Barren Something"}}
		assert.Equal(t, "Something", cp.PinTypeName(p))
		assert.Equal(t, "Something 21-111", cp.PinName(p))
	})
}

func TestCharacterPlanet_Fingerprint(t *testing.T) {
	t0 := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	water := &app.EveType{ID: 3645, Name: "Water"}
	aqueous := &app.EveType{ID: 2268, Name: "Aqueous Liquids"}
	// newColony returns a new colony with an extractor, a storage and a factory.
	newColony := func() *app.CharacterPlanet {
		return &app.CharacterPlanet{
			ID:           1,
			CharacterID:  2,
			EvePlanet:    &app.EvePlanet{ID: 3},
			LastUpdate:   t0,
			UpgradeLevel: 4,
			Pins: []*app.PlanetPin{
				{
					ID:                   1,
					Type:                 &app.EveType{ID: 2848},
					ExtractorProductType: optional.New(aqueous),
					ExtractorQtyPerCycle: optional.New[int64](1081),
					ExtractorCycleTime:   optional.New(30 * time.Minute),
					ExtractorNumHeads:    optional.New[int64](10),
					ExtractorHeadRadius:  optional.New(0.5),
					InstallTime:          optional.New(t0),
					ExpiryTime:           optional.New(t0.Add(4 * time.Hour)),
					LastCycleStart:       optional.New(t0),
				},
				{
					ID:   2,
					Type: &app.EveType{ID: 2541},
					Contents: []*app.PlanetPinContent{
						{Type: aqueous, Amount: 100},
						{Type: water, Amount: 20},
					},
				},
				{
					ID:        3,
					Type:      &app.EveType{ID: 2473},
					Schematic: optional.New(&app.EveSchematic{ID: 121}),
				},
			},
			Routes: []*app.PlanetRoute{
				{RouteID: 1, SourcePinID: 1, DestinationPinID: 2, ContentType: aqueous, Quantity: 10_000},
				{RouteID: 2, SourcePinID: 2, DestinationPinID: 3, ContentType: aqueous, Quantity: 3000},
			},
		}
	}
	want := newColony().Fingerprint()

	t.Run("should cover all fields", func(t *testing.T) {
		// When this fails, add the new field to Fingerprint or exclude it here.
		fieldNames := func(x any) []string {
			var s []string
			for f := range reflect.TypeOf(x).Fields() {
				s = append(s, f.Name)
			}
			return s
		}
		cases := []struct {
			value    any
			included []string
			excluded []string
		}{
			{
				app.CharacterPlanet{},
				[]string{"LastUpdate", "Pins", "Routes", "UpgradeLevel"},
				[]string{"ID", "CharacterID", "EvePlanet", "LastNotified"}, // keys and local data
			},
			{
				app.PlanetPin{},
				[]string{
					"ID", "Contents", "ExpiryTime", "ExtractorCycleTime", "ExtractorHeadRadius",
					"ExtractorNumHeads", "ExtractorProductType", "ExtractorQtyPerCycle", "FactorySchematic",
					"InstallTime", "LastCycleStart", "Schematic", "Type",
				},
				nil,
			},
			{app.PlanetPinContent{}, []string{"Amount", "Type"}, nil},
			{app.PlanetRoute{}, []string{"ContentType", "DestinationPinID", "Quantity", "RouteID", "SourcePinID"}, nil},
		}
		for _, tc := range cases {
			assert.ElementsMatch(t, slices.Concat(tc.included, tc.excluded), fieldNames(tc.value), "%T", tc.value)
		}
	})
	t.Run("should not depend on order", func(t *testing.T) {
		cp := newColony()
		slices.Reverse(cp.Pins)
		slices.Reverse(cp.Routes)
		slices.Reverse(cp.Pins[1].Contents)
		assert.Equal(t, want, cp.Fingerprint())
	})
	sameCases := map[string]func(cp *app.CharacterPlanet){
		"ID":            func(cp *app.CharacterPlanet) { cp.ID = 99 },
		"last notified": func(cp *app.CharacterPlanet) { cp.LastNotified = optional.New(t0) },
		"type name":     func(cp *app.CharacterPlanet) { cp.Pins[0].Type = &app.EveType{ID: 2848, Name: "Other"} },
	}
	for name, change := range sameCases {
		t.Run("should ignore "+name, func(t *testing.T) {
			cp := newColony()
			change(cp)
			assert.Equal(t, want, cp.Fingerprint())
		})
	}
	changeCases := map[string]func(cp *app.CharacterPlanet){
		"last update":         func(cp *app.CharacterPlanet) { cp.LastUpdate = t0.Add(time.Second) },
		"upgrade level":       func(cp *app.CharacterPlanet) { cp.UpgradeLevel = 5 },
		"added route":         func(cp *app.CharacterPlanet) { cp.Routes = append(cp.Routes, &app.PlanetRoute{RouteID: 3}) },
		"removed route":       func(cp *app.CharacterPlanet) { cp.Routes = cp.Routes[:1] },
		"route quantity":      func(cp *app.CharacterPlanet) { cp.Routes[0].Quantity = 1 },
		"route content":       func(cp *app.CharacterPlanet) { cp.Routes[0].ContentType = water },
		"route destination":   func(cp *app.CharacterPlanet) { cp.Routes[0].DestinationPinID = 3 },
		"removed pin":         func(cp *app.CharacterPlanet) { cp.Pins = cp.Pins[:2] },
		"pin type":            func(cp *app.CharacterPlanet) { cp.Pins[1].Type = &app.EveType{ID: 2542} },
		"content amount":      func(cp *app.CharacterPlanet) { cp.Pins[1].Contents[0].Amount = 101 },
		"content type":        func(cp *app.CharacterPlanet) { cp.Pins[1].Contents[1].Type = &app.EveType{ID: 1} },
		"last cycle start":    func(cp *app.CharacterPlanet) { cp.Pins[0].LastCycleStart = optional.New(t0.Add(time.Minute)) },
		"missing cycle start": func(cp *app.CharacterPlanet) { cp.Pins[0].LastCycleStart = optional.Optional[time.Time]{} },
		"expiry":              func(cp *app.CharacterPlanet) { cp.Pins[0].ExpiryTime = optional.New(t0) },
		"install time":        func(cp *app.CharacterPlanet) { cp.Pins[0].InstallTime = optional.New(t0.Add(-time.Hour)) },
		"product":             func(cp *app.CharacterPlanet) { cp.Pins[0].ExtractorProductType = optional.New(water) },
		"yield":               func(cp *app.CharacterPlanet) { cp.Pins[0].ExtractorQtyPerCycle = optional.New[int64](1) },
		"cycle time":          func(cp *app.CharacterPlanet) { cp.Pins[0].ExtractorCycleTime = optional.New(time.Hour) },
		"heads":               func(cp *app.CharacterPlanet) { cp.Pins[0].ExtractorNumHeads = optional.New[int64](9) },
		"head radius":         func(cp *app.CharacterPlanet) { cp.Pins[0].ExtractorHeadRadius = optional.New(0.6) },
		"schematic":           func(cp *app.CharacterPlanet) { cp.Pins[2].Schematic = optional.New(&app.EveSchematic{ID: 122}) },
		"factory schematic": func(cp *app.CharacterPlanet) {
			cp.Pins[2].FactorySchematic = optional.New(&app.EveSchematic{ID: 121})
		},
		"zero instead of missing yield": func(cp *app.CharacterPlanet) {
			cp.Pins[1].ExtractorQtyPerCycle = optional.New[int64](0)
		},
	}
	for name, change := range changeCases {
		t.Run("should change with "+name, func(t *testing.T) {
			cp := newColony()
			change(cp)
			assert.NotEqual(t, want, cp.Fingerprint())
		})
	}
}

func BenchmarkCharacterPlanet_Fingerprint(b *testing.B) {
	t0 := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	cp := &app.CharacterPlanet{LastUpdate: t0}
	for i := range int64(9) {
		cp.Pins = append(cp.Pins, &app.PlanetPin{
			ID:                   i + 1,
			Type:                 &app.EveType{ID: 2848},
			Contents:             []*app.PlanetPinContent{{Type: &app.EveType{ID: 1}, Amount: 100}, {Type: &app.EveType{ID: 2}, Amount: 20}},
			ExtractorProductType: optional.New(&app.EveType{ID: 3}),
			ExtractorQtyPerCycle: optional.New[int64](1081),
			ExtractorCycleTime:   optional.New(30 * time.Minute),
			InstallTime:          optional.New(t0),
			ExpiryTime:           optional.New(t0.Add(4 * time.Hour)),
			LastCycleStart:       optional.New(t0),
		})
	}
	for i := range int64(11) {
		cp.Routes = append(cp.Routes, &app.PlanetRoute{RouteID: i + 1, SourcePinID: 1, DestinationPinID: 2, ContentType: &app.EveType{ID: 1}, Quantity: 3000})
	}
	for b.Loop() {
		cp.Fingerprint()
	}
}
