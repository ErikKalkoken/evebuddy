package characterservice

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestCharacterService_UpdateLocationESI(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(Params{Storage: st})
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
			fmt.Sprintf("https://esi.evetech.net/characters/%d/location", c.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"solar_system_id": el.SolarSystem.ValueOrZero().ID,
				"station_id":      el.ID,
			}),
		)
		// when
		changed, err := s.updateLocationESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterLocation,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		c2, err := s.GetCharacter(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, el, c2.Location.MustValue())
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
			fmt.Sprintf("https://esi.evetech.net/characters/%d/location", c.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"solar_system_id": el.SolarSystem.ValueOrZero().ID,
				"structure_id":    el.ID,
			}),
		)
		// when
		changed, err := s.updateLocationESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterLocation,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		c2, err := s.GetCharacter(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, el, c2.Location.MustValue())
	})

	t.Run("should create new location for an unknown structure", func(t *testing.T) {
		// given
		const structureID = 1_999_999_999_999
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		es := factory.CreateEveSolarSystem()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/location", c.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"solar_system_id": es.ID,
				"structure_id":    structureID,
			}),
		)
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/universe/structures/%d", structureID),
			httpmock.NewJsonResponderOrPanic(http.StatusForbidden, map[string]any{
				"error": "forbidden",
			}),
		)
		// when
		changed, err := s.updateLocationESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterLocation,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		c2, err := s.GetCharacter(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, structureID, c2.Location.MustValue().ID)
		xassert.Equal(t, es, c2.Location.MustValue().SolarSystem.MustValue())
	})
}
