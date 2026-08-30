package contracts

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

// TODO: split into views for personal and corporation slots

const (
	slotsFreeSome = "Has free slots"
	slotsFreeNone = "No free slots"
)

type slotRow struct {
	characterID     int64
	characterName   string
	corporationID   int64
	corporationName string
	free            int
	isTotal         bool
	tags            set.Set[string]
	total           int
	used            int
}

func (r slotRow) characterDisplay() []widget.RichTextSegment {
	if r.isTotal {
		return xwidget.RichTextSegmentsFromText("Totals", widget.RichTextStyle{
			TextStyle: fyne.TextStyle{Bold: true},
		})
	}
	return xwidget.RichTextSegmentsFromText(r.characterName)
}

func (r slotRow) usedColor() fyne.ThemeColorName {
	var c fyne.ThemeColorName
	switch r.used {
	case 0:
		c = theme.ColorNameSuccess
	case r.total:
		c = theme.ColorNameError
	default:
		c = theme.ColorNameWarning
	}
	return c
}

func (r slotRow) freeColor() fyne.ThemeColorName {
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

// Slots is a view that shows the used and available contract capacity for all characters.
type Slots struct {
	widget.BaseWidget

	body              fyne.CanvasObject
	columnSorter      *xwidget.ColumnSorter[slotRow]
	corporationSlots  bool
	footer            *widget.Label
	rows              []slotRow
	rowsFiltered      []slotRow
	selectCorporation *kxwidget.FilterChipSelect
	selectFreeSlots   *kxwidget.FilterChipSelect
	selectTag         *kxwidget.FilterChipSelect
	sortChip *kxwidget.SortChip
	u                 baseUI
}

const (
	overviewColCharacter = iota + 1
	overviewColPersonal
	overviewColCorporation
	overviewColFree
	overviewColTotal
)

func NewSlots(u baseUI, corporationSlots bool) *Slots {
	const columnWidth = 75
	columns := xwidget.NewDataColumns([]xwidget.DataColumn[slotRow]{
		ui.MakeEveEntityColumn(ui.MakeEveEntityColumnParams[slotRow]{
			ColumnID: overviewColCharacter,
			EIS:      u.EVEImage(),
			GetEntity: func(r slotRow) *app.EveEntity {
				return &app.EveEntity{
					ID:       r.characterID,
					Name:     r.characterName,
					Category: app.EveEntityCharacter,
				}
			},
			IsAvatar: true,
			Label:    "Character",
		}), {
			ID:    overviewColPersonal,
			Label: "Used",
			Width: columnWidth,
			Sort: func(a, b slotRow) int {
				return cmp.Compare(a.used, b.used)
			},
			Update: func(r slotRow, co fyne.CanvasObject) {
				co.(*xwidget.RichText).SetWithText(fmt.Sprint(r.used), widget.RichTextStyle{
					Alignment: fyne.TextAlignTrailing,
					ColorName: r.usedColor(),
					TextStyle: fyne.TextStyle{Bold: r.isTotal},
				})
			},
		}, {
			ID:    overviewColFree,
			Label: "Free",
			Width: columnWidth,
			Sort: func(a, b slotRow) int {
				return cmp.Compare(a.free, b.free)
			},
			Update: func(r slotRow, co fyne.CanvasObject) {
				co.(*xwidget.RichText).SetWithText(fmt.Sprint(r.free), widget.RichTextStyle{
					Alignment: fyne.TextAlignTrailing,
					ColorName: r.freeColor(),
					TextStyle: fyne.TextStyle{Bold: r.isTotal},
				})
			},
		}, {
			ID:    overviewColTotal,
			Label: "Total",
			Width: columnWidth,
			Sort: func(a, b slotRow) int {
				return cmp.Compare(a.total, b.total)
			},
			Update: func(r slotRow, co fyne.CanvasObject) {
				co.(*xwidget.RichText).SetWithText(fmt.Sprint(r.total), widget.RichTextStyle{
					Alignment: fyne.TextAlignTrailing,
					TextStyle: fyne.TextStyle{Bold: r.isTotal},
				})
			},
		}})
	a := &Slots{
		columnSorter:     xwidget.NewColumnSorter(columns, overviewColCharacter, xwidget.SortAsc),
		corporationSlots: corporationSlots,
		footer:           ui.NewLabelWithWrapping(""),
		u:                u,
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
			func(col int, r slotRow) []widget.RichTextSegment {
				switch col {
				case overviewColCharacter:
					return r.characterDisplay()
				case overviewColPersonal:
					return xwidget.RichTextSegmentsFromText(fmt.Sprint(r.used), widget.RichTextStyle{
						Alignment: fyne.TextAlignTrailing,
						ColorName: r.usedColor(),
						TextStyle: fyne.TextStyle{Bold: r.isTotal},
					})
				case overviewColFree:
					return xwidget.RichTextSegmentsFromText(fmt.Sprint(r.free), widget.RichTextStyle{
						Alignment: fyne.TextAlignTrailing,
						ColorName: r.freeColor(),
						TextStyle: fyne.TextStyle{Bold: r.isTotal},
					})
				case overviewColTotal:
					return xwidget.RichTextSegmentsFromText(fmt.Sprint(r.total), widget.RichTextStyle{
						Alignment: fyne.TextAlignTrailing,
						TextStyle: fyne.TextStyle{Bold: r.isTotal},
					})
				}
				return xwidget.RichTextSegmentsFromText("?")
			},
		)
	}

	a.selectCorporation = kxwidget.NewFilterChipSelect("Corporation", []string{}, func(string) {
		a.filterRowsAsync(-1)
	})
	a.selectFreeSlots = kxwidget.NewFilterChipSelect("Free slots", []string{
		slotsFreeSome,
		slotsFreeNone,
	}, func(string) {
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
		switch arg.Section {
		case app.SectionCharacterContracts, app.SectionCharacterSkills:
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

func (a *Slots) CreateRenderer() fyne.WidgetRenderer {
	filter := container.NewHBox(a.selectCorporation, a.selectFreeSlots, a.selectTag)
	if a.u.IsMobile() {
		filter.Add(a.sortChip)
	}
	c := container.NewBorder(container.NewHScroll(filter), a.footer, nil, nil, a.body)
	return widget.NewSimpleRenderer(c)
}

func (a *Slots) makeDataTable(headers xwidget.DataColumns[slotRow], makeCell func(col int, r slotRow) []widget.RichTextSegment) *widget.Table {
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

func (a *Slots) filterRowsAsync(sortCol int) {
	totalRows := len(a.rows)
	rows := slices.Clone(a.rows)
	corporation := a.selectCorporation.Selected
	freeSlots := a.selectFreeSlots.Selected
	tag := a.selectTag.Selected
	sortCol, dir, doSort := a.columnSorter.CalcSort(sortCol)

	go func() {
		rows := slices.Clone(rows)
		// filter
		if freeSlots != "" {
			rows = slices.DeleteFunc(rows, func(r slotRow) bool {
				switch freeSlots {
				case slotsFreeSome:
					return r.free == 0
				case slotsFreeNone:
					return r.free > 0
				}
				return true
			})
		}
		if x := corporation; x != "" {
			rows = slices.DeleteFunc(rows, func(r slotRow) bool {
				return r.corporationName != x
			})
		}
		if tag != "" {
			rows = slices.DeleteFunc(rows, func(r slotRow) bool {
				return !r.tags.Contains(tag)
			})
		}
		a.columnSorter.SortRows(rows, sortCol, dir, doSort)

		footer := fmt.Sprintf("Showing %d / %d characters", len(rows), totalRows)
		// add totals
		var used, free, total int
		for _, r := range rows {
			used += r.used
			free += r.free
			total += r.total
		}
		rows = append(rows, slotRow{
			used:          used,
			characterName: "Total",
			free:          free,
			isTotal:       true,
			total:         total,
		})

		corporationOptions := xslices.Map(rows, func(r slotRow) string {
			return r.corporationName
		})
		tagOptions := slices.Sorted(set.Union(xslices.Map(rows, func(r slotRow) set.Set[string] {
			return r.tags
		})...).All())

		fyne.Do(func() {
			a.footer.Text = footer
			a.footer.Importance = widget.MediumImportance
			a.footer.Refresh()
			a.selectCorporation.SetOptions(corporationOptions)
			a.selectTag.SetOptions(tagOptions)
			a.rowsFiltered = rows
			a.body.Refresh()
		})
	}()
}

func (a *Slots) update(ctx context.Context) {
	rows, err := a.fetchData(ctx)
	if err != nil {
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
		a.filterRowsAsync(-1)
	})
}

func (a *Slots) fetchData(ctx context.Context) ([]slotRow, error) {
	var f func(context.Context) ([]app.CharacterContractSlots, error)
	if a.corporationSlots {
		f = a.u.Character().ListAllCharacterContractSlotsCorporation
	} else {
		f = a.u.Character().ListAllCharacterContractSlotsPersonal
	}
	oo, err := f(ctx)
	if err != nil {
		return nil, err
	}
	var rows []slotRow
	for _, o := range oo {
		r := slotRow{
			characterID:     o.CharacterID,
			characterName:   o.CharacterName,
			corporationName: o.CorporationName,
			used:            o.Used,
			free:            o.Free,
			total:           o.Total,
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
