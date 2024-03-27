package model

import (
	"example/esiapp/internal/api/images"
	"example/esiapp/internal/helper/set"
	"fmt"

	"fyne.io/fyne/v2"
)

// An entity in Eve Online
type EveEntity struct {
	Category string
	ID       int32
	Name     string
}

// Supported categories of EveEntity
const (
	EveEntityAlliance    = "alliance"
	EveEntityCharacter   = "character"
	EveEntityCorporation = "corporation"
	EveEntityMailList    = "mail_list"
)

var categories = set.NewFromSlice([]string{EveEntityAlliance, EveEntityCharacter, EveEntityCorporation, EveEntityMailList})

// Save updates or creates an eve entity.
func (e *EveEntity) Save() error {
	if !categories.Has(e.Category) {
		return fmt.Errorf("invalid category: %s", e.Category)
	}
	_, err := db.NamedExec(`
		INSERT INTO eve_entities (id, name, category)
		VALUES (:id, :name, :category)
		ON CONFLICT (id) DO
		UPDATE SET name=:name, category=:category;`,
		*e,
	)
	if err != nil {
		return err
	}
	return nil
}

// ImageURL returns an image URL for an entity
func (e *EveEntity) ImageURL(size int) fyne.URI {
	var u fyne.URI
	switch e.Category {
	case EveEntityCharacter:
		u, _ = images.CharacterPortraitURL(e.ID, size)
	case EveEntityCorporation:
		u, _ = images.CorporationLogoURL(e.ID, size)
	default:
		panic(fmt.Sprintf("ImageURL not defined for category %s", e.Category))
	}
	return u
}

// FetchEntityIDs returns all existing entity IDs.
func FetchEntityIDs() ([]int32, error) {
	var ids []int32
	err := db.Select(&ids, "SELECT id FROM eve_entities;")
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// FetchEveEntity return an EveEntity object if it exists or nil.
func FetchEveEntity(id int32) (*EveEntity, error) {
	var e EveEntity
	if err := db.Get(&e, "SELECT * FROM eve_entities WHERE id = ?;", id); err != nil {
		return nil, err
	}
	if e.ID == 0 {
		return nil, fmt.Errorf("EveEntity object not found for ID %d", id)
	}
	return &e, nil
}
