package asset

import (
	"fmt"

	"fyne.io/fyne/v2"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/xsync"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type assetIconEIS interface {
	InventoryTypeBPC(id int64, size int) (fyne.Resource, error)
	InventoryTypeBPO(id int64, size int) (fyne.Resource, error)
	InventoryTypeIcon(id int64, size int) (fyne.Resource, error)
	InventoryTypeSKIN(id int64, size int) (fyne.Resource, error)
}

// assetIconCache caches the images for asset icons.
var assetIconCache xsync.Map[string, fyne.Resource]

func loadAssetIconAsync(eis assetIconEIS, typeID int64, variant app.InventoryTypeVariant, setIcon func(r fyne.Resource)) {
	key := fmt.Sprintf("%d-%d", typeID, variant)
	xwidget.LoadResourceAsyncWithCache(
		icons.BlankSvg,
		func() (fyne.Resource, bool) {
			return assetIconCache.Load(key)
		},
		setIcon,
		func() (fyne.Resource, error) {
			switch variant {
			case app.VariantBPO:
				return eis.InventoryTypeBPO(typeID, app.IconPixelSize)
			case app.VariantBPC:
				return eis.InventoryTypeBPC(typeID, app.IconPixelSize)
			case app.VariantSKIN:
				return eis.InventoryTypeSKIN(typeID, app.IconPixelSize)
			default:
				return eis.InventoryTypeIcon(typeID, app.IconPixelSize)
			}
		},
		func(r fyne.Resource) {
			assetIconCache.Store(key, r)
		},
	)
}
