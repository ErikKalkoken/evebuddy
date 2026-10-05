package screens

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

const (
	colonyStorageFilterTier = "Tier"
	colonyStorageTierOther  = "Other" // for types without a known group
)

// colonyStorageItem is a type stored in an installation.
type colonyStorageItem struct {
	group    string // empty when unknown
	name     string
	quantity int64
	typeID   int64
	volume   float64 // total in m3
}

func (r colonyStorageItem) tier() string {
	return cmp.Or(r.group, colonyStorageTierOther)
}

// colonyStorageList shows the contents of an installation as a filterable and sortable list.
type colonyStorageList struct {
	widget.BaseWidget

	columnSorter *xwidget.ColumnSorter[colonyStorageItem]
	empty        *widget.Label
	filterChip   *xwidget.FilterChipCompact
	items        []colonyStorageItem
	list         *widget.List
	rowsFiltered []colonyStorageItem
	sortChip     *kxwidget.SortChip
	summary      *widget.Label
	u            baseUI
}

func newColonyStorageList(u baseUI) *colonyStorageList {
	// names break ties, so the order is stable
	columns := xwidget.NewDataColumns([]xwidget.DataColumn[colonyStorageItem]{{
		Label: "Name",
		Sort: func(a, b colonyStorageItem) int {
			return strings.Compare(a.name, b.name)
		},
	}, {
		Label: "Quantity",
		Sort: func(a, b colonyStorageItem) int {
			return cmp.Or(cmp.Compare(a.quantity, b.quantity), strings.Compare(a.name, b.name))
		},
	}, {
		Label: "Volume",
		Sort: func(a, b colonyStorageItem) int {
			return cmp.Or(cmp.Compare(a.volume, b.volume), strings.Compare(a.name, b.name))
		},
	}})
	a := &colonyStorageList{
		columnSorter: xwidget.NewColumnSorter(columns, "Volume", xwidget.SortDesc),
		empty:        widget.NewLabel(""),
		summary:      ui.NewLabelWithTruncation(""),
		u:            u,
	}
	a.ExtendBaseWidget(a)
	a.empty.Alignment = fyne.TextAlignCenter
	a.empty.Hide()
	a.list = a.makeList()
	a.filterChip = xwidget.NewFilterChipCompact(nil, func(_ map[string]string) {
		a.filterRows()
	})
	a.sortChip = a.columnSorter.NewSortChip(func() {
		a.filterRows()
	})
	return a
}

func (a *colonyStorageList) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(
		container.NewBorder(nil, nil, nil, container.NewHBox(a.filterChip, a.sortChip), a.summary),
		nil,
		nil,
		nil,
		container.NewStack(a.list, container.NewVBox(a.empty)),
	)
	return widget.NewSimpleRenderer(c)
}

func (a *colonyStorageList) makeList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.rowsFiltered)
		},
		func() fyne.CanvasObject {
			return newColonyTypeItemWidget(a.u.EVEImage().InventoryTypeIconAsync)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.rowsFiltered) {
				return
			}
			r := a.rowsFiltered[id]
			co.(*colonyTypeItemWidget).set(r.typeID, r.name, colonyStorageItemDetails(r), widget.MediumImportance)
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		defer l.UnselectAll()
		if id >= len(a.rowsFiltered) {
			return
		}
		a.u.InfoViewer().ShowType(a.rowsFiltered[id].typeID, 0)
	}
	return l
}

// set replaces the items. Must be called on the main thread.
func (a *colonyStorageList) set(items []colonyStorageItem) {
	a.items = items
	tiers := xslices.Map(items, func(r colonyStorageItem) string {
		return r.tier()
	})
	a.filterChip.SetOptions(xwidget.NewFilterOptionMultiChoice(colonyStorageFilterTier, tiers))
	a.filterRows()
}

// filterRows applies the filter and sorting to the items. Must be called on the main thread.
func (a *colonyStorageList) filterRows() {
	rows := slices.Clone(a.items)
	if x := a.filterChip.Selected()[colonyStorageFilterTier]; x != "" {
		rows = slices.DeleteFunc(rows, func(r colonyStorageItem) bool {
			return r.tier() != x
		})
	}
	sortCol, dir, doSort := a.columnSorter.CalcSort("")
	a.columnSorter.SortRows(rows, sortCol, dir, doSort)

	var volume float64
	for _, r := range rows {
		volume += r.volume
	}
	noun := "items"
	if len(rows) == 1 {
		noun = "item"
	}
	a.summary.SetText(fmt.Sprintf("%s %s • %s m3", ihumanize.Comma(len(rows)), noun, humanize.FormatFloat("#,###.##", volume)))

	switch {
	case len(a.items) == 0:
		a.empty.SetText("Empty")
		a.empty.Show()
	case len(rows) == 0:
		a.empty.SetText("No matching items")
		a.empty.Show()
	default:
		a.empty.Hide()
	}
	a.rowsFiltered = rows
	a.list.Refresh()
}

// colonyStorageItemDetails returns the second line of a stored type.
func colonyStorageItemDetails(r colonyStorageItem) string {
	return fmt.Sprintf("%s units • %s m3", ihumanize.Comma(r.quantity), humanize.FormatFloat("#,###.##", r.volume))
}
