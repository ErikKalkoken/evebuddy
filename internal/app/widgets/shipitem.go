package widgets

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/anthonynsimon/bild/effect"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

// The ShipItem widget is used to render items on the type info window.
type ShipItem struct {
	widget.BaseWidget
	image        *canvas.Image
	label        *widget.Label
	fallbackIcon fyne.Resource
	sv           app.EveImageService
}

func NewShipItem(sv app.EveImageService, fallbackIcon fyne.Resource) *ShipItem {
	upLeft := image.Point{0, 0}
	lowRight := image.Point{128, 128}
	image := canvas.NewImageFromImage(image.NewRGBA(image.Rectangle{upLeft, lowRight}))
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
	go func(img *canvas.Image) {
		r, err := w.sv.InventoryTypeRender(typeID, 256)
		if err != nil {
			return
		}
		i, _, err := image.Decode(bytes.NewReader(r.Content()))
		if err != nil {
			panic(err)
		}
		j := i.(*image.YCbCr)
		b := j.Bounds()
		m := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
		draw.Draw(m, m.Bounds(), j, b.Min, draw.Src)
		if !canFly {
			m = effect.Grayscale(m)
		}
		w.image.Image = m
		w.image.Refresh()
	}(w.image)
}

func adjustBrightness(c color.Color, f float32) color.Color {
	r, g, b, a := c.RGBA()
	r = uint32(float32(r) * f)
	g = uint32(float32(g) * f)
	b = uint32(float32(b) * f)
	return color.RGBA{
		uint8((r >> 9)),
		uint8((g >> 9)),
		uint8((b >> 9)),
		uint8((a >> 9)),
	}
}

func (w *ShipItem) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewVBox(w.image, w.label)
	return widget.NewSimpleRenderer(c)
}
