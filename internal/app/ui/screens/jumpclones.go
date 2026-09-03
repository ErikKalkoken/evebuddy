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

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/xlayout"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

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
	footer            *widget.Label
	changeOrigin      *widget.Button
	columnSorter      *xwidget.ColumnSorter[jumpCloneRow]
	origin            *app.EveSolarSystem
	originLabel       *xwidget.RichText
	routePref         app.EveRoutePreference
	rows              []jumpCloneRow
	rowsFiltered      []jumpCloneRow
	selectCharacter   *kxwidget.FilterChipSelect
	selectRegion      *kxwidget.FilterChipSelect
	selectSolarSystem *kxwidget.FilterChipSelect
	selectTag         *kxwidget.FilterChipSelect
	sortChip          *kxwidget.SortChip
	u                 baseUI
}

const (
	jumpClonesColLocation = iota + 1
	jumpClonesColRegion
	jumpClonesColImplants
	jumpClonesColCharacter
	jumpClonesColJumps
)

func NewJumpClones(u baseUI) *JumpClones {
	columns := xwidget.NewDataColumns([]xwidget.DataColumn[jumpCloneRow]{{
		ID:    jumpClonesColLocation,
		Label: "Location",
		Width: ui.ColumnWidthLocation,
		Sort: func(a, b jumpCloneRow) int {
			return cmp.Compare(a.jc.Location.DisplayName(), b.jc.Location.DisplayName())
		},
		Update: func(r jumpCloneRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).Set(r.jc.Location.DisplayRichText())
		},
	}, {
		ID:    jumpClonesColRegion,
		Label: "Region",
		Width: ui.ColumnWidthRegion,
		Sort: func(a, b jumpCloneRow) int {
			return cmp.Compare(a.jc.Location.RegionName(), b.jc.Location.RegionName())
		},
		Update: func(r jumpCloneRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.jc.Location.RegionName())
		},
	}, {
		ID:    jumpClonesColImplants,
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
		ColumnID: jumpClonesColCharacter,
		EIS:      u.EVEImage(),
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
		ID:    jumpClonesColJumps,
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
		columnSorter: xwidget.NewColumnSorter(columns, jumpClonesColLocation, xwidget.SortAsc),
		originLabel:  xwidget.NewRichTextWithText("(not set)"),
		footer:       ui.NewLabelWithTruncation(""),
		u:            u,
	}
	a.ExtendBaseWidget(a)
	a.originLabel.Truncation = fyne.TextTruncateClip
	a.changeOrigin = widget.NewButton("Origin", func() {
		a.setOrigin(a.u.MainWindow())
	})
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
			func(col int, r jumpCloneRow) []widget.RichTextSegment {
				var s []widget.RichTextSegment
				switch col {
				case jumpClonesColLocation:
					s = r.jc.Location.DisplayRichText()
				case jumpClonesColRegion:
					s = xwidget.RichTextSegmentsFromText(r.jc.Location.RegionName())
				case jumpClonesColImplants:
					s = xwidget.RichTextSegmentsFromText(fmt.Sprint(r.jc.ImplantsCount))
				case jumpClonesColCharacter:
					s = xwidget.RichTextSegmentsFromText(r.jc.Character.Name)
				case jumpClonesColJumps:
					s = xwidget.RichTextSegmentsFromText(r.jumps())
				}
				return s
			},
			func(r jumpCloneRow) {
				showCloneDetailWindow(a.u, r, a.origin, a.routePref)
			},
		)
	}

	a.selectRegion = kxwidget.NewFilterChipSelectWithSearch("Region", []string{}, func(string) {
		a.filterRowsAsync(-1)
	}, a.u.MainWindow())

	a.selectSolarSystem = kxwidget.NewFilterChipSelectWithSearch("System", []string{}, func(string) {
		a.filterRowsAsync(-1)
	}, a.u.MainWindow())

	a.selectCharacter = kxwidget.NewFilterChipSelect("Character", []string{}, func(string) {
		a.filterRowsAsync(-1)
	})
	a.selectTag = kxwidget.NewFilterChipSelect("Tag", []string{}, func(string) {
		a.filterRowsAsync(-1)
	})
	a.sortChip = a.columnSorter.NewSortChip(func() {
		a.filterRowsAsync(-1)
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
		a.selectCharacter,
		a.selectTag,
	)
	if a.u.IsMobile() {
		filters.Add(a.sortChip)
	}
	var topBox *fyne.Container
	if a.u.IsMobile() {
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

func (a *JumpClones) filterRowsAsync(sortCol int) {
	totalRows := len(a.rows)
	rows := slices.Clone(a.rows)
	character := a.selectCharacter.Selected
	region := a.selectRegion.Selected
	solarSystem := a.selectSolarSystem.Selected
	tag := a.selectTag.Selected
	sortCol, dir, doSort := a.columnSorter.CalcSort(sortCol)

	go func() {
		// filter
		if character != "" {
			rows = slices.DeleteFunc(rows, func(r jumpCloneRow) bool {
				return r.jc.Character.Name != character
			})
		}
		if region != "" {
			rows = slices.DeleteFunc(rows, func(r jumpCloneRow) bool {
				return r.jc.Location.RegionName() != region
			})
		}
		if solarSystem != "" {
			rows = slices.DeleteFunc(rows, func(r jumpCloneRow) bool {
				return r.jc.Location.SolarSystemName() != solarSystem
			})
		}
		if tag != "" {
			rows = slices.DeleteFunc(rows, func(r jumpCloneRow) bool {
				return !r.tags.Contains(tag)
			})
		}
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
			a.footer.Text = footer
			a.footer.Importance = widget.MediumImportance
			a.footer.Refresh()
			a.selectTag.SetOptions(tagOptions)
			a.selectCharacter.SetOptions(characterOptions)
			a.selectRegion.SetOptions(regionOptions)
			a.selectSolarSystem.SetOptions(solarSystemOptions)
			a.rowsFiltered = rows
			a.body.Refresh()
		})
	}()
}

func (a *JumpClones) update(ctx context.Context) {
	rows, err := a.fetchRows(ctx)
	if err != nil {
		slog.Error("Failed to refresh clones UI", "err", err)
		fyne.Do(func() {
			a.footer.Text = "ERROR: " + a.u.ErrorDisplay(err)
			a.footer.Importance = widget.DangerImportance
			a.footer.Refresh()
			xslices.Clear(&a.rows)
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
	go func() {
		routes, err := a.u.EVEUniverse().FetchRoutes(context.Background(), headers)
		if err != nil {
			slog.Error("failed to fetch routes", "error", err)
			fyne.Do(func() {
				s := "Failed to fetch routes: " + a.u.ErrorDisplay(err)
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
			a.columnSorter.Set(jumpClonesColJumps, xwidget.SortAsc)
			a.filterRowsAsync(-1)
		})
	}()
}

func (a *JumpClones) setOrigin(w fyne.Window) {
	showErrorDialog := func(search string, err error) {
		ui.ShowErrorAndLog("Failed to resolve search for "+search, err, a.u.IsDeveloperMode(), w)
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
				fyne.Do(func() {
					showErrorDialog("Could not load solar system", err)
				})
				return
			}
			fyne.Do(func() {
				a.origin = s
				a.routePref = app.EveRoutePreferenceFromString(routePref.Selected)
				a.originLabel.Set(xwidget.InlineRichTextSegments(
					s.DisplayRichTextWithRegion(),
					xwidget.RichTextSegmentsFromText(fmt.Sprintf(" [%s]", a.routePref.String())),
				))
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
	if a.u.IsMobile() {
		d.Resize(fyne.NewSize(s.Width, s.Height))
	} else {
		d.Resize(fyne.NewSize(600, max(400, s.Height*0.8)))
	}
	d.Show()
	w.Canvas().Focus(entry)
}
