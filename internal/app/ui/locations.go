package ui

import (
	"context"
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/infowindow"
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

type LocationOverview struct {
	widget.BaseWidget

	rows []locationCharacter
	body fyne.CanvasObject
	top  *widget.Label
	u    *BaseUI
}

func NewLocations(u *BaseUI) *LocationOverview {
	a := &LocationOverview{
		rows: make([]locationCharacter, 0),
		top:  MakeTopLabel(),
		u:    u,
	}
	a.ExtendBaseWidget(a)

	headers := []headerDef{
		{"Name", 200},
		{"Location", 250},
		{"System", 150},
		{"Sec.", 50},
		{"Region", 150},
		{"Ship", 150},
	}

	makeDataLabel := func(col int, r locationCharacter) (string, fyne.TextAlign, widget.Importance) {
		var align fyne.TextAlign
		var importance widget.Importance
		var text string
		switch col {
		case 0:
			text = r.name
		case 1:
			text = EntityNameOrFallback(r.location, "?")
		case 2:
			text = EntityNameOrFallback(r.solarSystem, "?")
		case 3:
			text = r.systemSecurityDisplay()
			importance = r.securityImportance
			align = fyne.TextAlignTrailing
		case 4:
			text = EntityNameOrFallback(r.region, "?")
		case 5:
			text = EntityNameOrFallback(r.ship, "?")
		}
		return text, align, importance
	}
	if a.u.IsDesktop() {
		a.body = makeDataTableForDesktop(headers, &a.rows, makeDataLabel, func(c int, r locationCharacter) {
			switch c {
			case 0:
				a.u.ShowInfoWindow(infowindow.Character, int64(r.id))
			case 1:
				a.u.ShowInfoWindow(infowindow.Location, int64(r.location.ID))
			case 2:
				a.u.ShowInfoWindow(infowindow.SolarSystem, int64(r.solarSystem.ID))
			case 5:
				a.u.ShowTypeInfoWindow(r.ship.ID)
			}
		})
	} else {
		a.body = makeDataTableForMobile(headers, &a.rows, makeDataLabel, func(r locationCharacter) {
			a.u.ShowInfoWindow(infowindow.Location, int64(r.location.ID))
		})
	}
	return a
}

func (a *LocationOverview) CreateRenderer() fyne.WidgetRenderer {
	top := container.NewVBox(a.top, widget.NewSeparator())
	c := container.NewBorder(top, nil, nil, nil, a.body)
	return widget.NewSimpleRenderer(c)
}

func (a *LocationOverview) Update() {
	t, i, err := func() (string, widget.Importance, error) {
		locations, err := a.updateCharacters()
		if err != nil {
			return "", 0, err
		}
		if len(a.rows) == 0 {
			return "No characters", widget.LowImportance, nil
		}
		s := fmt.Sprintf("%d characters • %d locations", len(a.rows), locations)
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

func (a *LocationOverview) updateCharacters() (int, error) {
	var err error
	ctx := context.TODO()
	mycc, err := a.u.CharacterService().ListCharacters(ctx)
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
	a.rows = cc
	return locationIDs.Size(), nil
}
