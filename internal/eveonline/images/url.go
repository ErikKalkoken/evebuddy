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

type imageVariant string

const (
	imageVariantLogo     imageVariant = "logo"
	imageVariantPortrait imageVariant = "portrait"
	imageVariantRender   imageVariant = "render"
	imageVariantIcon     imageVariant = "icon"
)

const PlaceholderCharacterID = 1
const PlaceholderCorporationID = 1
const baseURL = "https://images.evetech.net"

// AllianceLogoURL returns an image URL for an alliance logo
func AllianceLogoURL(id int32, size int) (fyne.URI, error) {
	return imageURL(categoryAlliance, imageVariantLogo, id, size)
}

// CharacterPortraitURL returns an image URL for a character portrait
func CharacterPortraitURL(id int32, size int) (fyne.URI, error) {
	return imageURL(categoryCharacter, imageVariantPortrait, id, size)
}

// CorporationLogoURL returns an image URL for a corporation logo
func CorporationLogoURL(id int32, size int) (fyne.URI, error) {
	return imageURL(categoryCorporation, imageVariantLogo, id, size)
}

// FactionLogoURL returns an image URL for a faction logo
func FactionLogoURL(id int32, size int) (fyne.URI, error) {
	return imageURL(categoryCorporation, imageVariantLogo, id, size)
}

// InventoryTypeRenderURL returns an image URL for inventory type render
func InventoryTypeRenderURL(id int32, size int) (fyne.URI, error) {
	return imageURL(categoryInventoryType, imageVariantRender, id, size)
}

// InventoryTypeIconURL returns an image URL for inventory type icon
func InventoryTypeIconURL(id int32, size int) (fyne.URI, error) {
	return imageURL(categoryInventoryType, imageVariantIcon, id, size)
}

func imageURL(c category, v imageVariant, id int32, size int) (fyne.URI, error) {
	switch size {
	case 32, 64, 128, 256, 512, 1024:
		// valid size
	default:
		return nil, fmt.Errorf("invalid size %d", size)
	}
	s := fmt.Sprintf("%s/%s/%d/%s?size=%d", baseURL, c, id, v, size)
	u, err := storage.ParseURI(s)
	if err != nil {
		return nil, err
	}
	return u, nil
}
