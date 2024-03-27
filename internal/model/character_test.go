package model_test

import (
	"example/esiapp/internal/helper/set"
	"example/esiapp/internal/model"
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

func createCharacter(args ...model.Character) model.Character {
	var c model.Character
	if len(args) > 0 {
		c = args[0]
	}
	if c.ID == 0 {
		ids, err := model.FetchCharacterIDs()
		if err != nil {
			panic(err)
		}
		if len(ids) == 0 {
			c.ID = 1
		} else {
			c.ID = slices.Max(ids) + 1
		}
	}
	if c.Name == "" {
		c.Name = fmt.Sprintf("Generated character #%d", c.ID)
	}
	if c.Corporation.ID == 0 {
		c.Corporation = createEveEntity(model.EveEntity{Category: model.EveEntityCorporation})
	}
	err := c.Save()
	if err != nil {
		panic(err)
	}
	return c
}

func TestCharacter(t *testing.T) {
	t.Run("can save new", func(t *testing.T) {
		// given
		model.TruncateTables()
		corp := createEveEntity(model.EveEntity{Category: model.EveEntityCorporation})
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
		c := createCharacter(model.Character{Name: "Erik"})
		c.Name = "John"
		assert.NoError(t, c.Save())
		// when
		got, err := model.FetchCharacter(c.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, c, *got)
		}
	})
	t.Run("can fetch character by ID", func(t *testing.T) {
		// given
		model.TruncateTables()
		createCharacter()
		c2 := createCharacter()
		assert.NoError(t, c2.Save())
		// when
		r, err := model.FetchCharacter(2)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, c2, *r)
		}
	})
	t.Run("can fetch all", func(t *testing.T) {
		// given
		model.TruncateTables()
		c1 := createCharacter()
		c2 := createCharacter()
		assert.NoError(t, c2.Save())
		// when
		got, err := model.FetchAllCharacters()
		// then
		if assert.NoError(t, err) {
			assert.Len(t, got, 2)
			gotIDs := set.NewFromSlice([]int32{got[0].ID, got[1].ID})
			wantIDs := set.NewFromSlice([]int32{c1.ID, c2.ID})
			assert.Equal(t, wantIDs, gotIDs)

		}
	})
	t.Run("can delete", func(t *testing.T) {
		// given
		model.TruncateTables()
		c := createCharacter()
		// when
		err := c.Delete()
		// then
		if assert.NoError(t, err) {
			_, err := model.FetchCharacter(c.ID)
			assert.Error(t, err)
		}
	})
}
