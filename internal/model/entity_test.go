package model_test

import (
	"database/sql"
	"example/evebuddy/internal/factory"
	"example/evebuddy/internal/helper/set"
	"example/evebuddy/internal/model"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEveEntities(t *testing.T) {
	t.Run("can save new", func(t *testing.T) {
		// given
		model.TruncateTables()
		o := model.EveEntity{ID: 1, Name: "Erik", Category: model.EveEntityCharacter}
		// when
		err := o.Save()
		// then
		assert.NoError(t, err)
	})
	t.Run("should return error when trying to save with invalid category", func(t *testing.T) {
		// given
		model.TruncateTables()
		o := model.EveEntity{ID: 1, Name: "Erik", Category: "django"}
		// when
		err := o.Save()
		// then
		assert.Error(t, err)
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		model.TruncateTables()
		o := factory.CreateEveEntity(model.EveEntity{ID: 42, Name: "alpha", Category: "character"})
		o.Name = "bravo"
		o.Category = "corporation"
		// when
		err := o.Save()
		// then
		assert.NoError(t, err)
		o2, err := model.GetEveEntity(42)
		assert.NoError(t, err)
		assert.Equal(t, o2.Name, "bravo")
		assert.Equal(t, o2.Category, model.EveEntityCorporation)
	})
	t.Run("can fetch existing", func(t *testing.T) {
		// given
		model.TruncateTables()
		o := factory.CreateEveEntity()
		// when
		r, err := model.GetEveEntity(o.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, o, r)
		}
	})
	t.Run("should return error when not found", func(t *testing.T) {
		// given
		model.TruncateTables()
		_, err := model.GetEveEntity(42)
		// then
		assert.Equal(t, sql.ErrNoRows, err)
	})
	t.Run("can return all existing IDs", func(t *testing.T) {
		// given
		model.TruncateTables()
		e1 := factory.CreateEveEntity()
		e2 := factory.CreateEveEntity()
		// when
		r, err := model.ListEveEntityIDs()
		// then
		if assert.NoError(t, err) {
			gotIDs := set.NewFromSlice([]int32{e1.ID, e2.ID})
			wantIDs := set.NewFromSlice(r)
			assert.Equal(t, wantIDs, gotIDs)
		}
	})
	t.Run("should return all character names in order", func(t *testing.T) {
		// given
		model.TruncateTables()
		factory.CreateEveEntity(model.EveEntity{Name: "Yalpha2", Category: "character"})
		factory.CreateEveEntity(model.EveEntity{Name: "Xalpha1", Category: "character"})
		factory.CreateEveEntity(model.EveEntity{Name: "charlie", Category: "character"})
		factory.CreateEveEntity(model.EveEntity{Name: "other", Category: "corporation"})
		// when
		ee, err := model.SearchEveEntitiesByName("ALPHA")
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

func TestEveEntitiesImageURL(t *testing.T) {
	model.TruncateTables()
	t.Run("can return for character", func(t *testing.T) {
		// given
		o := model.EveEntity{ID: 1, Name: "Erik", Category: "character"}
		// when
		err := o.Save()
		// then
		assert.NoError(t, err)
	})
}
