package model_test

import (
	"example/esiapp/internal/helper/set"
	"example/esiapp/internal/model"
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Initialize the test database for this test package
func init() {
	_, err := model.Initialize(":memory:", false)
	if err != nil {
		panic(err)
	}
}

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
	err := c.Save()
	if err != nil {
		panic(err)
	}
	return c
}

func TestCharacterCanSaveNew(t *testing.T) {
	// given
	model.TruncateTables()
	c := createCharacter()
	// when
	r, err := model.FetchFirstCharacter()
	// then
	if assert.NoError(t, err) {
		assert.Equal(t, c, *r)
	}
}

func TestCharacterCanUpdate(t *testing.T) {
	// given
	model.TruncateTables()
	c := createCharacter(model.Character{Name: "Erik"})
	c.Name = "John"
	assert.NoError(t, c.Save())
	// when
	got, err := model.FetchFirstCharacter()
	// then
	if assert.NoError(t, err) {
		assert.Equal(t, c, *got)
	}
}

func TestCharacterCanFetchByCharacterID(t *testing.T) {
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
}

func TestCharacterCanFetchAll(t *testing.T) {
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
}

func TestCharacterCanDelete(t *testing.T) {
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
}
