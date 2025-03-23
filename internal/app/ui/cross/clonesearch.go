package cross

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	kxlayout "github.com/ErikKalkoken/fyne-kx/layout"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/shared"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

type cloneSearchRow struct {
	c     *app.CharacterJumpClone2
	route []*app.EveSolarSystem
}

func (r cloneSearchRow) Jumps() string {
	if r.route == nil {
		return "?"
	}
	if len(r.route) == 0 {
		return "None"
	}
	return fmt.Sprint(len(r.route) - 1)
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
		{Text: "Jumps", Width: 75},
	}

	makeCell := func(col int, r cloneSearchRow) []widget.RichTextSegment {
		var s []widget.RichTextSegment
		switch col {
		case 0:
			s = r.c.Location.DisplayRichText()
		case 1:
			if r.c.Location.SolarSystem != nil {
				s = iwidget.NewRichTextSegmentFromText(
					r.c.Location.SolarSystem.Constellation.Region.Name,
					false,
				)
			}
		case 2:
			s = iwidget.NewRichTextSegmentFromText(r.c.Character.Name, false)
		case 3:
			s = iwidget.NewRichTextSegmentFromText(r.Jumps(), false)
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
			case 3:
				if len(r.route) == 0 {
					return
				}
				list := widget.NewList(
					func() int {
						return len(r.route)
					},
					func() fyne.CanvasObject {
						return widget.NewRichText()
					},
					func(id widget.ListItemID, co fyne.CanvasObject) {
						if id >= len(r.route) {
							return
						}
						s := r.route[id]
						x := co.(*widget.RichText)
						x.Segments = s.DisplayRichText()
						x.Refresh()
					},
				)
				list.OnSelected = func(id widget.ListItemID) {
					defer list.UnselectAll()
					if id >= len(r.route) {
						return
					}
					s := r.route[id]
					a.u.ShowInfoWindow(app.EveEntitySolarSystem, s.ID)

				}
				col := kxlayout.NewColumns(50)
				from := widget.NewRichText(a.origin.DisplayRichText()...)
				from.Wrapping = fyne.TextWrapWord
				to := widget.NewRichText(r.c.Location.DisplayRichText()...)
				to.Wrapping = fyne.TextWrapWord
				top := container.New(
					layout.NewCustomPaddedVBoxLayout(0),
					container.New(col, widget.NewLabel("From"), from),
					container.New(col, widget.NewLabel("To"), to),
					container.New(col, widget.NewLabel("Jump"), widget.NewLabel(r.Jumps())),
				)
				c := container.NewBorder(top, nil, nil, nil, list)
				w := a.u.App().NewWindow(fmt.Sprintf("Route: %s -> %s", a.origin.Name, r.c.Location.SolarSystem.Name))
				w.SetContent(c)
				w.Resize(fyne.NewSize(600, 400))
				w.Show()
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
		err := a.updateRows()
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
	if len(a.rows) > 0 {
		go a.updateRoutes()
	}
}

func (a *CloneSearch) updateRows() error {
	ctx := context.Background()
	oo, err := a.u.CharacterService().ListAllCharacterJumpClones(ctx)
	if err != nil {
		return err
	}
	slices.SortFunc(oo, func(a, b *app.CharacterJumpClone2) int {
		return cmp.Compare(a.SolarSystemName(), b.SolarSystemName())
	})
	system, err := a.u.EveUniverseService().GetOrCreateSolarSystemESI(ctx, 30002537)
	if err != nil {
		return err
	}
	a.origin = system
	iwidget.SetRichText(a.originLabel, iwidget.NewRichTextSegmentFromText(system.Name, false)...)
	a.rows = slices.Collect(xiter.MapSlice(oo, func(o *app.CharacterJumpClone2) cloneSearchRow {
		return cloneSearchRow{c: o}
	}))
	return nil
}

func (a *CloneSearch) updateRoutes() {
	ctx := context.Background()
	wg := new(sync.WaitGroup)
	for i, o := range a.rows {
		wg.Add(1)
		go func() {
			defer wg.Done()
			j, err := a.u.EveUniverseService().GetRouteESI(ctx, o.c.Location.SolarSystem, a.origin)
			if err == nil {
				a.rows[i].route = j
				a.body.Refresh()
			}
		}()
	}
	wg.Wait()
}
