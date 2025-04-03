package character

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
	"github.com/ErikKalkoken/evebuddy/internal/humanize"
	iwidgets "github.com/ErikKalkoken/evebuddy/internal/widget"
)

const (
	planetImageSize         = 256
	planetWidgetSizeDesktop = 120
	planetWidgetSizeMobile  = 60
	planetBackgroundColor   = theme.ColorNameInputBackground
)

type PlanetWidget struct {
	widget.BaseWidget

	bg         *canvas.Rectangle
	extracting *widget.Label
	image      *canvas.Image
	infoIcon   *widget.Icon
	location   *widget.RichText
	post       *widget.Label
	producing  *widget.Label
}

func NewPlanet() *PlanetWidget {
	image := iwidgets.NewImageFromResource(theme.BrokenImageIcon(), fyne.NewSquareSize(planetWidgetSizeDesktop))
	extracting := widget.NewLabel("")
	extracting.Wrapping = fyne.TextWrapWord
	producing := widget.NewLabel("")
	producing.Wrapping = fyne.TextWrapWord
	location := widget.NewRichText()
	location.Wrapping = fyne.TextWrapWord
	w := &PlanetWidget{
		bg:         canvas.NewRectangle(theme.Color(planetBackgroundColor)),
		extracting: extracting,
		image:      image,
		infoIcon:   widget.NewIcon(theme.InfoIcon()),
		location:   location,
		post:       widget.NewLabel(""),
		producing:  producing,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *PlanetWidget) Set(cp *app.CharacterPlanet) {
	if cp.EvePlanet != nil && cp.EvePlanet.Type != nil {
		res, ok := cp.EvePlanet.Type.Icon()
		if ok {
			w.image.Resource = res
			w.image.Refresh()
		}
	}
	title := fmt.Sprintf("  %s - %s - %d installations", cp.EvePlanet.Name, cp.EvePlanet.TypeDisplay(), len(cp.Pins))
	w.location.Segments = slices.Concat(
		cp.EvePlanet.SolarSystem.SecurityStatusRichText(),
		iwidgets.NewRichTextSegmentFromText(title),
	)
	w.location.Refresh()

	extracted := strings.Join(cp.ExtractedTypeNames(), ",")
	var deadline string
	isExpired := false
	expires := cp.ExtractionsExpiryTime()
	if expires.IsZero() {
		deadline = "?"
		w.post.Hide()
	} else {
		deadline = expires.Format(app.DateTimeFormat)
		if expires.Before(time.Now()) {
			isExpired = true
		}
		w.post.Show()
	}
	if isExpired {
		w.post.Text = "EXPIRED"
		w.post.Importance = widget.DangerImportance
		w.post.Refresh()
	} else {
		w.post.Text = humanize.RelTime(expires)
		w.post.Importance = widget.SuccessImportance
		w.post.Refresh()
	}
	if extracted != "" {
		w.extracting.SetText(fmt.Sprintf("%s by %s", extracted, deadline))
	} else {
		w.extracting.SetText("-")
	}

	produced := strings.Join(cp.ProducedSchematicNames(), ",")
	if produced == "" {
		produced = "-"
	}
	w.producing.SetText(produced)
}

func (w *PlanetWidget) Refresh() {
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	w.bg.FillColor = th.Color(planetBackgroundColor, v)
	w.bg.Refresh()
	w.BaseWidget.Refresh()

}

func (w *PlanetWidget) CreateRenderer() fyne.WidgetRenderer {
	data := container.NewVBox(
		container.NewStack(
			w.bg,
			container.NewBorder(nil, nil, nil, w.infoIcon, w.location),
		),
		widget.NewForm(
			widget.NewFormItem("Extracting", w.extracting),
			widget.NewFormItem("Extraction due", w.post),
			widget.NewFormItem("Producing", w.producing),
		),
	)
	if fyne.CurrentDevice().IsMobile() {
		return widget.NewSimpleRenderer(data)
	}
	c := container.NewBorder(
		nil,
		nil,
		container.NewVBox(w.image),
		nil,
		data,
	)
	return widget.NewSimpleRenderer(c)
}
