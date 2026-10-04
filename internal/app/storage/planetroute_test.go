package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestPlanetRoute(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create and list", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		planet := factory.CreateCharacterPlanet()
		contentType := factory.CreateEveType()
		// when
		err := st.CreatePlanetRoute(ctx, storage.CreatePlanetRouteParams{
			CharacterPlanetID: planet.ID,
			ContentTypeID:     contentType.ID,
			DestinationPinID:  2,
			Quantity:          20,
			RouteID:           4,
			SourcePinID:       1,
		})
		// then
		require.NoError(t, err)
		oo, err := st.ListPlanetRoutes(ctx, planet.ID)
		require.NoError(t, err)
		if assert.Len(t, oo, 1) {
			o := oo[0]
			xassert.Equal(t, contentType, o.ContentType)
			xassert.Equal(t, 2, o.DestinationPinID)
			xassert.Equal(t, 20, o.Quantity)
			xassert.Equal(t, 4, o.RouteID)
			xassert.Equal(t, 1, o.SourcePinID)
		}
	})
	t.Run("should return error when mandatory field missing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		planet := factory.CreateCharacterPlanet()
		// when
		err := st.CreatePlanetRoute(ctx, storage.CreatePlanetRouteParams{
			CharacterPlanetID: planet.ID,
		})
		// then
		assert.Error(t, err)
	})
	t.Run("should load routes with character planet", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		planet := factory.CreateCharacterPlanet()
		r := factory.CreatePlanetRoute(storage.CreatePlanetRouteParams{CharacterPlanetID: planet.ID})
		// when
		p, err := st.GetCharacterPlanet(ctx, planet.CharacterID, planet.EvePlanet.ID)
		// then
		require.NoError(t, err)
		if assert.Len(t, p.Routes, 1) {
			xassert.Equal(t, r, p.Routes[0])
		}
	})
}
