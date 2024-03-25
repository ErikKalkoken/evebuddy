package models_test

import (
	"example/esiapp/internal/models"
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

// createEveEntity is a test factory for EveEntity objects.
func createEveEntity(e models.EveEntity) models.EveEntity {
	if e.ID == 0 {
		ids, err := models.FetchEntityIDs()
		if err != nil {
			panic(err)
		}
		if len(ids) > 0 {
			e.ID = slices.Max(ids) + 1
		} else {
			e.ID = 1
		}
	}
	if e.Name == "" {
		e.Name = fmt.Sprintf("generated #%d", e.ID)
	}
	if e.Category == "" {
		e.Category = "character"
	}
	if err := e.Save(); err != nil {
		panic(err)
	}
	return e
}

func TestEntitiesCanSaveNew(t *testing.T) {
	// given
	models.TruncateTables()
	o := models.EveEntity{ID: 1, Name: "Erik", Category: "character"}
	// when
	err := o.Save()
	// then
	assert.NoError(t, err)
}

func TestEntitiesShouldReturnErrorWhenCategoryNotValid(t *testing.T) {
	// given
	models.TruncateTables()
	o := models.EveEntity{ID: 1, Name: "Erik", Category: "django"}
	// when
	err := o.Save()
	// then
	assert.Error(t, err)
}

func TestEntitiesCanUpdateExisting(t *testing.T) {
	// given
	models.TruncateTables()
	o := createEveEntity(models.EveEntity{ID: 42, Name: "alpha", Category: "character"})
	o.Name = "bravo"
	o.Category = "corporation"
	// when
	err := o.Save()
	// then
	assert.NoError(t, err)
	o2, err := models.FetchEveEntity(42)
	assert.NoError(t, err)
	assert.Equal(t, o2.Name, "bravo")
	assert.Equal(t, o2.Category, "corporation")
}

func TestEntitiesCanFetchById(t *testing.T) {
	// given
	models.TruncateTables()
	o := createEveEntity(models.EveEntity{ID: 42})
	// when
	r, err := models.FetchEveEntity(42)
	// then
	if assert.NoError(t, err) {
		assert.Equal(t, o, *r)
	}
}

func TestEntitiesShouldReturnErrorWhenNotFound(t *testing.T) {
	// given
	models.TruncateTables()
	r, err := models.FetchEveEntity(42)
	// then
	assert.Error(t, err)
	assert.Nil(t, r)
}

func TestEntitiesCanReturnAllIDs(t *testing.T) {
	// given
	models.TruncateTables()
	createEveEntity(models.EveEntity{ID: 42})
	createEveEntity(models.EveEntity{ID: 12})
	// when
	r, err := models.FetchEntityIDs()
	// then
	if assert.NoError(t, err) {
		assert.Equal(t, []int32{12, 42}, r)
	}
}
