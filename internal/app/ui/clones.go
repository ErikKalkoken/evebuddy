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
	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	awidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
	"github.com/ErikKalkoken/evebuddy/internal/app/xdialog"
	"github.com/ErikKalkoken/evebuddy/internal/xlayout"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type cloneRow struct {
	jc    *app.CharacterJumpClone2
	route []*app.EveSolarSystem
	tags  set.Set[string]
}

func (r cloneRow) id() string {
	if r.jc == nil {
		return ""
	}
	id := fmt.Sprint(r.jc.ID)
	for _, s := range r.route {
		id += fmt.Sprintf("-%d", s.ID)
	}
	return id
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
	footer            *widget.Label
	changeOrigin      *widget.Button
	columnSorter      *xwidget.ColumnSorter[cloneRow]
	origin            *app.EveSolarSystem
	originLabel       *xwidget.RichText
	routePref         app.EveRoutePreference
	rows              []cloneRow
	rowsFiltered      []cloneRow
	selectOwner       *kxwidget.FilterChipSelect
	selectRegion      *kxwidget.FilterChipSelect
	selectSolarSystem *kxwidget.FilterChipSelect
	selectTag         *kxwidget.FilterChipSelect
	sortButton        *xwidget.SortButton
	u                 *baseUI
}

const (
	clonesColLocation = iota + 1
	clonesColRegion
	clonesColImplants
	clonesColCharacter
	clonesColJumps
)

func newClones(u *baseUI) *clones {
	columns := xwidget.NewDataColumns([]xwidget.DataColumn[cloneRow]{{
		ID:    clonesColLocation,
		Label: "Location",
		Width: columnWidthLocation,
		Sort: func(a, b cloneRow) int {
			return cmp.Compare(a.jc.Location.DisplayName(), b.jc.Location.DisplayName())
		},
		Update: func(r cloneRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).Set(r.jc.Location.DisplayRichText())
		},
	}, {
		ID:    clonesColRegion,
		Label: "Region",
		Width: columnWidthRegion,
		Sort: func(a, b cloneRow) int {
			return cmp.Compare(a.jc.Location.RegionName(), b.jc.Location.RegionName())
		},
		Update: func(r cloneRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.jc.Location.RegionName())
		},
	}, {
		ID:    clonesColImplants,
		Label: "Impl.",
		Width: 100,
		Sort: func(a, b cloneRow) int {
			return cmp.Compare(a.jc.ImplantsCount, b.jc.ImplantsCount)
		},
		Update: func(r cloneRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(fmt.Sprint(r.jc.ImplantsCount), widget.RichTextStyle{
				Alignment: fyne.TextAlignTrailing,
			})
		},
	}, awidget.MakeEveEntityColumn(awidget.MakeEveEntityColumnParams[cloneRow]{
		ColumnID: clonesColCharacter,
		EIS:      u.eis,
		GetEntity: func(r cloneRow) *app.EveEntity {
			return &app.EveEntity{
				ID:       r.jc.Character.ID,
				Name:     r.jc.Character.Name,
				Category: app.EveEntityCharacter,
			}
		},
		IsAvatar: true,
		Label:    "Character",
	}), {
		ID:    clonesColJumps,
		Label: "Jumps",
		Width: 100,
		Sort: func(a, b cloneRow) int {
			return a.compare(b)
		},
		Update: func(r cloneRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.jumps(), widget.RichTextStyle{
				Alignment: fyne.TextAlignTrailing,
			})
		},
	}})
	a := &clones{
		columnSorter: xwidget.NewColumnSorter(columns, clonesColLocation, xwidget.SortAsc),
		originLabel:  xwidget.NewRichTextWithText("(not set)"),
		footer:       newLabelWithTruncation(),
		u:            u,
	}
	a.ExtendBaseWidget(a)
	a.originLabel.Truncation = fyne.TextTruncateClip
	a.changeOrigin = widget.NewButton("Origin", func() {
		a.setOrigin(a.u.MainWindow())
	})
	if !app.IsMobile() {
		a.body = xwidget.MakeDataTable(
			columns,
			&a.rowsFiltered,
			func() fyne.CanvasObject {
				x := xwidget.NewRichText()
				x.Truncation = fyne.TextTruncateClip
				return x
			},
			a.columnSorter,
			a.filterRowsAsync,
			func(c int, r cloneRow) {
				switch c {
				case 0:
					a.u.InfoWindow().ShowLocation(r.jc.Location.ID)
				case 1:
					if v, ok := r.jc.Location.SolarSystem.Value(); ok {
						a.u.InfoWindow().Show(app.EveEntityRegion, v.Constellation.Region.ID)
					}
				case 2:
					if r.jc == nil || r.jc.ImplantsCount == 0 {
						return
					}
					a.showCloneWindow(r.jc)
				case 3:
					a.u.InfoWindow().Show(app.EveEntityCharacter, r.jc.Character.ID)
				case 4:
					if len(r.route) == 0 {
						return
					}
					a.showRouteWindow(r)
				}
			},
		)
	} else {
		a.body = xwidget.MakeDataList(
			columns,
			&a.rowsFiltered,
			func(col int, r cloneRow) []widget.RichTextSegment {
				var s []widget.RichTextSegment
				switch col {
				case clonesColLocation:
					s = r.jc.Location.DisplayRichText()
				case clonesColRegion:
					s = xwidget.RichTextSegmentsFromText(r.jc.Location.RegionName())
				case clonesColImplants:
					s = xwidget.RichTextSegmentsFromText(fmt.Sprint(r.jc.ImplantsCount))
				case clonesColCharacter:
					s = xwidget.RichTextSegmentsFromText(r.jc.Character.Name)
				case clonesColJumps:
					s = xwidget.RichTextSegmentsFromText(r.jumps())
				}
				return s
			},
			func(r cloneRow) {
				if len(r.route) == 0 {
					return
				}
				a.showRouteWindow(r)
			},
		)
	}

	a.selectRegion = kxwidget.NewFilterChipSelectWithSearch("Region", []string{}, func(string) {
		a.filterRowsAsync(-1)
	}, a.u.window)

	a.selectSolarSystem = kxwidget.NewFilterChipSelectWithSearch("System", []string{}, func(string) {
		a.filterRowsAsync(-1)
	}, a.u.window)

	a.selectOwner = kxwidget.NewFilterChipSelect("Owner", []string{}, func(string) {
		a.filterRowsAsync(-1)
	})
	a.selectTag = kxwidget.NewFilterChipSelect("Tag", []string{}, func(string) {
		a.filterRowsAsync(-1)
	})
	a.sortButton = a.columnSorter.NewSortButton(func() {
		a.filterRowsAsync(-1)
	}, a.u.window)

	// signals
	a.u.signals.CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
		if arg.Section == app.SectionCharacterJumpClones {
			a.update(ctx)
		}
	})
	a.u.signals.CharacterAdded.AddListener(func(ctx context.Context, _ *app.Character) {
		a.update(ctx)
	})
	a.u.signals.CharacterRemoved.AddListener(func(ctx context.Context, _ *app.EntityShort) {
		a.update(ctx)
	})
	a.u.signals.TagsChanged.AddListener(func(ctx context.Context, s struct{}) {
		a.update(ctx)
	})
	return a
}

func (a *clones) CreateRenderer() fyne.WidgetRenderer {
	origin := container.NewBorder(
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
	if app.IsMobile() {
		filters.Add(a.sortButton)
	}
	var topBox *fyne.Container
	if app.IsMobile() {
		topBox = container.NewVBox(origin, container.NewHScroll(filters))
	} else {
		topBox = container.New(xlayout.NewColumnsByRatio(0.60), container.NewHScroll(filters), origin)
	}
	c := container.NewBorder(
		topBox,
		a.footer,
		nil,
		nil,
		a.body,
	)
	return widget.NewSimpleRenderer(c)
}

func (a *clones) filterRowsAsync(sortCol int) {
	totalRows := len(a.rows)
	rows := slices.Clone(a.rows)
	owner := a.selectOwner.Selected
	region := a.selectRegion.Selected
	solarSystem := a.selectSolarSystem.Selected
	tag := a.selectTag.Selected
	sortCol, dir, doSort := a.columnSorter.CalcSort(sortCol)

	go func() {
		// filter
		if owner != "" {
			rows = slices.DeleteFunc(rows, func(r cloneRow) bool {
				return r.jc.Character.Name != owner
			})
		}
		if region != "" {
			rows = slices.DeleteFunc(rows, func(r cloneRow) bool {
				return r.jc.Location.RegionName() != region
			})
		}
		if solarSystem != "" {
			rows = slices.DeleteFunc(rows, func(r cloneRow) bool {
				return r.jc.Location.SolarSystemName() != solarSystem
			})
		}
		if tag != "" {
			rows = slices.DeleteFunc(rows, func(r cloneRow) bool {
				return !r.tags.Contains(tag)
			})
		}
		a.columnSorter.SortRows(rows, sortCol, dir, doSort)
		// set data & refresh
		tagOptions := slices.Sorted(set.Union(xslices.Map(rows, func(r cloneRow) set.Set[string] {
			return r.tags
		})...).All())
		ownerOptions := xslices.Map(rows, func(r cloneRow) string {
			return r.jc.Character.Name
		})
		regionOptions := xslices.Map(rows, func(r cloneRow) string {
			return r.jc.Location.RegionName()
		})
		solarSystemOptions := xslices.Map(rows, func(r cloneRow) string {
			return r.jc.Location.SolarSystemName()
		})

		footer := fmt.Sprintf("Showing %d / %d clones", len(rows), totalRows)

		fyne.Do(func() {
			a.footer.Text = footer
			a.footer.Importance = widget.MediumImportance
			a.footer.Refresh()
			a.selectTag.SetOptions(tagOptions)
			a.selectOwner.SetOptions(ownerOptions)
			a.selectRegion.SetOptions(regionOptions)
			a.selectSolarSystem.SetOptions(solarSystemOptions)
			a.rowsFiltered = rows
			a.body.Refresh()
		})
	}()
}

func (a *clones) update(ctx context.Context) {
	rows, err := a.fetchRows(ctx)
	if err != nil {
		slog.Error("Failed to refresh clones UI", "err", err)
		fyne.Do(func() {
			a.footer.Text = "ERROR: " + app.ErrorDisplay(err)
			a.footer.Importance = widget.DangerImportance
			a.footer.Refresh()
			a.rows = xslices.Reset(a.rows)
			a.filterRowsAsync(-1)
		})
		return
	}
	fyne.Do(func() {
		a.rows = rows
		a.filterRowsAsync(-1)
		if len(rows) > 0 && a.origin != nil {
			a.updateRoutesAsync()
		}
	})
}

func (a *clones) fetchRows(ctx context.Context) ([]cloneRow, error) {
	oo, err := a.u.cs.ListAllJumpClones(ctx)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(oo, func(a, b *app.CharacterJumpClone2) int {
		return cmp.Compare(a.Location.SolarSystemName(), b.Location.SolarSystemName())
	})
	var rows []cloneRow
	for _, o := range oo {
		r := cloneRow{jc: o}
		tags, err := a.u.cs.ListTagsForCharacter(ctx, o.Character.ID)
		if err != nil {
			return nil, err
		}
		r.tags = tags
		rows = append(rows, r)
	}
	return rows, nil
}

func (a *clones) updateRoutesAsync() {
	if a.origin == nil {
		return
	}
	for i := range a.rows {
		a.rows[i].route = nil
	}
	a.body.Refresh()
	var headers []app.EveRouteHeader
	for _, r := range a.rows {
		destination, ok := r.jc.Location.SolarSystem.Value()
		if !ok {
			continue
		}
		headers = append(headers, app.EveRouteHeader{
			Origin:      a.origin,
			Destination: destination,
			Preference:  a.routePref,
		})
	}
	go func() {
		routes, err := a.u.eus.FetchRoutes(context.Background(), headers)
		if err != nil {
			slog.Error("failed to fetch routes", "error", err)
			fyne.Do(func() {
				s := "Failed to fetch routes: " + app.ErrorDisplay(err)
				a.originLabel.Set(xwidget.RichTextSegmentsFromText(s, widget.RichTextStyle{
					ColorName: theme.ColorNameError,
				}))
			})
			return
		}
		m := make(map[int64][]*app.EveSolarSystem)
		for h, route := range routes {
			m[h.Destination.ID] = route
		}
		fyne.Do(func() {
			for i, r := range a.rows {
				solarSystem, ok := r.jc.Location.SolarSystem.Value()
				if !ok {
					continue
				}
				a.rows[i].route = m[solarSystem.ID]
			}
			a.columnSorter.Set(clonesColJumps, xwidget.SortAsc)
			a.filterRowsAsync(-1)
		})
	}()
}

func (a *clones) setOrigin(w fyne.Window) {
	showErrorDialog := func(search string, err error) {
		slog.Error("Failed to resolve names", "search", search, "error", err)
		xdialog.ShowError("Something went wrong", err, w)
	}
	var d dialog.Dialog
	var results []*app.EveEntity
	routePref := widget.NewSelect(
		xslices.Map(app.EveRoutePreferences(), func(a app.EveRoutePreference) string {
			return a.String()
		}), nil,
	)
	routePref.Selected = app.RouteShorter.String()
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
		a.originLabel.Set(xwidget.InlineRichTextSegments(
			s.DisplayRichTextWithRegion(),
			xwidget.RichTextSegmentsFromText(fmt.Sprintf(" [%s]", a.routePref.String())),
		))
		a.updateRoutesAsync()
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
	if app.IsMobile() {
		d.Resize(fyne.NewSize(s.Width, s.Height))
	} else {
		d.Resize(fyne.NewSize(600, max(400, s.Height*0.8)))
	}
	d.Show()
	w.Canvas().Focus(entry)
}

func (a *clones) showRouteWindow(r cloneRow) {
	if r.jc == nil {
		return
	}
	title := fmt.Sprintf("Route: %s -> %s", a.origin.Name, r.jc.Location.SolarSystemName())
	w, ok := a.u.GetOrCreateWindow(fmt.Sprintf("route-%s", r.id()), title, r.jc.Character.Name)
	if !ok {
		w.Show()
		return
	}
	col := kxlayout.NewColumns(60)
	list := widget.NewList(
		func() int {
			return len(r.route)
		},
		func() fyne.CanvasObject {
			return container.New(col, widget.NewLabel(""), xwidget.NewRichText())
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(r.route) {
				return
			}
			s := r.route[id]
			border := co.(*fyne.Container).Objects
			num := border[0].(*widget.Label)
			num.SetText(fmt.Sprint(id))
			border[1].(*xwidget.RichText).Set(s.DisplayRichTextWithRegion())
		},
	)
	list.HideSeparators = true
	list.OnSelected = func(id widget.ListItemID) {
		defer list.UnselectAll()
		if id >= len(r.route) {
			return
		}
		s := r.route[id]
		a.u.InfoWindow().Show(app.EveEntitySolarSystem, s.ID)

	}

	var fromText []widget.RichTextSegment
	if a.origin != nil {
		fromText = a.origin.DisplayRichTextWithRegion()
	}
	from := xwidget.NewTappableRichText(fromText, func() {
		if a.origin != nil {
			a.u.InfoWindow().Show(app.EveEntitySolarSystem, a.origin.ID)
		}
	})
	from.Wrapping = fyne.TextWrapWord

	var toText []widget.RichTextSegment
	if v, ok := r.jc.Location.SolarSystem.Value(); ok {
		toText = v.DisplayRichTextWithRegion()
	}
	to := xwidget.NewTappableRichText(toText, func() {
		if v, ok := r.jc.Location.SolarSystem.Value(); ok {
			a.u.InfoWindow().Show(app.EveEntitySolarSystem, v.ID)
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
	setDetailWindow(detailWindowParams{
		title:   title,
		content: c,
		window:  w,
	})
	w.Show()
}

func (a *clones) showCloneWindow(jc *app.CharacterJumpClone2) {
	if jc == nil {
		return
	}
	title := fmt.Sprintf("Clone #%d", jc.CloneID)
	w, ok := a.u.GetOrCreateWindow(fmt.Sprintf("clone-%d-%d", jc.Character.ID, jc.ID), title, jc.Character.Name)
	if !ok {
		w.Show()
		return
	}
	clone, err := a.u.cs.GetJumpClone(context.Background(), jc.Character.ID, jc.CloneID)
	if err != nil {
		slog.Error("show clone", "error", err)
		xdialog.ShowError("failed to load clone", err, a.u.MainWindow())
		return
	}
	list := widget.NewList(
		func() int {
			return len(clone.Implants)
		},
		func() fyne.CanvasObject {
			icon := xwidget.NewImageFromResource(
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
			a.u.eis.InventoryTypeIconAsync(im.EveType.ID, app.IconPixelSize, func(r fyne.Resource) {
				icon.Resource = r
				icon.Refresh()
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
		a.u.InfoWindow().Show(app.EveEntityInventoryType, im.EveType.ID)

	}

	location := makeLocationLabel(jc.Location.ToShort(), a.u.InfoWindow().ShowLocation)
	character := makeLinkLabelWithWrap(jc.Character.Name, func() {
		a.u.InfoWindow().Show(app.EveEntityCharacter, jc.Character.ID)
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
	setDetailWindow(detailWindowParams{
		title:   title,
		content: c,
		window:  w,
	})
	w.Show()
}
