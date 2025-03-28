package widget

import (
	"image/color"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	indexUndefined = -1
)

type navItemVariant uint

const (
	navPage navItemVariant = iota
	navSectionLabel
)

type navItem struct {
	badge      string
	content    fyne.CanvasObject
	icon       fyne.Resource
	isSelected bool
	stackIndex int
	text       string
	variant    navItemVariant
}

func newNavItem(text string, icon fyne.Resource, content fyne.CanvasObject, variant navItemVariant) *navItem {
	it := &navItem{
		content:    content,
		icon:       icon,
		stackIndex: indexUndefined,
		text:       text,
		variant:    variant,
	}
	return it
}

func NewNavPage(text string, icon fyne.Resource, content fyne.CanvasObject) *navItem {
	return newNavItem(text, icon, content, navPage)
}

func NewNavSectionLabel(text string) *navItem {
	return newNavItem(text, nil, nil, navSectionLabel)
}

// Navigation drawers let people switch between UI views on larger devices.
type NavDrawer struct {
	widget.BaseWidget

	items    []*navItem
	list     *widget.List
	selected int
	pages    *fyne.Container
}

func NewNavDrawer(items ...*navItem) *NavDrawer {
	w := &NavDrawer{
		pages:    container.NewStack(),
		selected: indexUndefined,
	}
	w.ExtendBaseWidget(w)
	w.list = w.makeList()
	for _, p := range items {
		w.AddPage(p)
	}
	for id, it := range slices.Backward(w.items) {
		if it.variant == navPage {
			w.SelectIndex(id)
		}
	}
	return w
}

func (w *NavDrawer) AddPage(p *navItem) {
	if p.variant == navPage {
		p.content.Hide()
		w.pages.Add(p.content)
		p.stackIndex = len(w.pages.Objects) - 1
	}
	w.items = append(w.items, p)
}

func (w *NavDrawer) Select(item *navItem) {
	id, ok := w.findItem(item)
	if !ok {
		return
	}
	w.SelectIndex(id)
}

func (w *NavDrawer) SelectIndex(id int) {
	if id >= len(w.items) {
		return
	}
	w.list.Select(id)
}

func (w *NavDrawer) selectIndex(id int) {
	it := w.items[id]
	if it.stackIndex == indexUndefined {
		return
	}
	if w.selected != indexUndefined {
		si := w.items[w.selected]
		w.pages.Objects[si.stackIndex].Hide()
	}
	w.pages.Objects[it.stackIndex].Show()
	w.selected = id
}

func (w *NavDrawer) SelectedIndex() int {
	return w.selected
}

func (w *NavDrawer) SetItemBadge(item *navItem, text string) {
	id, ok := w.findItem(item)
	if !ok {
		return
	}
	w.items[id].badge = text
	w.list.RefreshItem(id)
}

func (w *NavDrawer) makeList() *widget.List {
	p := w.Theme().Size(theme.SizeNamePadding)
	list := widget.NewList(
		func() int {
			return len(w.items)
		},
		func() fyne.CanvasObject {
			spacer := canvas.NewRectangle(color.Transparent)
			spacer.SetMinSize(fyne.NewSize(p, 1))
			icon := widget.NewIcon(iconBlankSvg)
			text := widget.NewLabel("Template Template") // TODO: Make width a configuration
			badge := widget.NewLabel("Template")
			return container.NewBorder(
				widget.NewSeparator(),
				nil,
				nil,
				nil,
				container.NewHBox(spacer, icon, text, layout.NewSpacer(), badge),
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(w.items) {
				return
			}
			it := w.items[id]
			border := co.(*fyne.Container).Objects
			main := border[0].(*fyne.Container)
			separator := border[1]
			box := main.Objects
			spacer := box[0]
			icon := box[1].(*widget.Icon)
			title := box[2].(*widget.Label)
			badge := box[4].(*widget.Label)
			switch it.variant {
			case navPage:
				title.Text = it.text
				title.TextStyle.Bold = it.isSelected
				title.Refresh()
				icon.SetResource(it.icon)
				icon.Show()
				separator.Hide()
				spacer.Show()
				if it.badge != "" {
					badge.SetText(it.badge)
					badge.Show()
				} else {
					badge.Hide()
				}
			case navSectionLabel:
				title.SetText(it.text)
				icon.Hide()
				if id != 0 { // dont show for first item
					separator.Show()
				} else {
					separator.Hide()
				}
				spacer.Hide()
				badge.Hide()
			}
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		if id >= len(w.items) {
			list.UnselectAll()
			return
		}
		it := w.items[id]
		if it.variant != navPage {
			list.UnselectAll()
			return
		}
		w.selectIndex(id)
		for i, p := range w.items {
			if p.isSelected {
				p.isSelected = false
				list.RefreshItem(i)
			}
			if i == id {
				p.isSelected = true
				list.RefreshItem(i)
			}
		}
	}
	list.HideSeparators = true
	return list
}

func (w *NavDrawer) findItem(item *navItem) (int, bool) {
	for i, it := range w.items {
		if it == item {
			return i, true
		}
	}
	return 0, false
}

func (w *NavDrawer) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	c := container.New(layout.NewCustomPaddedLayout(-p, -p, 0, 0),
		container.NewBorder(
			nil,
			nil,
			container.NewHBox(w.list, widget.NewSeparator()),
			nil,
			container.New(layout.NewCustomPaddedLayout(-p, -p, -p, -p), w.pages),
		))
	return widget.NewSimpleRenderer(c)
}
