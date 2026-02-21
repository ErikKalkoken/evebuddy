package storage_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

func TestEveUniverseService_EveGroup(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	t.Run("can create new and get", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateEveCategory()
		arg := storage.CreateEveGroupParams{
			ID:          42,
			Name:        "name",
			CategoryID:  c.ID,
			IsPublished: true,
		}
		// when
		err := st.CreateEveGroup(t.Context(), arg)
		// then
		require.NoError(t, err)
		g, err := st.GetEveGroup(t.Context(), 42)
		require.NoError(t, err)
		xassert.Equal(t, 42, g.ID)
		xassert.Equal(t, "name", g.Name)
		xassert.Equal(t, true, g.IsPublished)
		xassert.Equal(t, c, g.Category)
	})
	t.Run("can get already existing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		g := factory.CreateEveGroup()
		// when
		got, err := st.GetOrCreateEveGroup(t.Context(), storage.CreateEveGroupParams{
			ID: g.ID,
		})
		// then
		require.NoError(t, err)
		xassert.Equal(t, g.ID, got.ID)
		xassert.Equal(t, g.Name, got.Name)
		xassert.Equal(t, g.IsPublished, got.IsPublished)
		xassert.Equal(t, g.Category, got.Category)
	})
	t.Run("can create new when not existing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateEveCategory()
		// when
		got, err := st.GetOrCreateEveGroup(t.Context(), storage.CreateEveGroupParams{
			ID:          42,
			Name:        "Alpha",
			CategoryID:  c.ID,
			IsPublished: true,
		})
		// then
		require.NoError(t, err)
		xassert.Equal(t, 42, got.ID)
		xassert.Equal(t, "Alpha", got.Name)
		assert.True(t, got.IsPublished)
		xassert.Equal(t, c, got.Category)
	})
	t.Run("can list for a category", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateEveCategory()
		g1 := factory.CreateEveGroup(storage.CreateEveGroupParams{
			CategoryID: c.ID,
		})
		g2 := factory.CreateEveGroup(storage.CreateEveGroupParams{
			CategoryID: c.ID,
		})
		factory.CreateEveGroup()
		// when
		oo, err := st.ListEveGroupsForCategory(t.Context(), c.ID)
		// then
		require.NoError(t, err)
		want := set.Of(g1.ID, g2.ID)
		got := set.Collect(xiter.MapSlice(oo, func(x *app.EveGroup) int64 {
			return x.ID
		}))
		xassert.Equal(t, want, got)
	})
	t.Run("can list skill groups", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		category := factory.CreateEveCategory(storage.CreateEveCategoryParams{ID: app.EveCategorySkill})
		group := factory.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: category.ID, IsPublished: true})
		factory.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: category.ID, IsPublished: false})
		factory.CreateEveGroup()
		// when
		xx, err := st.ListEveSkillGroups(t.Context())
		require.NoError(t, err)
		// then
		want := set.Of(group.ID)
		got := set.Collect(xiter.MapSlice(xx, func(x *app.EveSkillGroup) int64 {
			return x.ID
		}))
		xassert.Equal(t, want, got)
	})
}
