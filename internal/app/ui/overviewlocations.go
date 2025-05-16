package ui

import (
	"context"
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

type OverviewLocations struct {
	widget.BaseWidget

	rows []*app.Character
	body fyne.CanvasObject
	top  *widget.Label
	u    *BaseUI
}

func NewOverviewLocations(u *BaseUI) *OverviewLocations {
	a := &OverviewLocations{
		rows: make([]*app.Character, 0),
		top:  appwidget.MakeTopLabel(),
		u:    u,
	}
	a.ExtendBaseWidget(a)

	headers := []headerDef{
		{Text: "Character", Width: columnWidthCharacter},
		{Text: "Location", Width: columnWidthLocation},
		{Text: "Region", Width: columnWidthRegion},
		{Text: "Ship", Width: 150},
	}

	makeCell := func(col int, r *app.Character) []widget.RichTextSegment {
		switch col {
		case 0:
			return iwidget.NewRichTextSegmentFromText(r.EveCharacter.Name)
		case 1:
			if r.Location != nil {
				return r.Location.DisplayRichText()
			}
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
	if a.u.isDesktop {
		a.body = makeDataTableForDesktop(headers, &a.rows, makeCell, func(c int, r *app.Character) {
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
		a.body = makeDataTableForMobile(headers, &a.rows, makeCell, func(r *app.Character) {
			if r.Location != nil {
				a.u.ShowLocationInfoWindow(r.Location.ID)
			}
		})
	}
	return a
}

func (a *OverviewLocations) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(a.top, nil, nil, nil, a.body)
	return widget.NewSimpleRenderer(c)
}

func (a *OverviewLocations) update() {
	rows := make([]*app.Character, 0)
	t, i, err := func() (string, widget.Importance, error) {
		cc, count, err := a.fetchRows(a.u.services())
		if err != nil {
			return "", 0, err
		}
		if len(cc) == 0 {
			return "No characters", widget.LowImportance, nil
		}
		rows = cc
		s := fmt.Sprintf("%d characters • %d locations", len(cc), count)
		return s, widget.MediumImportance, nil
	}()
	if err != nil {
		slog.Error("Failed to refresh locations UI", "err", err)
		t = "ERROR: " + a.u.humanizeError(err)
		i = widget.DangerImportance
	}
	fyne.Do(func() {
		a.top.Text = t
		a.top.Importance = i
		a.top.Refresh()
	})
	fyne.Do(func() {
		a.rows = rows
		a.body.Refresh()
	})
}

func (*OverviewLocations) fetchRows(s services) ([]*app.Character, int, error) {
	ctx := context.TODO()
	cc, err := s.cs.ListCharacters(ctx)
	if err != nil {
		return nil, 0, err
	}
	u := xslices.Map(cc, func(x *app.Character) int64 {
		if x.Location != nil {
			return x.Location.ID
		}
		return 0
	})
	locationIDs := set.Of(u...)
	locationIDs.Delete(0)
	return cc, locationIDs.Size(), nil
}
