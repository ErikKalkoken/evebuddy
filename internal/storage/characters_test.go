package storage_test

import (
	"example/esiapp/internal/storage"
	"testing"

	"github.com/stretchr/testify/assert"
)

func init() {
	if err := storage.Initialize(":memory:"); err != nil {
		panic(err)
	}
}

func createCharacter(id int32, name string) storage.Character {
	c := storage.Character{ID: id, Name: name}
	err := c.Save()
	if err != nil {
		panic(err)
	}
	return c
}

func TestCharacterCanSaveNew(t *testing.T) {
	// given
	storage.TruncateTables()
	c := createCharacter(1, "Erik")
	// when
	r, err := storage.FetchFirstCharacter()
	// then
	if assert.NoError(t, err) {
		assert.Equal(t, c, *r)
	}
}

func TestCharacterCanUpdate(t *testing.T) {
	// given
	storage.TruncateTables()
	c := createCharacter(1, "Erik")
	c.Name = "John"
	assert.NoError(t, c.Save())
	// when
	got, err := storage.FetchFirstCharacter()
	// then
	if assert.NoError(t, err) {
		assert.Equal(t, c, *got)
	}
}

func TestCharacterCanFetchByCharacterID(t *testing.T) {
	// given
	storage.TruncateTables()
	c1 := storage.Character{ID: 1, Name: "Erik"}
	assert.NoError(t, c1.Save())
	c2 := storage.Character{ID: 2, Name: "Naoko"}
	assert.NoError(t, c2.Save())
	// when
	r, err := storage.FetchCharacter(2)
	// then
	if assert.NoError(t, err) {
		assert.Equal(t, c2, *r)
	}
}

func TestCharacterCanFetchAll(t *testing.T) {
	// given
	storage.TruncateTables()
	c1 := storage.Character{ID: 1, Name: "Naoko"}
	assert.NoError(t, c1.Save())
	c2 := storage.Character{ID: 2, Name: "Erik"}
	assert.NoError(t, c2.Save())
	// when
	got, err := storage.FetchAllCharacters()
	// then
	if assert.NoError(t, err) {
		assert.Equal(t, 2, len(got))
		assert.Equal(t, "Erik", got[0].Name)
		assert.Equal(t, "Naoko", got[1].Name)

	}
}
