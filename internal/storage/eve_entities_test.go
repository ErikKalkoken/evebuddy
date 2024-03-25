package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func createEveEntity(id int32, name string, category string) EveEntity {
	e := EveEntity{ID: id, Name: name, Category: category}
	err := e.Save()
	if err != nil {
		panic(err)
	}
	return e
}

func TestEntitiesCanSaveNew(t *testing.T) {
	// given
	truncateTables()
	o := EveEntity{ID: 1, Name: "Erik", Category: "character"}
	// when
	err := o.Save()
	// then
	assert.NoError(t, err)
}

func TestEntitiesShouldReturnErrorWhenCategoryNotValid(t *testing.T) {
	// given
	truncateTables()
	o := EveEntity{ID: 1, Name: "Erik", Category: "django"}
	// when
	err := o.Save()
	// then
	assert.Error(t, err)
}

func TestEntitiesCanUpdateExisting(t *testing.T) {
	// given
	truncateTables()
	o := createEveEntity(42, "alpha", "character")
	assert.NoError(t, o.Save())
	o.Name = "bravo"
	o.Category = "corporation"
	// when
	err := o.Save()
	// then
	assert.NoError(t, err)
	o2, err := FetchEveEntity(42)
	assert.NoError(t, err)
	assert.Equal(t, o2.Name, "bravo")
	assert.Equal(t, o2.Category, "corporation")
}

func TestEntitiesCanFetchById(t *testing.T) {
	// given
	truncateTables()
	o := createEveEntity(42, "dummy", "character")
	// when
	r, err := FetchEveEntity(42)
	// then
	if assert.NoError(t, err) {
		assert.Equal(t, o, *r)
	}
}

func TestEntitiesShouldReturnErrorWhenNotFound(t *testing.T) {
	// given
	truncateTables()
	r, err := FetchEveEntity(42)
	// then
	assert.Error(t, err)
	assert.Nil(t, r)
}

func TestEntitiesCanReturnAllIDs(t *testing.T) {
	// given
	truncateTables()
	createEveEntity(42, "alpha", "character")
	createEveEntity(12, "bravo", "character")
	// when
	r, err := FetchEntityIDs()
	// then
	if assert.NoError(t, err) {
		assert.Equal(t, []int32{12, 42}, r)
	}
}
