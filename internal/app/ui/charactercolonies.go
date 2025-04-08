package ui

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
	"github.com/ErikKalkoken/evebuddy/internal/humanize"
	iwidgets "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type CharacterColonies struct {
	widget.BaseWidget

	OnUpdate func(total, expired int)

	planets []*app.CharacterPlanet
	list    *widget.List
	top     *widget.Label
	u       *BaseUI
}

func NewCharacterColonies(u *BaseUI) *CharacterColonies {
	a := &CharacterColonies{
		planets: make([]*app.CharacterPlanet, 0),
		top:     appwidget.MakeTopLabel(),
		u:       u,
	}
	a.ExtendBaseWidget(a)
	a.list = a.makeList()
	return a
}

func (a *CharacterColonies) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(a.top, nil, nil, nil, a.list)
	return widget.NewSimpleRenderer(c)
}

func (a *CharacterColonies) makeList() *widget.List {
	var l *widget.List
	l = widget.NewList(
		func() int {
			return len(a.planets)
		},
		func() fyne.CanvasObject {
			return NewPlanet()
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.planets) || id < 0 {
				return
			}
			p := a.planets[id]
			o := co.(*PlanetWidget)
			o.Set(p)
			l.SetItemHeight(id, o.MinSize().Height)
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		defer l.UnselectAll()
		if id >= len(a.planets) || id < 0 {
			return
		}
		p := a.planets[id]
		a.u.ShowInfoWindow(app.EveEntitySolarSystem, p.EvePlanet.SolarSystem.ID)
	}
	return l
}

func (a *CharacterColonies) Update() {
	var t string
	var i widget.Importance
	if err := a.updateEntries(); err != nil {
		slog.Error("Failed to refresh wallet journal UI", "err", err)
		t = "ERROR"
		i = widget.DangerImportance
	} else {
		t, i = a.makeTopText()
	}
	a.top.Text = t
	a.top.Importance = i
	a.top.Refresh()
	a.list.Refresh()
	if a.OnUpdate != nil {
		var expiredCount int
		for _, p := range a.planets {
			if t := p.ExtractionsExpiryTime(); !t.IsZero() && t.Before(time.Now()) {
				expiredCount++
			}
		}
		a.OnUpdate(len(a.planets), expiredCount)
	}
}

func (a *CharacterColonies) makeTopText() (string, widget.Importance) {
	if !a.u.HasCharacter() {
		return "No character", widget.LowImportance
	}
	c := a.u.CurrentCharacter()
	hasData := a.u.StatusCacheService().CharacterSectionExists(c.ID, app.SectionPlanets)
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance
	}
	var max string
	s, err := a.u.CharacterService().GetSkill(context.Background(), c.ID, app.EveTypeInterplanetaryConsolidation)
	if errors.Is(err, app.ErrNotFound) {
		max = "1"
	} else if err != nil {
		max = "?"
		slog.Error("Trying to fetch skill for character", "character", c.ID, "error", err)
	} else {
		max = strconv.Itoa(s.ActiveSkillLevel + 1)
	}
	t := fmt.Sprintf("Installed: %d / %s", len(a.planets), max)
	return t, widget.MediumImportance
}

func (a *CharacterColonies) updateEntries() error {
	if !a.u.HasCharacter() {
		a.planets = make([]*app.CharacterPlanet, 0)
		return nil
	}
	characterID := a.u.CurrentCharacterID()
	var err error
	a.planets, err = a.u.CharacterService().ListPlanets(context.TODO(), characterID)
	if err != nil {
		return err
	}
	return nil
}

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
