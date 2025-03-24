package characterservice

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestUpdateCharacterPlanetsESI(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := newCharacterService(st)
	ctx := context.Background()
	t.Run("should update planets from scratch (minimal)", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		factory.CreateEvePlanet(storage.CreateEvePlanetParams{ID: 40023691})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 2254})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 2256})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/planets/", c.ID),
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
			fmt.Sprintf("https://esi.evetech.net/v3/characters/%d/planets/40023691/", c.ID),
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
						"latitude":  1.55087844973,
						"longitude": 0.717145933308,
						"pin_id":    1000000017021,
						"type_id":   2254,
					},
					{
						"latitude":  1.53360639935,
						"longitude": 0.709775584394,
						"pin_id":    1000000017022,
						"type_id":   2256,
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
			}))
		// when
		changed, err := s.updateCharacterPlanetsESI(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionPlanets,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			p, err := st.GetCharacterPlanet(ctx, c.ID, 40023691)
			if assert.NoError(t, err) {
				assert.Equal(t, time.Date(2016, 11, 28, 16, 42, 51, 0, time.UTC), p.LastUpdate)
				assert.Equal(t, 3, p.UpgradeLevel)
				pins, err := st.ListPlanetPins(ctx, p.ID)
				if assert.NoError(t, err) {
					got := make([]int64, 0)
					for _, x := range pins {
						got = append(got, x.ID)
					}
					assert.ElementsMatch(t, []int64{1000000017021, 1000000017022}, got)
				}
			}
		}
	})
	t.Run("should update planets from scratch (all field)", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		factory.CreateEvePlanet(storage.CreateEvePlanetParams{ID: 40023691})
		contentType := factory.CreateEveType()
		productType := factory.CreateEveType()
		pinType := factory.CreateEveType()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/planets/", c.ID),
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
			fmt.Sprintf("https://esi.evetech.net/v3/characters/%d/planets/40023691/", c.ID),
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
			}))
		// when
		changed, err := s.updateCharacterPlanetsESI(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionPlanets,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			p, err := st.GetCharacterPlanet(ctx, c.ID, 40023691)
			if assert.NoError(t, err) {
				assert.Equal(t, time.Date(2016, 11, 28, 16, 42, 51, 0, time.UTC), p.LastUpdate)
				assert.Equal(t, 3, p.UpgradeLevel)
				pins, err := st.ListPlanetPins(ctx, p.ID)
				if assert.NoError(t, err) {
					assert.Len(t, pins, 1)
					pin, err := st.GetPlanetPin(ctx, p.ID, 1000000017021)
					if assert.NoError(t, err) {
						assert.Equal(t, time.Date(2024, 12, 4, 9, 39, 8, 0, time.UTC), pin.ExpiryTime.ValueOrZero())
						assert.Equal(t, time.Date(2024, 12, 3, 7, 39, 8, 0, time.UTC), pin.InstallTime.ValueOrZero())
						assert.Equal(t, time.Date(2024, 12, 3, 7, 39, 12, 0, time.UTC), pin.LastCycleStart.ValueOrZero())
						assert.Equal(t, productType, pin.ExtractorProductType)
						assert.Equal(t, pinType, pin.Type)
					}
				}
			}
		}
	})
	t.Run("should update planets and remove obsoletes", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
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
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/planets/", c.ID),
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
			fmt.Sprintf("https://esi.evetech.net/v3/characters/%d/planets/40023691/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"links": []map[string]any{
						{
							"destination_pin_id": 1000000017022,
							"link_level":         0,
							"source_pin_id":      1000000017021,
						},
					},
					"pins": []map[string]any{
						{
							"latitude":  1.55087844973,
							"longitude": 0.717145933308,
							"pin_id":    1000000017021,
							"type_id":   2254,
						},
						{
							"latitude":  1.53360639935,
							"longitude": 0.709775584394,
							"pin_id":    1000000017022,
							"type_id":   2256,
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
				},
			}))
		// when
		changed, err := s.updateCharacterPlanetsESI(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionPlanets,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			oo, err := st.ListCharacterPlanets(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Len(t, oo, 1)
				o, err := st.GetCharacterPlanet(ctx, c.ID, 40023691)
				if assert.NoError(t, err) {
					assert.Equal(t, time.Date(2016, 11, 28, 16, 42, 51, 0, time.UTC), o.LastUpdate)
					assert.Equal(t, 3, o.UpgradeLevel)
				}
			}
		}
	})
}
