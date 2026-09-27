package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestEvePlanet(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		solarSystem := factory.CreateEveSolarSystem()
		et := factory.CreateEveType()
		arg := storage.CreateEvePlanetParams{
			ID:            42,
			Name:          "name",
			SolarSystemID: solarSystem.ID,
			TypeID:        et.ID,
		}
		// when
		err := st.CreateEvePlanet(ctx, arg)
		// then
		require.NoError(t, err)
		o, err := st.GetEvePlanet(ctx, 42)
		require.NoError(t, err)
		xassert.Equal(t, 42, o.ID)
		xassert.Equal(t, "name", o.Name)
		xassert.Equal(t, solarSystem, o.SolarSystem)
		xassert.Equal(t, et, o.Type)
	})
}
