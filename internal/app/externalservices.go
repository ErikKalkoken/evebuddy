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

type DictionaryService interface {
	Delete(string) error
	Float32(string) (float32, bool, error)
	Int(string) (int, bool, error)
	IntWithFallback(string, int) (int, error)
	SetFloat32(string, float32) error
	Float64(key string) (float64, bool, error)
	SetFloat64(key string, value float64) error
	SetInt(string, int) error
	SetString(string, string) error
	String(string) (string, bool, error)
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
