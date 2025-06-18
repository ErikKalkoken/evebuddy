package ui

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"

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
	"github.com/ErikKalkoken/evebuddy/internal/set"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

type cloneRow struct {
	jc       *app.CharacterJumpClone2
	route    []*app.EveSolarSystem
	routeErr error
	tags     set.Set[string]
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

type clones struct {
	widget.BaseWidget

	body              fyne.CanvasObject
	changeOrigin      *widget.Button
	columnSorter      *columnSorter
	origin            *app.EveSolarSystem
	originLabel       *iwidget.RichText
	routePref         app.EveRoutePreference
	rows              []cloneRow
	rowsFiltered      []cloneRow
	selectOwner       *kxwidget.FilterChipSelect
	selectRegion      *kxwidget.FilterChipSelect
	selectSolarSystem *kxwidget.FilterChipSelect
	selectTag         *kxwidget.FilterChipSelect
	sortButton        *sortButton
	top               *widget.Label
	u                 *baseUI
}

func newClones(u *baseUI) *clones {
	headers := []headerDef{
		{Label: "Location", Width: columnWidthLocation},
		{Label: "Region", Width: columnWidthRegion, NotSortable: true},
		{Label: "Impl.", Width: 100},
		{Label: "Character", Width: columnWidthCharacter},
		{Label: "Jumps", Width: 100},
	}
	a := &clones{
		columnSorter: newColumnSorter(headers),
		originLabel:  iwidget.NewRichTextWithText("(not set)"),
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
			s = r.jc.Location.DisplayRichText()
		case 1:
			s = iwidget.NewRichTextSegmentFromText(r.jc.Location.RegionName())
		case 2:
			s = iwidget.NewRichTextSegmentFromText(fmt.Sprint(r.jc.ImplantsCount))
		case 3:
			s = iwidget.NewRichTextSegmentFromText(r.jc.Character.Name)
		case 4:
			s = iwidget.NewRichTextSegmentFromText(r.jumps())
		}
		return s
	}
	if a.u.isDesktop {
		a.body = makeDataTable(headers, &a.rowsFiltered, makeCell, a.columnSorter, a.filterRows, func(c int, r cloneRow) {
			switch c {
			case 0:
				a.u.ShowLocationInfoWindow(r.jc.Location.ID)
			case 1:
				if r.jc.Location.SolarSystem != nil {
					a.u.ShowInfoWindow(app.EveEntityRegion, r.jc.Location.SolarSystem.Constellation.Region.ID)
				}
			case 2:
				if r.jc.ImplantsCount == 0 {
					return
				}
				a.showClone(r)
			case 3:
				a.u.ShowInfoWindow(app.EveEntityCharacter, r.jc.Character.ID)
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

	a.selectRegion = kxwidget.NewFilterChipSelectWithSearch("Region", []string{}, func(string) {
		a.filterRows(-1)
	}, a.u.window)

	a.selectSolarSystem = kxwidget.NewFilterChipSelectWithSearch("System", []string{}, func(string) {
		a.filterRows(-1)
	}, a.u.window)

	a.selectOwner = kxwidget.NewFilterChipSelect("Owner", []string{}, func(string) {
		a.filterRows(-1)
	})
	a.selectTag = kxwidget.NewFilterChipSelect("Tag", []string{}, func(string) {
		a.filterRows(-1)
	})
	a.sortButton = a.columnSorter.newSortButton(headers, func() {
		a.filterRows(-1)
	}, a.u.window)

	return a
}

func (a *clones) CreateRenderer() fyne.WidgetRenderer {
	route := container.NewBorder(
		nil,
		nil,
		a.changeOrigin,
		nil,
		a.originLabel,
	)
	filters := container.NewHBox(
		a.selectRegion,
		a.selectSolarSystem,
		a.selectOwner,
		a.selectTag,
	)
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

func (a *clones) filterRows(sortCol int) {
	rows := slices.Clone(a.rows)
	// filter
	if x := a.selectOwner.Selected; x != "" {
		rows = xslices.Filter(rows, func(r cloneRow) bool {
			return r.jc.Character.Name == x
		})
	}
	if x := a.selectRegion.Selected; x != "" {
		rows = xslices.Filter(rows, func(r cloneRow) bool {
			return r.jc.Location.RegionName() == x
		})
	}
	if x := a.selectSolarSystem.Selected; x != "" {
		rows = xslices.Filter(rows, func(r cloneRow) bool {
			return r.jc.Location.SolarSystemName() == x
		})
	}
	if x := a.selectTag.Selected; x != "" {
		rows = xslices.Filter(rows, func(r cloneRow) bool {
			return r.tags.Contains(x)
		})
	}
	// sort
	a.columnSorter.sort(sortCol, func(sortCol int, dir sortDir) {
		slices.SortFunc(rows, func(a, b cloneRow) int {
			var x int
			switch sortCol {
			case 0:
				x = cmp.Compare(a.jc.Location.DisplayName(), b.jc.Location.DisplayName())
			case 1:
				x = cmp.Compare(a.jc.Location.RegionName(), b.jc.Location.RegionName())
			case 2:
				x = cmp.Compare(a.jc.ImplantsCount, b.jc.ImplantsCount)
			case 3:
				x = cmp.Compare(a.jc.Character.Name, b.jc.Character.Name)
			case 4:
				x = a.compare(b)
			}
			if dir == sortAsc {
				return x
			} else {
				return -1 * x
			}
		})
	})
	// set data & refresh
	a.selectTag.SetOptions(slices.Sorted(set.Union(xslices.Map(rows, func(r cloneRow) set.Set[string] {
		return r.tags
	})...).All()))
	a.selectOwner.SetOptions(xslices.Map(rows, func(r cloneRow) string {
		return r.jc.Character.Name
	}))
	a.selectRegion.SetOptions(xslices.Map(rows, func(r cloneRow) string {
		return r.jc.Location.RegionName()
	}))
	a.selectSolarSystem.SetOptions(xslices.Map(rows, func(r cloneRow) string {
		return r.jc.Location.SolarSystemName()
	}))
	a.rowsFiltered = rows
	a.body.Refresh()
}

func (a *clones) update() {
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

func (*clones) fetchRows(s services) ([]cloneRow, error) {
	ctx := context.Background()
	oo, err := s.cs.ListAllJumpClones(ctx)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(oo, func(a, b *app.CharacterJumpClone2) int {
		return cmp.Compare(a.Location.SolarSystemName(), b.Location.SolarSystemName())
	})
	rows := make([]cloneRow, 0)
	for _, o := range oo {
		r := cloneRow{jc: o}
		tags, err := s.cs.ListTagsForCharacter(ctx, o.Character.ID)
		if err != nil {
			return nil, err
		}
		r.tags = set.Collect(xiter.MapSlice(tags, func(x *app.CharacterTag) string {
			return x.Name
		}))
		rows = append(rows, r)
	}
	return rows, nil
}

func (a *clones) updateRoutes() {
	fyne.Do(func() {
		for i := range a.rows {
			a.rows[i].route = nil
		}
		a.body.Refresh()
	})
	headers := make([]app.EveRouteHeader, 0)
	fyne.DoAndWait(func() {
		for _, r := range a.rows {
			destination := r.jc.Location.SolarSystem
			if destination == nil {
				continue
			}
			headers = append(headers, app.EveRouteHeader{
				Origin:      a.origin,
				Destination: destination,
				Preference:  a.routePref,
			})
		}
	})
	routes, err := a.u.eus.FetchRoutes(context.Background(), headers)
	if err != nil {
		slog.Error("failed to fetch routes", "error", err)
		fyne.Do(func() {
			s := "Failed to fetch routes: " + a.u.humanizeError(err)
			a.originLabel.Set(iwidget.NewRichTextSegmentFromText(s, widget.RichTextStyle{
				ColorName: theme.ColorNameError,
			}))
		})
		return
	}
	m := make(map[int32][]*app.EveSolarSystem)
	for h, route := range routes {
		m[h.Destination.ID] = route
	}
	fyne.Do(func() {
		for i, r := range a.rows {
			solarSystem := r.jc.Location.SolarSystem
			if solarSystem == nil {
				continue
			}
			a.rows[i].route = m[solarSystem.ID]
		}
		a.body.Refresh()
		a.columnSorter.set(4, sortOff)
		a.filterRows(4)
	})
}

func (a *clones) setOrigin(w fyne.Window) {
	showErrorDialog := func(search string, err error) {
		slog.Error("Failed to resolve names", "search", search, "error", err)
		a.u.showErrorDialog("Something went wrong", err, w)
	}
	var d dialog.Dialog
	results := make([]*app.EveEntity, 0)
	routePref := widget.NewSelect(
		xslices.Map(app.EveRoutePreferences(), func(a app.EveRoutePreference) string {
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
		a.routePref = app.EveRoutePreferenceFromString(routePref.Selected)
		a.originLabel.Set(iwidget.InlineRichTextSegments(
			s.DisplayRichTextWithRegion(),
			iwidget.NewRichTextSegmentFromText(fmt.Sprintf(" [%s]", a.routePref.String())),
		))
		go a.updateRoutes()
		d.Hide()
	}
	list.HideSeparators = true
	entry := widget.NewEntry()
	entry.PlaceHolder = "Type to start searching..."
	entry.ActionItem = kxwidget.NewIconButton(theme.CancelIcon(), func() {
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
			x := ee[app.SearchSolarSystem]
			slices.SortFunc(x, func(a, b *app.EveEntity) int {
				return a.Compare(b)
			})
			fyne.Do(func() {
				results = x
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
	_, s := w.Canvas().InteractiveArea()
	if !a.u.isDesktop {
		d.Resize(fyne.NewSize(s.Width, s.Height))
	} else {
		d.Resize(fyne.NewSize(600, max(400, s.Height*0.8)))
	}
	d.Show()
	w.Canvas().Focus(entry)
}

func (a *clones) showRoute(r cloneRow) {
	col := kxlayout.NewColumns(60)
	list := widget.NewList(
		func() int {
			return len(r.route)
		},
		func() fyne.CanvasObject {
			return container.New(col, widget.NewLabel(""), iwidget.NewRichText())
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(r.route) {
				return
			}
			s := r.route[id]
			border := co.(*fyne.Container).Objects
			num := border[0].(*widget.Label)
			num.SetText(fmt.Sprint(id))
			border[1].(*iwidget.RichText).Set(s.DisplayRichTextWithRegion())
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

	var fromText []widget.RichTextSegment
	if a.origin != nil {
		fromText = a.origin.DisplayRichTextWithRegion()
	}
	from := iwidget.NewTappableRichText(fromText, func() {
		if a.origin != nil {
			a.u.ShowInfoWindow(app.EveEntitySolarSystem, a.origin.ID)
		}
	})
	from.Wrapping = fyne.TextWrapWord

	var toText []widget.RichTextSegment
	if r.jc.Location.SolarSystem != nil {
		toText = r.jc.Location.SolarSystem.DisplayRichTextWithRegion()
	}
	to := iwidget.NewTappableRichText(toText, func() {
		if r.jc.Location.SolarSystem != nil {
			a.u.ShowInfoWindow(app.EveEntitySolarSystem, r.jc.Location.SolarSystem.ID)
		}
	})
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
	title := fmt.Sprintf("Route: %s -> %s", a.origin.Name, r.jc.Location.SolarSystemName())
	w := a.u.App().NewWindow(a.u.MakeWindowTitle(title))
	w.SetContent(c)
	w.Resize(fyne.NewSize(600, 400))
	w.Show()
}

func (a *clones) showClone(r cloneRow) {
	clone, err := a.u.cs.GetJumpClone(context.Background(), r.jc.Character.ID, r.jc.CloneID)
	if err != nil {
		slog.Error("show clone", "error", err)
		a.u.showErrorDialog("failed to load clone", err, a.u.MainWindow())
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

	location := makeLocationLabel(r.jc.Location.ToShort(), a.u.ShowLocationInfoWindow)
	character := makeLinkLabelWithWrap(r.jc.Character.Name, func() {
		a.u.ShowInfoWindow(app.EveEntityCharacter, r.jc.Character.ID)
	})
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
