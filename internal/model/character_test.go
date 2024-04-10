package model_test

import (
	"example/evebuddy/internal/factory"
	"example/evebuddy/internal/helper/set"
	"example/evebuddy/internal/model"

	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCharacter(t *testing.T) {
	t.Run("can save new", func(t *testing.T) {
		// given
		model.TruncateTables()
		corp := factory.CreateEveEntity(model.EveEntity{Category: model.EveEntityCorporation})
		c := model.Character{ID: 1, Name: "Erik", Corporation: corp}
		// when
		err := c.Save()
		// then
		if assert.NoError(t, err) {
			r, err := model.FetchCharacter(c.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, c, *r)
			}
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		model.TruncateTables()
		c := factory.CreateCharacter(model.Character{Name: "Erik"})
		c.Name = "John"
		assert.NoError(t, c.Save())
		// when
		got, err := model.FetchCharacter(c.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, c, *got)
		}
	})
	t.Run("can fetch character by ID with corporation", func(t *testing.T) {
		// given
		model.TruncateTables()
		factory.CreateCharacter()
		c2 := factory.CreateCharacter()
		assert.NoError(t, c2.Save())
		// when
		r, err := model.FetchCharacter(2)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, c2, *r)
			assert.Equal(t, c2.Corporation.Name, r.Corporation.Name)
		}
	})
	t.Run("can fetch all", func(t *testing.T) {
		// given
		model.TruncateTables()
		c1 := factory.CreateCharacter(model.Character{Name: "Bravo"})
		c2 := factory.CreateCharacter(model.Character{Name: "Alpha"})
		assert.NoError(t, c2.Save())
		// when
		got, err := model.FetchAllCharacters()
		// then
		if assert.NoError(t, err) {
			assert.Len(t, got, 2)
			gotIDs := set.NewFromSlice([]int32{got[0].ID, got[1].ID})
			wantIDs := set.NewFromSlice([]int32{c2.ID, c1.ID})
			assert.Equal(t, wantIDs, gotIDs)
			assert.Equal(t, c2.Corporation.Name, got[0].Corporation.Name)
		}
	})
	t.Run("can delete", func(t *testing.T) {
		// given
		model.TruncateTables()
		c := factory.CreateCharacter()
		// when
		err := c.Delete()
		// then
		if assert.NoError(t, err) {
			_, err := model.FetchCharacter(c.ID)
			assert.Error(t, err)
		}
	})
}
