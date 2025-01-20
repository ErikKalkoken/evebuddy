package desktop

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

type locationCharacter struct {
	id                 int32
	location           *app.EntityShort[int64]
	name               string
	region             *app.EntityShort[int32]
	ship               *app.EntityShort[int32]
	solarSystem        *app.EntityShort[int32]
	systemSecurity     optional.Optional[float32]
	securityImportance widget.Importance
}

// locationsArea is the UI area that shows an overview of all the user's characters.
type locationsArea struct {
	characters []locationCharacter
	content    *fyne.Container
	table      *widget.Table
	top        *widget.Label
	u          *DesktopUI
}

func (u *DesktopUI) newLocationsArea() *locationsArea {
	a := locationsArea{
		characters: make([]locationCharacter, 0),
		top:        widget.NewLabel(""),
		u:          u,
	}
	a.top.TextStyle.Bold = true

	top := container.NewVBox(a.top, widget.NewSeparator())
	a.table = a.makeTable()
	a.content = container.NewBorder(top, nil, nil, nil, a.table)
	return &a
}

func (a *locationsArea) makeTable() *widget.Table {
	var headers = []struct {
		text     string
		maxChars int
	}{
		{"Name", 20},
		{"Location", 35},
		{"System", 15},
		{"Sec.", 5},
		{"Region", 15},
		{"Ship", 15},
	}

	t := widget.NewTable(
		func() (rows int, cols int) {
			return len(a.characters), len(headers)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Template")
		},
		func(tci widget.TableCellID, co fyne.CanvasObject) {
			l := co.(*widget.Label)
			if tci.Row >= len(a.characters) || tci.Row < 0 {
				return
			}
			c := a.characters[tci.Row]
			l.Alignment = fyne.TextAlignLeading
			l.Importance = widget.MediumImportance
			l.Truncation = fyne.TextTruncateClip
			switch tci.Col {
			case 0:
				l.Text = c.name
				l.Truncation = fyne.TextTruncateEllipsis
			case 1:
				l.Text = ui.EntityNameOrFallback(c.location, "?")
				l.Truncation = fyne.TextTruncateEllipsis
			case 2:
				if c.solarSystem == nil || c.systemSecurity.IsEmpty() {
					l.Text = "?"
				} else {
					l.Text = c.solarSystem.Name
				}
			case 3:
				if c.systemSecurity.IsEmpty() {
					l.Text = "?"
					l.Importance = widget.LowImportance
				} else {
					if c.systemSecurity.IsEmpty() {
						l.Text = "?"
					} else {
						l.Text = fmt.Sprintf("%.1f", c.systemSecurity.ValueOrZero())
					}
					l.Importance = c.securityImportance
				}
				l.Alignment = fyne.TextAlignTrailing
			case 4:
				l.Text = ui.EntityNameOrFallback(c.region, "?")
			case 5:
				l.Text = ui.EntityNameOrFallback(c.ship, "?")
				l.Truncation = fyne.TextTruncateEllipsis
			}
			l.Refresh()
		},
	)
	t.ShowHeaderRow = true
	t.StickyColumnCount = 1
	t.CreateHeader = func() fyne.CanvasObject {
		return widget.NewLabel("Template")
	}
	t.UpdateHeader = func(tci widget.TableCellID, co fyne.CanvasObject) {
		s := headers[tci.Col]
		label := co.(*widget.Label)
		label.SetText(s.text)
	}
	t.OnSelected = func(tci widget.TableCellID) {
		defer t.UnselectAll()
	}

	for i, h := range headers {
		x := widget.NewLabel(strings.Repeat("w", h.maxChars))
		w := x.MinSize().Width
		t.SetColumnWidth(i, w)
	}
	return t
}

func (a *locationsArea) refresh() {
	t, i, err := func() (string, widget.Importance, error) {
		locations, err := a.updateCharacters()
		if err != nil {
			return "", 0, err
		}
		if len(a.characters) == 0 {
			return "No characters", widget.LowImportance, nil
		}
		s := fmt.Sprintf("%d characters • %d locations", len(a.characters), locations)
		return s, widget.MediumImportance, nil
	}()
	if err != nil {
		slog.Error("Failed to refresh overview UI", "err", err)
		t = "ERROR"
		i = widget.DangerImportance
	}
	a.top.Text = t
	a.top.Importance = i
	a.table.Refresh()
}

func (a *locationsArea) updateCharacters() (int, error) {
	var err error
	ctx := context.TODO()
	mycc, err := a.u.CharacterService.ListCharacters(ctx)
	if err != nil {
		return 0, err
	}
	locationIDs := set.New[int64]()
	cc := make([]locationCharacter, len(mycc))
	for i, m := range mycc {
		c := locationCharacter{
			id:   m.ID,
			name: m.EveCharacter.Name,
		}
		if m.Location != nil {
			locationIDs.Add(m.Location.ID)
			c.location = &app.EntityShort[int64]{
				ID:   m.Location.ID,
				Name: m.Location.DisplayName(),
			}
			if m.Location.SolarSystem != nil {
				c.solarSystem = &app.EntityShort[int32]{
					ID:   m.Location.SolarSystem.ID,
					Name: m.Location.SolarSystem.Name,
				}
				c.systemSecurity = optional.New(m.Location.SolarSystem.SecurityStatus)
				c.securityImportance = m.Location.SolarSystem.SecurityType().ToImportance()
				c.region = &app.EntityShort[int32]{
					ID:   m.Location.SolarSystem.Constellation.Region.ID,
					Name: m.Location.SolarSystem.Constellation.Region.Name,
				}
			}
		}
		if m.Ship != nil {
			c.ship = &app.EntityShort[int32]{
				ID:   m.Ship.ID,
				Name: m.Ship.Name,
			}
		}
		cc[i] = c
	}
	a.characters = cc
	return locationIDs.Size(), nil
}
