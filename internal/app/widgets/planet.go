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
	"github.com/ErikKalkoken/evebuddy/internal/humanize"
)

const (
	planetImageSize         = 256
	planetWidgetSizeDesktop = 120
	planetWidgetSizeMobile  = 60
)

type Planet struct {
	widget.BaseWidget
	extracting *widget.Label
	post       *widget.Label
	image      *canvas.Image
	producing  *widget.Label
	security   *widget.Label
	title      *widget.Label
}

func NewPlanet() *Planet {
	image := canvas.NewImageFromResource(theme.BrokenImageIcon())
	image.FillMode = canvas.ImageFillContain
	isMobile := fyne.CurrentDevice().IsMobile()
	var size float32
	if isMobile {
		size = planetWidgetSizeMobile
	} else {
		size = planetWidgetSizeDesktop
	}
	image.SetMinSize(fyne.Size{Width: size, Height: size})
	w := &Planet{
		post:       widget.NewLabel(""),
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
	expires := cp.ExtractionsExpiryTime()
	if expires.IsZero() {
		deadline = "?"
		w.post.Hide()
	} else {
		deadline = expires.Format(app.TimeDefaultFormat)
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

func (w *Planet) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(
		nil,
		nil,
		container.NewVBox(w.image),
		nil,
		container.NewVBox(
			container.NewStack(canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground)), container.NewHBox(w.security, w.title)),
			widget.NewForm(
				widget.NewFormItem("Extracting", container.NewHBox(w.extracting, w.post)),
				widget.NewFormItem("Producing", w.producing),
			),
		),
	)
	return widget.NewSimpleRenderer(c)
}
