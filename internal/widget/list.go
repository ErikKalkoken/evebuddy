package widget

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type ListItem struct {
	Action     func()
	Headline   string
	IsDisabled bool
	Leading    fyne.Resource
	Supporting string
	Trailing   fyne.Resource
}

func NewListItemWithIcon(headline string, leading fyne.Resource, action func()) *ListItem {
	return &ListItem{
		Action:   action,
		Leading:  leading,
		Headline: headline,
	}
}

func NewListItem(headline string, action func()) *ListItem {
	return &ListItem{
		Action:   action,
		Headline: headline,
	}
}

func NewListItemWithNavigator(nav *Navigator, ab *AppBar) *ListItem {
	w := NewListItem(ab.Title(), func() {
		nav.Push(ab)
	})
	return w
}

// A listItem is the widget that renders a [ListItem] in a [List].
type listItem struct {
	widget.BaseWidget

	headline        *canvas.Text
	supporting      *canvas.Text
	leading         *canvas.Image
	leadingWrapped  *fyne.Container
	trailingWrapped *fyne.Container
	trailing        *canvas.Image
	IsDisabled      bool
}

func newListItem(leading, trailing fyne.Resource, headline, supporting string) *listItem {
	if leading == nil {
		leading = iconBlankSvg
	}
	i1 := NewImageFromResource(leading, fyne.NewSquareSize(theme.Size(theme.SizeNameInlineIcon)))
	if trailing == nil {
		trailing = iconBlankSvg
	}
	i2 := NewImageFromResource(trailing, fyne.NewSquareSize(theme.Size(theme.SizeNameInlineIcon)))
	h := canvas.NewText(headline, theme.Color(theme.ColorNameForeground))
	h.TextSize = theme.Size(theme.SizeNameSubHeadingText)
	t := canvas.NewText(supporting, theme.Color(theme.ColorNameForeground))
	t.TextSize = theme.Size(theme.SizeNameText)
	w := &listItem{
		headline:   h,
		supporting: t,
		leading:    i1,
		trailing:   i2,
	}
	p := theme.Padding()
	w.leadingWrapped = container.NewCenter(container.New(layout.NewCustomPaddedLayout(0, 0, p, p), w.leading))
	w.trailingWrapped = container.NewCenter(container.New(layout.NewCustomPaddedLayout(0, 0, p, p), w.trailing))
	w.ExtendBaseWidget(w)
	w.updateVisibility(leading, trailing, supporting)
	return w
}

func (w *listItem) Set(it *ListItem) {
	w.leading.Resource = it.Leading
	w.trailing.Resource = it.Trailing
	w.headline.Text = it.Headline
	w.supporting.Text = it.Supporting
	w.IsDisabled = it.IsDisabled
	w.updateVisibility(it.Leading, it.Trailing, it.Supporting)
	w.Refresh()
}

func (w *listItem) updateVisibility(leading fyne.Resource, trailing fyne.Resource, supporting string) {
	if supporting == "" {
		w.supporting.Hide()
	} else {
		w.supporting.Show()
	}
	if leading == nil {
		w.leadingWrapped.Hide()
	} else {
		w.leadingWrapped.Show()
	}
	if trailing == nil {
		w.trailingWrapped.Hide()
	} else {
		w.trailingWrapped.Show()
	}
}

func (w *listItem) Refresh() {
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()

	if w.IsDisabled {
		c := th.Color(theme.ColorNameDisabled, v)
		w.headline.Color = c
		w.supporting.Color = c
	} else {
		c := th.Color(theme.ColorNameForeground, v)
		w.headline.Color = c
		w.supporting.Color = c
	}

	w.leading.Refresh()
	w.trailing.Refresh()
	w.headline.Refresh()
	w.supporting.Refresh()
	w.BaseWidget.Refresh()
}

func (w *listItem) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewPadded(container.NewBorder(
		nil,
		nil,
		w.leadingWrapped,
		w.trailingWrapped,
		container.NewVBox(
			layout.NewSpacer(),
			container.New(
				layout.NewCustomPaddedVBoxLayout(0),
				w.headline,
				w.supporting,
			),
			layout.NewSpacer(),
		)),
	)
	return widget.NewSimpleRenderer(c)
}

// List is a widget that renders a list of selectable items.
type List struct {
	widget.BaseWidget

	SelectDelay time.Duration

	title string
	items []*ListItem
}

func NewNavList(items ...*ListItem) *List {
	return NewNavListWithTitle("", items...)
}

func NewNavListWithTitle(title string, items ...*ListItem) *List {
	w := &List{
		items:       items,
		SelectDelay: 500 * time.Millisecond,
		title:       title,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *List) CreateRenderer() fyne.WidgetRenderer {
	list := widget.NewList(
		func() int {
			return len(w.items)
		},
		func() fyne.CanvasObject {
			return newListItem(iconBlankSvg, iconBlankSvg, "Headline", "Supporting")
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(w.items) {
				return
			}
			co.(*listItem).Set(w.items[id])
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		if id >= len(w.items) {
			list.UnselectAll()
			return
		}
		it := w.items[id]
		a := it.Action
		if a == nil || it.IsDisabled {
			list.UnselectAll()
			return
		}
		a()
		go func() {
			time.Sleep(w.SelectDelay)
			list.UnselectAll()
		}()
	}
	list.HideSeparators = true
	l := widget.NewLabel(w.title)
	l.TextStyle.Bold = true
	if w.title == "" {
		l.Hide()
	}
	c := container.NewBorder(l, nil, nil, nil, list)
	return widget.NewSimpleRenderer(c)
}
