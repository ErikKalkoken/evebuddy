// Package images provides image URLs for the Eve Online image server.
package images

import (
	"errors"
	"fmt"
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
	imageVariantBPO      imageVariant = "bp"
	imageVariantBPC      imageVariant = "bpc"
)

const (
	PlaceholderCharacterID   = 1
	PlaceholderCorporationID = 1
	baseURL                  = "https://images.evetech.net"
)

var ErrInvalidSize = errors.New("invalid size")

// AllianceLogoURL returns an image URL for an alliance logo
func AllianceLogoURL(id int32, size int) (string, error) {
	return imageURL(categoryAlliance, imageVariantLogo, id, size)
}

// CharacterPortraitURL returns an image URL for a character portrait
func CharacterPortraitURL(id int32, size int) (string, error) {
	return imageURL(categoryCharacter, imageVariantPortrait, id, size)
}

// CorporationLogoURL returns an image URL for a corporation logo
func CorporationLogoURL(id int32, size int) (string, error) {
	return imageURL(categoryCorporation, imageVariantLogo, id, size)
}

// FactionLogoURL returns an image URL for a faction logo
func FactionLogoURL(id int32, size int) (string, error) {
	return imageURL(categoryCorporation, imageVariantLogo, id, size)
}

// InventoryTypeRenderURL returns an image URL for inventory type render
func InventoryTypeRenderURL(id int32, size int) (string, error) {
	return imageURL(categoryInventoryType, imageVariantRender, id, size)
}

// InventoryTypeIconURL returns an image URL for inventory type icon
func InventoryTypeIconURL(id int32, size int) (string, error) {
	return imageURL(categoryInventoryType, imageVariantIcon, id, size)
}

// InventoryTypeBPOURL returns an image URL for inventory type bpo
func InventoryTypeBPOURL(id int32, size int) (string, error) {
	return imageURL(categoryInventoryType, imageVariantBPO, id, size)
}

// InventoryTypeBPCURL returns an image URL for inventory type bpc
func InventoryTypeBPCURL(id int32, size int) (string, error) {
	return imageURL(categoryInventoryType, imageVariantBPC, id, size)
}

func imageURL(c category, v imageVariant, id int32, size int) (string, error) {
	switch size {
	case 32, 64, 128, 256, 512, 1024:
		// valid size
	default:
		return "", fmt.Errorf("%d: %w", size, ErrInvalidSize)
	}
	url := fmt.Sprintf("%s/%s/%d/%s?size=%d", baseURL, c, id, v, size)
	return url, nil
}
