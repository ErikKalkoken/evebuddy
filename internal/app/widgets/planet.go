package widgets

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

const (
	planetImageSize  = 128
	planetWidgetSize = 40
)

type Planet struct {
	widget.BaseWidget
	image    *canvas.Image
	security *widget.Label
	title    *widget.Label
	sv       app.EveImageService
}

func NewPlanet(sv app.EveImageService) *Planet {
	image := canvas.NewImageFromResource(theme.BrokenImageIcon())
	image.FillMode = canvas.ImageFillContain
	image.SetMinSize(fyne.Size{Width: planetWidgetSize, Height: planetWidgetSize})
	w := &Planet{
		image:    image,
		security: widget.NewLabel(""),
		title:    widget.NewLabel(""),
		sv:       sv,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *Planet) Set(cp *app.CharacterPlanet) {
	refreshImageResourceAsync(w.image, func() (fyne.Resource, error) {
		return w.sv.InventoryTypeIcon(cp.EvePlanet.Type.ID, planetImageSize)

	})
	w.security.Text = fmt.Sprintf("%.1f", cp.EvePlanet.SolarSystem.SecurityStatus)
	w.security.Importance = cp.EvePlanet.SolarSystem.SecurityType().ToImportance()
	w.security.Refresh()
	w.title.SetText(fmt.Sprintf("%s - %s - %d installations", cp.EvePlanet.Name, cp.EvePlanet.TypeDisplay(), cp.NumPins))
}

func (w *Planet) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(
		nil,
		nil,
		w.image,
		nil,
		container.NewHBox(w.security, w.title),
	)
	return widget.NewSimpleRenderer(c)
}
