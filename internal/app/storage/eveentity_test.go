package storage_test

import (
	"context"
	"fmt"
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

func TestEveEntityUpdateOrCreate(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// given
		e1 := factory.CreateEveEntity(
			app.EveEntity{
				ID:       42,
				Name:     "Alpha",
				Category: app.EveEntityCharacter,
			})
		// when
		_, err := st.UpdateOrCreateEveEntity(ctx, storage.CreateEveEntityParams{
			ID:       e1.ID,
			Name:     "Erik",
			Category: app.EveEntityCorporation,
		})
		// then
		require.NoError(t, err)
		e2, err := st.GetEveEntity(ctx, e1.ID)
		require.NoError(t, err)
		xassert.Equal(t, e1.ID, e2.ID)
		xassert.Equal(t, "Erik", e2.Name)
		xassert.Equal(t, app.EveEntityCorporation, e2.Category)
	})
	t.Run("should not store with invalid ID 3", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		_, err := st.UpdateOrCreateEveEntity(ctx, storage.CreateEveEntityParams{
			ID:       0,
			Name:     "Dummy",
			Category: app.EveEntityAlliance,
		})
		// then
		assert.Error(t, err)
	})
}

func TestUpdateEveEntity(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// given
		e1 := factory.CreateEveEntity(
			app.EveEntity{
				ID:       42,
				Name:     "Alpha",
				Category: app.EveEntityCharacter,
			})
		// when
		err := st.UpdateEveEntity(ctx, e1.ID, "Erik")
		// then
		require.NoError(t, err)
		e2, err := st.GetEveEntity(ctx, e1.ID)
		require.NoError(t, err)
		xassert.Equal(t, e1.ID, e2.ID)
		xassert.Equal(t, "Erik", e2.Name)
		xassert.Equal(t, app.EveEntityCharacter, e2.Category)
	})
}

func TestEveEntity(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		_, err := st.CreateEveEntity(ctx, storage.CreateEveEntityParams{42, "Dummy", app.EveEntityAlliance})
		// then
		require.NoError(t, err)
		e, err := st.GetEveEntity(ctx, 42)
		require.NoError(t, err)
		xassert.Equal(t, e.ID, 42)
		xassert.Equal(t, e.Name, "Dummy")
		xassert.Equal(t, e.Category, app.EveEntityAlliance)
	})
	t.Run("can fetch existing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// given
		e1 := factory.CreateEveEntity(
			app.EveEntity{
				ID:       42,
				Name:     "Alpha",
				Category: app.EveEntityCharacter,
			})
		// when
		e2, err := st.GetEveEntity(ctx, e1.ID)
		// then
		require.NoError(t, err)
		xassert.Equal(t, e1.ID, e2.ID)
		xassert.Equal(t, "Alpha", e2.Name)
		xassert.Equal(t, app.EveEntityCharacter, e2.Category)
	})
	t.Run("should return error when no object found 1", func(t *testing.T) {
		_, err := st.GetEveEntity(ctx, 99)
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
	t.Run("should not store with invalid ID 1", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		_, err := st.CreateEveEntity(ctx, storage.CreateEveEntityParams{0, "Dummy", app.EveEntityAlliance})
		// then
		assert.Error(t, err)
	})
	t.Run("should not store with invalid ID 2", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		arg := storage.CreateEveEntityParams{
			ID:       0,
			Name:     "Dummy",
			Category: app.EveEntityAlliance,
		}
		_, err := st.GetOrCreateEveEntity(ctx, arg)
		// then
		assert.Error(t, err)
	})
}

func TestListEveEntityByNameAndCategory(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("should return objs with matching name and category", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		factory.CreateEveEntityCharacter(app.EveEntity{Name: "Erik"})
		factory.CreateEveEntityAlliance(app.EveEntity{Name: "Erik"})
		factory.CreateEveEntityCharacter(app.EveEntity{Name: "Other"})
		// when
		ee, err := st.ListEveEntityByNameAndCategory(ctx, "Erik", app.EveEntityCharacter)
		// then
		require.NoError(t, err)
		got := xslices.Map(ee, func(e *app.EveEntity) string {
			return e.Name
		})
		want := []string{"Erik"}
		xassert.Equal(t, want, got)
	})
	t.Run("should return error for empty name", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		_, err := st.ListEveEntityByNameAndCategory(ctx, "", app.EveEntityCharacter)
		// then
		assert.Error(t, err)
	})
	t.Run("should return error for undefined category", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		_, err := st.ListEveEntityByNameAndCategory(ctx, "Erik", app.EveEntityUndefined)
		// then
		assert.Error(t, err)
	})
}

func TestListEveEntitiesForIDs(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("should return objs with matching ids in requested order", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		factory.CreateEveEntity(app.EveEntity{ID: 1})
		factory.CreateEveEntity(app.EveEntity{ID: 2})
		factory.CreateEveEntity(app.EveEntity{ID: 3})
		factory.CreateEveEntity(app.EveEntity{ID: 4})
		// when
		ee, err := st.ListEveEntitiesForIDs(ctx, []int64{4, 1, 3})
		// then
		require.NoError(t, err)
		got := xslices.Map(ee, func(a *app.EveEntity) int64 {
			return a.ID
		})
		want := []int64{4, 1, 3}
		xassert.Equal(t, want, got)
	})
	t.Run("should return objs with matching ids in requested order", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		factory.CreateEveEntity(app.EveEntity{ID: 1})
		factory.CreateEveEntity(app.EveEntity{ID: 2})
		factory.CreateEveEntity(app.EveEntity{ID: 3})
		factory.CreateEveEntity(app.EveEntity{ID: 4})
		// when
		ee, err := st.ListEveEntitiesForIDs(ctx, []int64{4, 1, 3})
		// then
		require.NoError(t, err)
		got := xslices.Map(ee, func(a *app.EveEntity) int64 {
			return a.ID
		})
		want := []int64{4, 1, 3}
		xassert.Equal(t, want, got)
	})
	t.Run("should return objs with matching ids and chunking", func(t *testing.T) {
		// given
		old := st.MaxIDsPerQuery
		st.MaxIDsPerQuery = 2
		defer func() {
			st.MaxIDsPerQuery = old
		}()
		testutil.MustTruncateTables(db)
		factory.CreateEveEntity(app.EveEntity{ID: 1})
		factory.CreateEveEntity(app.EveEntity{ID: 2})
		factory.CreateEveEntity(app.EveEntity{ID: 3})
		factory.CreateEveEntity(app.EveEntity{ID: 4})
		// when
		ee, err := st.ListEveEntitiesForIDs(ctx, []int64{2, 3, 4})
		// then
		require.NoError(t, err)
		got := xslices.Map(ee, func(a *app.EveEntity) int64 {
			return a.ID
		})
		want := []int64{2, 3, 4}
		assert.ElementsMatch(t, want, got)
	})
	t.Run("should return error when one object can not be found", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		factory.CreateEveEntity(app.EveEntity{ID: 1})
		// when
		_, err := st.ListEveEntitiesForIDs(ctx, []int64{1, 2})
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
}

func TestListExistingEveEntitiesForIDs(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("should return objs with matching ids in requested order", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		factory.CreateEveEntity(app.EveEntity{ID: 1})
		factory.CreateEveEntity(app.EveEntity{ID: 2})
		factory.CreateEveEntity(app.EveEntity{ID: 3})
		factory.CreateEveEntity(app.EveEntity{ID: 4})
		// when
		ee, err := st.ListExistingEveEntitiesForIDs(ctx, []int64{4, 1, 3})
		// then
		require.NoError(t, err)
		got := xslices.Map(ee, func(a *app.EveEntity) int64 {
			return a.ID
		})
		want := []int64{4, 1, 3}
		xassert.Equal(t, want, got)
	})
	t.Run("should skip ids without object", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		factory.CreateEveEntity(app.EveEntity{ID: 1})
		factory.CreateEveEntity(app.EveEntity{ID: 3})
		// when
		ee, err := st.ListExistingEveEntitiesForIDs(ctx, []int64{2, 3, 4, 1})
		// then
		require.NoError(t, err)
		got := xslices.Map(ee, func(a *app.EveEntity) int64 {
			return a.ID
		})
		want := []int64{3, 1}
		xassert.Equal(t, want, got)
	})
	t.Run("should return empty slice when no object can be found", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		ee, err := st.ListExistingEveEntitiesForIDs(ctx, []int64{1, 2})
		// then
		require.NoError(t, err)
		assert.Empty(t, ee)
	})
	t.Run("should return empty slice when called with no ids", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		ee, err := st.ListExistingEveEntitiesForIDs(ctx, []int64{})
		// then
		require.NoError(t, err)
		assert.Empty(t, ee)
	})
}

func TestListEveEntities(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("should return objs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		o1 := factory.CreateEveEntity()
		o2 := factory.CreateEveEntity()
		// when
		got, err := st.ListEveEntities(ctx)
		// then
		require.NoError(t, err)
		ids := xslices.Map(got, func(a *app.EveEntity) int64 {
			return a.ID
		})
		want := []int64{o1.ID, o2.ID}
		xassert.Equal(t, want, ids)
	})
}

func TestListEveEntitiesForMailRecipients(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("should return valid recipients with matching names in order", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateEveEntityCharacter(app.EveEntity{Name: "Y_alpha2"})
		factory.CreateEveEntityAlliance(app.EveEntity{Name: "X_alpha1"})
		factory.CreateEveEntityCorporation(app.EveEntity{Name: "Z_alpha3"})
		factory.CreateEveEntityCharacter(app.EveEntity{Name: "other"})
		// when
		ee, err := st.ListEveEntitiesForMailRecipients(ctx, c.ID, "ALPHA")
		// then
		require.NoError(t, err)
		got := xslices.Map(ee, func(x *app.EveEntity) string {
			return x.Name
		})
		xassert.Equal(t, []string{"X_alpha1", "Y_alpha2", "Z_alpha3"}, got)
	})
	t.Run("should not return other categories", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		for _, category := range []app.EveEntityCategory{
			app.EveEntityConstellation,
			app.EveEntityFaction,
			app.EveEntityInventoryType,
			app.EveEntityRegion,
			app.EveEntitySolarSystem,
			app.EveEntityStation,
			app.EveEntityUnknown,
		} {
			factory.CreateEveEntity(app.EveEntity{Name: "alpha", Category: category})
		}
		// when
		ee, err := st.ListEveEntitiesForMailRecipients(ctx, c.ID, "alpha")
		// then
		require.NoError(t, err)
		assert.Empty(t, ee)
	})
	t.Run("should return only mailing lists of the character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		ml1 := factory.CreateEveEntity(app.EveEntity{Name: "alpha list 1", Category: app.EveEntityMailList})
		factory.CreateCharacterMailList(c.ID, *ml1)
		ml2 := factory.CreateEveEntity(app.EveEntity{Name: "alpha list 2", Category: app.EveEntityMailList})
		factory.CreateCharacterMailList(0, *ml2) // list of another character
		factory.CreateEveEntity(app.EveEntity{Name: "alpha list 3", Category: app.EveEntityMailList})
		// when
		ee, err := st.ListEveEntitiesForMailRecipients(ctx, c.ID, "alpha")
		// then
		require.NoError(t, err)
		xassert.Equal(t, []*app.EveEntity{ml1}, ee)
	})
	t.Run("should not return NPCs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateEveEntityCharacter(app.EveEntity{ID: 3_000_001, Name: "alpha agent"})
		factory.CreateEveEntityCorporation(app.EveEntity{ID: 1_000_001, Name: "alpha navy"})
		e := factory.CreateEveEntityCharacter(app.EveEntity{Name: "alpha player"})
		// when
		ee, err := st.ListEveEntitiesForMailRecipients(ctx, c.ID, "alpha")
		// then
		require.NoError(t, err)
		xassert.Equal(t, []*app.EveEntity{e}, ee)
	})
	t.Run("should match wildcard characters literally", func(t *testing.T) {
		cases := []struct {
			partial string
			want    string
		}{
			{"_", "a_c"},
			{"%", "a%c"},
			{"!", "a!c"},
		}
		for _, tc := range cases {
			t.Run(tc.partial, func(t *testing.T) {
				// given
				testutil.MustTruncateTables(db)
				c := factory.CreateCharacter()
				for _, name := range []string{"a_c", "a%c", "a!c", "abc"} {
					factory.CreateEveEntityCharacter(app.EveEntity{Name: name})
				}
				// when
				ee, err := st.ListEveEntitiesForMailRecipients(ctx, c.ID, tc.partial)
				// then
				require.NoError(t, err)
				got := xslices.Map(ee, func(x *app.EveEntity) string {
					return x.Name
				})
				xassert.Equal(t, []string{tc.want}, got)
			})
		}
	})
}

func TestEveEntityGetOrCreate(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("should create new when not exist", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		arg := storage.CreateEveEntityParams{
			ID:       42,
			Name:     "Dummy",
			Category: app.EveEntityAlliance,
		}
		// when
		_, err := st.GetOrCreateEveEntity(ctx, arg)
		// then
		require.NoError(t, err)
		e, err := st.GetEveEntity(ctx, 42)
		require.NoError(t, err)
		xassert.Equal(t, e.ID, 42)
		xassert.Equal(t, e.Name, "Dummy")
		xassert.Equal(t, e.Category, app.EveEntityAlliance)
	})
	t.Run("should get when exists", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// given
		factory.CreateEveEntity(
			app.EveEntity{
				ID:       42,
				Name:     "Alpha",
				Category: app.EveEntityCharacter,
			})
		arg := storage.CreateEveEntityParams{
			ID:       42,
			Name:     "Erik",
			Category: app.EveEntityCorporation,
		}
		// when
		e, err := st.GetOrCreateEveEntity(ctx, arg)
		// then
		require.NoError(t, err)
		xassert.Equal(t, 42, e.ID)
		xassert.Equal(t, "Alpha", e.Name)
		xassert.Equal(t, app.EveEntityCharacter, e.Category)
	})
}

func TestEveEntityIDs(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("should list existing entity IDs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		factory.CreateEveEntity(app.EveEntity{ID: 5})
		factory.CreateEveEntity(app.EveEntity{ID: 42})
		// when
		got, err := st.ListEveEntityIDs(ctx)
		// then
		require.NoError(t, err)
		want := set.Of[int64](5, 42)
		xassert.Equal(t, want, got)
	})
	t.Run("should return missing IDs and ignore IDs with zero value", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		factory.CreateEveEntity(app.EveEntity{ID: 42})
		// when
		got, err := st.MissingEveEntityIDs(ctx, set.Of[int64](42, 5, 0))
		// then
		require.NoError(t, err)
		want := set.Of[int64](5)
		xassert.Equal(t, want, got)
	})
}

func TestEveEntityCanCreateAllCategories(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	testutil.MustTruncateTables(db)
	var categories = []app.EveEntityCategory{
		app.EveEntityAlliance,
		app.EveEntityCharacter,
		app.EveEntityConstellation,
		app.EveEntityCorporation,
		app.EveEntityFaction,
		app.EveEntityInventoryType,
		app.EveEntityMailList,
		app.EveEntityRegion,
		app.EveEntitySolarSystem,
		app.EveEntityStation,
		app.EveEntityUnknown,
	}
	for _, c := range categories {
		t.Run(fmt.Sprintf("can create new with category %s", c), func(t *testing.T) {
			// when
			e1 := factory.CreateEveEntity(app.EveEntity{Category: c})
			// then
			e2, err := st.GetEveEntity(ctx, e1.ID)
			require.NoError(t, err)
			xassert.Equal(t, e2.Category, c)

		})
	}
}
