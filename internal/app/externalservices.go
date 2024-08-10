package app

import (
	"time"

	"fyne.io/fyne/v2"
)

// Defines a cache service
type CacheService interface {
	Clear()
	Delete(any)
	Exists(any) bool
	Get(any) (any, bool)
	Set(any, any, time.Duration)
}

type EveImageService interface {
	CharacterPortrait(int32, int) (fyne.Resource, error)
	CorporationLogo(int32, int) (fyne.Resource, error)
	ClearCache() (int, error)
	InventoryTypeBPO(int32, int) (fyne.Resource, error)
	InventoryTypeBPC(int32, int) (fyne.Resource, error)
	InventoryTypeIcon(int32, int) (fyne.Resource, error)
	InventoryTypeRender(int32, int) (fyne.Resource, error)
	InventoryTypeSKIN(int32, int) (fyne.Resource, error)
	Size() (int, error)
}
