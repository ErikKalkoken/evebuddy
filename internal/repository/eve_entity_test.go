package repository_test

import (
	"context"
	"example/evebuddy/internal/helper/set"
	"example/evebuddy/internal/repository"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEveEntity(t *testing.T) {
	db, r, factory := setUpDB()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		repository.TruncateTables(db)
		// when
		e, err := r.CreateEveEntity(ctx, 42, "Dummy", repository.EveEntityAlliance)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, e.ID, int32(42))
			assert.Equal(t, e.Name, "Dummy")
			assert.Equal(t, e.Category, repository.EveEntityAlliance)
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		repository.TruncateTables(db)
		// given
		e1 := factory.CreateEveEntity(
			repository.EveEntity{
				ID:       42,
				Name:     "Alpha",
				Category: repository.EveEntityCharacter,
			})
		// when
		_, err := r.UpdateOrCreateEveEntity(ctx, e1.ID, "Erik", repository.EveEntityCorporation)
		// then
		if assert.NoError(t, err) {
			e2, err := r.GetEveEntity(ctx, e1.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, e1.ID, e2.ID)
				assert.Equal(t, "Erik", e2.Name)
				assert.Equal(t, repository.EveEntityCorporation, e2.Category)
			}
		}
	})
	t.Run("should return error when no object found 1", func(t *testing.T) {
		_, err := r.GetEveEntity(ctx, 99)
		assert.ErrorIs(t, err, repository.ErrNotFound)
	})
	t.Run("should return error when no object found 2", func(t *testing.T) {
		_, err := r.GetEveEntityByNameAndCategory(ctx, "dummy", repository.EveEntityAlliance)
		assert.ErrorIs(t, err, repository.ErrNotFound)
	})
	t.Run("should return objs with matching names in order", func(t *testing.T) {
		// given
		repository.TruncateTables(db)
		factory.CreateEveEntityCharacter(repository.EveEntity{Name: "Yalpha2"})
		factory.CreateEveEntityAlliance(repository.EveEntity{Name: "Xalpha1"})
		factory.CreateEveEntityCharacter(repository.EveEntity{Name: "charlie"})
		factory.CreateEveEntityCharacter(repository.EveEntity{Name: "other"})
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
		repository.TruncateTables(db)
		factory.CreateEveEntity(repository.EveEntity{ID: 5})
		factory.CreateEveEntity(repository.EveEntity{ID: 42})
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
		repository.TruncateTables(db)
		factory.CreateEveEntity(repository.EveEntity{ID: 42})
		// when
		got, err := r.MissingEveEntityIDs(ctx, []int32{42, 5})
		// then
		if assert.NoError(t, err) {
			want := set.NewFromSlice([]int32{5})
			assert.Equal(t, want, got)
		}
	})
}
