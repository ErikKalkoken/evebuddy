package characterservice

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestUpdateCharacterPlanetsESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(Params{Storage: st})
	ctx := context.Background()
	// t.Run("should update planets from scratch (minimal)", func(t *testing.T) {
	// 	// given
	// 	testutil.MustTruncateTables(db)
	// 	httpmock.Reset()
	// 	c := factory.CreateCharacterFull()
	// 	factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
	// 	factory.CreateEvePlanet(storage.CreateEvePlanetParams{ID: 40023691})
	// 	factory.CreateEveType(storage.CreateEveTypeParams{ID: 2254})
	// 	factory.CreateEveType(storage.CreateEveTypeParams{ID: 2256})
	// 	httpmock.RegisterResponder(
	// 		"GET",
	// 		fmt.Sprintf("https://esi.evetech.net/characters/%d/planets", c.ID),
	// 		httpmock.NewJsonResponderOrPanic(200, []map[string]any{
	// 			{
	// 				"last_update":     "2016-11-28T16:42:51Z",
	// 				"num_pins":        77,
	// 				"owner_id":        c.ID,
	// 				"planet_id":       40023691,
	// 				"planet_type":     "plasma",
	// 				"solar_system_id": 30000379,
	// 				"upgrade_level":   3,
	// 			},
	// 		}))
	// 	httpmock.RegisterResponder(
	// 		"GET",
	// 		fmt.Sprintf("https://esi.evetech.net/characters/%d/planets/40023691", c.ID),
	// 		httpmock.NewJsonResponderOrPanic(200, map[string]any{

	// 			"links": []map[string]any{
	// 				{
	// 					"destination_pin_id": 1000000017022,
	// 					"link_level":         0,
	// 					"source_pin_id":      1000000017021,
	// 				},
	// 			},
	// 			"pins": []map[string]any{
	// 				{
	// 					"latitude":  1.55087844973,
	// 					"longitude": 0.717145933308,
	// 					"pin_id":    1000000017021,
	// 					"type_id":   2254,
	// 				},
	// 				{
	// 					"latitude":  1.53360639935,
	// 					"longitude": 0.709775584394,
	// 					"pin_id":    1000000017022,
	// 					"type_id":   2256,
	// 				},
	// 			},
	// 			"routes": []map[string]any{
	// 				{
	// 					"content_type_id":    2393,
	// 					"destination_pin_id": 1000000017030,
	// 					"quantity":           20,
	// 					"route_id":           4,
	// 					"source_pin_id":      1000000017029,
	// 				},
	// 			},
	// 		}))
	// 	// when
	// 	changed, err := s.updatePlanetsESI(ctx, CharacterSectionUpdateParams{
	// 		CharacterID: c.ID,
	// 		Section:     app.SectionCharacterPlanets,
	// 	})
	// 	// then
	// 	if assert.NoError(t, err) {
	// 		assert.True(t, changed)
	// 		p, err := st.GetCharacterPlanet(ctx, c.ID, 40023691)
	// 		if assert.NoError(t, err) {
	// 			xassert.Equal(t, time.Date(2016, 11, 28, 16, 42, 51, 0, time.UTC), p.LastUpdate)
	// 			xassert.Equal(t, 3, p.UpgradeLevel)
	// 			pins, err := st.ListPlanetPins(ctx, p.ID)
	// 			if assert.NoError(t, err) {
	// 				got := make([]int64, 0)
	// 				for _, x := range pins {
	// 					got = append(got, x.ID)
	// 				}
	// 				assert.ElementsMatch(t, []int64{1000000017021, 1000000017022}, got)
	// 			}
	// 		}
	// 	}
	// })
	t.Run("should update planets from scratch (all field)", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		factory.CreateEvePlanet(storage.CreateEvePlanetParams{ID: 40023691})
		contentType := factory.CreateEveType()
		productType := factory.CreateEveType()
		pinType := factory.CreateEveType()
		routeType := factory.CreateEveType(storage.CreateEveTypeParams{ID: 2393})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/planets", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"last_update":     "2016-11-28T16:42:51Z",
					"num_pins":        77,
					"owner_id":        c.ID,
					"planet_id":       40023691,
					"planet_type":     "plasma",
					"solar_system_id": 30000379,
					"upgrade_level":   3,
				},
			}))
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/planets/40023691", c.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"links": []map[string]any{
					{
						"destination_pin_id": 1000000017022,
						"link_level":         0,
						"source_pin_id":      1000000017021,
					},
				},
				"pins": []map[string]any{
					{
						"contents": []map[string]any{
							{
								"amount":  42,
								"type_id": contentType.ID,
							},
						},
						"expiry_time": "2024-12-04T09:39:08Z",
						"extractor_details": map[string]any{
							"cycle_time":  1800,
							"head_radius": 0.013043995015323162,
							"heads": []map[string]any{
								{
									"head_id":   0,
									"latitude":  1.7599653005599976,
									"longitude": 4.165635108947754,
								},
							},
							"product_type_id": productType.ID,
							"qty_per_cycle":   1081,
						},
						"install_time":     "2024-12-03T07:39:08Z",
						"last_cycle_start": "2024-12-03T07:39:12Z",
						"latitude":         1.7196671962738037,
						"longitude":        4.1244120597839355,
						"pin_id":           1000000017021,
						"type_id":          pinType.ID,
					},
				},
				"routes": []map[string]any{
					{
						"content_type_id":    2393,
						"destination_pin_id": 1000000017030,
						"quantity":           20,
						"route_id":           4,
						"source_pin_id":      1000000017029,
					},
				},
			}),
		)
		// when
		changed, err := s.updatePlanetsESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterPlanets,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		p, err := st.GetCharacterPlanet(ctx, c.ID, 40023691)
		require.NoError(t, err)
		xassert.Equal(t, time.Date(2016, 11, 28, 16, 42, 51, 0, time.UTC), p.LastUpdate)
		xassert.Equal(t, 3, p.UpgradeLevel)
		pins, err := st.ListPlanetPins(ctx, p.ID)
		require.NoError(t, err)
		assert.Len(t, pins, 1)
		pin, err := st.GetPlanetPin(ctx, p.ID, 1000000017021)
		require.NoError(t, err)
		xassert.Equal(t,
			time.Date(2024, 12, 4, 9, 39, 8, 0, time.UTC),
			pin.ExpiryTime.ValueOrZero(),
		)
		xassert.Equal(t,
			time.Date(2024, 12, 3, 7, 39, 8, 0, time.UTC),
			pin.InstallTime.ValueOrZero(),
		)
		xassert.Equal(t,
			time.Date(2024, 12, 3, 7, 39, 12, 0, time.UTC),
			pin.LastCycleStart.ValueOrZero(),
		)
		xassert.EqualOptional(t, productType, pin.ExtractorProductType)
		xassert.Equal(t, pinType, pin.Type)
		xassert.EqualOptional(t, 30*time.Minute, pin.ExtractorCycleTime)
		xassert.EqualOptional(t, 0.013043995015323162, pin.ExtractorHeadRadius)
		xassert.EqualOptional(t, 1, pin.ExtractorNumHeads)
		xassert.EqualOptional(t, 1081, pin.ExtractorQtyPerCycle)
		if assert.Len(t, pin.Contents, 1) {
			xassert.Equal(t, contentType, pin.Contents[0].Type)
			xassert.Equal(t, 42, pin.Contents[0].Amount)
		}
		if assert.Len(t, p.Routes, 1) {
			r := p.Routes[0]
			xassert.Equal(t, routeType, r.ContentType)
			xassert.Equal(t, 1000000017030, r.DestinationPinID)
			xassert.Equal(t, 20, r.Quantity)
			xassert.Equal(t, 4, r.RouteID)
			xassert.Equal(t, 1000000017029, r.SourcePinID)
		}
	})
	t.Run("should update planets and remove obsoletes", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		factory.CreateEvePlanet(storage.CreateEvePlanetParams{ID: 40023691})
		factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{
			CharacterID: c.ID,
			EvePlanetID: 40023691,
		})
		factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{
			CharacterID: c.ID,
		})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 2254})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 2256})
		contentType := factory.CreateEveType()
		productType := factory.CreateEveType()
		pinType := factory.CreateEveType()
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 2393})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/planets", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"last_update":     "2016-11-28T16:42:51Z",
					"num_pins":        77,
					"owner_id":        c.ID,
					"planet_id":       40023691,
					"planet_type":     "plasma",
					"solar_system_id": 30000379,
					"upgrade_level":   3,
				},
			}))
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/planets/40023691", c.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"links": []map[string]any{
					{
						"destination_pin_id": 1000000017022,
						"link_level":         0,
						"source_pin_id":      1000000017021,
					},
				},
				"pins": []map[string]any{
					{
						"contents": []map[string]any{
							{
								"amount":  42,
								"type_id": contentType.ID,
							},
						},
						"expiry_time": "2024-12-04T09:39:08Z",
						"extractor_details": map[string]any{
							"cycle_time":  1800,
							"head_radius": 0.013043995015323162,
							"heads": []map[string]any{
								{
									"head_id":   0,
									"latitude":  1.7599653005599976,
									"longitude": 4.165635108947754,
								},
							},
							"product_type_id": productType.ID,
							"qty_per_cycle":   1081,
						},
						"install_time":     "2024-12-03T07:39:08Z",
						"last_cycle_start": "2024-12-03T07:39:12Z",
						"latitude":         1.7196671962738037,
						"longitude":        4.1244120597839355,
						"pin_id":           1000000017021,
						"type_id":          pinType.ID,
					},
				},
				"routes": []map[string]any{
					{
						"content_type_id":    2393,
						"destination_pin_id": 1000000017030,
						"quantity":           20,
						"route_id":           4,
						"source_pin_id":      1000000017029,
					},
				},
			}),
		)
		// when
		changed, err := s.updatePlanetsESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterPlanets,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		oo, err := st.ListCharacterPlanets(ctx, c.ID)
		require.NoError(t, err)
		assert.Len(t, oo, 1)
		o, err := st.GetCharacterPlanet(ctx, c.ID, 40023691)
		require.NoError(t, err)
		xassert.Equal(t, time.Date(2016, 11, 28, 16, 42, 51, 0, time.UTC), o.LastUpdate)
		xassert.Equal(t, 3, o.UpgradeLevel)
	})
}

func TestGetPlanet(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := NewFake(Params{Storage: st})
	ctx := context.Background()
	t.Run("can return a planet", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		p := factory.CreateCharacterPlanet()
		// when
		got, err := s.GetPlanet(ctx, p.CharacterID, p.EvePlanet.ID)
		// then
		require.NoError(t, err)
		assert.Equal(t, p.EvePlanet.ID, got.EvePlanet.ID)
	})
	t.Run("should return own error when not found", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		_, err := s.GetPlanet(ctx, 1, 1)
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
}

func TestListAllPlanets(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := NewFake(Params{Storage: st})
	ctx := context.Background()
	t.Run("can list planets across all characters", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		factory.CreateCharacterPlanet()
		factory.CreateCharacterPlanet()
		// when
		got, err := s.ListAllPlanets(ctx)
		// then
		require.NoError(t, err)
		assert.Len(t, got, 2)
	})
}

func TestForecastPlanet(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	s := NewFake(Params{Storage: st})
	ctx := context.Background()
	t.Run("should forecast colony loaded from storage", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		t0 := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
		cp := factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{LastUpdate: t0})
		ecuGroup := factory.CreateEveGroup(storage.CreateEveGroupParams{ID: app.EveGroupExtractorControlUnits})
		ecuType := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: ecuGroup.ID})
		storageGroup := factory.CreateEveGroup(storage.CreateEveGroupParams{ID: app.EveGroupStorageFacilities})
		storageType := factory.CreateEveType(storage.CreateEveTypeParams{
			GroupID:  storageGroup.ID,
			Capacity: optional.New(12_000.0),
		})
		product := factory.CreateEveType(storage.CreateEveTypeParams{Volume: optional.New(0.01)})
		factory.CreatePlanetPin(storage.CreatePlanetPinParams{
			CharacterPlanetID:      cp.ID,
			PinID:                  1,
			TypeID:                 ecuType.ID,
			ExtractorProductTypeID: optional.New(product.ID),
			ExtractorQtyPerCycle:   optional.New[int64](1081),
			ExtractorCycleTime:     optional.New(30 * time.Minute),
			InstallTime:            optional.New(t0),
			ExpiryTime:             optional.New(t0.Add(4 * time.Hour)),
			LastCycleStart:         optional.New(t0),
		})
		factory.CreatePlanetPin(storage.CreatePlanetPinParams{
			CharacterPlanetID: cp.ID,
			PinID:             2,
			TypeID:            storageType.ID,
			Contents:          map[int64]int64{product.ID: 100},
		})
		factory.CreatePlanetRoute(storage.CreatePlanetRouteParams{
			CharacterPlanetID: cp.ID,
			SourcePinID:       1,
			DestinationPinID:  2,
			ContentTypeID:     product.ID,
			Quantity:          10_000,
		})
		p, err := s.GetPlanet(ctx, cp.CharacterID, cp.EvePlanet.ID)
		require.NoError(t, err)
		// when
		got := s.ForecastPlanet(p, t0.Add(65*time.Minute))
		// then
		assert.Equal(t, app.ColonyExtracting, got.Status)
		assert.Equal(t, app.PinExtracting, got.Pins[1].Status)
		assert.Equal(t, map[int64]int64{product.ID: 100 + 2467 + 2086}, got.Pins[2].Contents)
		assert.InDelta(t, float64(100+2467+2086)*0.01, got.Pins[2].CapacityUsed, 0.0001)
		xassert.EqualOptional(t, 12_000.0, got.Pins[2].Capacity)
		xassert.EqualOptional(t, t0.Add(4*time.Hour), got.WorkEndsAt)
	})
}
