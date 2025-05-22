package ui

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
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
}

type industrySlots struct {
	widget.BaseWidget

	body            fyne.CanvasObject
	bottom          *widget.Label
	columnSorter    *columnSorter
	rows            []industrySlotRow
	rowsFiltered    []industrySlotRow
	totals          industrySlotRow
	selectFreeSlots *iwidget.FilterChipSelect
	slotType        app.IndustrySlotType
	sortButton      *sortButton
	u               *baseUI
}

func newIndustrySlots(u *baseUI, slotType app.IndustrySlotType) *industrySlots {
	const columnWidthNumber = 75
	headers := []headerDef{
		{Text: "Character", Width: columnWidthCharacter},
		{Text: "Busy", Width: columnWidthNumber},
		{Text: "Ready", Width: columnWidthNumber},
		{Text: "Free", Width: columnWidthNumber},
		{Text: "Total", Width: columnWidthNumber},
	}
	a := &industrySlots{
		bottom:       makeTopLabel(),
		columnSorter: newColumnSorterWithInit(headers, 0, sortAsc),
		rows:         make([]industrySlotRow, 0),
		rowsFiltered: make([]industrySlotRow, 0),
		slotType:     slotType,
		u:            u,
	}
	a.ExtendBaseWidget(a)
	makeCell := func(col int, r industrySlotRow) []widget.RichTextSegment {
		switch col {
		case 0:
			if r.isSummary {
				return iwidget.NewRichTextSegmentFromText("Totals", widget.RichTextStyle{
					TextStyle: fyne.TextStyle{Bold: true},
				})
			}
			return iwidget.NewRichTextSegmentFromText(r.characterName)
		case 1:
			return iwidget.NewRichTextSegmentFromText(fmt.Sprint(r.busy), widget.RichTextStyle{
				Alignment: fyne.TextAlignTrailing,
				TextStyle: fyne.TextStyle{Bold: r.isSummary},
			})
		case 2:
			var c fyne.ThemeColorName
			switch {
			case r.ready > 0:
				c = theme.ColorNamePrimary
			default:
				c = theme.ColorNameForeground
			}
			return iwidget.NewRichTextSegmentFromText(fmt.Sprint(r.ready), widget.RichTextStyle{
				Alignment: fyne.TextAlignTrailing,
				ColorName: c,
				TextStyle: fyne.TextStyle{Bold: r.isSummary},
			})
		case 3:
			var c fyne.ThemeColorName
			switch {
			case r.free == r.total:
				c = theme.ColorNameSuccess
			case r.free > 0:
				c = theme.ColorNameWarning
			case r.free == 0:
				c = theme.ColorNameError
			}
			return iwidget.NewRichTextSegmentFromText(fmt.Sprint(r.free), widget.RichTextStyle{
				Alignment: fyne.TextAlignTrailing,
				ColorName: c,
				TextStyle: fyne.TextStyle{Bold: r.isSummary},
			})
		case 4:
			return iwidget.NewRichTextSegmentFromText(fmt.Sprint(r.total), widget.RichTextStyle{
				Alignment: fyne.TextAlignTrailing,
				TextStyle: fyne.TextStyle{Bold: r.isSummary},
			})
		}
		return iwidget.NewRichTextSegmentFromText("?")
	}
	if a.u.isDesktop {
		a.body = makeDataTable(
			headers,
			&a.rowsFiltered,
			makeCell,
			a.columnSorter,
			a.filterRows,
			nil,
		)
	} else {
		a.body = a.makeDataList(headers, makeCell)
	}

	a.selectFreeSlots = iwidget.NewFilterChipSelect("Free slots", []string{
		slotsFreeSome,
		slotsFreeNone,
	}, func(string) {
		a.filterRows(-1)
	},
	)

	a.sortButton = a.columnSorter.newSortButton(headers, func() {
		a.filterRows(-1)
	}, a.u.window,
	)
	return a
}

func (a *industrySlots) CreateRenderer() fyne.WidgetRenderer {
	filter := container.NewHBox(a.selectFreeSlots)
	if !a.u.isDesktop {
		filter.Add(a.sortButton)
	}
	c := container.NewBorder(container.NewHScroll(filter), a.bottom, nil, nil, a.body)
	return widget.NewSimpleRenderer(c)
}

func (a *industrySlots) makeDataList(headers []headerDef, makeCell func(col int, r industrySlotRow) []widget.RichTextSegment) *widget.Table {
	w := widget.NewTable(
		func() (rows int, cols int) {
			return len(a.rowsFiltered), 4
		},
		func() fyne.CanvasObject {
			return widget.NewRichText()
		},
		func(tci widget.TableCellID, co fyne.CanvasObject) {
			if tci.Row >= len(a.rowsFiltered) {
				return
			}
			r := a.rowsFiltered[tci.Row]
			iwidget.SetRichText(co.(*widget.RichText), makeCell(tci.Col, r)...)
		},
	)
	w.ShowHeaderRow = true
	w.StickyColumnCount = 1
	w.CreateHeader = func() fyne.CanvasObject {
		return widget.NewLabel("")
	}
	w.UpdateHeader = func(tci widget.TableCellID, co fyne.CanvasObject) {
		co.(*widget.Label).SetText(headers[tci.Col].Text)
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
	// sort
	a.columnSorter.sort(sortCol, func(sortCol int, dir sortDir) {
		slices.SortFunc(rows, func(a, b industrySlotRow) int {
			var x int
			switch sortCol {
			case 0:
				x = strings.Compare(a.characterName, b.characterName)
			case 1:
				x = cmp.Compare(a.busy, b.busy)
			case 2:
				x = cmp.Compare(a.ready, b.ready)
			case 3:
				x = cmp.Compare(a.free, b.free)
			case 4:
				x = cmp.Compare(a.total, b.total)
			}
			if dir == sortAsc {
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
		total += r.free
	}
	rows = append(rows, industrySlotRow{
		busy:      active,
		ready:     ready,
		free:      free,
		total:     total,
		isSummary: true,
	})
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

func (*industrySlots) fetchData(s services, slotType app.IndustrySlotType) ([]industrySlotRow, error) {
	oo, err := s.cs.ListAllCharactersIndustrySlots(context.Background(), slotType)
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
		rows = append(rows, r)
	}
	return rows, nil
}
