package model

import (
	"example/esiapp/internal/api/images"
	"example/esiapp/internal/helper/set"
	"fmt"

	"fyne.io/fyne/v2"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// An entity in Eve Online
type EveEntity struct {
	Category EveEntityCategory
	ID       int32
	Name     string
}

type EveEntityCategory string

// Supported categories of EveEntity
const (
	EveEntityAlliance    EveEntityCategory = "alliance"
	EveEntityCharacter   EveEntityCategory = "character"
	EveEntityCorporation EveEntityCategory = "corporation"
	EveEntityMailList    EveEntityCategory = "mail_list"
)

var categories = set.NewFromSlice([]EveEntityCategory{EveEntityAlliance, EveEntityCharacter, EveEntityCorporation, EveEntityMailList})
var categoryLabels = map[EveEntityCategory]string{
	EveEntityMailList: "Mailing List",
}

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

func (e *EveEntity) CategoryLabel() string {
	l, ok := categoryLabels[e.Category]
	if !ok {
		l = string(e.Category)
	}
	c := cases.Title(language.English)
	return c.String(string(l))
}

// FetchEveEntityIDs returns all existing entity IDs.
func FetchEveEntityIDs() ([]int32, error) {
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

// FetchEveEntityNameSearch returns all entities partially matching a string in ascending order.
func FetchEveEntityNameSearch(partial string) ([]EveEntity, error) {
	var ee []EveEntity
	err := db.Select(
		&ee,
		`SELECT *
		FROM eve_entities
		WHERE name LIKE '%'||?||'%'
		ORDER BY name
		COLLATE NOCASE;`,
		partial,
	)
	if err != nil {
		return nil, err
	}
	return ee, nil
}
