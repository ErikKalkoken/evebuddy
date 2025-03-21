package character

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"log/slog"
	"time"

	_ "image/jpeg"
	_ "image/png"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/anthonynsimon/bild/effect"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/ui/shared"
)

// The ShipItem widget is used to render items on the type info window.
type ShipItem struct {
	widget.BaseWidget

	cache        app.CacheService
	fallbackIcon fyne.Resource
	image        *canvas.Image
	label        *widget.Label
	sv           app.EveImageService
}

func NewShipItem(sv app.EveImageService, cache app.CacheService, fallbackIcon fyne.Resource) *ShipItem {
	upLeft := image.Point{0, 0}
	lowRight := image.Point{128, 128}
	image := canvas.NewImageFromImage(image.NewRGBA(image.Rectangle{upLeft, lowRight}))
	image.FillMode = canvas.ImageFillContain
	image.ScaleMode = appwidget.DefaultImageScaleMode
	image.SetMinSize(fyne.NewSquareSize(128))
	w := &ShipItem{
		image:        image,
		label:        widget.NewLabel("First line\nSecond Line\nThird Line"),
		fallbackIcon: fallbackIcon,
		sv:           sv,
		cache:        cache,
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
	go func() {
		// TODO: Move grayscale feature into general package
		key := fmt.Sprintf("ship-image-%d", typeID)
		var img *image.RGBA
		y, ok := w.cache.Get(key)
		if !ok {
			r, err := w.sv.InventoryTypeRender(typeID, 256)
			if err != nil {
				slog.Error("failed to fetch image for ship render", "error", err)
				return
			}
			j, _, err := image.Decode(bytes.NewReader(r.Content()))
			if err != nil {
				slog.Error("failed to decode image for ship render", "error", err)
				return
			}
			b := j.Bounds()
			img = image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
			draw.Draw(img, img.Bounds(), j, b.Min, draw.Src)
			w.cache.Set(key, img, 3600*time.Second)
		} else {
			img = y.(*image.RGBA)
		}
		if !canFly {
			img = effect.Grayscale(img)
		}
		w.image.Image = img
		w.image.Refresh()
	}()
}

func (w *ShipItem) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewVBox(container.NewPadded(w.image), w.label)
	return widget.NewSimpleRenderer(c)
}
