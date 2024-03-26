package model_test

import (
	"example/esiapp/internal/helper/set"
	"example/esiapp/internal/model"
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

// createEveEntity is a test factory for EveEntity objects.
func createEveEntity(args ...model.EveEntity) model.EveEntity {
	var e model.EveEntity
	if len(args) > 0 {
		e = args[0]
	}
	if e.ID == 0 {
		ids, err := model.FetchEntityIDs()
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
	model.TruncateTables()
	o := model.EveEntity{ID: 1, Name: "Erik", Category: "character"}
	// when
	err := o.Save()
	// then
	assert.NoError(t, err)
}

func TestEntitiesShouldReturnErrorWhenCategoryNotValid(t *testing.T) {
	// given
	model.TruncateTables()
	o := model.EveEntity{ID: 1, Name: "Erik", Category: "django"}
	// when
	err := o.Save()
	// then
	assert.Error(t, err)
}

func TestEveEntitiesCanUpdateExisting(t *testing.T) {
	// given
	model.TruncateTables()
	o := createEveEntity(model.EveEntity{ID: 42, Name: "alpha", Category: "character"})
	o.Name = "bravo"
	o.Category = "corporation"
	// when
	err := o.Save()
	// then
	assert.NoError(t, err)
	o2, err := model.FetchEveEntity(42)
	assert.NoError(t, err)
	assert.Equal(t, o2.Name, "bravo")
	assert.Equal(t, o2.Category, "corporation")
}

func TestCanFetchEveEntity(t *testing.T) {
	// given
	model.TruncateTables()
	o := createEveEntity()
	// when
	r, err := model.FetchEveEntity(o.ID)
	// then
	if assert.NoError(t, err) {
		assert.Equal(t, o, *r)
	}
}

func TestFetchEveEntityShouldReturnErrorWhenNotFound(t *testing.T) {
	// given
	model.TruncateTables()
	r, err := model.FetchEveEntity(42)
	// then
	assert.Error(t, err)
	assert.Nil(t, r)
}

func TestEntitiesCanReturnAllIDs(t *testing.T) {
	// given
	model.TruncateTables()
	e1 := createEveEntity()
	e2 := createEveEntity()
	// when
	r, err := model.FetchEntityIDs()
	// then
	if assert.NoError(t, err) {
		gotIDs := set.NewFromSlice([]int32{e1.ID, e2.ID})
		wantIDs := set.NewFromSlice(r)
		assert.Equal(t, wantIDs, gotIDs)
	}
}
