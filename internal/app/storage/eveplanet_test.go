package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
)

func TestEvePlanet(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		solarSystem := factory.CreateEveSolarSystem()
		type_ := factory.CreateEveType()
		arg := storage.CreateEvePlanetParams{
			ID:            42,
			Name:          "name",
			SolarSystemID: solarSystem.ID,
			TypeID:        type_.ID,
		}
		// when
		err := st.CreateEvePlanet(ctx, arg)
		// then
		if assert.NoError(t, err) {
			o, err := st.GetEvePlanet(ctx, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, int32(42), o.ID)
				assert.Equal(t, "name", o.Name)
				assert.Equal(t, solarSystem, o.SolarSystem)
				assert.Equal(t, type_, o.Type)
			}
		}
	})
}
