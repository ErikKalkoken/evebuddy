package storage_test

import (
	"context"
	"testing"

	"github.com/ErikKalkoken/kx/set"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestEveType(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		g := factory.CreateEveGroup()
		arg := storage.CreateEveTypeParams{
			ID:             42,
			Capacity:       3,
			Description:    "description",
			GraphicID:      4,
			GroupID:        g.ID,
			IconID:         5,
			IsPublished:    true,
			MarketGroupID:  6,
			Mass:           7,
			Name:           "name",
			PackagedVolume: 8,
			Radius:         9,
			Volume:         10,
		}
		// when
		err := st.CreateEveType(ctx, arg)
		// then
		if assert.NoError(t, err) {
			x, err := st.GetEveType(ctx, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, int32(42), x.ID)
				assert.Equal(t, float32(3), x.Capacity)
				assert.Equal(t, "description", x.Description)
				assert.Equal(t, int32(4), x.GraphicID)
				assert.Equal(t, int32(5), x.IconID)
				assert.Equal(t, true, x.IsPublished)
				assert.Equal(t, int32(6), x.MarketGroupID)
				assert.Equal(t, float32(7), x.Mass)
				assert.Equal(t, "name", x.Name)
				assert.Equal(t, float32(8), x.PackagedVolume)
				assert.Equal(t, float32(9), x.Radius)
				assert.Equal(t, float32(10), x.Volume)
				assert.Equal(t, g, x.Group)
			}
		}
	})
	t.Run("can get existing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		want := factory.CreateEveType()
		// when
		got, err := st.GetOrCreateEveType(ctx, storage.CreateEveTypeParams{
			ID: want.ID,
		})
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, want, got)
		}
	})
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		g := factory.CreateEveGroup()
		arg := storage.CreateEveTypeParams{
			ID:             42,
			Capacity:       3,
			Description:    "description",
			GraphicID:      4,
			GroupID:        g.ID,
			IconID:         5,
			IsPublished:    true,
			MarketGroupID:  6,
			Mass:           7,
			Name:           "name",
			PackagedVolume: 8,
			Radius:         9,
			Volume:         10,
		}
		// when
		x, err := st.GetOrCreateEveType(ctx, arg)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(42), x.ID)
			assert.Equal(t, float32(3), x.Capacity)
			assert.Equal(t, "description", x.Description)
			assert.Equal(t, int32(4), x.GraphicID)
			assert.Equal(t, int32(5), x.IconID)
			assert.Equal(t, true, x.IsPublished)
			assert.Equal(t, int32(6), x.MarketGroupID)
			assert.Equal(t, float32(7), x.Mass)
			assert.Equal(t, "name", x.Name)
			assert.Equal(t, float32(8), x.PackagedVolume)
			assert.Equal(t, float32(9), x.Radius)
			assert.Equal(t, float32(10), x.Volume)
			assert.Equal(t, g, x.Group)
			x2, err := st.GetEveType(ctx, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, x, x2)
			}
		}
	})
	t.Run("can list IDs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		x1 := factory.CreateEveType()
		x2 := factory.CreateEveType()
		// when
		got, err := st.ListEveTypeIDs(ctx)
		// then
		if assert.NoError(t, err) {
			want := set.Of(x1.ID, x2.ID)
			xassert.EqualSet(t, want, got)
		}
	})
	t.Run("can identify missing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 7})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 8})
		// when
		x, err := st.MissingEveTypes(ctx, set.Of[int32](7, 9))
		// then
		if assert.NoError(t, err) {
			assert.True(t, set.Of[int32](9).Equal(x))
		}
	})
}
