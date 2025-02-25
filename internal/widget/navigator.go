package widget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
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

// NewNavigator return a new Navigator and defines the root page.
func NewNavigator(ab *AppBar) *Navigator {
	n := &Navigator{
		stack: container.NewStack(ab),
	}
	n.ExtendBaseWidget(n)
	return n
}

// Push adds a new page and shows it.
func (n *Navigator) Push(ab *AppBar) {
	n.push(ab, false)
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
	if len(n.stack.Objects) == 1 {
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
