// Package ui provides globals for UI packages.
package ui

import (
	"log/slog"
	"net/url"

	"fyne.io/fyne/v2"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

// Width of common columns in data tables
const (
	ColumnWidthEntity   = 200
	ColumnWidthDateTime = 150
	ColumnWidthLocation = 350
	ColumnWidthRegion   = 150
)

// Global UI constants
const (
	IconPixelSize      = 64
	IconUnitSize       = 28
	FloatFormat        = "#,###.##"
	fallbackWebsiteURL = "https://github.com/ErikKalkoken/evebuddy"
)

const (
	SkipUIReason = "UI tests are flaky"
)

// EVEImageService defines which methods from the EVE image service is used in the UI.
type EVEImageService interface {
	AllianceLogo(id int64, size int) (fyne.Resource, error)
	AllianceLogoAsync(id int64, size int, setter func(r fyne.Resource))
	AssetIconAsync(id int64, variant app.InventoryTypeVariant, size int, setter func(r fyne.Resource))
	CharacterPortrait(id int64, size int) (fyne.Resource, error)
	CharacterPortraitAsync(id int64, size int, setter func(r fyne.Resource))
	CorporationLogo(id int64, size int) (fyne.Resource, error)
	CorporationLogoAsync(id int64, size int, setter func(r fyne.Resource))
	EveEntityLogo(o *app.EveEntity, size int) (fyne.Resource, error)
	EveEntityLogoAsync(o *app.EveEntity, size int, setter func(r fyne.Resource))
	FactionLogo(id int64, size int) (fyne.Resource, error)
	FactionLogoAsync(id int64, size int, setter func(r fyne.Resource))
	InventoryTypeBPC(id int64, size int) (fyne.Resource, error)
	InventoryTypeBPCAsync(id int64, size int, setter func(r fyne.Resource))
	InventoryTypeBPO(id int64, size int) (fyne.Resource, error)
	InventoryTypeBPOAsync(id int64, size int, setter func(r fyne.Resource))
	InventoryTypeIcon(id int64, size int) (fyne.Resource, error)
	InventoryTypeIconAsync(id int64, size int, setter func(r fyne.Resource))
	InventoryTypeRender(id int64, size int) (fyne.Resource, error)
	InventoryTypeRenderAsync(id int64, size int, setter func(r fyne.Resource))
	InventoryTypeSKIN(id int64, size int) (fyne.Resource, error)
	InventoryTypeSKINAsync(id int64, size int, setter func(r fyne.Resource))
}

// InfoViewer defines which methods from the info viewer is used in the UI.
type InfoViewer interface {
	Show(o *app.EveEntity)
	ShowLocation(id int64)
	ShowRace(id int64)
	ShowType(typeID, characterID int64)
}

// Name returns the name for this app.
func Name() string {
	info := fyne.CurrentApp().Metadata()
	name := info.Name
	if name == "" {
		return "EVE Buddy"
	}
	return name
}

// WebsiteRootURL returns the URL of the app's website.
func WebsiteRootURL() *url.URL {
	s := fyne.CurrentApp().Metadata().Custom["Website"]
	if s == "" {
		s = fallbackWebsiteURL
	}
	uri, err := url.Parse(s)
	if err != nil {
		slog.Error("parse main website URL")
		uri, _ = url.Parse(fallbackWebsiteURL)
	}
	return uri
}
