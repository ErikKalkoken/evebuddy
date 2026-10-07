package screens

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

const (
	industrySlotsFreeSome = "Has free slots"
	industrySlotsFreeNone = "No free slots"
)

func industrySlotsFreeSlotsOptions() []string {
	return []string{industrySlotsFreeSome, industrySlotsFreeNone}
}

// Names of the industry slot filters, used as labels on desktop and as option names on mobile.
const (
	industrySlotsFilterFreeSlots = "Free slots"
	industrySlotsFilterTag       = "Tag"
)

// industrySlotsFilter is the selected value of each industry slot filter. Empty means not filtered.
type industrySlotsFilter struct {
	freeSlots string
	tag       string
}

// match reports whether row r passes all filters.
func (f industrySlotsFilter) match(r industrySlotRow) bool {
	switch {
	case f.freeSlots == industrySlotsFreeSome && r.free == 0,
		f.freeSlots == industrySlotsFreeNone && r.free > 0,
		f.tag != "" && !r.tags.Contains(f.tag):
		return false
	}
	return true
}

type industrySlotRow struct {
	characterID   int64
	characterName string
	busy          int
	ready         int
	free          int
	total         int
	isTotal       bool
	tags          set.Set[string]
}

func (r industrySlotRow) characterDisplay() []widget.RichTextSegment {
	if r.isTotal {
		return xwidget.RichTextSegmentsFromText("Totals", widget.RichTextStyle{
			TextStyle: fyne.TextStyle{Bold: true},
		})
	}
	return xwidget.RichTextSegmentsFromText(r.characterName)
}

func (r industrySlotRow) busyColor() fyne.ThemeColorName {
	var c fyne.ThemeColorName
	switch r.busy {
	case 0:
		c = theme.ColorNameSuccess
	case r.total:
		c = theme.ColorNameError
	default:
		c = theme.ColorNameWarning
	}
	return c
}

func (r industrySlotRow) readyColor() fyne.ThemeColorName {
	var c fyne.ThemeColorName
	switch {
	case r.ready > 0:
		c = theme.ColorNameWarning
	case r.ready == 0:
		c = theme.ColorNameSuccess
	default:
		c = theme.ColorNameForeground
	}
	return c
}

func (r industrySlotRow) freeColor() fyne.ThemeColorName {
	var c fyne.ThemeColorName
	switch {
	case r.free == r.total:
		c = theme.ColorNameSuccess
	case r.free > 0:
		c = theme.ColorNameWarning
	case r.free == 0:
		c = theme.ColorNameError
	}
	return c
}

type IndustrySlots struct {
	widget.BaseWidget

	body            fyne.CanvasObject
	filterChip      *xwidget.FilterChipCompact // only on mobile
	filterRun       latestRun
	footer          *widget.Label
	columnSorter    *xwidget.ColumnSorter[industrySlotRow]
	rows            []industrySlotRow
	rowsFiltered    []industrySlotRow
	selectFreeSlots *kxwidget.FilterChipSelect // select chips only on desktop
	selectTag       *kxwidget.FilterChipSelect
	slotType        app.IndustryJobType
	sortChip        *kxwidget.SortChip
	u               baseUI
}

const (
	industrySlotsColCharacter = iota
	industrySlotsColBusy
	industrySlotsColReady
	industrySlotsColFree
	industrySlotsColTotal
)

func NewIndustrySlots(u baseUI, slotType app.IndustryJobType) *IndustrySlots {
	const columnWidthNumber = 75
	columns := xwidget.NewDataColumns([]xwidget.DataColumn[industrySlotRow]{
		ui.MakeEveEntityColumn(ui.MakeEveEntityColumnParams[industrySlotRow]{
			EIS: u.EVEImage(),
			GetEntity: func(r industrySlotRow) *app.EveEntity {
				return &app.EveEntity{
					ID:       r.characterID,
					Name:     r.characterName,
					Category: app.EveEntityCharacter,
				}
			},
			IsAvatar: true,
			Label:    "Character",
		}), {
			Label: "Busy",
			Width: columnWidthNumber,
			Sort: func(a, b industrySlotRow) int {
				return cmp.Compare(a.busy, b.busy)
			},
			Update: func(r industrySlotRow, co fyne.CanvasObject) {
				co.(*xwidget.RichText).SetWithText(fmt.Sprint(r.busy), widget.RichTextStyle{
					Alignment: fyne.TextAlignTrailing,
					ColorName: r.busyColor(),
					TextStyle: fyne.TextStyle{Bold: r.isTotal},
				})
			},
		}, {
			Label: "Ready",
			Width: columnWidthNumber,
			Sort: func(a, b industrySlotRow) int {
				return cmp.Compare(a.ready, b.ready)
			},
			Update: func(r industrySlotRow, co fyne.CanvasObject) {
				co.(*xwidget.RichText).SetWithText(fmt.Sprint(r.ready), widget.RichTextStyle{
					Alignment: fyne.TextAlignTrailing,
					ColorName: r.readyColor(),
					TextStyle: fyne.TextStyle{Bold: r.isTotal},
				})
			},
		}, {
			Label: "Free",
			Width: columnWidthNumber,
			Sort: func(a, b industrySlotRow) int {
				return cmp.Compare(a.free, b.free)
			},
			Update: func(r industrySlotRow, co fyne.CanvasObject) {
				co.(*xwidget.RichText).SetWithText(fmt.Sprint(r.free), widget.RichTextStyle{
					Alignment: fyne.TextAlignTrailing,
					ColorName: r.freeColor(),
					TextStyle: fyne.TextStyle{Bold: r.isTotal},
				})
			},
		}, {
			Label: "Total",
			Width: columnWidthNumber,
			Sort: func(a, b industrySlotRow) int {
				return cmp.Compare(a.total, b.total)
			},
			Update: func(r industrySlotRow, co fyne.CanvasObject) {
				co.(*xwidget.RichText).SetWithText(fmt.Sprint(r.total), widget.RichTextStyle{
					Alignment: fyne.TextAlignTrailing,
					TextStyle: fyne.TextStyle{Bold: r.isTotal},
				})
			},
		}})
	a := &IndustrySlots{
		footer:       ui.NewLabelWithWrapping(""),
		columnSorter: xwidget.NewColumnSorter(columns, "Character", xwidget.SortAsc),
		slotType:     slotType,
		u:            u,
	}
	a.ExtendBaseWidget(a)
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
			nil,
		)
	} else {
		a.body = a.makeDataTable(
			columns,
			func(col int, r industrySlotRow) []widget.RichTextSegment {
				switch col {
				case industrySlotsColCharacter:
					return r.characterDisplay()
				case industrySlotsColBusy:
					return xwidget.RichTextSegmentsFromText(fmt.Sprint(r.busy), widget.RichTextStyle{
						Alignment: fyne.TextAlignTrailing,
						ColorName: r.busyColor(),
						TextStyle: fyne.TextStyle{Bold: r.isTotal},
					})
				case industrySlotsColReady:
					return xwidget.RichTextSegmentsFromText(fmt.Sprint(r.ready), widget.RichTextStyle{
						Alignment: fyne.TextAlignTrailing,
						ColorName: r.readyColor(),
						TextStyle: fyne.TextStyle{Bold: r.isTotal},
					})
				case industrySlotsColFree:
					return xwidget.RichTextSegmentsFromText(fmt.Sprint(r.free), widget.RichTextStyle{
						Alignment: fyne.TextAlignTrailing,
						ColorName: r.freeColor(),
						TextStyle: fyne.TextStyle{Bold: r.isTotal},
					})
				case industrySlotsColTotal:
					return xwidget.RichTextSegmentsFromText(fmt.Sprint(r.total), widget.RichTextStyle{
						Alignment: fyne.TextAlignTrailing,
						TextStyle: fyne.TextStyle{Bold: r.isTotal},
					})
				}
				return xwidget.RichTextSegmentsFromText("?")
			},
		)
	}

	if a.u.IsMobile() {
		a.filterChip = xwidget.NewFilterChipCompact(nil, func(map[string]string) {
			a.filterRowsAsync("")
		})
	} else {
		a.selectFreeSlots = kxwidget.NewFilterChipSelect(industrySlotsFilterFreeSlots, industrySlotsFreeSlotsOptions(), func(string) {
			a.filterRowsAsync("")
		})
		a.selectFreeSlots.SortDisabled = true
		a.selectTag = kxwidget.NewFilterChipSelect(industrySlotsFilterTag, []string{}, func(string) {
			a.filterRowsAsync("")
		})
	}
	a.sortChip = a.columnSorter.NewSortChip(func() {
		a.filterRowsAsync("")
	})

	// signals
	a.u.Signals().AppInit.AddListener(func(ctx context.Context, _ struct{}) {
		a.update(ctx)
	})
	a.u.Signals().CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
		switch arg.Section {
		case app.SectionCharacterIndustryJobs, app.SectionCharacterSkills:
			a.update(ctx)
		}
	})
	a.u.Signals().CorporationSectionChanged.AddListener(func(ctx context.Context, arg app.CorporationSectionUpdated) {
		if arg.Section == app.SectionCorporationIndustryJobs {
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

func (a *IndustrySlots) CreateRenderer() fyne.WidgetRenderer {
	var top fyne.CanvasObject
	if a.u.IsMobile() {
		top = container.NewHBox(a.filterChip, a.sortChip)
	} else {
		top = container.NewHScroll(container.NewHBox(a.selectFreeSlots, a.selectTag))
	}
	c := container.NewBorder(top, a.footer, nil, nil, a.body)
	return widget.NewSimpleRenderer(c)
}

// currentFilter returns the selected filters: from the compact chip on mobile
// and from the filter chips on desktop.
func (a *IndustrySlots) currentFilter() industrySlotsFilter {
	if a.filterChip != nil {
		s := a.filterChip.Selected()
		return industrySlotsFilter{
			freeSlots: s[industrySlotsFilterFreeSlots],
			tag:       s[industrySlotsFilterTag],
		}
	}
	return industrySlotsFilter{
		freeSlots: a.selectFreeSlots.Selected,
		tag:       a.selectTag.Selected,
	}
}

func (a *IndustrySlots) makeDataTable(headers xwidget.DataColumns[industrySlotRow], makeCell func(col int, r industrySlotRow) []widget.RichTextSegment) *widget.Table {
	w := widget.NewTable(
		func() (rows int, cols int) {
			return len(a.rowsFiltered), 4
		},
		func() fyne.CanvasObject {
			return xwidget.NewRichText()
		},
		func(tci widget.TableCellID, co fyne.CanvasObject) {
			if tci.Row >= len(a.rowsFiltered) {
				return
			}
			r := a.rowsFiltered[tci.Row]
			co.(*xwidget.RichText).Set(makeCell(tci.Col, r))
		},
	)
	w.ShowHeaderRow = true
	w.StickyColumnCount = 1
	w.CreateHeader = func() fyne.CanvasObject {
		return widget.NewLabel("")
	}
	w.UpdateHeader = func(tci widget.TableCellID, co fyne.CanvasObject) {
		if col, ok := headers.ColumnByIndex(tci.Col); ok {
			co.(*widget.Label).SetText(col.Label)
		}
	}
	for id, width := range map[int]float32{
		0: 175,
		1: 50,
		2: 50,
		3: 50,
	} {
		w.SetColumnWidth(id, width)
	}
	return w
}

func (a *IndustrySlots) filterRowsAsync(sortCol string) {
	isLatest := a.filterRun.start()
	totalRows := len(a.rows)
	rows := slices.Clone(a.rows)
	filter := a.currentFilter()
	sortCol, dir, doSort := a.columnSorter.CalcSort(sortCol)

	runAsync(func() {
		rows := slices.Clone(rows)
		rows = slices.DeleteFunc(rows, func(r industrySlotRow) bool {
			return !filter.match(r)
		})
		a.columnSorter.SortRows(rows, sortCol, dir, doSort)

		footer := fmt.Sprintf("Showing %d / %d characters", len(rows), totalRows)
		// add totals
		var busy, ready, free, total int
		for _, r := range rows {
			busy += r.busy
			ready += r.ready
			free += r.free
			total += r.total
		}
		rows = append(rows, industrySlotRow{
			busy:          busy,
			characterName: "Total",
			free:          free,
			isTotal:       true,
			ready:         ready,
			total:         total,
		})

		tagOptions := slices.Sorted(set.Union(xslices.Map(rows, func(r industrySlotRow) set.Set[string] {
			return r.tags
		})...).All())

		fyne.Do(func() {
			if !isLatest() {
				return
			}
			a.footer.Text = footer
			a.footer.Importance = widget.MediumImportance
			a.footer.Refresh()
			if a.filterChip != nil {
				a.filterChip.SetOptions(
					xwidget.NewFilterOptionMultiChoiceOrdered(industrySlotsFilterFreeSlots, industrySlotsFreeSlotsOptions()),
					xwidget.NewFilterOptionMultiChoice(industrySlotsFilterTag, tagOptions),
				)
			} else {
				a.selectTag.SetOptions(tagOptions)
			}
			a.rowsFiltered = rows
			a.body.Refresh()
		})
	})
}

func (a *IndustrySlots) update(ctx context.Context) {
	rows, err := a.fetchData(ctx, a.slotType)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		slog.Error("Failed to refresh industrySlots UI", "err", err)
		fyne.Do(func() {
			a.footer.Text = "ERROR: " + a.u.ErrorDisplay(err)
			a.footer.Importance = widget.DangerImportance
			a.footer.Refresh()
		})
		return
	}
	fyne.Do(func() {
		a.rows = rows
		a.filterRowsAsync("")
	})
}

func (a *IndustrySlots) fetchData(ctx context.Context, slotType app.IndustryJobType) ([]industrySlotRow, error) {
	oo, err := a.u.Character().ListAllCharactersIndustrySlots(ctx, slotType)
	if err != nil {
		return nil, err
	}
	var rows []industrySlotRow
	for _, o := range oo {
		r := industrySlotRow{
			characterID:   o.CharacterID,
			characterName: o.CharacterName,
			busy:          o.Busy,
			ready:         o.Ready,
			free:          o.Free,
			total:         o.Total,
		}
		tags, err := a.u.Character().ListTagsForCharacter(ctx, o.CharacterID)
		if err != nil {
			return nil, err
		}
		r.tags = tags
		rows = append(rows, r)
	}
	return rows, nil
}
