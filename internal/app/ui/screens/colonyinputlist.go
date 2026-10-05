package screens

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

// colonyInputItem is an input of a processor.
type colonyInputItem struct {
	demand   int64 // per cycle
	inStock  int64
	isRouted bool
	name     string
	typeID   int64
}

// colonyInputList shows the inputs of a processor.
type colonyInputList struct {
	widget.BaseWidget

	empty *widget.Label
	items []colonyInputItem
	list  *widget.List
	u     baseUI
}

func newColonyInputList(u baseUI) *colonyInputList {
	a := &colonyInputList{
		empty: widget.NewLabel(""),
		u:     u,
	}
	a.ExtendBaseWidget(a)
	a.empty.Alignment = fyne.TextAlignCenter
	a.empty.Hide()
	a.list = widget.NewList(
		func() int {
			return len(a.items)
		},
		func() fyne.CanvasObject {
			return newColonyTypeItemWidget(a.u.EVEImage().InventoryTypeIconAsync)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.items) {
				return
			}
			r := a.items[id]
			importance := widget.MediumImportance
			if !r.isRouted {
				importance = widget.DangerImportance
			}
			co.(*colonyTypeItemWidget).set(r.typeID, r.name, colonyInputItemDetails(r), importance)
		},
	)
	a.list.OnSelected = func(id widget.ListItemID) {
		defer a.list.UnselectAll()
		if id >= len(a.items) {
			return
		}
		a.u.InfoViewer().ShowType(a.items[id].typeID, 0)
	}
	return a
}

func (a *colonyInputList) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewStack(a.list, container.NewVBox(a.empty)))
}

// set replaces the items and shows emptyText when there are none. Must be called on the main thread.
func (a *colonyInputList) set(items []colonyInputItem, emptyText string) {
	items = slices.Clone(items)
	slices.SortFunc(items, func(a, b colonyInputItem) int {
		return cmp.Or(strings.Compare(a.name, b.name), cmp.Compare(b.demand, a.demand))
	})
	a.items = items
	if len(items) == 0 {
		a.empty.SetText(emptyText)
		a.empty.Show()
	} else {
		a.empty.Hide()
	}
	a.list.Refresh()
}

// colonyInputItemDetails returns the second line of an input.
func colonyInputItemDetails(r colonyInputItem) string {
	s := fmt.Sprintf("Needs %s • %s in stock", ihumanize.Comma(r.demand), ihumanize.Comma(r.inStock))
	if !r.isRouted {
		s += " • not routed"
	}
	return s
}

// colonyTypeItemWidget shows a type with its icon, name and a line of details.
type colonyTypeItemWidget struct {
	widget.BaseWidget

	details  *widget.Label
	icon     *canvas.Image
	loadIcon func(id int64, size int, setter func(r fyne.Resource))
	name     *widget.Label
}

func newColonyTypeItemWidget(loadIcon func(id int64, size int, setter func(r fyne.Resource))) *colonyTypeItemWidget {
	w := &colonyTypeItemWidget{
		details:  widget.NewLabel(""),
		icon:     xwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(ui.IconUnitSize)),
		loadIcon: loadIcon,
		name:     widget.NewLabel(""),
	}
	w.details.Truncation = fyne.TextTruncateEllipsis
	w.name.Truncation = fyne.TextTruncateEllipsis
	w.ExtendBaseWidget(w)
	return w
}

func (w *colonyTypeItemWidget) set(typeID int64, name, details string, importance widget.Importance) {
	w.name.SetText(name)
	w.details.Text = details
	w.details.Importance = importance
	w.details.Refresh()
	w.loadIcon(typeID, ui.IconPixelSize, func(res fyne.Resource) {
		w.icon.Resource = res
		w.icon.Refresh()
	})
}

func (w *colonyTypeItemWidget) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	first := container.New(layout.NewCustomPaddedLayout(0, -2*p, 0, 0), w.name)
	second := container.New(layout.NewCustomPaddedLayout(-2*p, 0, 0, 0), w.details)
	main := container.New(layout.NewCustomPaddedVBoxLayout(0), first, second)
	c := container.NewBorder(nil, nil, container.NewPadded(w.icon), nil, main)
	return widget.NewSimpleRenderer(c)
}
