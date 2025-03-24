package collectivewidget

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

type LocationOverview struct {
	widget.BaseWidget

	rows []*app.Character
	body fyne.CanvasObject
	top  *widget.Label
	u    app.UI
}

func NewLocationOverview(u app.UI) *LocationOverview {
	a := &LocationOverview{
		rows: make([]*app.Character, 0),
		top:  appwidget.MakeTopLabel(),
		u:    u,
	}
	a.ExtendBaseWidget(a)

	headers := []iwidget.HeaderDef{
		{Text: "Character", Width: characterColumnWidth},
		{Text: "Location", Width: locationColumnWidth},
		{Text: "Region", Width: regionColumnWidth},
		{Text: "Ship", Width: 150},
	}

	makeCell := func(col int, r *app.Character) []widget.RichTextSegment {
		switch col {
		case 0:
			return iwidget.NewRichTextSegmentFromText(r.EveCharacter.Name)
		case 1:
			return r.Location.DisplayRichText()
		case 2:
			if r.Location != nil {
				return iwidget.NewRichTextSegmentFromText(r.Location.SolarSystem.Constellation.Region.Name)
			}
		case 3:
			if r.Ship != nil {
				return iwidget.NewRichTextSegmentFromText(r.Ship.Name)
			}
		}
		return iwidget.NewRichTextSegmentFromText("?")

	}
	if a.u.IsDesktop() {
		a.body = iwidget.MakeDataTableForDesktop2(headers, &a.rows, makeCell, func(c int, r *app.Character) {
			switch c {
			case 0:
				a.u.ShowInfoWindow(app.EveEntityCharacter, r.ID)
			case 1:
				if r.Location != nil {
					a.u.ShowLocationInfoWindow(r.Location.ID)
				}
			case 2:
				if r.Location != nil {
					a.u.ShowInfoWindow(app.EveEntityRegion, r.Location.SolarSystem.Constellation.Region.ID)
				}
			case 3:
				if r.Ship != nil {
					a.u.ShowTypeInfoWindow(r.Ship.ID)
				}
			}
		})
	} else {
		a.body = iwidget.MakeDataTableForMobile2(headers, &a.rows, makeCell, func(r *app.Character) {
			if r.Location != nil {
				a.u.ShowLocationInfoWindow(r.Location.ID)
			}
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
	a.rows = mycc
	locationIDs := set.NewFromSlice(slices.Collect(xiter.MapSlice(mycc, func(x *app.Character) int64 {
		if x.Location != nil {
			return x.Location.ID
		}
		return 0
	})))
	locationIDs.Remove(0)
	return locationIDs.Size(), nil
}
