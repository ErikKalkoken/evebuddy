// Package ui provides globals for UI packages.
package ui

import (
	"fyne.io/fyne/v2"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

type EVEImageService interface {
	AllianceLogo(id int64, size int) (fyne.Resource, error)
	AllianceLogoAsync(id int64, size int, setter func(r fyne.Resource))
	CharacterPortrait(id int64, size int) (fyne.Resource, error)
	CharacterPortraitAsync(id int64, size int, setter func(r fyne.Resource))
	CorporationLogo(id int64, size int) (fyne.Resource, error)
	CorporationLogoAsync(id int64, size int, setter func(r fyne.Resource))
	FactionLogo(id int64, size int) (fyne.Resource, error)
	FactionLogoAsync(id int64, size int, setter func(r fyne.Resource))
	InventoryTypeRender(id int64, size int) (fyne.Resource, error)
	InventoryTypeRenderAsync(id int64, size int, setter func(r fyne.Resource))
	InventoryTypeIcon(id int64, size int) (fyne.Resource, error)
	InventoryTypeIconAsync(id int64, size int, setter func(r fyne.Resource))
	InventoryTypeBPO(id int64, size int) (fyne.Resource, error)
	InventoryTypeBPOAsync(id int64, size int, setter func(r fyne.Resource))
	InventoryTypeBPC(id int64, size int) (fyne.Resource, error)
	InventoryTypeBPCAsync(id int64, size int, setter func(r fyne.Resource))
	InventoryTypeSKIN(id int64, size int) (fyne.Resource, error)
	InventoryTypeSKINAsync(id int64, size int, setter func(r fyne.Resource))
}

type InfoWindow interface {
	Show(o *app.EveEntity)
	ShowLocation(id int64)
	ShowRace(id int64)
	ShowType(typeID, characterID int64)
}
