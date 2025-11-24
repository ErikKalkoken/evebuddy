package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

func TestEveSolarSystem(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateEveConstellation()
		arg := storage.CreateEveSolarSystemParams{
			ID:              42,
			ConstellationID: c.ID,
			Name:            "name",
			SecurityStatus:  -8.5,
		}
		// when
		err := st.CreateEveSolarSystem(ctx, arg)
		// then
		if assert.NoError(t, err) {
			g, err := st.GetEveSolarSystem(ctx, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, int32(42), g.ID)
				assert.Equal(t, "name", g.Name)
				assert.Equal(t, c, g.Constellation)
				assert.Equal(t, float32(-8.5), g.SecurityStatus)
			}
		}
	})
	t.Run("can list IDs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		o1 := factory.CreateEveSolarSystem()
		o2 := factory.CreateEveSolarSystem()
		// when
		got, err := st.ListEveSolarSystemIDs(ctx)
		if assert.NoError(t, err) {
			want := set.Of(o1.ID, o2.ID)
			assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
		}
	})
	t.Run("can return missing IDs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		r1 := factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 42})
		// when
		got, err := st.MissingEveSolarSystems(ctx, set.Of(r1.ID, 99))
		if assert.NoError(t, err) {
			want := set.Of[int32](99)
			assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
		}
	})
}
