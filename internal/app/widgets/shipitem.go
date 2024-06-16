package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// The ShipItem widget is used to render items on the type info window.
type ShipItem struct {
	widget.BaseWidget
	image        *canvas.Image
	label        *widget.Label
	fallbackIcon fyne.Resource
	sv           InventoryTypeImageProvider
}

func NewShipItem(sv InventoryTypeImageProvider, fallbackIcon fyne.Resource) *ShipItem {
	image := canvas.NewImageFromResource(theme.BrokenImageIcon())
	image.FillMode = canvas.ImageFillContain
	image.SetMinSize(fyne.Size{Width: 128, Height: 128})
	w := &ShipItem{
		image:        image,
		label:        widget.NewLabel("First line\nSecond Line\nThird Line"),
		fallbackIcon: fallbackIcon,
		sv:           sv,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *ShipItem) Set(typeID int32, label string, canFly bool) {
	w.label.Importance = widget.MediumImportance
	w.label.Text = label
	w.label.Wrapping = fyne.TextWrapWord
	var i widget.Importance
	if canFly {
		i = widget.MediumImportance
	} else {
		i = widget.LowImportance
	}
	w.label.Importance = i
	w.label.Refresh()
	refreshImageResourceAsync(w.image, func() (fyne.Resource, error) {
		return w.sv.InventoryTypeRender(typeID, 256)
	})
}

func (w *ShipItem) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewVBox(w.image, w.label)
	return widget.NewSimpleRenderer(c)
}
