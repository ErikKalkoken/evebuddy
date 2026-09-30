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
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
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
			Capacity:       optional.New(3.0),
			Description:    "description",
			GraphicID:      optional.New[int64](4),
			GroupID:        g.ID,
			IconID:         optional.New[int64](5),
			IsPublished:    true,
			MarketGroupID:  optional.New[int64](6),
			Mass:           optional.New(7.0),
			Name:           "name",
			PackagedVolume: optional.New(8.0),
			Radius:         optional.New(9.0),
			Volume:         optional.New(10.0),
		}
		// when
		err := st.CreateEveType(ctx, arg)
		// then
		require.NoError(t, err)
		x, err := st.GetEveType(ctx, 42)
		require.NoError(t, err)
		xassert.Equal(t, 42, x.ID)
		xassert.EqualOptional(t, 3.0, x.Capacity)
		xassert.Equal(t, "description", x.Description)
		xassert.EqualOptional(t, 4, x.GraphicID)
		xassert.EqualOptional(t, 5, x.IconID)
		xassert.Equal(t, true, x.IsPublished)
		xassert.EqualOptional(t, 6, x.MarketGroupID)
		xassert.EqualOptional(t, 7.0, x.Mass)
		xassert.Equal(t, "name", x.Name)
		xassert.EqualOptional(t, 8.0, x.PackagedVolume)
		xassert.EqualOptional(t, 9.0, x.Radius)
		xassert.EqualOptional(t, 10.0, x.Volume)
		xassert.Equal(t, g, x.Group)
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
		require.NoError(t, err)
		xassert.Equal(t, want, got)
	})
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		g := factory.CreateEveGroup()
		arg := storage.CreateEveTypeParams{
			ID:             42,
			Capacity:       optional.New(3.0),
			Description:    "description",
			GraphicID:      optional.New[int64](4),
			GroupID:        g.ID,
			IconID:         optional.New[int64](5),
			IsPublished:    true,
			MarketGroupID:  optional.New[int64](6),
			Mass:           optional.New(7.0),
			Name:           "name",
			PackagedVolume: optional.New(8.0),
			Radius:         optional.New(9.0),
			Volume:         optional.New(10.0),
		}
		// when
		x, err := st.GetOrCreateEveType(ctx, arg)
		// then
		require.NoError(t, err)
		xassert.Equal(t, 42, x.ID)
		xassert.EqualOptional(t, 3.0, x.Capacity)
		xassert.Equal(t, "description", x.Description)
		xassert.EqualOptional(t, 4, x.GraphicID)
		xassert.EqualOptional(t, 5, x.IconID)
		xassert.Equal(t, true, x.IsPublished)
		xassert.EqualOptional(t, 6, x.MarketGroupID)
		xassert.EqualOptional(t, 7.0, x.Mass)
		xassert.Equal(t, "name", x.Name)
		xassert.EqualOptional(t, 8.0, x.PackagedVolume)
		xassert.EqualOptional(t, 9.0, x.Radius)
		xassert.EqualOptional(t, 10.0, x.Volume)
		xassert.Equal(t, g, x.Group)
		x2, err := st.GetEveType(ctx, 42)
		require.NoError(t, err)
		xassert.Equal(t, x, x2)
	})
	t.Run("can list IDs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		x1 := factory.CreateEveType()
		x2 := factory.CreateEveType()
		// when
		got, err := st.ListEveTypeIDs(ctx)
		// then
		require.NoError(t, err)
		want := set.Of(x1.ID, x2.ID)
		xassert.Equal(t, want, got)
	})
	t.Run("can list types", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		x1 := factory.CreateEveType()
		x2 := factory.CreateEveType()
		// when
		got, err := st.ListEveTypes(ctx)
		// then
		require.NoError(t, err)
		assert.ElementsMatch(t, []*app.EveType{x1, x2}, got)
	})
	t.Run("can identify missing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 7})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 8})
		// when
		x, err := st.MissingEveTypes(ctx, set.Of[int64](7, 9))
		// then
		require.NoError(t, err)
		assert.True(t, set.Of[int64](9).Equal(x))
	})
	t.Run("can list skills", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		category := factory.CreateEveCategory(storage.CreateEveCategoryParams{ID: app.EveCategorySkill})
		group := factory.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: category.ID, IsPublished: true})
		o1 := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID, IsPublished: true})
		factory.CreateEveType()
		// when
		oo, err := st.ListEveSkills(ctx)
		// then
		require.NoError(t, err)
		want := set.Of(o1.ID)
		got := set.Collect(xiter.MapSlice(oo, func(x *app.EveType) int64 {
			return x.ID
		}))
		xassert.Equal(t, want, got)
	})
}

func TestListEveTypesForIDs(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("should return objs with matching ids in requested order", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 1})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 2})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 3})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 4})
		// when
		oo, err := st.ListEveTypesForIDs(ctx, []int64{4, 1, 3})
		// then
		require.NoError(t, err)
		got := xslices.Map(oo, func(a *app.EveType) int64 {
			return a.ID
		})
		xassert.Equal(t, []int64{4, 1, 3}, got)
	})
	t.Run("should return fully mapped objs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		want := factory.CreateEveType()
		// when
		oo, err := st.ListEveTypesForIDs(ctx, []int64{want.ID})
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
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 1})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 2})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 3})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 4})
		// when
		oo, err := st.ListEveTypesForIDs(ctx, []int64{2, 3, 4})
		// then
		require.NoError(t, err)
		got := xslices.Map(oo, func(a *app.EveType) int64 {
			return a.ID
		})
		assert.ElementsMatch(t, []int64{2, 3, 4}, got)
	})
	t.Run("should return error when one object can not be found", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 1})
		// when
		_, err := st.ListEveTypesForIDs(ctx, []int64{1, 2})
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
	t.Run("should return empty slice for empty ids", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		oo, err := st.ListEveTypesForIDs(ctx, []int64{})
		// then
		require.NoError(t, err)
		assert.Empty(t, oo)
	})
}
