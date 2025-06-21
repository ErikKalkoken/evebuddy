package storage_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestEveEntityUpdateOrCreate(t *testing.T) {
	db, r, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// given
		e1 := factory.CreateEveEntity(
			app.EveEntity{
				ID:       42,
				Name:     "Alpha",
				Category: app.EveEntityCharacter,
			})
		// when
		_, err := r.UpdateOrCreateEveEntity(ctx, storage.CreateEveEntityParams{
			ID:       e1.ID,
			Name:     "Erik",
			Category: app.EveEntityCorporation,
		})
		// then
		if assert.NoError(t, err) {
			e2, err := r.GetEveEntity(ctx, e1.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, e1.ID, e2.ID)
				assert.Equal(t, "Erik", e2.Name)
				assert.Equal(t, app.EveEntityCorporation, e2.Category)
			}
		}
	})
	t.Run("should not store with invalid ID 3", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		_, err := r.UpdateOrCreateEveEntity(ctx, storage.CreateEveEntityParams{
			ID:       0,
			Name:     "Dummy",
			Category: app.EveEntityAlliance,
		})
		// then
		assert.Error(t, err)
	})
}

func TestUpdateEveEntity(t *testing.T) {
	db, r, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// given
		e1 := factory.CreateEveEntity(
			app.EveEntity{
				ID:       42,
				Name:     "Alpha",
				Category: app.EveEntityCharacter,
			})
		// when
		err := r.UpdateEveEntity(ctx, e1.ID, "Erik")
		// then
		if assert.NoError(t, err) {
			e2, err := r.GetEveEntity(ctx, e1.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, e1.ID, e2.ID)
				assert.Equal(t, "Erik", e2.Name)
				assert.Equal(t, app.EveEntityCharacter, e2.Category)
			}
		}
	})
}

func TestEveEntity(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		_, err := st.CreateEveEntity(ctx, storage.CreateEveEntityParams{42, "Dummy", app.EveEntityAlliance})
		// then
		if assert.NoError(t, err) {
			e, err := st.GetEveEntity(ctx, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, e.ID, int32(42))
				assert.Equal(t, e.Name, "Dummy")
				assert.Equal(t, e.Category, app.EveEntityAlliance)
			}
		}
	})
	t.Run("can fetch existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
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
		if assert.NoError(t, err) {
			assert.Equal(t, e1.ID, e2.ID)
			assert.Equal(t, "Alpha", e2.Name)
			assert.Equal(t, app.EveEntityCharacter, e2.Category)
		}
	})
	t.Run("should return error when no object found 1", func(t *testing.T) {
		_, err := st.GetEveEntity(ctx, 99)
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
	t.Run("should return objs with matching names in order", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		factory.CreateEveEntityCharacter(app.EveEntity{Name: "Y_alpha2"})
		factory.CreateEveEntityAlliance(app.EveEntity{Name: "X_alpha1"})
		factory.CreateEveEntityCharacter(app.EveEntity{Name: "charlie"})
		factory.CreateEveEntityCharacter(app.EveEntity{Name: "other"})
		// when
		ee, err := st.ListEveEntitiesByPartialName(ctx, "%ALPHA%")
		// then
		if assert.NoError(t, err) {
			var got []string
			for _, e := range ee {
				got = append(got, e.Name)
			}
			want := []string{"X_alpha1", "Y_alpha2"}
			assert.Equal(t, want, got)
		}
	})
	t.Run("should not store with invalid ID 1", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		_, err := st.CreateEveEntity(ctx, storage.CreateEveEntityParams{0, "Dummy", app.EveEntityAlliance})
		// then
		assert.Error(t, err)
	})
	t.Run("should not store with invalid ID 2", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
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

func TestListEveEntitiesForIDs(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("should return objs with matching ids in requested order", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		factory.CreateEveEntity(app.EveEntity{ID: 1})
		factory.CreateEveEntity(app.EveEntity{ID: 2})
		factory.CreateEveEntity(app.EveEntity{ID: 3})
		factory.CreateEveEntity(app.EveEntity{ID: 4})
		// when
		ee, err := st.ListEveEntitiesForIDs(ctx, []int32{4, 1, 3})
		// then
		if assert.NoError(t, err) {
			got := xslices.Map(ee, func(a *app.EveEntity) int32 {
				return a.ID
			})
			want := []int32{4, 1, 3}
			assert.Equal(t, want, got)
		}
	})
	t.Run("should return objs with matching ids in requested order", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		factory.CreateEveEntity(app.EveEntity{ID: 1})
		factory.CreateEveEntity(app.EveEntity{ID: 2})
		factory.CreateEveEntity(app.EveEntity{ID: 3})
		factory.CreateEveEntity(app.EveEntity{ID: 4})
		// when
		ee, err := st.ListEveEntitiesForIDs(ctx, []int32{4, 1, 3})
		// then
		if assert.NoError(t, err) {
			got := xslices.Map(ee, func(a *app.EveEntity) int32 {
				return a.ID
			})
			want := []int32{4, 1, 3}
			assert.Equal(t, want, got)
		}
	})
	t.Run("should return objs with matching ids and chunking", func(t *testing.T) {
		// given
		old := st.MaxListEveEntitiesForIDs
		st.MaxListEveEntitiesForIDs = 2
		defer func() {
			st.MaxListEveEntitiesForIDs = old
		}()
		testutil.TruncateTables(db)
		factory.CreateEveEntity(app.EveEntity{ID: 1})
		factory.CreateEveEntity(app.EveEntity{ID: 2})
		factory.CreateEveEntity(app.EveEntity{ID: 3})
		factory.CreateEveEntity(app.EveEntity{ID: 4})
		// when
		ee, err := st.ListEveEntitiesForIDs(ctx, []int32{2, 3, 4})
		// then
		if assert.NoError(t, err) {
			got := xslices.Map(ee, func(a *app.EveEntity) int32 {
				return a.ID
			})
			want := []int32{2, 3, 4}
			assert.ElementsMatch(t, want, got)
		}
	})
	t.Run("should return error when one object can not be found", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		factory.CreateEveEntity(app.EveEntity{ID: 1})
		// when
		_, err := st.ListEveEntitiesForIDs(ctx, []int32{1, 2})
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
}

func TestListEveEntities(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("should return objs", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		o1 := factory.CreateEveEntity()
		o2 := factory.CreateEveEntity()
		// when
		got, err := st.ListEveEntities(ctx)
		// then
		if assert.NoError(t, err) {
			got := xslices.Map(got, func(a *app.EveEntity) int32 {
				return a.ID
			})
			want := []int32{o1.ID, o2.ID}
			assert.Equal(t, want, got)
		}
	})
}

func TestEveEntityGetOrCreate(t *testing.T) {
	db, r, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("should create new when not exist", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		arg := storage.CreateEveEntityParams{
			ID:       42,
			Name:     "Dummy",
			Category: app.EveEntityAlliance,
		}
		// when
		_, err := r.GetOrCreateEveEntity(ctx, arg)
		// then
		if assert.NoError(t, err) {
			e, err := r.GetEveEntity(ctx, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, e.ID, int32(42))
				assert.Equal(t, e.Name, "Dummy")
				assert.Equal(t, e.Category, app.EveEntityAlliance)
			}
		}
	})
	t.Run("should get when exists", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
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
		e, err := r.GetOrCreateEveEntity(ctx, arg)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(42), e.ID)
			assert.Equal(t, "Alpha", e.Name)
			assert.Equal(t, app.EveEntityCharacter, e.Category)
		}
	})
}

func TestEveEntityIDs(t *testing.T) {
	db, r, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("should list existing entity IDs", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		factory.CreateEveEntity(app.EveEntity{ID: 5})
		factory.CreateEveEntity(app.EveEntity{ID: 42})
		// when
		got, err := r.ListEveEntityIDs(ctx)
		// then
		if assert.NoError(t, err) {
			want := set.Of[int32](5, 42)
			assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
		}
	})
	t.Run("should return missing IDs and ignore IDs with zero value", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		factory.CreateEveEntity(app.EveEntity{ID: 42})
		// when
		got, err := r.MissingEveEntityIDs(ctx, set.Of[int32](42, 5, 0))
		// then
		if assert.NoError(t, err) {
			want := set.Of[int32](5)
			assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
		}
	})
}

func TestEveEntityCanCreateAllCategories(t *testing.T) {
	db, r, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	testutil.TruncateTables(db)
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
			e2, err := r.GetEveEntity(ctx, e1.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, e2.Category, c)
			}

		})
	}
}
