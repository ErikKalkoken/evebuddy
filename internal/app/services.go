package app

import (
	"time"

	"fyne.io/fyne/v2"
)

// Defines a cache service
type CacheService interface {
	Clear()
	Delete(string)
	Exists(string) bool
	Get(string) (any, bool)
	Set(string, any, time.Duration)
}

type EveImageService interface {
	AllianceLogo(int32, int) (fyne.Resource, error)
	CharacterPortrait(int32, int) (fyne.Resource, error)
	CorporationLogo(int32, int) (fyne.Resource, error)
	ClearCache() error
	EntityIcon(int32, string, int) (fyne.Resource, error)
	InventoryTypeBPO(int32, int) (fyne.Resource, error)
	InventoryTypeBPC(int32, int) (fyne.Resource, error)
	InventoryTypeIcon(int32, int) (fyne.Resource, error)
	InventoryTypeRender(int32, int) (fyne.Resource, error)
	InventoryTypeSKIN(int32, int) (fyne.Resource, error)
}
