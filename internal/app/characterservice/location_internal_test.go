package characterservice

import (
	"context"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
)

func TestCharacterService_UpdateLocationESI(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(st)
	ctx := context.Background()

	t.Run("should create new location for a station", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		el := factory.CreateEveLocationStation()
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/characters/\d+/location/`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"solar_system_id": el.SolarSystem.ID,
				"station_id":      el.ID,
			}),
		)
		// when
		changed, err := s.updateLocationESI(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterLocation,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		c2, err := s.GetCharacter(ctx, c.ID)
		require.NoError(t, err)
		assert.Equal(t, el, c2.Location)
	})

	t.Run("should create new location for a known structure", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		el := factory.CreateEveLocationStructure()
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/characters/\d+/location/`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"solar_system_id": el.SolarSystem.ID,
				"structure_id":    el.ID,
			}),
		)
		// when
		changed, err := s.updateLocationESI(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterLocation,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		c2, err := s.GetCharacter(ctx, c.ID)
		require.NoError(t, err)
		assert.Equal(t, el, c2.Location)
	})

	t.Run("should create new location for an unknown structure", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		es := factory.CreateEveSolarSystem()
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/characters/\d+/location/`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"solar_system_id": es.ID,
				"structure_id":    1_999_999_999_999,
			}),
		)
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/universe/structures/\d+/`,
			httpmock.NewJsonResponderOrPanic(http.StatusForbidden, map[string]any{
				"error": "forbidden",
			}),
		)
		// when
		changed, err := s.updateLocationESI(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterLocation,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		c2, err := s.GetCharacter(ctx, c.ID)
		require.NoError(t, err)
		assert.EqualValues(t, 1_999_999_999_999, c2.Location.ID)
		assert.Equal(t, es, c2.Location.SolarSystem)
	})
}
