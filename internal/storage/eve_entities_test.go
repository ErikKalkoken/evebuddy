package storage_test

import (
	"example/esiapp/internal/storage"
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

type EveEntityArgs struct {
	Category string
	ID       int32
	Name     string
}

// createEveEntity creates a test objects.
func createEveEntity(p EveEntityArgs) storage.EveEntity {
	e := storage.EveEntity{ID: p.ID, Category: p.Category, Name: p.Name}
	if e.ID == 0 {
		ids, err := storage.FetchEntityIDs()
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
	e.MustSave()
	return e
}

func TestEntitiesCanSaveNew(t *testing.T) {
	// given
	storage.TruncateTables()
	o := storage.EveEntity{ID: 1, Name: "Erik", Category: "character"}
	// when
	err := o.Save()
	// then
	assert.NoError(t, err)
}

func TestEntitiesShouldReturnErrorWhenCategoryNotValid(t *testing.T) {
	// given
	storage.TruncateTables()
	o := storage.EveEntity{ID: 1, Name: "Erik", Category: "django"}
	// when
	err := o.Save()
	// then
	assert.Error(t, err)
}

func TestEntitiesCanUpdateExisting(t *testing.T) {
	// given
	storage.TruncateTables()
	o := storage.EveEntity{ID: 42, Name: "alpha", Category: "character"}
	o.MustSave()
	o.Name = "bravo"
	o.Category = "corporation"
	// when
	err := o.Save()
	// then
	assert.NoError(t, err)
	o2, err := storage.FetchEveEntity(42)
	assert.NoError(t, err)
	assert.Equal(t, o2.Name, "bravo")
	assert.Equal(t, o2.Category, "corporation")
}

func TestEntitiesCanFetchById(t *testing.T) {
	// given
	storage.TruncateTables()
	o := createEveEntity(EveEntityArgs{ID: 42})
	// when
	r, err := storage.FetchEveEntity(42)
	// then
	if assert.NoError(t, err) {
		assert.Equal(t, o, *r)
	}
}

func TestEntitiesShouldReturnErrorWhenNotFound(t *testing.T) {
	// given
	storage.TruncateTables()
	r, err := storage.FetchEveEntity(42)
	// then
	assert.Error(t, err)
	assert.Nil(t, r)
}

func TestEntitiesCanReturnAllIDs(t *testing.T) {
	// given
	storage.TruncateTables()
	createEveEntity(EveEntityArgs{ID: 42})
	createEveEntity(EveEntityArgs{ID: 12})
	// when
	r, err := storage.FetchEntityIDs()
	// then
	if assert.NoError(t, err) {
		assert.Equal(t, []int32{12, 42}, r)
	}
}
