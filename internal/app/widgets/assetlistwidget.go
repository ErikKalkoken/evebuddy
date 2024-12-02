package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/dustin/go-humanize"
)

const assetListIconSize = 64

type AssetListWidget struct {
	widget.BaseWidget
	icon         *canvas.Image
	name         *widget.Label
	quantity     *widget.Label
	fallbackIcon fyne.Resource
	sv           app.EveImageService
}

func NewAssetListWidget(sv app.EveImageService, fallbackIcon fyne.Resource) *AssetListWidget {
	icon := canvas.NewImageFromResource(fallbackIcon)
	icon.FillMode = canvas.ImageFillContain
	icon.SetMinSize(fyne.Size{Width: 40, Height: 40})
	item := &AssetListWidget{
		icon:         icon,
		name:         widget.NewLabel("Asset Template Name XXX\nAsset Template Name XXX"),
		quantity:     widget.NewLabel("99.999"),
		fallbackIcon: fallbackIcon,
		sv:           sv,
	}
	item.ExtendBaseWidget(item)
	return item
}

func (o *AssetListWidget) SetAsset(ca *app.CharacterAsset) {
	o.name.Text = ca.DisplayName()
	o.name.Wrapping = fyne.TextWrapWord
	o.name.Refresh()

	if !ca.IsSingleton {
		o.quantity.SetText(humanize.Comma(int64(ca.Quantity)))
		o.quantity.Show()
	} else {
		o.quantity.Hide()
	}

	o.icon.Resource = o.fallbackIcon
	o.icon.Refresh()

	refreshImageResourceAsync(o.icon, func() (fyne.Resource, error) {
		switch ca.Variant() {
		case app.VariantSKIN:
			return o.sv.InventoryTypeSKIN(ca.EveType.ID, assetListIconSize)
		case app.VariantBPO:
			return o.sv.InventoryTypeBPO(ca.EveType.ID, assetListIconSize)
		case app.VariantBPC:
			return o.sv.InventoryTypeBPC(ca.EveType.ID, assetListIconSize)
		default:
			return o.sv.InventoryTypeIcon(ca.EveType.ID, assetListIconSize)
		}
	})
}

func (o *AssetListWidget) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(nil, nil, o.icon, o.quantity, o.name)
	return widget.NewSimpleRenderer(c)
}
