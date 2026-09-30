package storage_test

import (
	"context"
	"testing"

	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
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
		require.NoError(t, err)
		g, err := st.GetEveSolarSystem(ctx, 42)
		require.NoError(t, err)
		xassert.Equal(t, 42, g.ID)
		xassert.Equal(t, "name", g.Name)
		xassert.Equal(t, c, g.Constellation)
		xassert.Equal(t, float32(-8.5), g.SecurityStatus)
	})
	t.Run("can list IDs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		o1 := factory.CreateEveSolarSystem()
		o2 := factory.CreateEveSolarSystem()
		// when
		got, err := st.ListEveSolarSystemIDs(ctx)
		require.NoError(t, err)
		want := set.Of(o1.ID, o2.ID)
		xassert.Equal(t, want, got)
	})
	t.Run("can return missing IDs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		r1 := factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 42})
		// when
		got, err := st.MissingEveSolarSystems(ctx, set.Of(r1.ID, 99))
		require.NoError(t, err)
		want := set.Of[int64](99)
		xassert.Equal(t, want, got)
	})
}

func TestListEveSolarSystemsForIDs(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("should return objs with matching ids in requested order", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 1})
		factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 2})
		factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 3})
		factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 4})
		// when
		oo, err := st.ListEveSolarSystemsForIDs(ctx, []int64{4, 1, 3})
		// then
		require.NoError(t, err)
		got := xslices.Map(oo, func(a *app.EveSolarSystem) int64 {
			return a.ID
		})
		xassert.Equal(t, []int64{4, 1, 3}, got)
	})
	t.Run("should return fully mapped objs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		want := factory.CreateEveSolarSystem()
		// when
		oo, err := st.ListEveSolarSystemsForIDs(ctx, []int64{want.ID})
		// then
		require.NoError(t, err)
		require.Len(t, oo, 1)
		xassert.Equal(t, want, oo[0])
	})
	t.Run("should return objs with matching ids and chunking", func(t *testing.T) {
		// given
		old := st.MaxIDsPerQuery
		st.MaxIDsPerQuery = 2
		defer func() {
			st.MaxIDsPerQuery = old
		}()
		testutil.MustTruncateTables(db)
		factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 1})
		factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 2})
		factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 3})
		factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 4})
		// when
		oo, err := st.ListEveSolarSystemsForIDs(ctx, []int64{2, 3, 4})
		// then
		require.NoError(t, err)
		got := xslices.Map(oo, func(a *app.EveSolarSystem) int64 {
			return a.ID
		})
		assert.ElementsMatch(t, []int64{2, 3, 4}, got)
	})
	t.Run("should return error when one object can not be found", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{ID: 1})
		// when
		_, err := st.ListEveSolarSystemsForIDs(ctx, []int64{1, 2})
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
	t.Run("should return empty slice for empty ids", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		oo, err := st.ListEveSolarSystemsForIDs(ctx, []int64{})
		// then
		require.NoError(t, err)
		assert.Empty(t, oo)
	})
}
