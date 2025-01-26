package ui

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
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

func (c locationCharacter) systemSecurityDisplay() string {
	if c.systemSecurity.IsEmpty() {
		return "?"
	}
	return fmt.Sprintf("%.1f", c.systemSecurity.ValueOrZero())
}

// LocationsArea is the UI area that shows an overview of all the user's characters.
//
// It generates output which is customized for either desktop or mobile.
type LocationsArea struct {
	Content fyne.CanvasObject

	characters []locationCharacter
	body       fyne.CanvasObject
	top        *widget.Label
	u          *BaseUI
}

func (u *BaseUI) NewLocationsArea() *LocationsArea {
	a := LocationsArea{
		characters: make([]locationCharacter, 0),
		top:        widget.NewLabel(""),
		u:          u,
	}
	a.top.TextStyle.Bold = true

	top := container.NewVBox(a.top, widget.NewSeparator())
	if a.u.IsDesktop() {
		a.body = a.makeTable()
	} else {
		a.body = a.makeList()
	}
	a.Content = container.NewBorder(top, nil, nil, nil, a.body)
	return &a
}

func (a *LocationsArea) makeTable() *widget.Table {
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
				l.Text = EntityNameOrFallback(c.location, "?")
				l.Truncation = fyne.TextTruncateEllipsis
			case 2:
				l.Text = EntityNameOrFallback(c.solarSystem, "?")
			case 3:
				l.Text = c.systemSecurityDisplay()
				l.Importance = c.securityImportance
				l.Alignment = fyne.TextAlignTrailing
			case 4:
				l.Text = EntityNameOrFallback(c.region, "?")
			case 5:
				l.Text = EntityNameOrFallback(c.ship, "?")
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

func (a *LocationsArea) makeList() *widget.List {
	p := theme.Padding()
	removeVPadding := layout.NewCustomPaddedLayout(-p, -p, 0, 0)
	makeXLabel := func(style ...fyne.TextStyle) fyne.CanvasObject {
		x := widget.NewLabel("Template")
		if len(style) > 0 {
			x.TextStyle = style[0]
		}
		x.Truncation = fyne.TextTruncateEllipsis
		c := container.New(removeVPadding, x)
		return c
	}
	unfurlXLabel := func(co fyne.CanvasObject) *widget.Label {
		return co.(*fyne.Container).Objects[0].(*widget.Label)
	}
	l := widget.NewList(
		func() int {
			return len(a.characters)
		},
		func() fyne.CanvasObject {
			return container.New(
				layout.NewCustomPaddedVBoxLayout(0),
				makeXLabel(fyne.TextStyle{Bold: true}),
				makeXLabel(),
				makeXLabel(),
				makeXLabel(),
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			vbox := co.(*fyne.Container).Objects
			if id >= len(a.characters) || id < 0 {
				return
			}
			c := a.characters[id]
			location := c.systemSecurityDisplay()
			if c.location != nil {
				location += " " + c.location.Name
			} else {
				location += " " + EntityNameOrFallback(c.solarSystem, "?")
			}
			unfurlXLabel(vbox[0]).SetText(c.name)
			unfurlXLabel(vbox[1]).SetText(location)
			unfurlXLabel(vbox[2]).SetText(EntityNameOrFallback(c.region, "?"))
			unfurlXLabel(vbox[3]).SetText(EntityNameOrFallback(c.ship, "?"))
		},
	)
	return l
}

func (a *LocationsArea) Refresh() {
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
	a.body.Refresh()
}

func (a *LocationsArea) updateCharacters() (int, error) {
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
