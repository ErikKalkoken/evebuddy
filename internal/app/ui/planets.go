package ui

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

// planetArea is the UI area that shows the skillqueue
type planetArea struct {
	content *fyne.Container
	planets []*app.CharacterPlanet
	table   *widget.Table
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
	a.table = a.makeTable()
	top := container.NewVBox(a.top, widget.NewSeparator())
	a.content = container.NewBorder(top, nil, nil, nil, a.table)
	return &a
}

func (a *planetArea) makeTable() *widget.Table {
	var headers = []struct {
		text  string
		width float32
	}{
		{"Region", 150},
		{"Constellation", 150},
		{"System", 150},
		{"Sec.", 50},
		{"Planet", 150},
		{"Type", 150},
		{"Installations", 100},
		{"Upgrade Lvl.", 100},
		{"Last Update", 100},
	}
	t := widget.NewTable(
		func() (rows int, cols int) {
			return len(a.planets), len(headers)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Template Template")
		},
		func(tci widget.TableCellID, co fyne.CanvasObject) {
			l := co.(*widget.Label)
			l.Importance = widget.MediumImportance
			l.Alignment = fyne.TextAlignLeading
			l.Truncation = fyne.TextTruncateOff
			if tci.Row >= len(a.planets) || tci.Row < 0 {
				return
			}
			p := a.planets[tci.Row]
			switch tci.Col {
			case 0:
				l.Text = p.EvePlanet.SolarSystem.Constellation.Region.Name
			case 1:
				l.Text = p.EvePlanet.SolarSystem.Constellation.Name
			case 2:
				l.Text = p.EvePlanet.SolarSystem.Name
			case 3:
				l.Text = fmt.Sprintf("%.1f", p.EvePlanet.SolarSystem.SecurityStatus)
				l.Importance = p.EvePlanet.SolarSystem.SecurityType().ToImportance()
			case 4:
				l.Text = p.EvePlanet.Name
			case 5:
				l.Text = p.EvePlanet.TypeDisplay()
			case 6:
				l.Alignment = fyne.TextAlignTrailing
				l.Text = strconv.Itoa(p.NumPins)
			case 7:
				l.Alignment = fyne.TextAlignTrailing
				l.Text = strconv.Itoa(p.UpgradeLevel)
			case 8:
				l.Text = humanize.Time(p.LastUpdate)
			}
			l.Refresh()
		},
	)
	t.ShowHeaderRow = true
	t.CreateHeader = func() fyne.CanvasObject {
		return widget.NewLabel("Template")
	}
	t.UpdateHeader = func(tci widget.TableCellID, co fyne.CanvasObject) {
		s := headers[tci.Col]
		co.(*widget.Label).SetText(s.text)
	}
	for i, h := range headers {
		t.SetColumnWidth(i, h.width)
	}
	t.OnSelected = func(tci widget.TableCellID) {
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
	a.table.Refresh()
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
	s := fmt.Sprintf("Planets: %s", t)
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
