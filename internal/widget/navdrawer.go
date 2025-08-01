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
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	indexUndefined = -1
)

type navItemVariant uint

const (
	navUndefined navItemVariant = iota
	navPage
	navSectionLabel
	navSeparator
)

// NavDrawer let people switch between UI views on larger devices.
type NavDrawer struct {
	widget.DisableableWidget

	MinWidth     float32        // minimum width of the navigation area
	OnSelectItem func(*NavItem) // called when an item is selected
	Title        string         // Text of a title. Title will not be rendered if left blank.

	items    []*NavItem
	list     *widget.List
	pages    *fyne.Container
	selected int
	title    *widget.Label
}

func NewNavDrawer(items ...*NavItem) *NavDrawer {
	label := widget.NewLabel("")
	label.SizeName = theme.SizeNameSubHeadingText
	w := &NavDrawer{
		pages:    container.NewStack(),
		selected: indexUndefined,
		title:    label,
	}
	label.Hide()
	w.ExtendBaseWidget(w)
	w.list = w.makeList()
	for _, p := range items {
		if p.variant == navPage {
			p.content.Hide()
			w.pages.Add(p.content)
			p.stackIndex = len(w.pages.Objects) - 1
		}
		w.items = append(w.items, p)
	}
	for id, it := range slices.Backward(w.items) {
		if it.variant == navPage {
			w.SelectIndex(id)
		}
	}
	return w
}

func (w *NavDrawer) ScrollToTop() {
	w.list.ScrollToTop()
}

func (w *NavDrawer) makeList() *widget.List {
	p := w.Theme().Size(theme.SizeNamePadding)
	var list *widget.List
	list = widget.NewList(
		func() int {
			return len(w.items)
		},
		func() fyne.CanvasObject {
			spacer := canvas.NewRectangle(color.Transparent)
			spacer.SetMinSize(fyne.NewSize(p, 1))
			icon := widget.NewIcon(iconBlankSvg)
			title := widget.NewLabel("Template")
			title.Truncation = fyne.TextTruncateEllipsis
			badge := widget.NewLabel("999+")
			return container.NewStack(
				container.New(layout.NewCustomPaddedLayout(0, 0, p, p),
					container.NewBorder(nil, nil, container.NewHBox(spacer, icon), badge, title),
				),
				container.New(layout.NewCustomPaddedLayout(0, 0, 2*p, 2*p),
					widget.NewSeparator(),
				),
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(w.items) {
				return
			}
			it := w.items[id]
			stack := co.(*fyne.Container).Objects
			separator := stack[1]
			main := stack[0]
			if it.variant == navSeparator {
				separator.Show()
				main.Hide()
			} else {
				separator.Hide()
				main.Show()
			}
			border := main.(*fyne.Container).Objects[0].(*fyne.Container).Objects
			title := border[0].(*widget.Label)
			box := border[1].(*fyne.Container).Objects
			spacer := box[0]
			icon := box[1].(*widget.Icon)
			badge := border[2].(*widget.Label)
			showIcon := func() {
				var r fyne.Resource
				if it.isDisabled {
					r = theme.NewDisabledResource(it.icon)
				} else {
					r = it.icon
				}
				icon.SetResource(r)
				icon.Show()
				spacer.Show()
			}
			updateBadge := func() {
				if it.badge != "" {
					badge.Text = it.badge
					if it.isDisabled {
						badge.Importance = widget.LowImportance
					} else {
						badge.Importance = widget.MediumImportance
					}
					badge.Refresh()
					badge.Show()
				} else {
					badge.Hide()
				}
			}
			switch it.variant {
			case navPage:
				title.SizeName = theme.SizeNameText
				title.Text = it.text
				title.TextStyle.Bold = it.isSelected
				if it.isDisabled {
					title.Importance = widget.LowImportance
				} else {
					title.Importance = widget.MediumImportance
				}
				title.Refresh()
				showIcon()
				updateBadge()
			case navSectionLabel:
				title.SizeName = theme.SizeNameScrollBar
				toUpper := cases.Upper(language.English)
				title.Text = toUpper.String(it.text)
				if it.isDisabled {
					title.Importance = widget.LowImportance
				} else {
					title.Importance = widget.MediumImportance
				}
				title.Refresh()
				icon.Hide()
				spacer.Hide()
				updateBadge()
			}
			list.SetItemHeight(id, co.(*fyne.Container).MinSize().Height)
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		if id >= len(w.items) {
			list.UnselectAll()
			return
		}
		it := w.items[id]
		if it.isDisabled || it.variant == navSeparator || it.variant == navSectionLabel {
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

func (w *NavDrawer) Disable() {
	if w.Disabled() {
		return
	}
	w.Select(w.items[0])
	w.ScrollToTop()
	for _, it := range w.items {
		it.isDisabled = true
	}
	w.DisableableWidget.Disable()
}

func (w *NavDrawer) Enable() {
	if !w.Disabled() {
		return
	}
	for _, it := range w.items {
		it.isDisabled = false
	}
	w.DisableableWidget.Enable()
	w.Select(w.items[0])
	w.ScrollToTop()
}

func (w *NavDrawer) findItem(item *NavItem) (int, bool) {
	for i, it := range w.items {
		if it == item {
			return i, true
		}
	}
	return 0, false
}

func (w *NavDrawer) Select(item *NavItem) {
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
	if w.OnSelectItem != nil {
		w.OnSelectItem(it)
	}
}

func (w *NavDrawer) SelectedIndex() int {
	return w.selected
}

func (w *NavDrawer) SetTitle(text string) {
	w.Title = text
	w.Refresh()
}

func (w *NavDrawer) SetItemBadge(item *NavItem, text string) {
	id, ok := w.findItem(item)
	if !ok {
		return
	}
	w.items[id].badge = text
	w.list.RefreshItem(id)
}

func (w *NavDrawer) SetItemText(item *NavItem, text string) {
	id, ok := w.findItem(item)
	if !ok {
		return
	}
	w.items[id].text = text
	w.list.RefreshItem(id)
}

func (w *NavDrawer) Refresh() {
	w.updateTitle()
	w.list.Refresh()
}

func (w *NavDrawer) updateTitle() {
	if w.Title != "" {
		w.title.SetText(w.Title)
		w.title.Show()
	} else {
		w.title.Hide()
	}
}

func (w *NavDrawer) CreateRenderer() fyne.WidgetRenderer {
	w.updateTitle()
	p := theme.Padding()
	spacer := canvas.NewRectangle(color.Transparent)
	spacer.SetMinSize(fyne.NewSize(w.MinWidth, 1))
	c := container.New(layout.NewCustomPaddedLayout(-p, -p, 0, 0),
		container.NewBorder(
			nil,
			nil,
			container.NewHBox(
				container.NewBorder(
					w.title,
					nil,
					nil,
					nil,
					container.NewStack(spacer, w.list),
				),
				widget.NewSeparator(),
			),
			nil,
			w.pages,
		))
	return widget.NewSimpleRenderer(c)
}

type NavItem struct {
	badge      string
	content    fyne.CanvasObject
	icon       fyne.Resource
	isDisabled bool
	isSelected bool
	stackIndex int
	text       string
	variant    navItemVariant
}

func NewNavPage(text string, icon fyne.Resource, content fyne.CanvasObject) *NavItem {
	it := newNavItem(navPage)
	it.text = text
	it.icon = icon
	it.content = content
	return it
}

func NewNavSectionLabel(text string) *NavItem {
	it := newNavItem(navSectionLabel)
	it.text = text
	return it
}

func NewNavSeparator() *NavItem {
	return newNavItem(navSeparator)
}

func newNavItem(variant navItemVariant) *NavItem {
	it := &NavItem{
		stackIndex: indexUndefined,
		variant:    variant,
	}
	return it
}

func (ni *NavItem) Enable() {
	ni.isDisabled = false
}

func (ni *NavItem) Disable() {
	ni.isDisabled = true
}
