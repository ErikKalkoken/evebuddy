package collectivewidget

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
	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

type cloneSearchRow struct {
	c     *app.CharacterJumpClone2
	route []*app.EveSolarSystem
}

func (r cloneSearchRow) compare(other cloneSearchRow) int {
	return cmp.Compare(r.sortValue(), other.sortValue())
}

func (r cloneSearchRow) sortValue() int {
	if r.route == nil {
		return 10_000
	}
	if len(r.route) == 0 {
		return 10_000_000
	}
	return len(r.route) - 1
}

func (r cloneSearchRow) jumps() string {
	if r.route == nil {
		return "?"
	}
	if len(r.route) == 0 {
		return "No route"
	}
	return fmt.Sprint(len(r.route) - 1)
}

type CloneSearch struct {
	widget.BaseWidget

	body         fyne.CanvasObject
	originButton *widget.Button
	originLabel  *widget.RichText
	origin       *app.EveSolarSystem
	routePref    *widget.Select
	rows         []cloneSearchRow
	top          *widget.Label
	u            app.UI
	colSort      []sortDir
}

func NewCloneSearch(u app.UI) *CloneSearch {
	headers := []iwidget.HeaderDef{
		{Text: "Location", Width: 350},
		{Text: "Region", Width: 150},
		{Text: "Impl.", Width: 100},
		{Text: "Character", Width: 200},
		{Text: "Jumps", Width: 100},
	}
	a := &CloneSearch{
		colSort:     make([]sortDir, len(headers)),
		originLabel: widget.NewRichTextWithText("?"),
		rows:        make([]cloneSearchRow, 0),
		top:         appwidget.MakeTopLabel(),
		u:           u,
	}
	a.ExtendBaseWidget(a)
	a.originLabel.Wrapping = fyne.TextWrapWord
	a.originButton = widget.NewButton("Origin", func() {
		a.changeOrigin(a.u.MainWindow())
	})
	xx := slices.Collect(xiter.MapSlice(app.RoutePreferences(), func(a app.RoutePreference) string {
		return a.String()
	}))
	a.routePref = widget.NewSelect(xx, func(s string) {})
	a.routePref.Selected = app.RouteShortest.String()
	a.routePref.OnChanged = func(s string) {
		go a.updateRoutes(app.RoutePreference(s))
	}
	makeCell := func(col int, r cloneSearchRow) []widget.RichTextSegment {
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
	if a.u.IsDesktop() {
		t := iwidget.MakeDataTableForDesktop2(headers, &a.rows, makeCell, func(c int, r cloneSearchRow) {
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
		iconSortAsc := theme.NewPrimaryThemedResource(icons.SortAscendingSvg)
		iconSortDesc := theme.NewPrimaryThemedResource(icons.SortDescendingSvg)
		iconSortOff := theme.NewThemedResource(icons.SortSvg)
		t.CreateHeader = func() fyne.CanvasObject {
			icon := widget.NewIcon(iconSortOff)
			label := kxwidget.NewTappableLabel("XXX", nil)
			return container.NewBorder(nil, nil, nil, icon, label)
		}
		t.UpdateHeader = func(tci widget.TableCellID, co fyne.CanvasObject) {
			h := headers[tci.Col]
			row := co.(*fyne.Container).Objects
			label := row[0].(*kxwidget.TappableLabel)
			label.SetText(h.Text)
			label.OnTapped = func() {
				a.processData(tci.Col)
			}
			icon := row[1].(*widget.Icon)
			switch a.colSort[tci.Col] {
			case sortOff:
				icon.SetResource(iconSortOff)
			case sortAsc:
				icon.SetResource(iconSortAsc)
			case sortDesc:
				icon.SetResource(iconSortDesc)
			}
		}
		a.body = t
	} else {
		a.body = iwidget.MakeDataTableForMobile2(headers, &a.rows, makeCell, func(r cloneSearchRow) {
			if len(r.route) == 0 {
				return
			}
			a.showRoute(r)
		})
	}
	return a
}

func (a *CloneSearch) CreateRenderer() fyne.WidgetRenderer {
	var route *fyne.Container
	routeLabel := widget.NewLabelWithStyle("Route:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	if a.u.IsDesktop() {
		route = container.NewBorder(
			nil,
			nil,
			container.NewHBox(routeLabel, a.routePref, a.originButton),
			nil,
			a.originLabel,
		)
	} else {
		route = container.NewVBox(
			container.NewHBox(routeLabel, a.routePref),
			container.NewBorder(nil, nil, a.originButton, nil, a.originLabel),
		)
	}
	top := container.NewVBox(
		a.top,
		widget.NewSeparator(),
		route,
	)
	c := container.NewBorder(
		top,
		nil,
		nil,
		nil,
		a.body,
	)
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
	if len(a.rows) > 0 && a.origin != nil {
		go a.updateRoutes(app.RoutePreference(a.routePref.Selected))
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
	a.rows = slices.Collect(xiter.MapSlice(oo, func(o *app.CharacterJumpClone2) cloneSearchRow {
		return cloneSearchRow{c: o}
	}))
	return nil
}

func (a *CloneSearch) updateRoutes(flag app.RoutePreference) {
	if a.origin == nil {
		return
	}
	for i := range a.rows {
		a.rows[i].route = nil
	}
	a.body.Refresh()
	ctx := context.Background()
	wg := new(sync.WaitGroup)
	for i, o := range a.rows {
		wg.Add(1)
		go func() {
			defer wg.Done()
			dest := o.c.Location.SolarSystem
			origin := a.origin
			j, err := a.u.EveUniverseService().GetRouteESI(ctx, dest, origin, flag)
			if err != nil {
				slog.Error("Failed to get route", "origin", origin.ID, "destination", dest.ID, "error", err)
				return
			}
			a.rows[i].route = j
			a.body.Refresh()
		}()
	}
	wg.Wait()
	slices.SortFunc(a.rows, func(a, b cloneSearchRow) int {
		return a.compare(b)
	})
	for i := range a.colSort {
		a.colSort[i] = sortOff
	}
	a.colSort[4] = sortAsc
	a.body.Refresh()
}

func (a *CloneSearch) changeOrigin(w fyne.Window) {
	showErrorDialog := func(search string, err error) {
		slog.Error("Failed to resolve names", "search", search, "error", err)
		a.u.ShowErrorDialog("Something went wrong", err, w)
	}
	var d dialog.Dialog
	results := make([]*app.EveEntity, 0)
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
		s, err := a.u.EveUniverseService().GetOrCreateSolarSystemESI(context.Background(), r.ID)
		if err != nil {
			showErrorDialog("Could not load solar system", err)
			return
		}
		a.origin = s
		a.originLabel.Segments = s.DisplayRichTextWithRegion()
		a.originLabel.Refresh()
		go a.updateRoutes(app.RoutePreference(a.routePref.Selected))
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
			ee, _, err := a.u.CharacterService().SearchESI(
				ctx,
				a.u.CurrentCharacterID(),
				search,
				[]app.SearchCategory{app.SearchSolarSystem},
				false,
			)
			if err != nil {
				showErrorDialog(search, err)
				return
			}
			results = ee[app.SearchSolarSystem]
			slices.SortFunc(results, func(a, b *app.EveEntity) int {
				return a.Compare(b)
			})
			list.Refresh()
		}()
	}
	note := widget.NewLabel("Select solar system from results list to change origin.")
	note.Importance = widget.LowImportance
	c := container.NewBorder(
		container.NewBorder(
			nil,
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

func (a *CloneSearch) processData(sortCol int) {
	var order sortDir
	if sortCol >= 0 {
		order = a.colSort[sortCol]
		order++
		if order > sortDesc {
			order = sortOff
		}
		for i := range a.colSort {
			a.colSort[i] = sortOff
		}
		a.colSort[sortCol] = order
	} else {
		for i := range a.colSort {
			if a.colSort[i] != sortOff {
				order = a.colSort[i]
				sortCol = i
				break
			}
		}
	}
	if sortCol >= 0 && order != sortOff {
		slices.SortFunc(a.rows, func(a, b cloneSearchRow) int {
			var x int
			switch sortCol {
			case 0:
				x = cmp.Compare(a.c.Location.DisplayName(), b.c.Location.DisplayName())
			case 1:
				x = cmp.Compare(
					a.c.Location.SolarSystem.Constellation.Region.Name,
					b.c.Location.SolarSystem.Constellation.Region.Name)
			case 2:
				x = cmp.Compare(a.c.Character.Name, b.c.Character.Name)
			case 3:
				x = cmp.Compare(a.sortValue(), b.sortValue())
			}
			if order == sortAsc {
				return x
			} else {
				return -1 * x
			}
		})
	}
	a.body.Refresh()
}

func (a *CloneSearch) showRoute(r cloneSearchRow) {
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
	from := widget.NewRichText(a.origin.DisplayRichTextWithRegion()...)
	from.Wrapping = fyne.TextWrapWord
	to := widget.NewRichText(r.c.Location.SolarSystem.DisplayRichTextWithRegion()...)
	to.Wrapping = fyne.TextWrapWord
	jumps := widget.NewLabel(fmt.Sprintf("%s (%s)", r.jumps(), a.routePref.Selected))
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
	w := a.u.App().NewWindow(fmt.Sprintf("Route: %s -> %s", a.origin.Name, r.c.Location.SolarSystem.Name))
	w.SetContent(c)
	w.Resize(fyne.NewSize(600, 400))
	w.Show()
}

func (a *CloneSearch) showClone(r cloneSearchRow) {
	clone, err := a.u.CharacterService().GetCharacterJumpClone(context.Background(), r.c.Character.ID, r.c.CloneID)
	if err != nil {
		panic(err) // TODO
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
			appwidget.RefreshImageResourceAsync(icon, func() (fyne.Resource, error) {
				return a.u.EveImageService().InventoryTypeIcon(im.EveType.ID, app.IconPixelSize)
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
	location := widget.NewRichText(r.c.Location.DisplayRichText()...)
	location.Wrapping = fyne.TextWrapWord
	character := widget.NewLabel(r.c.Character.Name)
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
	w := a.u.App().NewWindow("Clone")
	w.SetContent(c)
	w.Resize(fyne.NewSize(600, 400))
	w.Show()
}
