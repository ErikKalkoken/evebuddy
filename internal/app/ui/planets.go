package ui

import (
	"context"
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/widgets"
)

// planetArea is the UI area that shows the skillqueue
type planetArea struct {
	content *fyne.Container
	planets []*app.CharacterPlanet
	list    *widget.List
	top     *widget.Label
	u       *UI
}

func (u *UI) newPlanetArea() *planetArea {
	a := planetArea{
		planets: make([]*app.CharacterPlanet, 0),
		top:     widget.NewLabel(""),
		u:       u,
	}

	a.top.TextStyle.Bold = true
	a.list = a.makeList()
	top := container.NewVBox(a.top, widget.NewSeparator())
	a.content = container.NewBorder(top, nil, nil, nil, a.list)
	return &a
}

func (a *planetArea) makeList() *widget.List {
	t := widget.NewList(
		func() int {
			return len(a.planets)
		},
		func() fyne.CanvasObject {
			return widgets.NewPlanet(a.u.EveImageService)
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

func (a *planetArea) refresh() {
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
}

func (a *planetArea) makeTopText() (string, widget.Importance) {
	if !a.u.hasCharacter() {
		return "No character", widget.LowImportance
	}
	c := a.u.currentCharacter()
	hasData := a.u.StatusCacheService.CharacterSectionExists(c.ID, app.SectionPlanets)
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance
	}
	t := humanize.Comma(int64(len(a.planets)))
	s := fmt.Sprintf("Colonies: %s", t)
	return s, widget.MediumImportance
}

func (a *planetArea) updateEntries() error {
	if !a.u.hasCharacter() {
		a.planets = make([]*app.CharacterPlanet, 0)
		return nil
	}
	characterID := a.u.characterID()
	var err error
	a.planets, err = a.u.CharacterService.ListCharacterPlanets(context.TODO(), characterID)
	if err != nil {
		return fmt.Errorf("fetch planets for character %d: %w", characterID, err)
	}
	return nil
}
