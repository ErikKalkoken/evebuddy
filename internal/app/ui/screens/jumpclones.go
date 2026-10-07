package screens

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/ErikKalkoken/go-set"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/xlayout"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

// Names of the jump clone filters, used as labels on desktop and as option names on mobile.
const (
	jumpClonesFilterCharacter = "Character"
	jumpClonesFilterRegion    = "Region"
	jumpClonesFilterSystem    = "System"
	jumpClonesFilterTag       = "Tag"
)

// jumpClonesFilter is the selected value of each jump clone filter. Empty means not filtered.
type jumpClonesFilter struct {
	character   string
	region      string
	solarSystem string
	tag         string
}

// match reports whether row r passes all filters.
func (f jumpClonesFilter) match(r jumpCloneRow) bool {
	switch {
	case f.character != "" && r.jc.Character.Name != f.character,
		f.region != "" && r.jc.Location.RegionName() != f.region,
		f.solarSystem != "" && r.jc.Location.SolarSystemName() != f.solarSystem,
		f.tag != "" && !r.tags.Contains(f.tag):
		return false
	}
	return true
}

type jumpCloneRow struct {
	jc    *app.CharacterJumpClone2
	route []*app.EveSolarSystem
	tags  set.Set[string]
}

func (r jumpCloneRow) compare(other jumpCloneRow) int {
	return cmp.Compare(r.sortValue(), other.sortValue())
}

func (r jumpCloneRow) sortValue() int {
	if r.route == nil {
		return 10_000
	}
	if len(r.route) == 0 {
		return 10_000_000
	}
	return len(r.route) - 1
}

func (r jumpCloneRow) jumps() string {
	if r.route == nil {
		return "?"
	}
	if len(r.route) == 0 {
		return "No route"
	}
	return fmt.Sprint(len(r.route) - 1)
}

type JumpClones struct {
	widget.BaseWidget

	body              fyne.CanvasObject
	filterChip        *xwidget.FilterChipCompact // only on mobile
	filterRun         latestRun
	footer            *widget.Label
	changeOrigin      *ttwidget.Button
	columnSorter      *xwidget.ColumnSorter[jumpCloneRow]
	origin            *app.EveSolarSystem
	originLabel       *widget.Label
	originSecurity    *xwidget.RichText
	routeCancel       context.CancelFunc
	routePref         app.EveRoutePreference
	rows              []jumpCloneRow
	rowsFiltered      []jumpCloneRow
	selectCharacter   *kxwidget.FilterChipSelect // select chips only on desktop
	selectRegion      *kxwidget.FilterChipSelect
	selectSolarSystem *kxwidget.FilterChipSelect
	selectTag         *kxwidget.FilterChipSelect
	sortChip          *kxwidget.SortChip
	u                 baseUI
}

func NewJumpClones(u baseUI) *JumpClones {
	columns := xwidget.NewDataColumns([]xwidget.DataColumn[jumpCloneRow]{{
		Label: "Location",
		Width: ui.ColumnWidthLocation,
		Sort: func(a, b jumpCloneRow) int {
			return cmp.Compare(a.jc.Location.DisplayName(), b.jc.Location.DisplayName())
		},
		Update: func(r jumpCloneRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).Set(r.jc.Location.DisplayRichText())
		},
	}, {
		Label: "Region",
		Width: ui.ColumnWidthRegion,
		Sort: func(a, b jumpCloneRow) int {
			return cmp.Compare(a.jc.Location.RegionName(), b.jc.Location.RegionName())
		},
		Update: func(r jumpCloneRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.jc.Location.RegionName())
		},
	}, {
		Label: "Impl.",
		Width: 100,
		Sort: func(a, b jumpCloneRow) int {
			return cmp.Compare(a.jc.ImplantsCount, b.jc.ImplantsCount)
		},
		Update: func(r jumpCloneRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(fmt.Sprint(r.jc.ImplantsCount), widget.RichTextStyle{
				Alignment: fyne.TextAlignTrailing,
			})
		},
	}, ui.MakeEveEntityColumn(ui.MakeEveEntityColumnParams[jumpCloneRow]{
		EIS: u.EVEImage(),
		GetEntity: func(r jumpCloneRow) *app.EveEntity {
			return &app.EveEntity{
				ID:       r.jc.Character.ID,
				Name:     r.jc.Character.Name,
				Category: app.EveEntityCharacter,
			}
		},
		IsAvatar: true,
		Label:    "Character",
	}), {
		Label: "Jumps",
		Width: 100,
		Sort: func(a, b jumpCloneRow) int {
			return a.compare(b)
		},
		Update: func(r jumpCloneRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.jumps(), widget.RichTextStyle{
				Alignment: fyne.TextAlignTrailing,
			})
		},
	}})
	a := &JumpClones{
		columnSorter:   xwidget.NewColumnSorter(columns, "Location", xwidget.SortAsc),
		originLabel:    ui.NewLabelWithTruncation("(not set)"),
		originSecurity: xwidget.NewRichText(),
		footer:         ui.NewLabelWithTruncation(""),
		u:              u,
	}
	a.ExtendBaseWidget(a)
	a.originSecurity.Hide()
	a.changeOrigin = ttwidget.NewButtonWithIcon("", theme.NewThemedResource(icons.MapMarkerSvg), func() {
		a.setOrigin(a.u.MainWindow())
	})
	a.changeOrigin.SetToolTip("Change origin")
	if !a.u.IsMobile() {
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
			func(_ int, r jumpCloneRow) {
				showCloneDetailWindow(a.u, r, a.origin, a.routePref)
			},
		)
	} else {
		a.body = xwidget.MakeDataList(
			columns,
			&a.rowsFiltered,
			func(col string, r jumpCloneRow) []widget.RichTextSegment {
				var s []widget.RichTextSegment
				switch col {
				case "Location":
					s = r.jc.Location.DisplayRichText()
				case "Region":
					s = xwidget.RichTextSegmentsFromText(r.jc.Location.RegionName())
				case "Impl.":
					s = xwidget.RichTextSegmentsFromText(fmt.Sprint(r.jc.ImplantsCount))
				case "Character":
					s = xwidget.RichTextSegmentsFromText(r.jc.Character.Name)
				case "Jumps":
					s = xwidget.RichTextSegmentsFromText(r.jumps())
				}
				return s
			},
			func(r jumpCloneRow) {
				showCloneDetailWindow(a.u, r, a.origin, a.routePref)
			},
		)
	}

	if a.u.IsMobile() {
		a.filterChip = xwidget.NewFilterChipCompact(nil, func(map[string]string) {
			a.filterRowsAsync("")
		})
	} else {
		makeSelect := func(label string) *kxwidget.FilterChipSelect {
			return kxwidget.NewFilterChipSelect(label, []string{}, func(string) {
				a.filterRowsAsync("")
			})
		}
		makeSelectWithSearch := func(label string) *kxwidget.FilterChipSelect {
			return kxwidget.NewFilterChipSelectWithSearch(label, []string{}, func(string) {
				a.filterRowsAsync("")
			}, a.u.MainWindow())
		}
		a.selectRegion = makeSelectWithSearch(jumpClonesFilterRegion)
		a.selectSolarSystem = makeSelectWithSearch(jumpClonesFilterSystem)
		a.selectCharacter = makeSelect(jumpClonesFilterCharacter)
		a.selectTag = makeSelect(jumpClonesFilterTag)
	}
	a.sortChip = a.columnSorter.NewSortChip(func() {
		a.filterRowsAsync("")
	})

	// signals
	a.u.Signals().AppInit.AddListener(func(ctx context.Context, _ struct{}) {
		a.update(ctx)
	})

	a.u.Signals().CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
		if arg.Section == app.SectionCharacterJumpClones {
			a.update(ctx)
		}
	})
	a.u.Signals().CharacterAdded.AddListener(func(ctx context.Context, _ *app.Character) {
		a.update(ctx)
	})
	a.u.Signals().CharacterRemoved.AddListener(func(ctx context.Context, _ *app.EntityShort) {
		a.update(ctx)
	})
	a.u.Signals().TagsChanged.AddListener(func(ctx context.Context, _ struct{}) {
		a.update(ctx)
	})
	return a
}

func (a *JumpClones) CreateRenderer() fyne.WidgetRenderer {
	originText := container.New(originTextLayout{}, a.originSecurity, a.originLabel)
	var topBox *fyne.Container
	if a.u.IsMobile() {
		topBox = container.NewBorder(nil, nil, a.changeOrigin, container.NewHBox(a.filterChip, a.sortChip), originText)
	} else {
		origin := container.NewBorder(
			nil,
			nil,
			a.changeOrigin,
			nil,
			originText,
		)
		filters := container.NewHBox(
			a.selectRegion,
			a.selectSolarSystem,
			a.selectCharacter,
			a.selectTag,
		)
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

// currentFilter returns the selected filters: from the compact chip on mobile
// and from the filter chips on desktop.
func (a *JumpClones) currentFilter() jumpClonesFilter {
	if a.filterChip != nil {
		s := a.filterChip.Selected()
		return jumpClonesFilter{
			character:   s[jumpClonesFilterCharacter],
			region:      s[jumpClonesFilterRegion],
			solarSystem: s[jumpClonesFilterSystem],
			tag:         s[jumpClonesFilterTag],
		}
	}
	return jumpClonesFilter{
		character:   a.selectCharacter.Selected,
		region:      a.selectRegion.Selected,
		solarSystem: a.selectSolarSystem.Selected,
		tag:         a.selectTag.Selected,
	}
}

func (a *JumpClones) filterRowsAsync(sortCol string) {
	isLatest := a.filterRun.start()
	totalRows := len(a.rows)
	rows := slices.Clone(a.rows)
	filter := a.currentFilter()
	sortCol, dir, doSort := a.columnSorter.CalcSort(sortCol)

	runAsync(func() {
		rows = slices.DeleteFunc(rows, func(r jumpCloneRow) bool {
			return !filter.match(r)
		})
		a.columnSorter.SortRows(rows, sortCol, dir, doSort)
		// set data & refresh
		tagOptions := slices.Sorted(set.Union(xslices.Map(rows, func(r jumpCloneRow) set.Set[string] {
			return r.tags
		})...).All())
		characterOptions := xslices.Map(rows, func(r jumpCloneRow) string {
			return r.jc.Character.Name
		})
		regionOptions := xslices.Map(rows, func(r jumpCloneRow) string {
			return r.jc.Location.RegionName()
		})
		solarSystemOptions := xslices.Map(rows, func(r jumpCloneRow) string {
			return r.jc.Location.SolarSystemName()
		})

		footer := fmt.Sprintf("Showing %d / %d clones", len(rows), totalRows)

		fyne.Do(func() {
			if !isLatest() {
				return
			}
			a.footer.Text = footer
			a.footer.Importance = widget.MediumImportance
			a.footer.Refresh()
			if a.filterChip != nil {
				a.filterChip.SetOptions(
					xwidget.NewFilterOptionMultiChoiceWithSearch(jumpClonesFilterRegion, regionOptions),
					xwidget.NewFilterOptionMultiChoiceWithSearch(jumpClonesFilterSystem, solarSystemOptions),
					xwidget.NewFilterOptionMultiChoice(jumpClonesFilterCharacter, characterOptions),
					xwidget.NewFilterOptionMultiChoice(jumpClonesFilterTag, tagOptions),
				)
			} else {
				a.selectTag.SetOptions(tagOptions)
				a.selectCharacter.SetOptions(characterOptions)
				a.selectRegion.SetOptions(regionOptions)
				a.selectSolarSystem.SetOptions(solarSystemOptions)
			}
			a.rowsFiltered = rows
			a.body.Refresh()
		})
	})
}

func (a *JumpClones) update(ctx context.Context) {
	rows, err := a.fetchRows(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		slog.Error("Failed to refresh clones UI", "err", err)
		fyne.Do(func() {
			a.footer.Text = "ERROR: " + a.u.ErrorDisplay(err)
			a.footer.Importance = widget.DangerImportance
			a.footer.Refresh()
			xslices.Clear(&a.rows)
			a.filterRowsAsync("")
		})
		return
	}
	fyne.Do(func() {
		a.rows = rows
		a.filterRowsAsync("")
		if len(rows) > 0 && a.origin != nil {
			a.updateRoutesAsync()
		}
	})
}

func (a *JumpClones) fetchRows(ctx context.Context) ([]jumpCloneRow, error) {
	oo, err := a.u.Character().ListAllJumpClones(ctx)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(oo, func(a, b *app.CharacterJumpClone2) int {
		return cmp.Compare(a.Location.SolarSystemName(), b.Location.SolarSystemName())
	})
	var rows []jumpCloneRow
	for _, o := range oo {
		r := jumpCloneRow{jc: o}
		tags, err := a.u.Character().ListTagsForCharacter(ctx, o.Character.ID)
		if err != nil {
			return nil, err
		}
		r.tags = tags
		rows = append(rows, r)
	}
	return rows, nil
}

func (a *JumpClones) updateRoutesAsync() {
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
	if a.routeCancel != nil {
		a.routeCancel() // supersede any still-running fetch
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.routeCancel = cancel
	go func() {
		routes, err := a.u.EVEUniverse().FetchRoutes(ctx, headers)
		if ctx.Err() != nil {
			return // superseded
		}
		if err != nil {
			slog.Error("failed to fetch routes", "error", err)
			fyne.Do(func() {
				a.originSecurity.Hide()
				a.originLabel.Text = "Failed to fetch routes: " + a.u.ErrorDisplay(err)
				a.originLabel.Importance = widget.DangerImportance
				a.originLabel.Refresh()
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
			a.columnSorter.Set("Jumps", xwidget.SortAsc)
			a.filterRowsAsync("")
		})
	}()
}

func (a *JumpClones) setOrigin(w fyne.Window) {
	if a.u.IsOffline() {
		a.u.DisplaySnackbar("Can't set origin while offline")
		return
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
		go func() {
			s, err := a.u.EVEUniverse().GetOrCreateSolarSystemESI(context.Background(), r.ID)
			if err != nil {
				ui.NotifyErrorAndLog("Failed to load solar system. Try again later.", err, a.u.DisplaySnackbar)
				return
			}
			fyne.Do(func() {
				a.origin = s
				a.routePref = app.EveRoutePreferenceFromString(routePref.Selected)
				a.originSecurity.Set(s.SecurityStatusRichText())
				a.originSecurity.Show()
				a.originLabel.Text = fmt.Sprintf("  %s [%s]", s.Name, a.routePref.String())
				a.originLabel.Importance = widget.MediumImportance
				a.originLabel.Refresh()
				a.updateRoutesAsync()
				d.Hide()
			})
		}()
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
			ee, _, err := a.u.Character().SearchESI(
				ctx,
				search,
				[]app.SearchCategory{app.SearchSolarSystem},
				false,
			)
			if err != nil {
				ui.NotifyErrorAndLog("Failed to resolve search. Try again later.", err, a.u.DisplaySnackbar)
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
	if a.u.IsMobile() {
		d.Resize(fyne.NewSize(s.Width, s.Height))
	} else {
		d.Resize(fyne.NewSize(600, max(400, s.Height*0.8)))
	}
	d.Show()
	w.Canvas().Focus(entry)
}

// originTextLayout places the security status and the origin label so they read as one text,
// while only the label is truncated.
type originTextLayout struct{}

// overlap returns how far the label is moved over the security status,
// so the inner paddings between them don't add extra space.
func (originTextLayout) overlap(objects []fyne.CanvasObject) float32 {
	if !objects[0].Visible() {
		return 0
	}
	return objects[0].MinSize().Width - 2*theme.Size(theme.SizeNameInnerPadding)
}

func (l originTextLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	security, label := objects[0], objects[1]
	x := l.overlap(objects)
	if security.Visible() {
		security.Move(fyne.NewPos(0, 0))
		security.Resize(fyne.NewSize(security.MinSize().Width, size.Height))
	}
	label.Move(fyne.NewPos(x, 0))
	label.Resize(fyne.NewSize(size.Width-x, size.Height))
}

func (l originTextLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	security, label := objects[0].MinSize(), objects[1].MinSize()
	return fyne.NewSize(l.overlap(objects)+label.Width, max(security.Height, label.Height))
}
