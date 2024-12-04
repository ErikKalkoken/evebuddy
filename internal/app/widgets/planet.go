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
	planetImageSize  = 256
	planetWidgetSize = 120
)

type Planet struct {
	widget.BaseWidget
	extracting *widget.Label
	expired    *widget.Label
	image      *canvas.Image
	producing  *widget.Label
	security   *widget.Label
	title      *widget.Label
}

func NewPlanet() *Planet {
	image := canvas.NewImageFromResource(theme.BrokenImageIcon())
	image.FillMode = canvas.ImageFillContain
	image.SetMinSize(fyne.Size{Width: planetWidgetSize, Height: planetWidgetSize})
	offline := widget.NewLabel("OFFLINE")
	offline.Importance = widget.WarningImportance
	w := &Planet{
		expired:    offline,
		extracting: widget.NewLabel(""),
		image:      image,
		producing:  widget.NewLabel(""),
		security:   widget.NewLabel(""),
		title:      widget.NewLabel(""),
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *Planet) Set(cp *app.CharacterPlanet) {
	if cp.EvePlanet != nil && cp.EvePlanet.Type != nil {
		res, ok := cp.EvePlanet.Type.Icon()
		if ok {
			w.image.Resource = res
			w.image.Refresh()
		}
	}
	w.security.Text = fmt.Sprintf("%.1f", cp.EvePlanet.SolarSystem.SecurityStatus)
	w.security.Importance = cp.EvePlanet.SolarSystem.SecurityType().ToImportance()
	w.security.Refresh()
	s := fmt.Sprintf("%s - %s - %d installations", cp.EvePlanet.Name, cp.EvePlanet.TypeDisplay(), len(cp.Pins))
	w.title.SetText(s)

	extracted := strings.Join(cp.ExtractedTypeNames(), ",")
	var deadline string
	isExpired := false
	if x := cp.ExtractionsExpiryTime(); x.IsZero() {
		deadline = "?"
	} else {
		deadline = x.Format(app.TimeDefaultFormat)
		if x.Before(time.Now()) {
			isExpired = true
		}
	}
	if isExpired {
		w.expired.Show()
	} else {
		w.expired.Hide()
	}
	var x string
	if extracted != "" {
		x = fmt.Sprintf("%s by %s", extracted, deadline)
	} else {
		x = "-"
	}
	w.extracting.SetText(x)

	produced := strings.Join(cp.ProducedSchematicNames(), ",")
	if produced == "" {
		produced = "-"
	}
	w.producing.SetText(produced)
}

func (w *Planet) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(
		nil,
		nil,
		container.NewVBox(w.image),
		nil,
		container.NewVBox(
			container.NewStack(canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground)), container.NewHBox(w.security, w.title)),
			widget.NewForm(
				widget.NewFormItem("Extracting", container.NewHBox(w.extracting, w.expired)),
				widget.NewFormItem("Producing", w.producing),
			),
		),
	)
	return widget.NewSimpleRenderer(c)
}
