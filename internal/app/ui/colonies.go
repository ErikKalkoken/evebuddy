package ui

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/dustin/go-humanize"
)

type colonyRow struct {
	character          string
	due                string
	dueImportance      widget.Importance
	extracting         string
	planet             string
	planetType         string
	producing          string
	region             string
	security           string
	securityImportance widget.Importance
}

// coloniesArea is the UI area that shows the skillqueue
type coloniesArea struct {
	content *fyne.Container
	rows    []colonyRow
	table   *widget.Table
	top     *widget.Label
	u       *UI
}

func (u *UI) newColoniesArea() *coloniesArea {
	a := coloniesArea{
		top:  widget.NewLabel(""),
		rows: make([]colonyRow, 0),
		u:    u,
	}
	a.top.TextStyle.Bold = true

	top := container.NewVBox(a.top, widget.NewSeparator())
	a.table = a.makeTable()
	a.content = container.NewBorder(top, nil, nil, nil, a.table)
	return &a
}

func (a *coloniesArea) makeTable() *widget.Table {
	var headers = []struct {
		text  string
		width float32
	}{
		{"Planet", 150},
		{"Sec.", 50},
		{"Type", 100},
		{"Extracting", 200},
		{"Due", 150},
		{"Producing", 200},
		{"Region", 150},
		{"Character", 150},
	}
	t := widget.NewTable(
		func() (rows int, cols int) {
			return len(a.rows), len(headers)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Template")
		},
		func(tci widget.TableCellID, co fyne.CanvasObject) {
			l := co.(*widget.Label)
			l.Importance = widget.MediumImportance
			l.Alignment = fyne.TextAlignLeading
			l.Truncation = fyne.TextTruncateOff
			if tci.Row >= len(a.rows) || tci.Row < 0 {
				return
			}
			w := a.rows[tci.Row]
			switch tci.Col {
			case 0:
				l.Text = w.planet
			case 1:
				l.Text = w.security
				l.Importance = w.securityImportance
				l.Alignment = fyne.TextAlignTrailing
			case 2:
				l.Text = w.planetType
			case 3:
				l.Text = w.extracting
				l.Truncation = fyne.TextTruncateEllipsis
			case 4:
				l.Text = w.due
				l.Importance = w.dueImportance
			case 5:
				l.Text = w.producing
				l.Truncation = fyne.TextTruncateEllipsis
			case 6:
				l.Text = w.region
			case 7:
				l.Text = w.character
				l.Truncation = fyne.TextTruncateEllipsis
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
	t.OnSelected = func(id widget.TableCellID) {
		t.UnselectAll()
	}
	return t
}

func (a *coloniesArea) refresh() {
	var t string
	var i widget.Importance
	if err := a.updateEntries(); err != nil {
		slog.Error("Failed to refresh wallet transaction UI", "err", err)
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

func (a *coloniesArea) makeTopText() (string, widget.Importance) {
	t := humanize.Comma(int64(len(a.rows)))
	s := fmt.Sprintf("Colonies: %s", t)
	return s, widget.MediumImportance
}

func (a *coloniesArea) updateEntries() error {
	pp, err := a.u.CharacterService.ListAllCharacterPlanets(context.TODO())
	if err != nil {
		return err
	}
	rows := make([]colonyRow, len(pp))
	for i, p := range pp {
		r := colonyRow{
			character:          a.u.StatusCacheService.CharacterName(p.CharacterID),
			planet:             p.EvePlanet.Name,
			planetType:         p.EvePlanet.TypeDisplay(),
			region:             p.EvePlanet.SolarSystem.Constellation.Region.Name,
			security:           fmt.Sprintf("%0.1f", p.EvePlanet.SolarSystem.SecurityStatus),
			securityImportance: p.EvePlanet.SolarSystem.SecurityType().ToImportance(),
		}
		extractions := strings.Join(p.ExtractedTypeNames(), ", ")
		if extractions == "" {
			extractions = "-"
		}
		r.extracting = extractions
		productions := strings.Join(p.ProducedSchematicNames(), ", ")
		if productions == "" {
			productions = "-"
		}
		r.producing = productions
		due := p.ExtractionsExpiryTime()
		if due.IsZero() {
			r.due = "-"
		} else if due.Before(time.Now()) {
			r.due = "OFFLINE"
			r.dueImportance = widget.WarningImportance
		} else {
			r.due = due.Format(app.TimeDefaultFormat)
		}
		rows[i] = r
	}
	a.rows = rows
	return nil
}
