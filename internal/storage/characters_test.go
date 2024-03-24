package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func init() {
	initializeTest()
}

func createCharacter(id int32, name string) Character {
	c := Character{ID: id, Name: name}
	err := c.Save()
	if err != nil {
		panic(err)
	}
	return c
}

func TestCharacterCanSaveNew(t *testing.T) {
	// given
	truncateTables()
	c := createCharacter(1, "Erik")
	// when
	r, err := FetchFirstCharacter()
	// then
	assert.Nil(t, err)
	assert.Equal(t, c, *r)
}

func TestCharacterCanUpdate(t *testing.T) {
	// given
	truncateTables()
	c := createCharacter(1, "Erik")
	c.Name = "John"
	assert.Nil(t, c.Save())
	// when
	got, err := FetchFirstCharacter()
	// then
	assert.Nil(t, err)
	assert.Equal(t, c, *got)
}

func TestCharacterCanFetchByCharacterID(t *testing.T) {
	// given
	truncateTables()
	c1 := Character{ID: 1, Name: "Erik"}
	assert.Nil(t, c1.Save())
	c2 := Character{ID: 2, Name: "Naoko"}
	assert.Nil(t, c2.Save())
	// when
	r, err := FetchCharacter(2)
	// then
	assert.Nil(t, err)
	assert.Equal(t, c2, *r)
}

func TestCharacterCanFetchAll(t *testing.T) {
	// given
	truncateTables()
	c1 := Character{ID: 1, Name: "Naoko"}
	assert.Nil(t, c1.Save())
	c2 := Character{ID: 2, Name: "Erik"}
	assert.Nil(t, c2.Save())
	// when
	got, err := FetchAllCharacters()
	// then
	assert.Nil(t, err)
	assert.Equal(t, 2, len(got))
	assert.Equal(t, "Erik", got[0].Name)
	assert.Equal(t, "Naoko", got[1].Name)
}
