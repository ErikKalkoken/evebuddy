package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/dustin/go-humanize"
)

type InventoryTypeImageProvider interface {
	InventoryTypeBPO(int32, int) (fyne.Resource, error)
	InventoryTypeBPC(int32, int) (fyne.Resource, error)
	InventoryTypeIcon(int32, int) (fyne.Resource, error)
}

type assetListWidget struct {
	widget.BaseWidget
	icon         *canvas.Image
	name         *widget.Label
	quantity     *widget.Label
	fallbackIcon fyne.Resource
	sv           InventoryTypeImageProvider
}

func NewAssetListWidget(sv InventoryTypeImageProvider, fallbackIcon fyne.Resource) *assetListWidget {
	icon := canvas.NewImageFromResource(fallbackIcon)
	icon.FillMode = canvas.ImageFillContain
	icon.SetMinSize(fyne.Size{Width: 40, Height: 40})
	item := &assetListWidget{
		icon:         icon,
		name:         widget.NewLabel("Asset Template Name XXX\nAsset Template Name XXX"),
		quantity:     widget.NewLabel("99.999"),
		fallbackIcon: fallbackIcon,
		sv:           sv,
	}
	item.ExtendBaseWidget(item)
	return item
}

func (o *assetListWidget) SetAsset(ca *model.CharacterAsset) {
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
		if ca.IsSKIN() {
			return resourceSkinicon64pxPng, nil
		} else if ca.IsBPO() {
			return o.sv.InventoryTypeBPO(ca.EveType.ID, 64)
		} else if ca.IsBlueprintCopy {
			return o.sv.InventoryTypeBPC(ca.EveType.ID, 64)
		} else {
			return o.sv.InventoryTypeIcon(ca.EveType.ID, 64)
		}
	})
}

func (o *assetListWidget) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(nil, nil, o.icon, o.quantity, o.name)
	return widget.NewSimpleRenderer(c)
}
