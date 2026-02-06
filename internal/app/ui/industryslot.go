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

func (r industrySlotRow) characterDisplay() []widget.RichTextSegment {
	if r.isSummary {
		return iwidget.RichTextSegmentsFromText("Totals", widget.RichTextStyle{
			TextStyle: fyne.TextStyle{Bold: true},
		})
	}
	return iwidget.RichTextSegmentsFromText(r.characterName)
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

type industrySlots struct {
	widget.BaseWidget

	body            fyne.CanvasObject
	bottom          *widget.Label
	columnSorter    *iwidget.ColumnSorter[industrySlotRow]
	rows            []industrySlotRow
	rowsFiltered    []industrySlotRow
	selectFreeSlots *kxwidget.FilterChipSelect
	selectTag       *kxwidget.FilterChipSelect
	slotType        app.IndustryJobType
	sortButton      *iwidget.SortButton
	u               *baseUI
}

const (
	industrySlotsColCharacter = iota + 1
	industrySlotsColBusy
	industrySlotsColReady
	industrySlotsColFree
	industrySlotsColTotal
)

func newIndustrySlots(u *baseUI, slotType app.IndustryJobType) *industrySlots {
	const columnWidthNumber = 75
	columns := iwidget.NewDataColumns([]iwidget.DataColumn[industrySlotRow]{{
		ID:    industrySlotsColCharacter,
		Label: "Character",
		Width: columnWidthEntity,
		Sort: func(a, b industrySlotRow) int {
			return xstrings.CompareIgnoreCase(a.characterName, b.characterName)
		},
		Update: func(r industrySlotRow, co fyne.CanvasObject) {
			co.(*iwidget.RichText).Set(r.characterDisplay())
		},
	}, {
		ID:    industrySlotsColBusy,
		Label: "Busy",
		Width: columnWidthNumber,
		Sort: func(a, b industrySlotRow) int {
			return cmp.Compare(a.busy, b.busy)
		},
		Update: func(r industrySlotRow, co fyne.CanvasObject) {
			co.(*iwidget.RichText).SetWithText(fmt.Sprint(r.busy), widget.RichTextStyle{
				Alignment: fyne.TextAlignTrailing,
				ColorName: r.busyColor(),
				TextStyle: fyne.TextStyle{Bold: r.isSummary},
			})
		},
	}, {
		ID:    industrySlotsColReady,
		Label: "Ready",
		Width: columnWidthNumber,
		Sort: func(a, b industrySlotRow) int {
			return cmp.Compare(a.ready, b.ready)
		},
		Update: func(r industrySlotRow, co fyne.CanvasObject) {
			co.(*iwidget.RichText).SetWithText(fmt.Sprint(r.ready), widget.RichTextStyle{
				Alignment: fyne.TextAlignTrailing,
				ColorName: r.readyColor(),
				TextStyle: fyne.TextStyle{Bold: r.isSummary},
			})
		},
	}, {
		ID:    industrySlotsColFree,
		Label: "Free",
		Width: columnWidthNumber,
		Sort: func(a, b industrySlotRow) int {
			return cmp.Compare(a.free, b.free)
		},
		Update: func(r industrySlotRow, co fyne.CanvasObject) {
			co.(*iwidget.RichText).SetWithText(fmt.Sprint(r.free), widget.RichTextStyle{
				Alignment: fyne.TextAlignTrailing,
				ColorName: r.freeColor(),
				TextStyle: fyne.TextStyle{Bold: r.isSummary},
			})
		},
	}, {
		ID:    industrySlotsColTotal,
		Label: "Total",
		Width: columnWidthNumber,
		Sort: func(a, b industrySlotRow) int {
			return cmp.Compare(a.total, b.total)
		},
		Update: func(r industrySlotRow, co fyne.CanvasObject) {
			co.(*iwidget.RichText).SetWithText(fmt.Sprint(r.total), widget.RichTextStyle{
				Alignment: fyne.TextAlignTrailing,
				TextStyle: fyne.TextStyle{Bold: r.isSummary},
			})
		},
	}})
	a := &industrySlots{
		bottom:       makeTopLabel(),
		columnSorter: iwidget.NewColumnSorter(columns, industrySlotsColCharacter, iwidget.SortAsc),
		rows:         make([]industrySlotRow, 0),
		rowsFiltered: make([]industrySlotRow, 0),
		slotType:     slotType,
		u:            u,
	}
	a.ExtendBaseWidget(a)
	if !a.u.isMobile {
		a.body = iwidget.MakeDataTable(
			columns,
			&a.rowsFiltered,
			func() fyne.CanvasObject {
				x := iwidget.NewRichText()
				x.Truncation = fyne.TextTruncateClip
				return x
			},
			a.columnSorter,
			a.filterRows,
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
					return iwidget.RichTextSegmentsFromText(fmt.Sprint(r.busy), widget.RichTextStyle{
						Alignment: fyne.TextAlignTrailing,
						ColorName: r.busyColor(),
						TextStyle: fyne.TextStyle{Bold: r.isSummary},
					})
				case industrySlotsColReady:
					return iwidget.RichTextSegmentsFromText(fmt.Sprint(r.ready), widget.RichTextStyle{
						Alignment: fyne.TextAlignTrailing,
						ColorName: r.readyColor(),
						TextStyle: fyne.TextStyle{Bold: r.isSummary},
					})
				case industrySlotsColFree:
					return iwidget.RichTextSegmentsFromText(fmt.Sprint(r.free), widget.RichTextStyle{
						Alignment: fyne.TextAlignTrailing,
						ColorName: r.freeColor(),
						TextStyle: fyne.TextStyle{Bold: r.isSummary},
					})
				case industrySlotsColTotal:
					return iwidget.RichTextSegmentsFromText(fmt.Sprint(r.total), widget.RichTextStyle{
						Alignment: fyne.TextAlignTrailing,
						TextStyle: fyne.TextStyle{Bold: r.isSummary},
					})
				}
				return iwidget.RichTextSegmentsFromText("?")
			},
		)
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
	a.u.characterAdded.AddListener(func(_ context.Context, _ *app.Character) {
		a.update()
	})
	a.u.characterRemoved.AddListener(func(_ context.Context, _ *app.EntityShort[int32]) {
		a.update()
	})
	a.u.tagsChanged.AddListener(func(ctx context.Context, s struct{}) {
		a.update()
	})
	return a
}

func (a *industrySlots) CreateRenderer() fyne.WidgetRenderer {
	filter := container.NewHBox(a.selectFreeSlots, a.selectTag)
	if a.u.isMobile {
		filter.Add(a.sortButton)
	}
	c := container.NewBorder(container.NewHScroll(filter), a.bottom, nil, nil, a.body)
	return widget.NewSimpleRenderer(c)
}

func (a *industrySlots) makeDataTable(headers iwidget.DataColumns[industrySlotRow], makeCell func(col int, r industrySlotRow) []widget.RichTextSegment) *widget.Table {
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

func (a *industrySlots) filterRows(sortCol int) {
	rows := slices.Clone(a.rows)
	freeSlots := a.selectFreeSlots.Selected
	tag := a.selectTag.Selected
	sortCol, dir, doSort := a.columnSorter.CalcSort(sortCol)

	go func() {
		rows := slices.Clone(rows)
		// filter
		if freeSlots != "" {
			rows = slices.DeleteFunc(rows, func(r industrySlotRow) bool {
				switch freeSlots {
				case slotsFreeSome:
					return r.free == 0
				case slotsFreeNone:
					return r.free > 0
				}
				return true
			})
		}
		if tag != "" {
			rows = slices.DeleteFunc(rows, func(r industrySlotRow) bool {
				return !r.tags.Contains(tag)
			})
		}
		a.columnSorter.SortRows(rows, sortCol, dir, doSort)
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
		tagOptions := slices.Sorted(set.Union(xslices.Map(rows, func(r industrySlotRow) set.Set[string] {
			return r.tags
		})...).All())

		fyne.Do(func() {
			a.selectTag.SetOptions(tagOptions)
			a.rowsFiltered = rows
			a.body.Refresh()
		})
	}()
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
