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

	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

const (
	colonyRouteFilterDirection = "Direction"
	colonyRouteFilterItem      = "Item"
	colonyRouteIncoming        = "Incoming"
	colonyRouteOutgoing        = "Outgoing"
)

// colonyRouteItem is a route from or to an installation.
type colonyRouteItem struct {
	isIncoming bool
	name       string // of the type
	onSelected func() // shows the connected installation, nil when not available
	otherName  string // of the connected installation, empty when unknown
	quantity   int64
	typeID     int64
}

func (r colonyRouteItem) direction() string {
	if r.isIncoming {
		return colonyRouteIncoming
	}
	return colonyRouteOutgoing
}

// colonyRouteList shows the routes of an installation as a filterable and sortable list.
type colonyRouteList struct {
	widget.BaseWidget

	columnSorter *xwidget.ColumnSorter[colonyRouteItem]
	empty        *widget.Label
	filterChip   *xwidget.FilterChipCompact
	items        []colonyRouteItem
	list         *widget.List
	rowsFiltered []colonyRouteItem
	sortChip     *kxwidget.SortChip
	summary      *widget.Label
	u            baseUI
}

func newColonyRouteList(u baseUI) *colonyRouteList {
	// the other fields break ties, so the order is stable
	columns := xwidget.NewDataColumns([]xwidget.DataColumn[colonyRouteItem]{{
		Label: "Item",
		Sort:  compareColonyRoutes,
	}, {
		Label: "Quantity",
		Sort: func(a, b colonyRouteItem) int {
			return cmp.Or(cmp.Compare(a.quantity, b.quantity), compareColonyRoutes(a, b))
		},
	}, {
		Label: "Installation",
		Sort: func(a, b colonyRouteItem) int {
			return cmp.Or(strings.Compare(a.otherName, b.otherName), compareColonyRoutes(a, b))
		},
	}})
	a := &colonyRouteList{
		columnSorter: xwidget.NewColumnSorter(columns, "Item", xwidget.SortAsc),
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

func (a *colonyRouteList) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(
		container.NewBorder(nil, nil, nil, container.NewHBox(a.filterChip, a.sortChip), a.summary),
		nil,
		nil,
		nil,
		container.NewStack(a.list, container.NewVBox(a.empty)),
	)
	return widget.NewSimpleRenderer(c)
}

func (a *colonyRouteList) makeList() *widget.List {
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
			co.(*colonyTypeItemWidget).set(r.typeID, r.name, colonyRouteItemDetails(r), widget.MediumImportance)
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		defer l.UnselectAll()
		if id >= len(a.rowsFiltered) {
			return
		}
		if f := a.rowsFiltered[id].onSelected; f != nil {
			f()
		}
	}
	return l
}

// set replaces the items. Must be called on the main thread.
func (a *colonyRouteList) set(items []colonyRouteItem) {
	a.items = items
	directions := xslices.Map(items, func(r colonyRouteItem) string {
		return r.direction()
	})
	names := xslices.Map(items, func(r colonyRouteItem) string {
		return r.name
	})
	a.filterChip.SetOptions(
		xwidget.NewFilterOptionMultiChoice(colonyRouteFilterDirection, directions),
		xwidget.NewFilterOptionMultiChoice(colonyRouteFilterItem, names),
	)
	a.filterRows()
}

// filterRows applies the filter and sorting to the items. Must be called on the main thread.
func (a *colonyRouteList) filterRows() {
	rows := slices.Clone(a.items)
	filter := a.filterChip.Selected()
	if x := filter[colonyRouteFilterDirection]; x != "" {
		rows = slices.DeleteFunc(rows, func(r colonyRouteItem) bool {
			return r.direction() != x
		})
	}
	if x := filter[colonyRouteFilterItem]; x != "" {
		rows = slices.DeleteFunc(rows, func(r colonyRouteItem) bool {
			return r.name != x
		})
	}
	sortCol, dir, doSort := a.columnSorter.CalcSort("")
	a.columnSorter.SortRows(rows, sortCol, dir, doSort)

	noun := "routes"
	if len(rows) == 1 {
		noun = "route"
	}
	a.summary.SetText(fmt.Sprintf("%s %s", ihumanize.Comma(len(rows)), noun))

	switch {
	case len(a.items) == 0:
		a.empty.SetText("No routes")
		a.empty.Show()
	case len(rows) == 0:
		a.empty.SetText("No matching routes")
		a.empty.Show()
	default:
		a.empty.Hide()
	}
	a.rowsFiltered = rows
	a.list.Refresh()
}

// compareColonyRoutes orders routes by item, incoming first, larger quantity first and installation.
func compareColonyRoutes(a, b colonyRouteItem) int {
	return cmp.Or(
		strings.Compare(a.name, b.name),
		strings.Compare(a.direction(), b.direction()), // incoming first
		cmp.Compare(b.quantity, a.quantity),
		strings.Compare(a.otherName, b.otherName),
	)
}

// colonyRouteItemDetails returns the second line of a route.
func colonyRouteItemDetails(r colonyRouteItem) string {
	direction := "to"
	if r.isIncoming {
		direction = "from"
	}
	other := cmp.Or(r.otherName, "unknown installation")
	return fmt.Sprintf("%s units %s %s", ihumanize.Comma(r.quantity), direction, other)
}
