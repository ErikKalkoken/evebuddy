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

// Navigator is a container that allows the user to navigate to a new page
// and return back to the previous one.
// Navigation between pages will replace the shown content of the container.
// Supports nested navigation.
type Navigator struct {
	widget.BaseWidget

	NavBar *NavBar // Current navbar. Required for hide feature.

	stack *fyne.Container // stack of pages. First object is the root page.
}

// NewNavigatorWithAppBar return a new Navigator and defines the root page.
func NewNavigatorWithAppBar(ab *AppBar) *Navigator {
	n := &Navigator{
		stack: container.NewStack(),
	}
	n.ExtendBaseWidget(n)
	if ab != nil {
		n.stack.Add(ab)
	}
	return n
}

// NewNavigator return a new Navigator.
func NewNavigator() *Navigator {
	return NewNavigatorWithAppBar(nil)
}

// Push adds a new page and shows it.
func (n *Navigator) Push(ab *AppBar) {
	n.push(ab, false)
}

func (n *Navigator) Set(ab *AppBar) {
	n.stack.RemoveAll()
	n.stack.Add(ab)
	n.Refresh()
}

func (n *Navigator) IsEmpty() bool {
	return len(n.stack.Objects) == 0
}

// PushHideNavBar adds a new page and shows it while hiding the navbar.
//
// Will panic if pushed under an existing page with an already deactivated nav bar.
func (n *Navigator) PushHideNavBar(ab *AppBar) {
	n.push(ab, true)
}

func (n *Navigator) push(ab *AppBar, hideNavBar bool) {
	ab.Navigator = n
	if hideNavBar && n.NavBar != nil {
		n.NavBar.HideBar()
	}
	previous := n.topPage()
	n.stack.Add(ab)
	previous.Hide()
}

// Pop removes the current page and shows the previous page.
// Does nothing when the root page is shown.
func (n *Navigator) Pop() {
	if len(n.stack.Objects) < 2 {
		return
	}
	n.stack.Remove(n.topPage())
	n.topPage().Show()
	if n.NavBar != nil {
		n.NavBar.ShowBar()
	}
}

// PopAll removes all additional pages and shows the root page.
// Does nothing when the root page is shown.
func (n *Navigator) PopAll() {
	if len(n.stack.Objects) == 0 {
		return
	}
	for len(n.stack.Objects) > 1 {
		n.stack.Remove(n.topPage())
	}
	n.topPage().Show()
	if n.NavBar != nil {
		n.NavBar.ShowBar()
	}
}

func (n *Navigator) topPage() fyne.CanvasObject {
	return n.stack.Objects[len(n.stack.Objects)-1]
}

func (n *Navigator) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(n.stack)
}

// An AppBar displays navigation, actions, and text at the top of a screen.
//
// AppBars can be used for both mobile and desktop UIs.
type AppBar struct {
	widget.BaseWidget

	Navigator *Navigator

	bg       *canvas.Rectangle
	body     fyne.CanvasObject
	isMobile bool
	items    []*IconButton
	title    *Label
}

// NewAppBar returns a new AppBar. The toolbar items are optional.
func NewAppBar(title string, body fyne.CanvasObject, items ...*IconButton) *AppBar {
	w := &AppBar{
		body:     body,
		isMobile: fyne.CurrentDevice().IsMobile(),
		items:    items,
	}
	w.ExtendBaseWidget(w)
	w.bg = canvas.NewRectangle(theme.Color(colorBarBackground))
	w.bg.SetMinSize(fyne.NewSize(10, 45))
	if !w.isMobile {
		w.bg.Hide()
	}
	w.title = NewLabelWithSize(title, theme.SizeNameSubHeadingText)
	w.title.Truncation = fyne.TextTruncateEllipsis
	return w
}

func (w *AppBar) SetTitle(text string) {
	w.title.SetText(text)
}

func (w *AppBar) Title() string {
	return w.title.Text
}

func (w *AppBar) Refresh() {
	if w.isMobile {
		th := w.Theme()
		v := fyne.CurrentApp().Settings().ThemeVariant()
		w.bg.FillColor = th.Color(colorBarBackground, v)
		w.bg.Refresh()
	}
	w.title.Refresh()
	w.body.Refresh()
	w.BaseWidget.Refresh()
}

func (w *AppBar) CreateRenderer() fyne.WidgetRenderer {
	var left, right fyne.CanvasObject
	if w.Navigator != nil {
		left = NewIconButton(theme.NavigateBackIcon(), func() {
			w.Navigator.Pop()
		})
	}
	p := theme.Padding()
	is := theme.IconInlineSize()
	if len(w.items) > 0 {
		icons := container.New(layout.NewCustomPaddedHBoxLayout(is))
		for _, ib := range w.items {
			icons.Add(ib)
		}
		right = container.New(layout.NewCustomPaddedLayout(0, 0, 0, p), icons)
	}
	row := container.NewBorder(nil, nil, left, right, w.title)
	var top, main fyne.CanvasObject
	if w.isMobile {
		top = container.New(
			layout.NewCustomPaddedLayout(-p, -2*p, -p, -p),
			container.NewStack(w.bg, container.NewPadded(row)),
		)
		main = container.New(layout.NewCustomPaddedLayout(p, p, 0, 0), w.body)
	} else {
		top = container.NewVBox(
			row,
			canvas.NewRectangle(color.Transparent),
		)
		main = w.body
	}
	c := container.NewBorder(top, nil, nil, nil, main)
	return widget.NewSimpleRenderer(c)
}
