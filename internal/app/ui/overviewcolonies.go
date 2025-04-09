package ui

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/app"
	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type colonyRow struct {
	character     string
	due           string
	dueColor      fyne.ThemeColorName
	extracting    string
	isExpired     bool
	planet        string
	planetType    app.EntityShort[int32]
	producing     string
	region        app.EntityShort[int32]
	security      string
	securityColor fyne.ThemeColorName
	solarSystemID int32
	characterID   int32
}

type OverviewColonies struct {
	widget.BaseWidget

	OnUpdate func(total, expired int)

	body fyne.CanvasObject
	rows []colonyRow
	top  *widget.Label
	u    *BaseUI
}

func NewOverviewColonies(u *BaseUI) *OverviewColonies {
	a := &OverviewColonies{
		rows: make([]colonyRow, 0),
		top:  appwidget.MakeTopLabel(),
		u:    u,
	}
	a.ExtendBaseWidget(a)
	headers := []iwidget.HeaderDef{
		{Text: "Planet", Width: 150},
		{Text: "Type", Width: 100},
		{Text: "Extracting", Width: 200},
		{Text: "Due", Width: 150},
		{Text: "Producing", Width: 200},
		{Text: "Region", Width: 150},
		{Text: "Character", Width: characterColumnWidth},
	}
	makeCell := func(col int, r colonyRow) []widget.RichTextSegment {
		switch col {
		case 0:
			return slices.Concat(
				iwidget.NewRichTextSegmentFromText(r.security, widget.RichTextStyle{
					ColorName: r.securityColor,
					Inline:    true,
				}),
				iwidget.NewRichTextSegmentFromText("  "+r.planet),
			)
		case 1:
			return iwidget.NewRichTextSegmentFromText(r.planetType.Name)
		case 2:
			return iwidget.NewRichTextSegmentFromText(r.extracting)
		case 3:
			return iwidget.NewRichTextSegmentFromText(r.due, widget.RichTextStyle{
				ColorName: r.dueColor,
			})
		case 4:
			return iwidget.NewRichTextSegmentFromText(r.producing)
		case 5:
			return iwidget.NewRichTextSegmentFromText(r.region.Name)
		case 6:
			return iwidget.NewRichTextSegmentFromText(r.character)
		}
		return iwidget.NewRichTextSegmentFromText("?")
	}
	if a.u.IsDesktop() {
		a.body = iwidget.MakeDataTableForDesktop2(headers, &a.rows, makeCell, func(col int, r colonyRow) {
			switch col {
			case 0:
				a.u.ShowInfoWindow(app.EveEntitySolarSystem, r.solarSystemID)
			case 1:
				a.u.ShowInfoWindow(app.EveEntityInventoryType, r.planetType.ID)
			case 5:
				a.u.ShowInfoWindow(app.EveEntityRegion, r.region.ID)
			case 6:
				a.u.ShowInfoWindow(app.EveEntityCharacter, r.characterID)
			}
		})
	} else {
		a.body = iwidget.MakeDataTableForMobile2(headers, &a.rows, makeCell, func(r colonyRow) {
			a.u.ShowInfoWindow(app.EveEntitySolarSystem, r.solarSystemID)
		})
	}
	return a
}

func (a *OverviewColonies) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(a.top, nil, nil, nil, a.body)
	return widget.NewSimpleRenderer(c)
}

func (a *OverviewColonies) Update() {
	var s string
	var i widget.Importance
	var total, expired int
	if err := a.updateEntries(); err != nil {
		slog.Error("Failed to refresh wallet transaction UI", "err", err)
		s = "ERROR"
		i = widget.DangerImportance
	} else {
		total = len(a.rows)
		for _, c := range a.rows {
			if c.isExpired {
				expired++
			}
		}
		s = fmt.Sprintf("%d colonies", total)
		if expired > 0 {
			s += fmt.Sprintf(" • %d expired", expired)
		}
	}
	a.top.Text = s
	a.top.Importance = i
	a.top.Refresh()
	a.body.Refresh()
	if a.OnUpdate != nil {
		a.OnUpdate(total, expired)
	}
}

func (a *OverviewColonies) updateEntries() error {
	pp, err := a.u.CharacterService().ListAllPlanets(context.TODO())
	if err != nil {
		return err
	}
	rows := make([]colonyRow, len(pp))
	for i, p := range pp {
		r := colonyRow{
			character:   a.u.StatusCacheService().CharacterName(p.CharacterID),
			characterID: p.CharacterID,
			dueColor:    theme.ColorNameForeground, // default
			planet:      p.EvePlanet.Name,
			planetType: app.EntityShort[int32]{
				ID:   p.EvePlanet.Type.ID,
				Name: p.EvePlanet.TypeDisplay(),
			},
			region: app.EntityShort[int32]{
				ID:   p.EvePlanet.SolarSystem.Constellation.Region.ID,
				Name: p.EvePlanet.SolarSystem.Constellation.Region.Name,
			},
			security:      fmt.Sprintf("%0.1f", p.EvePlanet.SolarSystem.SecurityStatus),
			securityColor: p.EvePlanet.SolarSystem.SecurityType().ToColorName(),
			solarSystemID: p.EvePlanet.SolarSystem.ID,
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
			r.dueColor = theme.ColorNameError
			r.isExpired = true
		} else {
			r.due = due.Format(app.DateTimeFormat)
		}
		rows[i] = r
	}
	a.rows = rows
	return nil
}
