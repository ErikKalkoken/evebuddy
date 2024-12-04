package widgets

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/set"
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
	extractions := set.New[string]()
	expireTimes := make([]time.Time, 0)
	productions := set.New[string]()
	for _, p := range cp.Pins {
		if x := p.ExpiryTime.ValueOrZero(); x.After(time.Now()) {
			expireTimes = append(expireTimes, x)
		}
		switch p.Type.Group.ID {
		case app.EveGroupProcessors:
			if p.Schematic != nil {
				productions.Add(p.Schematic.Name)
			}
		case app.EveGroupExtractorControlUnits:
			if p.ExtractorProductType != nil {
				extractions.Add(p.ExtractorProductType.Name)
			}
		}
	}
	extractions2 := extractions.ToSlice()
	slices.Sort(extractions2)
	ex := strings.Join(extractions2, ",")
	var deadline string
	if len(expireTimes) == 0 {
		deadline = "EXPIRED"
	} else {
		slices.SortFunc(expireTimes, func(a, b time.Time) int {
			return b.Compare(a)
		})
		deadline = expireTimes[0].Format(app.TimeDefaultFormat)
	}
	w.extracting.SetText(fmt.Sprintf("%s by %s", ex, deadline))
	productions2 := productions.ToSlice()
	slices.Sort(productions2)
	prd := strings.Join(productions2, ",")
	w.producing.SetText(fmt.Sprintf("%s", prd))
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
