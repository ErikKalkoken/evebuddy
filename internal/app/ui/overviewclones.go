package ui

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxlayout "github.com/ErikKalkoken/fyne-kx/layout"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

type cloneRow struct {
	c     *app.CharacterJumpClone2
	route []*app.EveSolarSystem
}

func (r cloneRow) compare(other cloneRow) int {
	return cmp.Compare(r.sortValue(), other.sortValue())
}

func (r cloneRow) sortValue() int {
	if r.route == nil {
		return 10_000
	}
	if len(r.route) == 0 {
		return 10_000_000
	}
	return len(r.route) - 1
}

func (r cloneRow) jumps() string {
	if r.route == nil {
		return "?"
	}
	if len(r.route) == 0 {
		return "No route"
	}
	return fmt.Sprint(len(r.route) - 1)
}

type overviewClones struct {
	widget.BaseWidget

	body              fyne.CanvasObject
	changeOrigin      *widget.Button
	columnSorter      *columnSorter
	origin            *app.EveSolarSystem
	originLabel       *widget.RichText
	routePref         app.RoutePreference
	rows              []cloneRow
	rowsFiltered      []cloneRow
	sortButton        *sortButton
	top               *widget.Label
	u                 *BaseUI
	selectOwner       *selectFilter
	selectRegion      *selectFilter
	selectSolarSystem *selectFilter
}

func newOverviewClones(u *BaseUI) *overviewClones {
	headers := []headerDef{
		{Text: "Location", Width: columnWidthLocation},
		{Text: "Region", Width: columnWidthRegion, NotSortable: true},
		{Text: "Impl.", Width: 100},
		{Text: "Character", Width: columnWidthCharacter},
		{Text: "Jumps", Width: 100},
	}
	a := &overviewClones{
		columnSorter: newColumnSorter(headers),
		originLabel:  widget.NewRichTextWithText("(not set)"),
		rows:         make([]cloneRow, 0),
		rowsFiltered: make([]cloneRow, 0),
		top:          makeTopLabel(),
		u:            u,
	}
	a.ExtendBaseWidget(a)
	a.originLabel.Wrapping = fyne.TextWrapWord
	a.changeOrigin = widget.NewButton("Route from", func() {
		a.setOrigin(a.u.MainWindow())
	})
	makeCell := func(col int, r cloneRow) []widget.RichTextSegment {
		var s []widget.RichTextSegment
		switch col {
		case 0:
			s = r.c.Location.DisplayRichText()
		case 1:
			if r.c.Location.SolarSystem != nil {
				s = iwidget.NewRichTextSegmentFromText(r.c.Location.SolarSystem.Constellation.Region.Name)
			}
		case 2:
			s = iwidget.NewRichTextSegmentFromText(fmt.Sprint(r.c.ImplantsCount))
		case 3:
			s = iwidget.NewRichTextSegmentFromText(r.c.Character.Name)
		case 4:
			s = iwidget.NewRichTextSegmentFromText(r.jumps())
		}
		return s
	}
	if a.u.isDesktop {
		a.body = makeDataTable(headers, &a.rowsFiltered, makeCell, a.columnSorter, a.filterRows, func(c int, r cloneRow) {
			switch c {
			case 0:
				a.u.ShowLocationInfoWindow(r.c.Location.ID)
			case 1:
				if r.c.Location.SolarSystem != nil {
					a.u.ShowInfoWindow(app.EveEntityRegion, r.c.Location.SolarSystem.Constellation.Region.ID)
				}
			case 2:
				if r.c.ImplantsCount == 0 {
					return
				}
				a.showClone(r)
			case 3:
				a.u.ShowInfoWindow(app.EveEntityCharacter, r.c.Character.ID)
			case 4:
				if len(r.route) == 0 {
					return
				}
				a.showRoute(r)
			}
		})
	} else {
		a.body = makeDataList(headers, &a.rowsFiltered, makeCell, func(r cloneRow) {
			if len(r.route) == 0 {
				return
			}
			a.showRoute(r)
		})
	}

	a.selectRegion = newSelectFilter("Any region", func() {
		a.filterRows(-1)
	})

	a.selectSolarSystem = newSelectFilter("Any system", func() {
		a.filterRows(-1)
	})

	a.selectOwner = newSelectFilter("Any owner", func() {
		a.filterRows(-1)
	})

	a.sortButton = a.columnSorter.newSortButton(headers, func() {
		a.filterRows(-1)
	}, a.u.window)

	return a
}

func (a *overviewClones) CreateRenderer() fyne.WidgetRenderer {
	route := container.NewBorder(
		nil,
		nil,
		a.changeOrigin,
		nil,
		a.originLabel,
	)
	filters := container.NewHBox(a.selectRegion, a.selectSolarSystem, a.selectOwner)
	if !a.u.isDesktop {
		filters.Add(a.sortButton)
	}
	c := container.NewBorder(
		container.NewVBox(
			a.top,
			route,
			container.NewHScroll(filters),
		),
		nil,
		nil,
		nil,
		a.body,
	)
	return widget.NewSimpleRenderer(c)
}

func (a *overviewClones) update() {
	rows := make([]cloneRow, 0)
	t, i, err := func() (string, widget.Importance, error) {
		rows2, err := a.fetchRows(a.u.services())
		if err != nil {
			return "", 0, err
		}
		if len(rows2) == 0 {
			return "No clones", widget.LowImportance, nil
		}
		rows = rows2
		s := fmt.Sprintf("%d clones", len(rows2))
		return s, widget.MediumImportance, nil
	}()
	if err != nil {
		slog.Error("Failed to refresh clones UI", "err", err)
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
		a.filterRows(-1)
		if len(rows) > 0 && a.origin != nil {
			go a.updateRoutes()
		}
	})
}

func (*overviewClones) fetchRows(s services) ([]cloneRow, error) {
	ctx := context.Background()
	oo, err := s.cs.ListAllJumpClones(ctx)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(oo, func(a, b *app.CharacterJumpClone2) int {
		return cmp.Compare(a.SolarSystemName(), b.SolarSystemName())
	})
	rows := xslices.Map(oo, func(o *app.CharacterJumpClone2) cloneRow {
		return cloneRow{c: o}
	})
	return rows, nil
}

func (a *overviewClones) updateRoutes() {
	if a.origin == nil {
		return
	}
	fyne.Do(func() {
		for i := range a.rows {
			a.rows[i].route = nil
		}
		a.body.Refresh()
	})
	ctx := context.Background()
	wg := new(sync.WaitGroup)
	for i, o := range a.rows {
		wg.Add(1)
		go func() {
			defer wg.Done()
			dest := o.c.Location.SolarSystem
			origin := a.origin
			j, err := a.u.eus.FetchRoute(ctx, dest, origin, a.routePref)
			if err != nil {
				slog.Error("Failed to get route", "origin", origin.ID, "destination", dest.ID, "error", err)
				return
			}
			fyne.Do(func() {
				a.rows[i].route = j
				a.body.Refresh()
			})
		}()
	}
	wg.Wait()
	slices.SortFunc(a.rows, func(a, b cloneRow) int {
		return a.compare(b)
	})
	fyne.Do(func() {
		a.columnSorter.set(4, sortAsc)
		a.filterRows(-1)
	})
}

func (a *overviewClones) setOrigin(w fyne.Window) {
	showErrorDialog := func(search string, err error) {
		slog.Error("Failed to resolve names", "search", search, "error", err)
		a.u.ShowErrorDialog("Something went wrong", err, w)
	}
	var d dialog.Dialog
	results := make([]*app.EveEntity, 0)
	routePref := widget.NewSelect(
		xslices.Map(app.RoutePreferences(), func(a app.RoutePreference) string {
			return a.String()
		}), nil,
	)
	routePref.Selected = app.RouteShortest.String()
	list := widget.NewList(
		func() int {
			return len(results)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(results) {
				return
			}
			o := results[id]
			co.(*widget.Label).SetText(o.Name)
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		if id >= len(results) {
			return
		}
		r := results[id]
		s, err := a.u.eus.GetOrCreateSolarSystemESI(context.Background(), r.ID)
		if err != nil {
			showErrorDialog("Could not load solar system", err)
			return
		}
		a.origin = s
		a.routePref = app.RoutePreference(routePref.Selected)
		a.originLabel.Segments = iwidget.InlineRichTextSegments(
			s.DisplayRichTextWithRegion(),
			iwidget.NewRichTextSegmentFromText(fmt.Sprintf(" [%s]", a.routePref.String())),
		)
		a.originLabel.Refresh()
		go a.updateRoutes()
		d.Hide()
	}
	list.HideSeparators = true
	entry := widget.NewEntry()
	entry.PlaceHolder = "Type to start searching..."
	entry.ActionItem = iwidget.NewIconButton(theme.CancelIcon(), func() {
		entry.SetText("")
	})
	entry.OnChanged = func(search string) {
		if len(search) < 3 {
			results = results[:0]
			list.Refresh()
			return
		}
		go func() {
			ctx := context.Background()
			ee, _, err := a.u.cs.SearchESI(
				ctx,
				a.u.currentCharacterID(),
				search,
				[]app.SearchCategory{app.SearchSolarSystem},
				false,
			)
			if err != nil {
				fyne.Do(func() {
					showErrorDialog(search, err)
				})
				return
			}
			results = ee[app.SearchSolarSystem]
			slices.SortFunc(results, func(a, b *app.EveEntity) int {
				return a.Compare(b)
			})
			fyne.Do(func() {
				list.Refresh()
			})
		}()
	}
	note := widget.NewLabel("Select solar system from results list to change origin.")
	note.Importance = widget.LowImportance
	c := container.NewBorder(
		container.NewBorder(
			container.NewHBox(widget.NewLabel("Route preference:"), routePref),
			nil,
			nil,
			widget.NewButton("Cancel", func() {
				d.Hide()
			}),
			entry,
		),
		note,
		nil,
		nil,
		list,
	)
	d = dialog.NewCustomWithoutButtons("Change origin", c, w)
	d.Resize(fyne.NewSize(600, 400))
	d.Show()
	w.Canvas().Focus(entry)
}

func (a *overviewClones) filterRows(sortCol int) {
	rows := slices.Clone(a.rows)
	// filter
	a.selectOwner.applyFilter(func(selected string) {
		rows = xslices.Filter(rows, func(o cloneRow) bool {
			return o.c.CharacterName() == selected
		})
	})
	a.selectRegion.applyFilter(func(selected string) {
		rows = xslices.Filter(rows, func(o cloneRow) bool {
			return o.c.RegionName() == selected
		})
	})
	a.selectSolarSystem.applyFilter(func(selected string) {
		rows = xslices.Filter(rows, func(o cloneRow) bool {
			return o.c.SolarSystemName() == selected
		})
	})

	// sort
	a.columnSorter.sort(sortCol, func(sortCol int, dir sortDir) {
		slices.SortFunc(rows, func(a, b cloneRow) int {
			var x int
			switch sortCol {
			case 0:
				x = cmp.Compare(a.c.Location.DisplayName(), b.c.Location.DisplayName())
			case 1:
				x = cmp.Compare(
					a.c.Location.SolarSystem.Constellation.Region.Name,
					b.c.Location.SolarSystem.Constellation.Region.Name)
			case 2:
				x = cmp.Compare(a.c.ImplantsCount, b.c.ImplantsCount)
			case 3:
				x = cmp.Compare(a.c.Character.Name, b.c.Character.Name)
			case 4:
				x = cmp.Compare(a.sortValue(), b.sortValue())
			}
			if dir == sortAsc {
				return x
			} else {
				return -1 * x
			}
		})
	})
	a.selectOwner.setOptions(xiter.MapSlice(rows, func(o cloneRow) string {
		return o.c.CharacterName()
	}))
	a.selectRegion.setOptions(xiter.MapSlice(rows, func(o cloneRow) string {
		return o.c.RegionName()
	}))
	a.selectSolarSystem.setOptions(xiter.MapSlice(rows, func(o cloneRow) string {
		return o.c.SolarSystemName()
	}))
	a.rowsFiltered = rows
	a.body.Refresh()
}

func (a *overviewClones) showRoute(r cloneRow) {
	col := kxlayout.NewColumns(60)
	list := widget.NewList(
		func() int {
			return len(r.route)
		},
		func() fyne.CanvasObject {
			return container.New(col, widget.NewLabel(""), widget.NewRichText())
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(r.route) {
				return
			}
			s := r.route[id]
			border := co.(*fyne.Container).Objects
			num := border[0].(*widget.Label)
			num.SetText(fmt.Sprint(id))
			name := border[1].(*widget.RichText)
			name.Segments = s.DisplayRichTextWithRegion()
			name.Refresh()
		},
	)
	list.HideSeparators = true
	list.OnSelected = func(id widget.ListItemID) {
		defer list.UnselectAll()
		if id >= len(r.route) {
			return
		}
		s := r.route[id]
		a.u.ShowInfoWindow(app.EveEntitySolarSystem, s.ID)

	}
	from := iwidget.NewTappableRichText(
		func() {
			a.u.ShowInfoWindow(app.EveEntitySolarSystem, a.origin.ID)
		},
		a.origin.DisplayRichTextWithRegion()...)
	from.Wrapping = fyne.TextWrapWord
	to := iwidget.NewTappableRichText(
		func() {
			a.u.ShowInfoWindow(app.EveEntitySolarSystem, r.c.Location.SolarSystem.ID)
		},
		r.c.Location.SolarSystem.DisplayRichTextWithRegion()...)
	to.Wrapping = fyne.TextWrapWord
	jumps := widget.NewLabel(fmt.Sprintf("%s (%s)", r.jumps(), a.routePref.String()))
	top := container.New(
		layout.NewCustomPaddedVBoxLayout(0),
		container.New(
			col,
			widget.NewLabelWithStyle("From", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			from,
		),
		container.New(
			col,
			widget.NewLabelWithStyle("To", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			to,
		),
		container.New(
			col,
			widget.NewLabelWithStyle("Jumps", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			jumps,
		),
	)
	c := container.NewBorder(
		container.NewVBox(top, widget.NewSeparator()),
		nil,
		nil,
		nil,
		list,
	)
	title := fmt.Sprintf("Route: %s -> %s", a.origin.Name, r.c.Location.SolarSystem.Name)
	w := a.u.App().NewWindow(a.u.MakeWindowTitle(title))
	w.SetContent(c)
	w.Resize(fyne.NewSize(600, 400))
	w.Show()
}

func (a *overviewClones) showClone(r cloneRow) {
	clone, err := a.u.cs.GetJumpClone(context.Background(), r.c.Character.ID, r.c.CloneID)
	if err != nil {
		slog.Error("show clone", "error", err)
		a.u.ShowErrorDialog("failed to load clone", err, a.u.MainWindow())
		return
	}
	list := widget.NewList(
		func() int {
			return len(clone.Implants)
		},
		func() fyne.CanvasObject {
			icon := iwidget.NewImageFromResource(
				icons.Characterplaceholder64Jpeg,
				fyne.NewSquareSize(app.IconUnitSize),
			)
			name := widget.NewLabel("")
			name.Truncation = fyne.TextTruncateEllipsis
			return container.NewBorder(
				nil,
				nil,
				icon,
				nil,
				name,
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(clone.Implants) {
				return
			}
			im := clone.Implants[id]
			border := co.(*fyne.Container).Objects
			icon := border[1].(*canvas.Image)
			iwidget.RefreshImageAsync(icon, func() (fyne.Resource, error) {
				return a.u.eis.InventoryTypeIcon(im.EveType.ID, app.IconPixelSize)
			})
			name := border[0]
			name.(*widget.Label).SetText(im.EveType.Name)
		},
	)
	list.HideSeparators = true
	list.OnSelected = func(id widget.ListItemID) {
		defer list.UnselectAll()
		if id >= len(clone.Implants) {
			return
		}
		im := clone.Implants[id]
		a.u.ShowInfoWindow(app.EveEntityInventoryType, im.EveType.ID)

	}
	location := iwidget.NewTappableRichText(
		func() {
			a.u.ShowLocationInfoWindow(r.c.Location.ID)
		},
		r.c.Location.DisplayRichText()...)
	location.Wrapping = fyne.TextWrapWord
	character := kxwidget.NewTappableLabel(r.c.Character.Name, func() {
		a.u.ShowInfoWindow(app.EveEntityCharacter, r.c.Character.ID)
	})
	character.Wrapping = fyne.TextWrapWord
	implants := widget.NewLabel(fmt.Sprint(len(clone.Implants)))
	col := kxlayout.NewColumns(80)
	top := container.New(
		layout.NewCustomPaddedVBoxLayout(0),
		container.New(
			col,
			widget.NewLabelWithStyle("Location", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			location,
		),
		container.New(
			col,
			widget.NewLabelWithStyle("Character", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			character,
		),
		container.New(
			col,
			widget.NewLabelWithStyle("Implants", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			implants,
		),
	)
	c := container.NewBorder(
		container.NewVBox(top, widget.NewSeparator()),
		nil,
		nil,
		nil,
		list,
	)
	w := a.u.App().NewWindow(a.u.MakeWindowTitle("Clone"))
	w.SetContent(c)
	w.Resize(fyne.NewSize(600, 400))
	w.Show()
}
