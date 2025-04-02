package app

import "fyne.io/fyne/v2"

type EveImageService interface {
	AllianceLogo(int32, int) (fyne.Resource, error)
	CharacterPortrait(int32, int) (fyne.Resource, error)
	CorporationLogo(int32, int) (fyne.Resource, error)
	ClearCache() error
	EntityIcon(int32, string, int) (fyne.Resource, error)
	FactionLogo(id int32, size int) (fyne.Resource, error)
	InventoryTypeBPO(int32, int) (fyne.Resource, error)
	InventoryTypeBPC(int32, int) (fyne.Resource, error)
	InventoryTypeIcon(int32, int) (fyne.Resource, error)
	InventoryTypeRender(int32, int) (fyne.Resource, error)
	InventoryTypeSKIN(int32, int) (fyne.Resource, error)
}
