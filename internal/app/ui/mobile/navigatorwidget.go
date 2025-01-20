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

	mu     sync.Mutex
	stack  *fyne.Container // stack of pages. First object is the root page.
	titles []string
}

// NewNavigator return a new Navigator and defines the root page.
func NewNavigator(title string, content fyne.CanvasObject) *Navigator {
	n := &Navigator{
		stack:  container.NewStack(content),
		titles: make([]string, 0),
	}
	n.titles = append(n.titles, title)
	n.ExtendBaseWidget(n)
	return n
}

// Push adds a new page and shows it.
func (n *Navigator) Push(title string, content fyne.CanvasObject) {
	func() {
		n.mu.Lock()
		defer n.mu.Unlock()
		link := widget.NewHyperlink("< "+n.topTitle(), nil)
		link.OnTapped = func() {
			n.Pop()
		}
		previous := n.topPage()
		page := container.NewBorder(link, nil, nil, nil, content)
		n.stack.Add(page)
		n.titles = append(n.titles, title)
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
		n.titles = n.titles[:len(n.titles)-1]
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
			n.titles = n.titles[:len(n.titles)-2]
		}
	}()
	n.stack.Refresh()
}

func (n *Navigator) topPage() fyne.CanvasObject {
	return n.stack.Objects[len(n.stack.Objects)-1]
}

func (n *Navigator) topTitle() string {
	return n.titles[len(n.titles)-1]
}

func (n *Navigator) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(n.stack)
}
