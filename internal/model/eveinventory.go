package model

import (
	"fyne.io/fyne/v2"

	"example/evebuddy/internal/api/images"
)

// EveCategory is a category in Eve Online.
type EveCategory struct {
	ID          int32
	IsPublished bool
	Name        string
}

// EveGroup is a group in Eve Online.
type EveGroup struct {
	ID          int32
	Category    *EveCategory
	IsPublished bool
	Name        string
}

// EveType is a type in Eve Online.
type EveType struct {
	ID          int32
	Description string
	Group       *EveGroup
	IsPublished bool
	Name        string
}

// IconURL returns the URL for an icon image of a type.
func (t *EveType) IconURL(size int) (fyne.URI, error) {
	return images.InventoryTypeRenderURL(t.ID, size)
}
