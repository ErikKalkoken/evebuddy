package widget

import (
	"image/color"

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
	title      string
	icon       fyne.Resource
	content    fyne.CanvasObject
	variant    navItemVariant
	stackIndex int
	isSelected bool
}

func NewNavPage(title string, icon fyne.Resource, content fyne.CanvasObject) *navItem {
	p := &navItem{
		title:      title,
		icon:       icon,
		content:    content,
		variant:    navPage,
		stackIndex: indexUndefined,
	}
	return p
}

func NewNavSectionLabel(title string) *navItem {
	p := &navItem{
		title:      title,
		variant:    navSectionLabel,
		stackIndex: indexUndefined,
	}
	return p
}

// Navigation drawers let people switch between UI views on larger devices.
type NavDrawer struct {
	widget.BaseWidget

	current int
	items   []*navItem
	stack   *fyne.Container
}

func NewNavDrawer(pages ...*navItem) *NavDrawer {
	w := &NavDrawer{
		current: indexUndefined,
		stack:   container.NewStack(),
	}
	w.ExtendBaseWidget(w)
	for _, p := range pages {
		w.AddPage(p)
	}
	if len(pages) > 0 {
		w.Select(0)
	}
	return w
}

func (w *NavDrawer) AddPage(p *navItem) {
	if p.variant == navPage {
		p.content.Hide()
		w.stack.Add(p.content)
		p.stackIndex = len(w.stack.Objects) - 1
	}
	w.items = append(w.items, p)
}

func (w *NavDrawer) Select(id int) {
	if id >= len(w.items) {
		return
	}
	it := w.items[id]
	if it.stackIndex == indexUndefined {
		return
	}
	if w.current != indexUndefined {
		w.stack.Objects[w.current].Hide()
	}
	w.stack.Objects[it.stackIndex].Show()
	w.current = it.stackIndex
}

func (w *NavDrawer) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	list := widget.NewList(
		func() int {
			return len(w.items)
		},
		func() fyne.CanvasObject {
			spacer := canvas.NewRectangle(color.Transparent)
			spacer.SetMinSize(fyne.NewSize(p, 1))
			return container.NewBorder(
				widget.NewSeparator(),
				nil,
				nil,
				nil,
				container.NewHBox(spacer, widget.NewIcon(iconBlankSvg), widget.NewLabel("Template Template")),
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
			switch it.variant {
			case navPage:
				title.Text = it.title
				title.TextStyle.Bold = it.isSelected
				title.Refresh()
				icon.SetResource(it.icon)
				icon.Show()
				separator.Hide()
				spacer.Show()
			case navSectionLabel:
				title.SetText(it.title)
				icon.Hide()
				if id != 0 { // dont show for first item
					separator.Show()
				} else {
					separator.Hide()
				}
				spacer.Hide()
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
		w.Select(id)
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
	c := container.NewBorder(
		nil,
		nil,
		container.NewHBox(list, widget.NewSeparator()),
		nil,
		container.New(layout.NewCustomPaddedLayout(-p, -p, -p, -p), w.stack),
	)
	return widget.NewSimpleRenderer(c)
}
