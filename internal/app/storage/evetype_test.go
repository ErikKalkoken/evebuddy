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
		xassert.Equal(t, 3.0, x.Capacity.ValueOrZero())
		xassert.Equal(t, "description", x.Description)
		xassert.Equal(t, 4, x.GraphicID.ValueOrZero())
		xassert.Equal(t, 5, x.IconID.ValueOrZero())
		xassert.Equal(t, true, x.IsPublished)
		xassert.Equal(t, 6, x.MarketGroupID.ValueOrZero())
		xassert.Equal(t, 7.0, x.Mass.ValueOrZero())
		xassert.Equal(t, "name", x.Name)
		xassert.Equal(t, 8.0, x.PackagedVolume.ValueOrZero())
		xassert.Equal(t, 9.0, x.Radius.ValueOrZero())
		xassert.Equal(t, 10.0, x.Volume.ValueOrZero())
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
		xassert.Equal(t, 3.0, x.Capacity.ValueOrZero())
		xassert.Equal(t, "description", x.Description)
		xassert.Equal(t, 4, x.GraphicID.ValueOrZero())
		xassert.Equal(t, 5, x.IconID.ValueOrZero())
		xassert.Equal(t, true, x.IsPublished)
		xassert.Equal(t, 6, x.MarketGroupID.ValueOrZero())
		xassert.Equal(t, 7.0, x.Mass.ValueOrZero())
		xassert.Equal(t, "name", x.Name)
		xassert.Equal(t, 8.0, x.PackagedVolume.ValueOrZero())
		xassert.Equal(t, 9.0, x.Radius.ValueOrZero())
		xassert.Equal(t, 10.0, x.Volume.ValueOrZero())
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
