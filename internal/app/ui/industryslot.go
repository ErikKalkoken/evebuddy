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
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/app/uiservices"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

const (
	slotsFreeSome = "Has free slots"
	slotsFreeNone = "No free slots"
)

type industrySlotRow struct {
	characterID   int64
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
	footer          *widget.Label
	columnSorter    *xwidget.ColumnSorter[industrySlotRow]
	rows            []industrySlotRow
	rowsFiltered    []industrySlotRow
	selectFreeSlots *kxwidget.FilterChipSelect
	selectTag       *kxwidget.FilterChipSelect
	slotType        app.IndustryJobType
	sortButton      *xwidget.SortButton
	u         uiservices.UIServices
}

const (
	industrySlotsColCharacter = iota + 1
	industrySlotsColBusy
	industrySlotsColReady
	industrySlotsColFree
	industrySlotsColTotal
)

func NewIndustrySlots(u         uiservices.UIServices, slotType app.IndustryJobType) *IndustrySlots {
	const columnWidthNumber = 75
	columns := xwidget.NewDataColumns([]xwidget.DataColumn[industrySlotRow]{{
		ID:    industrySlotsColCharacter,
		Label: "Character",
		Width: columnWidthEntity,
		Sort: func(a, b industrySlotRow) int {
			return xstrings.CompareIgnoreCase(a.characterName, b.characterName)
		},
		Create: func() fyne.CanvasObject {
			icon := xwidget.NewImageFromResource(
				icons.Characterplaceholder64Jpeg,
				fyne.NewSquareSize(app.IconUnitSize),
			)
			icon.CornerRadius = app.IconUnitSize / 2
			name := widget.NewLabel("Template")
			name.Truncation = fyne.TextTruncateClip
			return container.NewBorder(nil, nil, icon, nil, name)
		},
		Update: func(r industrySlotRow, co fyne.CanvasObject) {
			border := co.(*fyne.Container).Objects
			label := border[0].(*widget.Label)
			icon := border[1].(*canvas.Image)
			if r.isSummary {
				label.Text = "Total"
				label.TextStyle = fyne.TextStyle{Bold: true}
				label.Refresh()
				icon.Resource = icons.BlankSvg
				icon.Refresh()
			} else {
				label.Text = r.characterName
				label.TextStyle = fyne.TextStyle{}
				label.Refresh()
				u.EVEImage().CharacterPortraitAsync(r.characterID, app.IconPixelSize, func(r fyne.Resource) {
					icon.Resource = r
					icon.Refresh()
				})
			}
		},
	}, {
		ID:    industrySlotsColBusy,
		Label: "Busy",
		Width: columnWidthNumber,
		Sort: func(a, b industrySlotRow) int {
			return cmp.Compare(a.busy, b.busy)
		},
		Update: func(r industrySlotRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(fmt.Sprint(r.busy), widget.RichTextStyle{
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
			co.(*xwidget.RichText).SetWithText(fmt.Sprint(r.ready), widget.RichTextStyle{
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
			co.(*xwidget.RichText).SetWithText(fmt.Sprint(r.free), widget.RichTextStyle{
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
			co.(*xwidget.RichText).SetWithText(fmt.Sprint(r.total), widget.RichTextStyle{
				Alignment: fyne.TextAlignTrailing,
				TextStyle: fyne.TextStyle{Bold: r.isSummary},
			})
		},
	}})
	a := &IndustrySlots{
		footer:       newLabelWithWrapping(),
		columnSorter: xwidget.NewColumnSorter(columns, industrySlotsColCharacter, xwidget.SortAsc),
		slotType:     slotType,
		u:            u,
	}
	a.ExtendBaseWidget(a)
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
						TextStyle: fyne.TextStyle{Bold: r.isSummary},
					})
				case industrySlotsColReady:
					return xwidget.RichTextSegmentsFromText(fmt.Sprint(r.ready), widget.RichTextStyle{
						Alignment: fyne.TextAlignTrailing,
						ColorName: r.readyColor(),
						TextStyle: fyne.TextStyle{Bold: r.isSummary},
					})
				case industrySlotsColFree:
					return xwidget.RichTextSegmentsFromText(fmt.Sprint(r.free), widget.RichTextStyle{
						Alignment: fyne.TextAlignTrailing,
						ColorName: r.freeColor(),
						TextStyle: fyne.TextStyle{Bold: r.isSummary},
					})
				case industrySlotsColTotal:
					return xwidget.RichTextSegmentsFromText(fmt.Sprint(r.total), widget.RichTextStyle{
						Alignment: fyne.TextAlignTrailing,
						TextStyle: fyne.TextStyle{Bold: r.isSummary},
					})
				}
				return xwidget.RichTextSegmentsFromText("?")
			},
		)
	}

	a.selectFreeSlots = kxwidget.NewFilterChipSelect("Free slots", []string{
		slotsFreeSome,
		slotsFreeNone,
	}, func(string) {
		a.filterRowsAsync(-1)
	})
	a.selectTag = kxwidget.NewFilterChipSelect("Tag", []string{}, func(string) {
		a.filterRowsAsync(-1)
	})
	a.sortButton = a.columnSorter.NewSortButton(func() {
		a.filterRowsAsync(-1)
	}, a.u.MainWindow())

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
	a.u.Signals().TagsChanged.AddListener(func(ctx context.Context, s struct{}) {
		a.update(ctx)
	})
	return a
}

func (a *IndustrySlots) CreateRenderer() fyne.WidgetRenderer {
	filter := container.NewHBox(a.selectFreeSlots, a.selectTag)
	if app.IsMobile() {
		filter.Add(a.sortButton)
	}
	c := container.NewBorder(container.NewHScroll(filter), a.footer, nil, nil, a.body)
	return widget.NewSimpleRenderer(c)
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
			id, ok := headers.IDLookup(tci.Col)
			if !ok {
				return
			}
			r := a.rowsFiltered[tci.Row]
			co.(*xwidget.RichText).Set(makeCell(id, r))
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

func (a *IndustrySlots) filterRowsAsync(sortCol int) {
	totalRows := len(a.rows)
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

		footer := fmt.Sprintf("Showing %d / %d characters", len(rows), totalRows)
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

		tagOptions := slices.Sorted(set.Union(xslices.Map(rows, func(r industrySlotRow) set.Set[string] {
			return r.tags
		})...).All())

		fyne.Do(func() {
			a.footer.Text = footer
			a.footer.Importance = widget.MediumImportance
			a.footer.Refresh()
			a.selectTag.SetOptions(tagOptions)
			a.rowsFiltered = rows
			a.body.Refresh()
		})
	}()
}

func (a *IndustrySlots) update(ctx context.Context) {
	rows, err := a.fetchData(ctx, a.slotType)
	if err != nil {
		slog.Error("Failed to refresh industrySlots UI", "err", err)
		fyne.Do(func() {
			a.footer.Text = "ERROR: " + app.ErrorDisplay(err)
			a.footer.Importance = widget.DangerImportance
			a.footer.Refresh()
		})
		return
	}
	fyne.Do(func() {
		a.rows = rows
		a.filterRowsAsync(-1)
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
