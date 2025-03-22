package cross

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
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/shared"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type colonyRow struct {
	character          string
	due                string
	dueImportance      widget.Importance
	extracting         string
	isExpired          bool
	planet             string
	planetType         string
	producing          string
	region             string
	security           string
	securityImportance widget.Importance
	solarSystemID      int32
	characterID        int32
}

type ColonyOverview struct {
	widget.BaseWidget

	OnUpdate func(top string)

	body fyne.CanvasObject
	rows []colonyRow
	top  *widget.Label
	u    app.UI
}

func NewColonyOverview(u app.UI) *ColonyOverview {
	a := &ColonyOverview{
		rows: make([]colonyRow, 0),
		top:  shared.MakeTopLabel(),
		u:    u,
	}
	a.ExtendBaseWidget(a)
	headers := []iwidget.HeaderDef{
		{Text: "Planet", Width: 150},
		{Text: "Sec.", Width: 50},
		{Text: "Type", Width: 100},
		{Text: "Extracting", Width: 200},
		{Text: "Due", Width: 150},
		{Text: "Producing", Width: 200},
		{Text: "Region", Width: 150},
		{Text: "Character", Width: 150},
	}
	makeDataLabel := func(col int, w colonyRow) (string, fyne.TextAlign, widget.Importance) {
		var align fyne.TextAlign
		var importance widget.Importance
		var text string
		switch col {
		case 0:
			text = w.planet
		case 1:
			text = w.security
			importance = w.securityImportance
			align = fyne.TextAlignTrailing
		case 2:
			text = w.planetType
		case 3:
			text = w.extracting
		case 4:
			text = w.due
			importance = w.dueImportance
		case 5:
			text = w.producing
		case 6:
			text = w.region
		case 7:
			text = w.character
		}
		return text, align, importance
	}
	if a.u.IsDesktop() {
		a.body = iwidget.MakeDataTableForDesktop(headers, &a.rows, makeDataLabel, func(col int, r colonyRow) {
			switch col {
			case 0, 1, 2, 3, 4, 5, 6:
				a.u.ShowInfoWindow(app.EveEntitySolarSystem, r.solarSystemID)
			case 7:
				a.u.ShowInfoWindow(app.EveEntityCharacter, r.characterID)
			}
		})
	} else {
		a.body = iwidget.MakeDataTableForMobile(headers, &a.rows, makeDataLabel, nil)
	}
	return a
}

func (a *ColonyOverview) CreateRenderer() fyne.WidgetRenderer {
	top := container.NewVBox(a.top, widget.NewSeparator())
	c := container.NewBorder(top, nil, nil, nil, a.body)
	return widget.NewSimpleRenderer(c)
}

func (a *ColonyOverview) Update() {
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
	a.body.Refresh()
	if a.OnUpdate != nil {
		a.OnUpdate(t)
	}
}

func (a *ColonyOverview) makeTopText() (string, widget.Importance) {
	var expiredCount int
	for _, c := range a.rows {
		if c.isExpired {
			expiredCount++
		}
	}
	s := fmt.Sprintf("%d colonies", len(a.rows))
	if expiredCount > 0 {
		s += fmt.Sprintf(" • %d expired", expiredCount)
	}
	return s, widget.MediumImportance
}

func (a *ColonyOverview) updateEntries() error {
	pp, err := a.u.CharacterService().ListAllCharacterPlanets(context.TODO())
	if err != nil {
		return err
	}
	rows := make([]colonyRow, len(pp))
	for i, p := range pp {
		r := colonyRow{
			character:          a.u.StatusCacheService().CharacterName(p.CharacterID),
			characterID:        p.CharacterID,
			planet:             p.EvePlanet.Name,
			planetType:         p.EvePlanet.TypeDisplay(),
			region:             p.EvePlanet.SolarSystem.Constellation.Region.Name,
			solarSystemID:      p.EvePlanet.SolarSystem.ID,
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
			r.isExpired = true
		} else {
			r.due = due.Format(app.DateTimeFormat)
		}
		rows[i] = r
	}
	a.rows = rows
	return nil
}
