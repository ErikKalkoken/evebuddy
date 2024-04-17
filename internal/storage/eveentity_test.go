package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"example/evebuddy/internal/helper/set"
	"example/evebuddy/internal/model"
	"example/evebuddy/internal/storage"
	"example/evebuddy/internal/testutil"
)

func TestEveEntity(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		_, err := r.CreateEveEntity(ctx, 42, "Dummy", model.EveEntityAlliance)
		// then
		if assert.NoError(t, err) {
			e, err := r.GetEveEntity(ctx, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, e.ID, int32(42))
				assert.Equal(t, e.Name, "Dummy")
				assert.Equal(t, e.Category, model.EveEntityAlliance)
			}
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// given
		e1 := factory.CreateEveEntity(
			model.EveEntity{
				ID:       42,
				Name:     "Alpha",
				Category: model.EveEntityCharacter,
			})
		// when
		_, err := r.UpdateOrCreateEveEntity(ctx, e1.ID, "Erik", model.EveEntityCorporation)
		// then
		if assert.NoError(t, err) {
			e2, err := r.GetEveEntity(ctx, e1.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, e1.ID, e2.ID)
				assert.Equal(t, "Erik", e2.Name)
				assert.Equal(t, model.EveEntityCorporation, e2.Category)
			}
		}
	})
	t.Run("can fetch existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// given
		e1 := factory.CreateEveEntity(
			model.EveEntity{
				ID:       42,
				Name:     "Alpha",
				Category: model.EveEntityCharacter,
			})
		// when
		e2, err := r.GetEveEntity(ctx, e1.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, e1.ID, e2.ID)
			assert.Equal(t, "Alpha", e2.Name)
			assert.Equal(t, model.EveEntityCharacter, e2.Category)
		}
	})
	t.Run("should return error when no object found 1", func(t *testing.T) {
		_, err := r.GetEveEntity(ctx, 99)
		assert.ErrorIs(t, err, storage.ErrNotFound)
	})
	t.Run("should return error when no object found 2", func(t *testing.T) {
		_, err := r.GetEveEntityByNameAndCategory(ctx, "dummy", model.EveEntityAlliance)
		assert.ErrorIs(t, err, storage.ErrNotFound)
	})

	t.Run("should return objs with matching names in order", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		factory.CreateEveEntityCharacter(model.EveEntity{Name: "Yalpha2"})
		factory.CreateEveEntityAlliance(model.EveEntity{Name: "Xalpha1"})
		factory.CreateEveEntityCharacter(model.EveEntity{Name: "charlie"})
		factory.CreateEveEntityCharacter(model.EveEntity{Name: "other"})
		// when
		ee, err := r.ListEveEntitiesByPartialName(ctx, "%ALPHA%")
		// then
		if assert.NoError(t, err) {
			var got []string
			for _, e := range ee {
				got = append(got, e.Name)
			}
			want := []string{"Xalpha1", "Yalpha2"}
			assert.Equal(t, want, got)
		}
	})
	t.Run("should not store with invalid ID 1", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		_, err := r.CreateEveEntity(ctx, 0, "Dummy", model.EveEntityAlliance)
		// then
		assert.Error(t, err)
	})
	t.Run("should not store with invalid ID 2", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		_, err := r.GetOrCreateEveEntity(ctx, 0, "Dummy", model.EveEntityAlliance)
		// then
		assert.Error(t, err)
	})
	t.Run("should not store with invalid ID 3", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		_, err := r.UpdateOrCreateEveEntity(ctx, 0, "Dummy", model.EveEntityAlliance)
		// then
		assert.Error(t, err)
	})
}
func TestEveEntityGetOrCreate(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("should create new when not exist", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		_, err := r.GetOrCreateEveEntity(ctx, 42, "Dummy", model.EveEntityAlliance)
		// then
		if assert.NoError(t, err) {
			e, err := r.GetEveEntity(ctx, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, e.ID, int32(42))
				assert.Equal(t, e.Name, "Dummy")
				assert.Equal(t, e.Category, model.EveEntityAlliance)
			}
		}
	})
	t.Run("should get when exists", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// given
		factory.CreateEveEntity(
			model.EveEntity{
				ID:       42,
				Name:     "Alpha",
				Category: model.EveEntityCharacter,
			})
		// when
		e, err := r.GetOrCreateEveEntity(ctx, 42, "Erik", model.EveEntityCorporation)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(42), e.ID)
			assert.Equal(t, "Alpha", e.Name)
			assert.Equal(t, model.EveEntityCharacter, e.Category)
		}
	})
}

func TestEveEntityIDs(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("should list existing entity IDs", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		factory.CreateEveEntity(model.EveEntity{ID: 5})
		factory.CreateEveEntity(model.EveEntity{ID: 42})
		// when
		got, err := r.ListEveEntityIDs(ctx)
		// then
		if assert.NoError(t, err) {
			want := []int32{5, 42}
			assert.Equal(t, want, got)
		}
	})
	t.Run("should return missing IDs", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		factory.CreateEveEntity(model.EveEntity{ID: 42})
		// when
		got, err := r.MissingEveEntityIDs(ctx, []int32{42, 5})
		// then
		if assert.NoError(t, err) {
			want := set.NewFromSlice([]int32{5})
			assert.Equal(t, want, got)
		}
	})
}
