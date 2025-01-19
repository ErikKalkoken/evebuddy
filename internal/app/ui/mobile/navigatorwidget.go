package mobile

import (
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type page struct {
	title   string
	content fyne.CanvasObject
}

// Navigator is a container that allows the user to navigate to a new page
// and return back to the previous one.
// Navigation between pages will replace the shown content of the container.
// Supports nested navigation.
type Navigator struct {
	widget.BaseWidget

	mu    sync.Mutex
	stack *fyne.Container // stack of pages. First object is the root page.
	title string
}

// NewNavigator return a new Navigator and defines the root page.
func NewNavigator(title string, content fyne.CanvasObject) *Navigator {
	n := &Navigator{
		stack: container.NewStack(content),
		title: title,
	}
	n.ExtendBaseWidget(n)
	return n
}

// Push adds a new page and shows it.
func (n *Navigator) Push(title string, content fyne.CanvasObject) {
	func() {
		n.mu.Lock()
		defer n.mu.Unlock()
		link := widget.NewHyperlink("< "+n.title, nil)
		link.OnTapped = func() {
			n.Pop()
		}
		n.title = title
		previous := n.topPage()
		page := container.NewBorder(link, nil, nil, nil, content)
		n.stack.Add(page)
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
