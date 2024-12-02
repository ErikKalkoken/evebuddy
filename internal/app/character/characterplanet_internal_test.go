package character

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
	t.Run("should update planets from scratch", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		factory.CreateEvePlanet(storage.CreateEvePlanetParams{ID: 40023691})
		data := []map[string]any{
			{
				"last_update":     "2016-11-28T16:42:51Z",
				"num_pins":        77,
				"owner_id":        c.ID,
				"planet_id":       40023691,
				"planet_type":     "plasma",
				"solar_system_id": 30000379,
				"upgrade_level":   3,
			},
		}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/planets/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, data))

		// when
		changed, err := s.updateCharacterPlanetsESI(ctx, UpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionPlanets,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			o, err := st.GetCharacterPlanet(ctx, c.ID, 40023691)
			if assert.NoError(t, err) {
				assert.Equal(t, time.Date(2016, 11, 28, 16, 42, 51, 0, time.UTC), o.LastUpdate)
				assert.Equal(t, 77, o.NumPins)
				assert.Equal(t, 3, o.UpgradeLevel)
			}
		}
	})
	t.Run("should replace planets", func(t *testing.T) {
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
		data := []map[string]any{
			{
				"last_update":     "2016-11-28T16:42:51Z",
				"num_pins":        77,
				"owner_id":        c.ID,
				"planet_id":       40023691,
				"planet_type":     "plasma",
				"solar_system_id": 30000379,
				"upgrade_level":   3,
			},
		}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/planets/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, data))

		// when
		changed, err := s.updateCharacterPlanetsESI(ctx, UpdateSectionParams{
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
					assert.Equal(t, 77, o.NumPins)
					assert.Equal(t, 3, o.UpgradeLevel)
				}
			}
		}
	})
}
