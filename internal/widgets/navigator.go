package widgets

import (
	"sync"

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

	navBar *NavBar

	mu    sync.Mutex
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
	n.push(ab, nil)
}

// PushNoNavBar adds a new page and shows it without a navbar.
//
// Will panic if pushed under an existing page with an already deactivated nav bar.
func (n *Navigator) PushNoNavBar(ab *AppBar, nb *NavBar) {
	n.push(ab, nb)
}

// Push adds a new page and shows it.
func (n *Navigator) push(ab *AppBar, nb *NavBar) {
	ab.Navigator = n
	func() {
		n.mu.Lock()
		defer n.mu.Unlock()
		if nb != nil {
			if n.navBar != nil {
				panic("Can not create modal page behind existing modal page")
			}
			n.navBar = nb
			nb.HideBar()
		}
		previous := n.topPage()
		n.stack.Add(ab)
		previous.Hide()
	}()
	n.stack.Refresh()
}

// Pop removes the current page and shows the previous page.
// Does nothing when the root page is shown.
func (n *Navigator) Pop() {
	func() {
		n.mu.Lock()
		defer n.mu.Unlock()
		if len(n.stack.Objects) == 1 {
			return
		}
		n.stack.Remove(n.topPage())
		n.topPage().Show()
		if n.navBar != nil {
			n.navBar.ShowBar()
			n.navBar = nil
		}
	}()
	n.stack.Refresh()
}

// PopAll removes all additional pages and shows the root page.
// Does nothing when the root page is shown.
func (n *Navigator) PopAll() {
	func() {
		n.mu.Lock()
		defer n.mu.Unlock()
		for len(n.stack.Objects) > 1 {
			n.stack.Remove(n.topPage())
		}
		n.topPage().Show()
	}()
	n.stack.Refresh()
}

func (n *Navigator) topPage() fyne.CanvasObject {
	return n.stack.Objects[len(n.stack.Objects)-1]
}

func (n *Navigator) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(n.stack)
}
