package xwidget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/stack"
)

// Navigator is a container that allows the user to navigate to a new page
// and return back to the previous one.
// Supports nested navigation.
type Navigator struct {
	widget.BaseWidget

	NavBar *NavBar // Current navbar. Required for hide feature.

	pages      *fyne.Container // stack of pages. First object is the root page.
	hideNavBar stack.Stack[bool]
}

// NewNavigator returns a new Navigator and defines the root page.
func NewNavigator(ab *AppBar) *Navigator {
	if ab == nil {
		panic("must provide an AppBar")
	}
	n := &Navigator{
		pages: container.NewStack(),
	}
	n.ExtendBaseWidget(n)
	n.pages.Add(ab)
	n.hideNavBar.Push(false)
	return n
}

// Push adds a new page and shows it.
func (n *Navigator) Push(ab *AppBar) {
	n.push(ab, false)
}

func (n *Navigator) Set(ab *AppBar) {
	n.pages.RemoveAll()
	n.pages.Add(ab)
	n.hideNavBar.Clear()
	n.hideNavBar.Push(false)
	n.Refresh()
}

func (n *Navigator) IsEmpty() bool {
	return len(n.pages.Objects) == 0
}

// PushAndHideNavBar adds a new page and shows it while hiding the navbar.
func (n *Navigator) PushAndHideNavBar(ab *AppBar) {
	n.push(ab, true)
}

func (n *Navigator) push(ab *AppBar, hideNavBar bool) {
	ab.Navigator = n
	if hideNavBar && n.NavBar != nil {
		n.NavBar.HideBar()
	}
	previous := n.Current()
	n.pages.Add(ab)
	n.hideNavBar.Push(hideNavBar)
	previous.Hide()
}

// Pop tries remove the current page and show the previous page.
//
// Returns the removed page or nil.
// It does not remove the root page.
func (n *Navigator) Pop() fyne.CanvasObject {
	if len(n.pages.Objects) < 2 {
		return nil
	}
	page := n.Current()
	n.pages.Remove(page)
	n.hideNavBar.Pop()
	n.Current().Show()
	n.showNavBarWhenRequired()
	return page
}

// PopAll removes all non-root pages and shows the root page.
// Does nothing when the root page is shown.
func (n *Navigator) PopAll() {
	if len(n.pages.Objects) == 0 {
		return
	}
	for len(n.pages.Objects) > 1 {
		n.pages.Remove(n.Current())
		n.hideNavBar.Pop()
	}
	n.Current().Show()
	n.showNavBarWhenRequired()
}

func (n *Navigator) showNavBarWhenRequired() {
	if n.NavBar == nil {
		return
	}
	if v, ok := n.hideNavBar.Peek(); ok && !v {
		n.NavBar.ShowBar()
	}
}

// Current returns the current page.
func (n *Navigator) Current() fyne.CanvasObject {
	return n.pages.Objects[len(n.pages.Objects)-1]
}

func (n *Navigator) IsRoot() bool {
	return len(n.pages.Objects) == 1
}

func (n *Navigator) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(n.pages)
}
