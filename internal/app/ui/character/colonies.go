package character

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
)

type Colonies struct {
	widget.BaseWidget

	OnUpdate func(total, expired int)

	planets []*app.CharacterPlanet
	list    *widget.List
	top     *widget.Label
	u       app.UI
}

func NewColonies(u app.UI) *Colonies {
	a := &Colonies{
		planets: make([]*app.CharacterPlanet, 0),
		top:     appwidget.MakeTopLabel(),
		u:       u,
	}
	a.ExtendBaseWidget(a)
	a.list = a.makeList()
	return a
}

func (a *Colonies) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(a.top, nil, nil, nil, a.list)
	return widget.NewSimpleRenderer(c)
}

func (a *Colonies) makeList() *widget.List {
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

func (a *Colonies) Update() {
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

func (a *Colonies) makeTopText() (string, widget.Importance) {
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

func (a *Colonies) updateEntries() error {
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
