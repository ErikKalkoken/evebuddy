package cross

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/shared"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type jumpClone struct {
	id                 int64
	location           *app.EntityShort[int64]
	character          *app.EntityShort[int32]
	region             *app.EntityShort[int32]
	solarSystem        *app.EntityShort[int32]
	systemSecurity     optional.Optional[float32]
	securityImportance widget.Importance
}

func (c jumpClone) systemSecurityDisplay() string {
	if c.systemSecurity.IsEmpty() {
		return "?"
	}
	return fmt.Sprintf("%.1f", c.systemSecurity.ValueOrZero())
}

type CloneSearch struct {
	widget.BaseWidget

	rows []jumpClone
	body fyne.CanvasObject
	top  *widget.Label
	u    app.UI
}

func NewCloneSearch(u app.UI) *CloneSearch {
	a := &CloneSearch{
		rows: make([]jumpClone, 0),
		top:  shared.MakeTopLabel(),
		u:    u,
	}
	a.ExtendBaseWidget(a)

	headers := []iwidget.HeaderDef{
		{Text: "Location", Width: 250},
		{Text: "System", Width: 150},
		{Text: "Sec.", Width: 50},
		{Text: "Region", Width: 150},
		{Text: "Character", Width: 200},
	}

	makeDataLabel := func(col int, r jumpClone) (string, fyne.TextAlign, widget.Importance) {
		var align fyne.TextAlign
		var importance widget.Importance
		var text string
		switch col {
		case 0:
			text = EntityNameOrFallback(r.location, "?")
		case 1:
			text = EntityNameOrFallback(r.solarSystem, "?")
		case 2:
			text = r.systemSecurityDisplay()
			importance = r.securityImportance
			align = fyne.TextAlignTrailing
		case 3:
			text = EntityNameOrFallback(r.region, "?")
		case 4:
			text = r.character.Name
		}
		return text, align, importance
	}
	if a.u.IsDesktop() {
		a.body = iwidget.MakeDataTableForDesktop(headers, &a.rows, makeDataLabel, func(c int, r jumpClone) {
			switch c {
			case 0:
				a.u.ShowLocationInfoWindow(r.location.ID)
			case 1:
				a.u.ShowInfoWindow(app.EveEntitySolarSystem, r.solarSystem.ID)
			case 4:
				a.u.ShowInfoWindow(app.EveEntityCharacter, r.character.ID)
			}
		})
	} else {
		a.body = iwidget.MakeDataTableForMobile(headers, &a.rows, makeDataLabel, func(r jumpClone) {
			a.u.ShowLocationInfoWindow(r.location.ID)
		})
	}
	return a
}

func (a *CloneSearch) CreateRenderer() fyne.WidgetRenderer {
	top := container.NewVBox(a.top, widget.NewSeparator())
	c := container.NewBorder(top, nil, nil, nil, a.body)
	return widget.NewSimpleRenderer(c)
}

func (a *CloneSearch) Update() {
	t, i, err := func() (string, widget.Importance, error) {
		locations, err := a.updateRows()
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

func (a *CloneSearch) updateRows() (int, error) {
	var err error
	oo, err := a.u.CharacterService().ListAllCharacterJumpClones(context.Background())
	if err != nil {
		return 0, err
	}
	locationIDs := set.New[int64]()
	rows := make([]jumpClone, len(oo))
	for i, o := range oo {
		c := jumpClone{
			id:        o.ID,
			character: o.Character,
		}
		if o.Location != nil {
			locationIDs.Add(o.Location.ID)
			c.location = &app.EntityShort[int64]{
				ID:   o.Location.ID,
				Name: o.Location.Name,
			}
			if o.Location.SolarSystem != nil {
				c.solarSystem = &app.EntityShort[int32]{
					ID:   o.Location.SolarSystem.ID,
					Name: o.Location.SolarSystem.Name,
				}
				c.systemSecurity = optional.New(o.Location.SolarSystem.SecurityStatus)
				c.securityImportance = o.Location.SolarSystem.SecurityType().ToImportance()
				c.region = &app.EntityShort[int32]{
					ID:   o.Location.SolarSystem.Constellation.Region.ID,
					Name: o.Location.SolarSystem.Constellation.Region.Name,
				}
			}
		}
		rows[i] = c
	}
	slices.SortFunc(rows, func(a, b jumpClone) int {
		return cmp.Compare(a.solarSystem.Name, b.solarSystem.Name)
	})
	a.rows = rows
	return locationIDs.Size(), nil
}
