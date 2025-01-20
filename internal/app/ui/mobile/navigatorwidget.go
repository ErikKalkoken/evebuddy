package mobile

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

	mu    sync.Mutex
	stack *fyne.Container // stack of pages. First object is the root page.
}

// NewNavigator return a new Navigator and defines the root page.
func NewNavigator(title string, content fyne.CanvasObject) *Navigator {
	n := &Navigator{
		stack: container.NewStack(NewAppBar(title, content, nil)),
	}
	n.ExtendBaseWidget(n)
	return n
}

// Push adds a new page and shows it.
func (n *Navigator) Push(title string, content fyne.CanvasObject) {
	func() {
		n.mu.Lock()
		defer n.mu.Unlock()
		previous := n.topPage()
		n.stack.Add(NewAppBar(title, content, n))
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
	}()
	n.stack.Refresh()
}

func (n *Navigator) topPage() fyne.CanvasObject {
	return n.stack.Objects[len(n.stack.Objects)-1]
}

func (n *Navigator) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(n.stack)
}
