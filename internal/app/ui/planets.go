package ui

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
	"github.com/ErikKalkoken/evebuddy/internal/app/character"
	"github.com/ErikKalkoken/evebuddy/internal/app/widgets"
)

// PlanetArea is the UI area that shows the skillqueue
type PlanetArea struct {
	Content *fyne.Container

	OnRefresh func(count int)

	planets []*app.CharacterPlanet
	list    *widget.List
	top     *widget.Label
	u       *BaseUI
}

func (u *BaseUI) NewPlanetArea() *PlanetArea {
	a := PlanetArea{
		planets: make([]*app.CharacterPlanet, 0),
		top:     makeTopLabel(),
		u:       u,
	}
	a.list = a.makeList()
	top := container.NewVBox(a.top, widget.NewSeparator())
	a.Content = container.NewBorder(top, nil, nil, nil, a.list)
	return &a
}

func (a *PlanetArea) makeList() *widget.List {
	t := widget.NewList(
		func() int {
			return len(a.planets)
		},
		func() fyne.CanvasObject {
			return widgets.NewPlanet()
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.planets) || id < 0 {
				return
			}
			o := co.(*widgets.Planet)
			p := a.planets[id]
			o.Set(p)
		},
	)
	t.OnSelected = func(id widget.ListItemID) {
		defer t.UnselectAll()
	}
	return t
}

func (a *PlanetArea) Refresh() {
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
	if a.OnRefresh != nil {
		var expiredCount int
		for _, p := range a.planets {
			if t := p.ExtractionsExpiryTime(); !t.IsZero() && t.Before(time.Now()) {
				expiredCount++
			}
		}
		a.OnRefresh(expiredCount)
	}
}

func (a *PlanetArea) makeTopText() (string, widget.Importance) {
	if !a.u.HasCharacter() {
		return "No character", widget.LowImportance
	}
	c := a.u.CurrentCharacter()
	hasData := a.u.StatusCacheService.CharacterSectionExists(c.ID, app.SectionPlanets)
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance
	}
	var max string
	s, err := a.u.CharacterService.GetCharacterSkill(context.Background(), c.ID, app.EveTypeInterplanetaryConsolidation)
	if errors.Is(err, character.ErrNotFound) {
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

func (a *PlanetArea) updateEntries() error {
	if !a.u.HasCharacter() {
		a.planets = make([]*app.CharacterPlanet, 0)
		return nil
	}
	characterID := a.u.CharacterID()
	var err error
	a.planets, err = a.u.CharacterService.ListCharacterPlanets(context.TODO(), characterID)
	if err != nil {
		return err
	}
	return nil
}
