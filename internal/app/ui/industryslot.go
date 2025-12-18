package ui

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
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
)

const (
	slotsFreeSome = "Has free slots"
	slotsFreeNone = "No free slots"
)

type industrySlotRow struct {
	characterName string
	busy          int
	ready         int
	free          int
	total         int
	isSummary     bool
	tags          set.Set[string]
}

type industrySlots struct {
	widget.BaseWidget

	body            fyne.CanvasObject
	bottom          *widget.Label
	columnSorter    *iwidget.ColumnSorter
	rows            []industrySlotRow
	rowsFiltered    []industrySlotRow
	selectFreeSlots *kxwidget.FilterChipSelect
	selectTag       *kxwidget.FilterChipSelect
	slotType        app.IndustryJobType
	sortButton      *iwidget.SortButton
	u               *baseUI
}

const (
	industrySlotsColCharacter = 0
	industrySlotsColBusy      = 1
	industrySlotsColReady     = 2
	industrySlotsColFree      = 3
	industrySlotsColTotal     = 4
)

func newIndustrySlots(u *baseUI, slotType app.IndustryJobType) *industrySlots {
	const columnWidthNumber = 75
	headers := iwidget.NewDataTableDef([]iwidget.ColumnDef{{
		Col:   industrySlotsColCharacter,
		Label: "Character",
		Width: columnWidthEntity,
	}, {
		Col:   industrySlotsColBusy,
		Label: "Busy",
		Width: columnWidthNumber,
	}, {
		Col:   industrySlotsColReady,
		Label: "Ready",
		Width: columnWidthNumber,
	}, {
		Col:   industrySlotsColFree,
		Label: "Free",
		Width: columnWidthNumber,
	}, {
		Col:   industrySlotsColTotal,
		Label: "Total",
		Width: columnWidthNumber,
	}})
	a := &industrySlots{
		bottom:       makeTopLabel(),
		columnSorter: headers.NewColumnSorter(industrySlotsColCharacter, iwidget.SortAsc),
		rows:         make([]industrySlotRow, 0),
		rowsFiltered: make([]industrySlotRow, 0),
		slotType:     slotType,
		u:            u,
	}
	a.ExtendBaseWidget(a)
	makeCell := func(col int, r industrySlotRow) []widget.RichTextSegment {
		switch col {
		case industrySlotsColCharacter:
			if r.isSummary {
				return iwidget.RichTextSegmentsFromText("Totals", widget.RichTextStyle{
					TextStyle: fyne.TextStyle{Bold: true},
				})
			}
			return iwidget.RichTextSegmentsFromText(r.characterName)
		case industrySlotsColBusy:
			var c fyne.ThemeColorName
			switch r.busy {
			case 0:
				c = theme.ColorNameSuccess
			case r.total:
				c = theme.ColorNameError
			default:
				c = theme.ColorNameWarning
			}
			return iwidget.RichTextSegmentsFromText(fmt.Sprint(r.busy), widget.RichTextStyle{
				Alignment: fyne.TextAlignTrailing,
				ColorName: c,
				TextStyle: fyne.TextStyle{Bold: r.isSummary},
			})
		case industrySlotsColReady:
			var c fyne.ThemeColorName
			switch {
			case r.ready > 0:
				c = theme.ColorNameWarning
			case r.ready == 0:
				c = theme.ColorNameSuccess
			default:
				c = theme.ColorNameForeground
			}
			return iwidget.RichTextSegmentsFromText(fmt.Sprint(r.ready), widget.RichTextStyle{
				Alignment: fyne.TextAlignTrailing,
				ColorName: c,
				TextStyle: fyne.TextStyle{Bold: r.isSummary},
			})
		case industrySlotsColFree:
			var c fyne.ThemeColorName
			switch {
			case r.free == r.total:
				c = theme.ColorNameSuccess
			case r.free > 0:
				c = theme.ColorNameWarning
			case r.free == 0:
				c = theme.ColorNameError
			}
			return iwidget.RichTextSegmentsFromText(fmt.Sprint(r.free), widget.RichTextStyle{
				Alignment: fyne.TextAlignTrailing,
				ColorName: c,
				TextStyle: fyne.TextStyle{Bold: r.isSummary},
			})
		case industrySlotsColTotal:
			return iwidget.RichTextSegmentsFromText(fmt.Sprint(r.total), widget.RichTextStyle{
				Alignment: fyne.TextAlignTrailing,
				TextStyle: fyne.TextStyle{Bold: r.isSummary},
			})
		}
		return iwidget.RichTextSegmentsFromText("?")
	}
	if a.u.isDesktop {
		a.body = iwidget.MakeDataTable(
			headers,
			&a.rowsFiltered,
			makeCell,
			a.columnSorter,
			a.filterRows,
			nil,
		)
	} else {
		a.body = a.makeDataTable(headers, makeCell)
	}

	a.selectFreeSlots = kxwidget.NewFilterChipSelect("Free slots", []string{
		slotsFreeSome,
		slotsFreeNone,
	}, func(string) {
		a.filterRows(-1)
	})
	a.selectTag = kxwidget.NewFilterChipSelect("Tag", []string{}, func(string) {
		a.filterRows(-1)
	})
	a.sortButton = a.columnSorter.NewSortButton(func() {
		a.filterRows(-1)
	}, a.u.window)

	a.u.characterSectionChanged.AddListener(func(_ context.Context, arg characterSectionUpdated) {
		switch arg.section {
		case app.SectionCharacterIndustryJobs, app.SectionCharacterSkills:
			a.update()
		}
	})
	a.u.corporationSectionChanged.AddListener(func(_ context.Context, arg corporationSectionUpdated) {
		if arg.section == app.SectionCorporationIndustryJobs {
			a.update()
		}
	})
	return a
}

func (a *industrySlots) CreateRenderer() fyne.WidgetRenderer {
	filter := container.NewHBox(a.selectFreeSlots, a.selectTag)
	if !a.u.isDesktop {
		filter.Add(a.sortButton)
	}
	c := container.NewBorder(container.NewHScroll(filter), a.bottom, nil, nil, a.body)
	return widget.NewSimpleRenderer(c)
}

func (a *industrySlots) makeDataTable(headers iwidget.DataTableDef, makeCell func(col int, r industrySlotRow) []widget.RichTextSegment) *widget.Table {
	w := widget.NewTable(
		func() (rows int, cols int) {
			return len(a.rowsFiltered), 4
		},
		func() fyne.CanvasObject {
			return iwidget.NewRichText()
		},
		func(tci widget.TableCellID, co fyne.CanvasObject) {
			if tci.Row >= len(a.rowsFiltered) {
				return
			}
			r := a.rowsFiltered[tci.Row]
			co.(*iwidget.RichText).Set(makeCell(tci.Col, r))
		},
	)
	w.ShowHeaderRow = true
	w.StickyColumnCount = 1
	w.CreateHeader = func() fyne.CanvasObject {
		return widget.NewLabel("")
	}
	w.UpdateHeader = func(tci widget.TableCellID, co fyne.CanvasObject) {
		co.(*widget.Label).SetText(headers.Column(tci.Col).Label)
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

func (a *industrySlots) filterRows(sortCol int) {
	rows := slices.Clone(a.rows)
	// filter
	if x := a.selectFreeSlots.Selected; x != "" {
		rows = xslices.Filter(rows, func(r industrySlotRow) bool {
			switch x {
			case slotsFreeSome:
				return r.free > 0
			case slotsFreeNone:
				return r.free == 0
			}
			return false
		})
	}
	if x := a.selectTag.Selected; x != "" {
		rows = xslices.Filter(rows, func(r industrySlotRow) bool {
			return r.tags.Contains(x)
		})
	}
	// sort
	a.columnSorter.Sort(sortCol, func(sortCol int, dir iwidget.SortDir) {
		slices.SortFunc(rows, func(a, b industrySlotRow) int {
			var x int
			switch sortCol {
			case industrySlotsColCharacter:
				x = xstrings.CompareIgnoreCase(a.characterName, b.characterName)
			case industrySlotsColBusy:
				x = cmp.Compare(a.busy, b.busy)
			case industrySlotsColReady:
				x = cmp.Compare(a.ready, b.ready)
			case industrySlotsColFree:
				x = cmp.Compare(a.free, b.free)
			case industrySlotsColTotal:
				x = cmp.Compare(a.total, b.total)
			}
			if dir == iwidget.SortAsc {
				return x
			} else {
				return -1 * x
			}
		})
	})
	// add totals
	var active, ready, free, total int
	for _, r := range rows {
		active += r.busy
		ready += r.ready
		free += r.free
		total += r.total
	}
	rows = append(rows, industrySlotRow{
		busy:      active,
		ready:     ready,
		free:      free,
		total:     total,
		isSummary: true,
	})
	// set data & refresh
	a.selectTag.SetOptions(slices.Sorted(set.Union(xslices.Map(rows, func(r industrySlotRow) set.Set[string] {
		return r.tags
	})...).All()))
	a.rowsFiltered = rows
	a.body.Refresh()
}

func (a *industrySlots) update() {
	rows := make([]industrySlotRow, 0)
	t, i, err := func() (string, widget.Importance, error) {
		rr, err := a.fetchData(a.u.services(), a.slotType)
		if err != nil {
			return "", 0, err
		}
		if len(rr) == 0 {
			return "No characters", widget.LowImportance, nil
		}
		rows = rr
		return "", widget.MediumImportance, nil
	}()
	if err != nil {
		slog.Error("Failed to refresh industrySlots UI", "err", err)
		t = "ERROR: " + a.u.humanizeError(err)
		i = widget.DangerImportance
	}
	fyne.Do(func() {
		if t != "" {
			a.bottom.Text = t
			a.bottom.Importance = i
			a.bottom.Refresh()
			a.bottom.Show()
		} else {
			a.bottom.Hide()
		}
	})
	fyne.Do(func() {
		a.rows = rows
		a.filterRows(-1)
	})
}

func (*industrySlots) fetchData(s services, slotType app.IndustryJobType) ([]industrySlotRow, error) {
	ctx := context.Background()
	oo, err := s.cs.ListAllCharactersIndustrySlots(ctx, slotType)
	if err != nil {
		return nil, err
	}
	rows := make([]industrySlotRow, 0)
	for _, o := range oo {
		r := industrySlotRow{
			characterName: o.CharacterName,
			busy:          o.Busy,
			ready:         o.Ready,
			free:          o.Free,
			total:         o.Total,
		}
		tags, err := s.cs.ListTagsForCharacter(ctx, o.CharacterID)
		if err != nil {
			return nil, err
		}
		r.tags = tags
		rows = append(rows, r)
	}
	return rows, nil
}
