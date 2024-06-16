package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/dustin/go-humanize"
)

type EveTypeVariant uint

const (
	VariantRegular EveTypeVariant = iota
	VariantBPO
	VariantBPC
	VariantSKIN
)

const iconSize = 64

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

func (o *AssetListWidget) SetAsset(name string, quantity int32, isSingleton bool, typeID int32, variant EveTypeVariant) {
	o.name.Text = name
	o.name.Wrapping = fyne.TextWrapWord
	o.name.Refresh()

	if !isSingleton {
		o.quantity.SetText(humanize.Comma(int64(quantity)))
		o.quantity.Show()
	} else {
		o.quantity.Hide()
	}

	o.icon.Resource = o.fallbackIcon
	o.icon.Refresh()

	refreshImageResourceAsync(o.icon, func() (fyne.Resource, error) {
		switch variant {
		case VariantSKIN:
			return o.sv.InventoryTypeSKIN(typeID, iconSize)
		case VariantBPO:
			return o.sv.InventoryTypeBPO(typeID, iconSize)
		case VariantBPC:
			return o.sv.InventoryTypeBPC(typeID, iconSize)
		default:
			return o.sv.InventoryTypeIcon(typeID, iconSize)
		}
	})
}

func (o *AssetListWidget) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(nil, nil, o.icon, o.quantity, o.name)
	return widget.NewSimpleRenderer(c)
}
