package storage_test

import (
	"context"
	"example/evebuddy/internal/helper/set"
	"example/evebuddy/internal/storage"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEveEntity(t *testing.T) {
	db, r, factory := setUpDB()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		storage.TruncateTables(db)
		// when
		e, err := r.CreateEveEntity(ctx, 42, "Dummy", storage.EveEntityAlliance)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, e.ID, int32(42))
			assert.Equal(t, e.Name, "Dummy")
			assert.Equal(t, e.Category, storage.EveEntityAlliance)
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		storage.TruncateTables(db)
		// given
		e1 := factory.CreateEveEntity(
			storage.EveEntity{
				ID:       42,
				Name:     "Alpha",
				Category: storage.EveEntityCharacter,
			})
		// when
		_, err := r.UpdateOrCreateEveEntity(ctx, e1.ID, "Erik", storage.EveEntityCorporation)
		// then
		if assert.NoError(t, err) {
			e2, err := r.GetEveEntity(ctx, e1.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, e1.ID, e2.ID)
				assert.Equal(t, "Erik", e2.Name)
				assert.Equal(t, storage.EveEntityCorporation, e2.Category)
			}
		}
	})
	t.Run("can fetch existing", func(t *testing.T) {
		// given
		storage.TruncateTables(db)
		// given
		e1 := factory.CreateEveEntity(
			storage.EveEntity{
				ID:       42,
				Name:     "Alpha",
				Category: storage.EveEntityCharacter,
			})
		// when
		e2, err := r.GetEveEntity(ctx, e1.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, e1.ID, e2.ID)
			assert.Equal(t, "Alpha", e2.Name)
			assert.Equal(t, storage.EveEntityCharacter, e2.Category)
		}
	})
	t.Run("should return error when no object found 1", func(t *testing.T) {
		_, err := r.GetEveEntity(ctx, 99)
		assert.ErrorIs(t, err, storage.ErrNotFound)
	})
	t.Run("should return error when no object found 2", func(t *testing.T) {
		_, err := r.GetEveEntityByNameAndCategory(ctx, "dummy", storage.EveEntityAlliance)
		assert.ErrorIs(t, err, storage.ErrNotFound)
	})
	t.Run("should return objs with matching names in order", func(t *testing.T) {
		// given
		storage.TruncateTables(db)
		factory.CreateEveEntityCharacter(storage.EveEntity{Name: "Yalpha2"})
		factory.CreateEveEntityAlliance(storage.EveEntity{Name: "Xalpha1"})
		factory.CreateEveEntityCharacter(storage.EveEntity{Name: "charlie"})
		factory.CreateEveEntityCharacter(storage.EveEntity{Name: "other"})
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
}

func TestEveEntityIDs(t *testing.T) {
	db, r, factory := setUpDB()
	defer db.Close()
	ctx := context.Background()
	t.Run("should list existing entity IDs", func(t *testing.T) {
		// given
		storage.TruncateTables(db)
		factory.CreateEveEntity(storage.EveEntity{ID: 5})
		factory.CreateEveEntity(storage.EveEntity{ID: 42})
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
		storage.TruncateTables(db)
		factory.CreateEveEntity(storage.EveEntity{ID: 42})
		// when
		got, err := r.MissingEveEntityIDs(ctx, []int32{42, 5})
		// then
		if assert.NoError(t, err) {
			want := set.NewFromSlice([]int32{5})
			assert.Equal(t, want, got)
		}
	})
}
