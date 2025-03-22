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
	"github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type cloneSearchRow struct {
	c     *app.CharacterJumpClone2
	jumps optional.Optional[int]
}

type CloneSearch struct {
	widget.BaseWidget

	body         fyne.CanvasObject
	changeOrigin *widget.Button
	originLabel  *widget.RichText
	origin       *app.EveSolarSystem
	rows         []cloneSearchRow
	top          *widget.Label
	u            app.UI
}

func NewCloneSearch(u app.UI) *CloneSearch {
	a := &CloneSearch{
		rows:        make([]cloneSearchRow, 0),
		top:         shared.MakeTopLabel(),
		originLabel: widget.NewRichText(),
		changeOrigin: widget.NewButton("Change", func() {

		}),
		u: u,
	}
	a.ExtendBaseWidget(a)

	headers := []iwidget.HeaderDef{
		{Text: "Location", Width: 350},
		{Text: "Region", Width: 150},
		{Text: "Character", Width: 200},
		{Text: "Jumps", Width: 50},
	}

	makeCell := func(col int, r cloneSearchRow) []widget.RichTextSegment {
		s := make([]widget.RichTextSegment, 0)
		switch col {
		case 0:
			if r.c.Location.SolarSystem != nil {
				s = append(s, r.c.Location.SolarSystem.SecurityStatusRichText(true))
				s = append(s, NewRichTextSegmentFromText("  "+r.c.Location.DisplayName()))
			}
		case 1:
			if r.c.Location.SolarSystem != nil {
				s = append(s, NewRichTextSegmentFromText(r.c.Location.SolarSystem.Constellation.Region.Name))
			}
		case 2:
			s = append(s, NewRichTextSegmentFromText(r.c.Character.Name))
		case 3:
			s = append(s, NewRichTextSegmentFromText(humanize.Optional(r.jumps, "?")))
		}
		return s
	}
	if a.u.IsDesktop() {
		a.body = iwidget.MakeDataTableForDesktop2(headers, &a.rows, makeCell, func(c int, r cloneSearchRow) {
			switch c {
			case 0:
				a.u.ShowLocationInfoWindow(r.c.Location.ID)
			case 1:
				if r.c.Location.SolarSystem != nil {
					a.u.ShowInfoWindow(app.EveEntityRegion, r.c.Location.SolarSystem.Constellation.Region.ID)
				}
			case 2:
				a.u.ShowInfoWindow(app.EveEntityCharacter, r.c.Character.ID)
			}
		})
	} else {
		a.body = iwidget.MakeDataTableForMobile2(headers, &a.rows, makeCell, func(r cloneSearchRow) {
			a.u.ShowLocationInfoWindow(r.c.Location.ID)
		})
	}
	return a
}

func (a *CloneSearch) CreateRenderer() fyne.WidgetRenderer {
	top := container.NewVBox(
		a.top,
		widget.NewSeparator(),
		container.NewBorder(nil, nil, nil, a.changeOrigin, a.originLabel),
	)
	c := container.NewBorder(top, nil, nil, nil, a.body)
	return widget.NewSimpleRenderer(c)
}

func (a *CloneSearch) Update() {
	t, i, err := func() (string, widget.Importance, error) {
		err := a.updateData()
		if err != nil {
			return "", 0, err
		}
		if len(a.rows) == 0 {
			return "No clones", widget.LowImportance, nil
		}
		s := fmt.Sprintf("%d clones", len(a.rows))
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

func (a *CloneSearch) updateData() error {
	oo, err := a.u.CharacterService().ListAllCharacterJumpClones(context.Background())
	if err != nil {
		return err
	}
	slices.SortFunc(oo, func(a, b *app.CharacterJumpClone2) int {
		return cmp.Compare(a.SolarSystemName(), b.SolarSystemName())
	})
	rows := make([]cloneSearchRow, len(oo))
	for i, o := range oo {
		rows[i] = cloneSearchRow{
			c: o,
		}
	}
	a.rows = rows
	system, err := a.u.EveUniverseService().GetOrCreateSolarSystemESI(context.Background(), 30002537)
	if err != nil {
		return err
	}
	a.origin = system
	iwidget.SetRichText(a.originLabel, NewRichTextSegmentFromText(system.Name))
	return nil
}

func NewRichTextSegmentFromText(s string) widget.RichTextSegment {
	return &widget.TextSegment{Text: s}
}
