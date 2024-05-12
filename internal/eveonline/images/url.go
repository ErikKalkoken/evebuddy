// Package images provides image URLs for the Eve Online image server.
package images

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
)

type category string

const (
	categoryCharacter     category = "characters"
	categoryCorporation   category = "corporations"
	categoryAlliance      category = "alliances"
	categoryInventoryType category = "types"
)

const PlaceholderCharacterID = 1
const PlaceholderCorporationID = 1
const baseURL = "https://images.evetech.net"

// AllianceLogoURL returns an image URL for an alliance logo
func AllianceLogoURL(id int32, size int) (fyne.URI, error) {
	return imageURL(categoryAlliance, id, size)
}

// CharacterPortraitURL returns an image URL for a character portrait
func CharacterPortraitURL(id int32, size int) (fyne.URI, error) {
	return imageURL(categoryCharacter, id, size)
}

// CorporationLogoURL returns an image URL for a corporation logo
func CorporationLogoURL(id int32, size int) (fyne.URI, error) {
	return imageURL(categoryCorporation, id, size)
}

// FactionLogoURL returns an image URL for a faction logo
func FactionLogoURL(id int32, size int) (fyne.URI, error) {
	return imageURL(categoryCorporation, id, size)
}

// InventoryTypeRenderURL returns an image URL for inventory type render
func InventoryTypeRenderURL(id int32, size int) (fyne.URI, error) {
	return imageURL(categoryInventoryType, id, size)
}

func imageURL(c category, id int32, size int) (fyne.URI, error) {
	switch size {
	case 32, 64, 128, 256, 512, 1024:
		// valid size
	default:
		return nil, fmt.Errorf("invalid size %d", size)
	}
	category2Class := map[category]string{
		categoryAlliance:      "logo",
		categoryCharacter:     "portrait",
		categoryCorporation:   "logo",
		categoryInventoryType: "render",
	}
	class, ok := category2Class[c]
	if !ok {
		return nil, fmt.Errorf("class not defined for this category: %v", c)
	}
	s := fmt.Sprintf("%s/%s/%d/%s?size=%d", baseURL, c, id, class, size)
	u, err := storage.ParseURI(s)
	if err != nil {
		return nil, err
	}
	return u, nil
}
