package widgets

import (
	"fmt"
	"slices"
	"strings"

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
	image      *canvas.Image
	security   *widget.Label
	title      *widget.Label
	extracting *widget.Label
	producing  *widget.Label
}

func NewPlanet() *Planet {
	image := canvas.NewImageFromResource(theme.BrokenImageIcon())
	image.FillMode = canvas.ImageFillContain
	image.SetMinSize(fyne.Size{Width: planetWidgetSize, Height: planetWidgetSize})
	w := &Planet{
		image:      image,
		extracting: widget.NewLabel(""),
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

	extracted := extractedStringsSorted(cp.ExtractedTypes(), func(a *app.EveType) string {
		return a.Name
	})
	extracted2 := strings.Join(extracted, ",")
	var deadline string
	if x := cp.ExtractionsExpiryTime(); x.IsZero() {
		deadline = "EXPIRED"
	} else {
		deadline = x.Format(app.TimeDefaultFormat)
	}
	w.extracting.SetText(fmt.Sprintf("%s by %s", extracted2, deadline))

	produced := extractedStringsSorted(cp.ProducedSchematics(), func(a *app.EveSchematic) string {
		return a.Name
	})
	produced2 := strings.Join(produced, ",")
	w.producing.SetText(fmt.Sprintf("%s", produced2))
}

func extractedStringsSorted[T any](s []T, extract func(a T) string) []string {
	s2 := make([]string, 0)
	for _, x := range s {
		s2 = append(s2, extract(x))
	}
	slices.Sort(s2)
	return s2
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
				widget.NewFormItem("Extracting", w.extracting),
				widget.NewFormItem("Producing", w.producing),
			),
		),
	)
	return widget.NewSimpleRenderer(c)
}
