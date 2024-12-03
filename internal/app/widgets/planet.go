package widgets

import (
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

const (
	planetImageSize  = 128
	planetWidgetSize = 80
)

type Planet struct {
	widget.BaseWidget
	image    *canvas.Image
	security *widget.Label
	title    *widget.Label
	content  *widget.Label
	sv       app.EveImageService
}

func NewPlanet(sv app.EveImageService) *Planet {
	image := canvas.NewImageFromResource(theme.BrokenImageIcon())
	image.FillMode = canvas.ImageFillContain
	image.SetMinSize(fyne.Size{Width: planetWidgetSize, Height: planetWidgetSize})
	w := &Planet{
		image:    image,
		content:  widget.NewLabel("first\nsecond"),
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
	s := fmt.Sprintf("%s - %s - %d installations", cp.EvePlanet.Name, cp.EvePlanet.TypeDisplay(), len(cp.Pins))
	w.title.SetText(s)
	extractors := make(map[string]time.Time)
	processors := make(map[string]bool)
	for _, p := range cp.Pins {
		switch p.Type.Group.ID {
		case app.EveGroupProcessors:
			if p.Schematic != nil {
				processors[p.Schematic.Name] = true
			}
		case app.EveGroupExtractorControlUnits:
			if p.ExtractorProductType != nil {
				extractors[p.ExtractorProductType.Name] = p.ExpiryTime.ValueOrZero()
			}
		}
	}
	l := make([]string, 0)
	for name, expiry := range extractors {
		var exp string
		if expiry.After(time.Now()) {
			exp = expiry.Format(app.TimeDefaultFormat)
		} else {
			exp = "EXPIRED"
		}
		l = append(l, fmt.Sprintf("Extraction: %s by %s", name, exp))
	}
	for name := range processors {
		l = append(l, fmt.Sprintf("Production: %s", name))
	}
	w.content.SetText(strings.Join(l, "\n"))
}

func (w *Planet) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(
		nil,
		nil,
		w.image,
		nil,
		container.NewVBox(
			container.NewHBox(w.security, w.title),
			w.content,
		),
	)
	return widget.NewSimpleRenderer(c)
}
