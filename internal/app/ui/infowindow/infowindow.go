package infowindow

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"github.com/ErikKalkoken/evebuddy/internal/app/icon"

	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
)

const (
	defaultIconPixelSize = 64
	defaultIconUnitSize  = 32
	logoZoomFactor       = 1.3
	zoomImagePixelSize   = 512
)

func showZoomWindow(title string, id int32, load func(int32, int) (fyne.Resource, error)) {
	w := fyne.CurrentApp().NewWindow(title)
	s := float32(zoomImagePixelSize) / w.Canvas().Scale()
	i := appwidget.NewImageResourceAsync(icon.QuestionmarkSvg, fyne.NewSquareSize(s), func() (fyne.Resource, error) {
		return load(id, zoomImagePixelSize)
	})
	p := theme.Padding()
	w.SetContent(container.New(layout.NewCustomPaddedLayout(-p, -p, -p, -p), i))
	w.Show()
}
